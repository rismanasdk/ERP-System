package purchasing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/products"

	"github.com/lib/pq"
)

type inventoryRepository interface {
	GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, inventory *inventory.Inventory) (int64, error)
	UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error
	CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *inventory.StockMovement) (int64, error)
}

type productService interface {
	GetByID(ctx context.Context, id int64) (*products.Product, error)
}

type branchService interface {
	EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error
	ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error)
}

type auditService interface {
	RecordWithTx(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) (int64, error)
}

type repositoryInterface interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreatePurchaseWithTx(ctx context.Context, tx *sql.Tx, purchase *Purchase) (int64, error)
	CreatePurchaseItemWithTx(ctx context.Context, tx *sql.Tx, item *PurchaseItem) (int64, error)
	GetSupplierByID(ctx context.Context, id int64) (*Supplier, error)
	GetPurchaseByID(ctx context.Context, id int64) (*Purchase, error)
	ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error)
}

type Service struct {
	repo          repositoryInterface
	inventoryRepo inventoryRepository
	productSvc    productService
	branchSvc     branchService
	authChecker   auth.PermissionChecker
	auditSvc      auditService
}

const (
	PurchaseCreatePermission   = "purchases.create"
	PurchaseStatusCompleted    = "COMPLETED"
	purchaseNumberRetryLimit   = 3
	purchaseNumberRandomPrefix = 8
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrForbidden              = errors.New("permission denied")
	ErrPurchaseNotFound       = errors.New("purchase not found")
	ErrSupplierNotFound       = errors.New("supplier not found")
	ErrSupplierInactive       = errors.New("supplier is inactive")
	ErrProductNotFound        = errors.New("product not found")
	ErrProductInactive        = errors.New("product is inactive")
	ErrDuplicateProduct       = errors.New("duplicate product in purchase items")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type CreatePurchaseItemInput struct {
	ProductID int64   `json:"product_id"`
	Quantity  int64   `json:"quantity"`
	UnitCost  float64 `json:"unit_cost"`
}

type CreatePurchaseInput struct {
	BranchID   int64                     `json:"branch_id"`
	SupplierID int64                     `json:"supplier_id"`
	Notes      *string                   `json:"notes,omitempty"`
	Items      []CreatePurchaseItemInput `json:"items"`
}

func NewService(repo repositoryInterface, inventoryRepo inventoryRepository, productSvc productService, branchSvc branchService, authChecker auth.PermissionChecker, auditSvc auditService) *Service {
	return &Service{
		repo:          repo,
		inventoryRepo: inventoryRepo,
		productSvc:    productSvc,
		branchSvc:     branchSvc,
		authChecker:   authChecker,
		auditSvc:      auditSvc,
	}
}

func (s *Service) CreatePurchase(ctx context.Context, input CreatePurchaseInput) (purchaseID int64, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}

	var allowed bool
	if allowed, err = s.authChecker.HasPermission(ctx, userID, PurchaseCreatePermission); err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}

	if len(input.Items) == 0 {
		return 0, &ValidationError{Field: "items", Message: "must contain at least one item"}
	}

	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, input.BranchID, true); err != nil {
		return 0, err
	}

	supplier, err := s.repo.GetSupplierByID(ctx, input.SupplierID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrSupplierNotFound
		}
		return 0, err
	}
	if !supplier.IsActive {
		return 0, ErrSupplierInactive
	}

	existingProducts := map[int64]struct{}{}
	items := make([]PurchaseItem, len(input.Items))
	totalAmount := float64(0)

	for i, itemInput := range input.Items {
		if itemInput.ProductID == 0 {
			return 0, &ValidationError{Field: "product_id", Message: "must be provided"}
		}
		if _, found := existingProducts[itemInput.ProductID]; found {
			return 0, ErrDuplicateProduct
		}
		existingProducts[itemInput.ProductID] = struct{}{}

		if itemInput.Quantity <= 0 {
			return 0, &ValidationError{Field: "quantity", Message: "must be greater than zero"}
		}
		if itemInput.UnitCost < 0 {
			return 0, &ValidationError{Field: "unit_cost", Message: "must be greater than or equal to 0"}
		}

		product, err := s.productSvc.GetByID(ctx, itemInput.ProductID)
		if err != nil {
			if errors.Is(err, products.ErrProductNotFound) {
				return 0, ErrProductNotFound
			}
			return 0, err
		}
		if !product.IsActive {
			return 0, ErrProductInactive
		}

		subtotal := float64(itemInput.Quantity) * itemInput.UnitCost
		totalAmount += subtotal
		items[i] = PurchaseItem{
			ProductID: itemInput.ProductID,
			Quantity:  itemInput.Quantity,
			UnitCost:  itemInput.UnitCost,
			Subtotal:  subtotal,
		}
	}

	purchaseNumber := generatePurchaseNumber()

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	purchase := &Purchase{
		BranchID:       input.BranchID,
		SupplierID:     input.SupplierID,
		PurchaseNumber: purchaseNumber,
		Status:         PurchaseStatusCompleted,
		TotalAmount:    totalAmount,
		Notes:          input.Notes,
		CreatedBy:      userID,
	}

	purchaseID, err = s.insertPurchaseWithRetry(ctx, tx, purchase)
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		item.PurchaseID = purchaseID
		if _, err = s.repo.CreatePurchaseItemWithTx(ctx, tx, &item); err != nil {
			return 0, err
		}

		inventoryRow, invErr := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, input.BranchID)
		if invErr != nil {
			if errors.Is(invErr, sql.ErrNoRows) {
				_, err = s.inventoryRepo.CreateWithTx(ctx, tx, &inventory.Inventory{ProductID: item.ProductID, BranchID: input.BranchID, Quantity: item.Quantity})
				if err != nil {
					return 0, err
				}
			} else {
				return 0, invErr
			}
		}
		if inventoryRow != nil {
			newQuantity := inventoryRow.Quantity + item.Quantity
			if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inventoryRow.ID, newQuantity); err != nil {
				return 0, err
			}
		}

		movement := &inventory.StockMovement{
			ProductID:     item.ProductID,
			BranchID:      input.BranchID,
			MovementType:  inventory.MovementTypeIN,
			QuantityDelta: item.Quantity,
			ReferenceType: inventory.PtrString("purchase"),
			ReferenceID:   inventory.PtrInt64(purchaseID),
			ActorUserID:   actorUserIDFromContext(ctx),
			Metadata: map[string]any{
				"purchase_number": purchase.PurchaseNumber,
				"branch_id":       input.BranchID,
				"product_id":      item.ProductID,
				"supplier_id":     input.SupplierID,
			},
		}
		if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		auditMetadata := map[string]any{
			"purchase_id":     purchaseID,
			"purchase_number": purchase.PurchaseNumber,
			"branch_id":       input.BranchID,
			"supplier_id":     input.SupplierID,
			"total_amount":    totalAmount,
			"item_count":      len(items),
		}
		resourceID := fmt.Sprintf("%d", purchaseID)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "purchase.create",
			Resource:    "purchase",
			ResourceID:  &resourceID,
			Metadata:    auditMetadata,
		}); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return purchaseID, nil
}

