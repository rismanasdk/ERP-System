package sales

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
)

type inventoryRepository interface {
	GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error)
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
	CreateSaleWithTx(ctx context.Context, tx *sql.Tx, sale *Sale) (int64, error)
	CreateSaleItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleItem) (int64, error)
	GetSaleByID(ctx context.Context, id int64) (*Sale, error)
	GetSaleByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Sale, error)
	GetSaleByNumber(ctx context.Context, number string) (*Sale, error)
	ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error)
	ListSaleItemsBySaleID(ctx context.Context, saleID int64) ([]SaleItem, error)
	ListSaleItemsBySaleIDWithTx(ctx context.Context, tx *sql.Tx, saleID int64) ([]SaleItem, error)
	UpdateSaleStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error
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
	SaleCreatePermission   = "sales.create"
	SaleReadPermission     = "sales.read"
	SaleCompletePermission = "sales.complete"
	SaleCancelPermission   = "sales.cancel"
	SaleStatusDraft        = "DRAFT"
	SaleStatusCompleted    = "COMPLETED"
	SaleStatusCancelled    = "CANCELLED"
	saleNumberRetryLimit   = 3
	saleNumberRandomPrefix = 8
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrForbidden              = errors.New("permission denied")
	ErrSaleNotFound           = errors.New("sale not found")
	ErrProductNotFound        = errors.New("product not found")
	ErrProductInactive        = errors.New("product is inactive")
	ErrSaleHasNoItems         = errors.New("sale has no items")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrInventoryNotFound      = errors.New("inventory not found")
	ErrInvalidSaleTransition  = errors.New("invalid sale status transition")
	ErrSaleAlreadyCompleted   = errors.New("sale already completed")
	ErrSaleAlreadyCancelled   = errors.New("sale already cancelled")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type CreateSaleItemInput struct {
	ProductID int64   `json:"product_id"`
	Quantity  int64   `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type CreateSaleInput struct {
	BranchID int64                 `json:"branch_id"`
	Notes    *string               `json:"notes,omitempty"`
	Items    []CreateSaleItemInput `json:"items"`
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

func (s *Service) CreateSale(ctx context.Context, input CreateSaleInput) (saleID int64, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCreatePermission)
	if err != nil {
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

	existingProducts := map[int64]struct{}{}
	items := make([]SaleItem, 0, len(input.Items))
	totalAmount := float64(0)
	for _, itemInput := range input.Items {
		if itemInput.ProductID == 0 {
			return 0, &ValidationError{Field: "product_id", Message: "must be provided"}
		}
		if _, found := existingProducts[itemInput.ProductID]; found {
			return 0, errors.New("duplicate product in sale items")
		}
		existingProducts[itemInput.ProductID] = struct{}{}
		if itemInput.Quantity <= 0 {
			return 0, &ValidationError{Field: "quantity", Message: "must be greater than zero"}
		}
		if itemInput.UnitPrice < 0 {
			return 0, &ValidationError{Field: "unit_price", Message: "must be greater than or equal to 0"}
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

		subtotal := float64(itemInput.Quantity) * itemInput.UnitPrice
		totalAmount += subtotal
		items = append(items, SaleItem{
			ProductID: itemInput.ProductID,
			Quantity:  itemInput.Quantity,
			UnitPrice: itemInput.UnitPrice,
			Subtotal:  subtotal,
		})
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale := &Sale{
		BranchID:    input.BranchID,
		SaleNumber:  generateSaleNumber(),
		Status:      SaleStatusDraft,
		TotalAmount: totalAmount,
		Notes:       input.Notes,
		CreatedBy:   userID,
	}
	for attempt := 1; attempt < saleNumberRetryLimit; attempt++ {
		if _, err = s.repo.GetSaleByNumber(ctx, sale.SaleNumber); err == nil {
			sale.SaleNumber = generateSaleNumber()
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		break
	}

	saleID, err = s.repo.CreateSaleWithTx(ctx, tx, sale)
	if err != nil {
		return 0, err
	}
	for i := range items {
		items[i].SaleID = saleID
		if _, err = s.repo.CreateSaleItemWithTx(ctx, tx, &items[i]); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.create",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"branch_id":    input.BranchID,
				"total_amount": totalAmount,
				"item_count":   len(items),
			},
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return saleID, nil
}

func (s *Service) GetSale(ctx context.Context, id int64) (sale *Sale, items []SaleItem, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleReadPermission)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, ErrForbidden
	}

	sale, err = s.repo.GetSaleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrSaleNotFound
		}
		return nil, nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return nil, nil, err
	}

	items, err = s.repo.ListSaleItemsBySaleID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return sale, items, nil
}

func (s *Service) ListSales(ctx context.Context, filter SaleFilter) (sales []Sale, err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleReadPermission)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}

	if filter.BranchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *filter.BranchID, true); err != nil {
			return nil, err
		}
		return s.repo.ListSales(ctx, filter)
	}

	branchesList, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	if len(branchesList) == 0 {
		return []Sale{}, nil
	}
	var allSales []Sale
	for _, branch := range branchesList {
		filterCopy := filter
		filterCopy.BranchID = &branch.ID
		branchSales, err := s.repo.ListSales(ctx, filterCopy)
		if err != nil {
			return nil, err
		}
		allSales = append(allSales, branchSales...)
	}
	return allSales, nil
}

func (s *Service) CompleteSale(ctx context.Context, saleID int64) (err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCompletePermission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSaleNotFound
		}
		return err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return err
	}

	switch sale.Status {
	case SaleStatusDraft:
		// allowed
	case SaleStatusCompleted:
		return ErrSaleAlreadyCompleted
	case SaleStatusCancelled:
		return ErrInvalidSaleTransition
	default:
		return ErrInvalidSaleTransition
	}

	items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ErrSaleHasNoItems
	}

	for _, item := range items {
		inventoryRow, err := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, sale.BranchID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInventoryNotFound
			}
			return err
		}
		if inventoryRow.Quantity < item.Quantity {
			return ErrInsufficientStock
		}
		newQuantity := inventoryRow.Quantity - item.Quantity
		if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inventoryRow.ID, newQuantity); err != nil {
			return err
		}
		movement := &inventory.StockMovement{
			ProductID:     item.ProductID,
			BranchID:      sale.BranchID,
			MovementType:  inventory.MovementTypeOUT,
			QuantityDelta: -item.Quantity,
			ReferenceType: inventory.PtrString("sale"),
			ReferenceID:   inventory.PtrInt64(saleID),
			ActorUserID:   actorUserIDFromContext(ctx),
			Metadata: map[string]any{
				"sale_id":    saleID,
				"branch_id":  sale.BranchID,
				"product_id": item.ProductID,
				"quantity":   item.Quantity,
			},
		}
		if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return err
		}
	}

	if err = s.repo.UpdateSaleStatusWithTx(ctx, tx, saleID, SaleStatusCompleted); err != nil {
		return err
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.complete",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sale_id":   saleID,
				"branch_id": sale.BranchID,
				"status":    SaleStatusCompleted,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) CancelSale(ctx context.Context, saleID int64) (err error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, SaleCancelPermission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	sale, err := s.repo.GetSaleByIDForUpdate(ctx, tx, saleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSaleNotFound
		}
		return err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, sale.BranchID, true); err != nil {
		return err
	}

	switch sale.Status {
	case SaleStatusDraft:
		if _, err = tx.ExecContext(ctx, `
            UPDATE sales
            SET status = $1, updated_at = NOW()
            WHERE id = $2
        `, SaleStatusCancelled, saleID); err != nil {
			return err
		}
	case SaleStatusCompleted:
		items, err := s.repo.ListSaleItemsBySaleIDWithTx(ctx, tx, saleID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrSaleHasNoItems
		}
		for _, item := range items {
			inventoryRow, err := s.inventoryRepo.GetByProductAndBranchForUpdate(ctx, tx, item.ProductID, sale.BranchID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrInventoryNotFound
				}
				return err
			}
			newQuantity := inventoryRow.Quantity + item.Quantity
			if err = s.inventoryRepo.UpdateQuantityWithTx(ctx, tx, inventoryRow.ID, newQuantity); err != nil {
				return err
			}
			movement := &inventory.StockMovement{
				ProductID:     item.ProductID,
				BranchID:      sale.BranchID,
				MovementType:  inventory.MovementTypeIN,
				QuantityDelta: item.Quantity,
				ReferenceType: inventory.PtrString("sale_cancel"),
				ReferenceID:   inventory.PtrInt64(saleID),
				ActorUserID:   actorUserIDFromContext(ctx),
				Metadata: map[string]any{
					"sale_id":    saleID,
					"branch_id":  sale.BranchID,
					"product_id": item.ProductID,
					"quantity":   item.Quantity,
				},
			}
			if _, err = s.inventoryRepo.CreateMovementWithTx(ctx, tx, movement); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `
            UPDATE sales
            SET status = $1, updated_at = NOW()
            WHERE id = $2
        `, SaleStatusCancelled, saleID); err != nil {
			return err
		}
	case SaleStatusCancelled:
		return ErrSaleAlreadyCancelled
	default:
		return ErrInvalidSaleTransition
	}

	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", saleID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "sale.cancel",
			Resource:    "sale",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"sale_id":   saleID,
				"branch_id": sale.BranchID,
				"status":    SaleStatusCancelled,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func generateSaleNumber() string {
	buf := make([]byte, saleNumberRandomPrefix)
	_, err := rand.Read(buf)
	if err != nil {
		return fmt.Sprintf("SALE-%d", time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("SALE-%s-%s", time.Now().UTC().Format("20060102150405"), hex.EncodeToString(buf))
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