func (s *Service) insertPurchaseWithRetry(ctx context.Context, tx *sql.Tx, purchase *Purchase) (int64, error) {
	for attempt := 0; attempt < purchaseNumberRetryLimit; attempt++ {
		if attempt > 0 {
			purchase.PurchaseNumber = generatePurchaseNumber()
		}
		id, err := s.repo.CreatePurchaseWithTx(ctx, tx, purchase)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				continue
			}
			return 0, err
		}
		return id, nil
	}
	return 0, errors.New("failed to generate unique purchase number")
}

func (s *Service) GetPurchase(ctx context.Context, id int64) (*Purchase, error) {
	purchase, err := s.repo.GetPurchaseByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPurchaseNotFound
		}
		return nil, err
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, purchase.BranchID, true); err != nil {
		return nil, err
	}
	return purchase, nil
}

func (s *Service) ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}

	if filter.BranchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *filter.BranchID, true); err != nil {
			return nil, err
		}
		return s.repo.ListPurchases(ctx, filter)
	}

	branches, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return []Purchase{}, nil
	}

	var purchases []Purchase
	for _, b := range branches {
		filterCopy := filter
		filterCopy.BranchID = &b.ID
		items, err := s.repo.ListPurchases(ctx, filterCopy)
		if err != nil {
			return nil, err
		}
		purchases = append(purchases, items...)
	}
	return purchases, nil
}

func generatePurchaseNumber() string {
	buf := make([]byte, purchaseNumberRandomPrefix)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("PO-%d", time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("PO-%s-%s", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(buf))
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
