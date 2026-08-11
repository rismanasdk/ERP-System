package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/master/products"

	"github.com/lib/pq"
)

type repository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, inventory *Inventory) (int64, error)
	GetByID(ctx context.Context, id int64) (*Inventory, error)
	GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error)
	List(ctx context.Context, branchID, productID *int64) ([]Inventory, error)
	UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error
	CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *StockMovement) (int64, error)
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

const (
	MovementTypeIN         = "IN"
	MovementTypeOUT        = "OUT"
	MovementTypeAdjustment = "ADJUSTMENT"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrForbidden              = errors.New("permission denied")
	ErrProductNotFound        = errors.New("product not found")
	ErrProductInactive        = errors.New("product is inactive")
	ErrInventoryNotFound      = errors.New("inventory not found")
	ErrInventoryConflict      = errors.New("inventory already exists for product and branch")
	ErrInvalidMovementType    = errors.New("invalid movement type")
	ErrInvalidQuantityDelta   = errors.New("quantity_delta must not be zero")
	ErrInsufficientStock      = errors.New("insufficient stock")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type Service struct {
	repo        repository
	productSvc  productService
	branchSvc   branchService
	authChecker auth.PermissionChecker
	auditSvc    auditService
}

func NewService(repo repository, productSvc productService, branchSvc branchService, authChecker auth.PermissionChecker, auditSvc auditService) *Service {
	return &Service{
		repo:        repo,
		productSvc:  productSvc,
		branchSvc:   branchSvc,
		authChecker: authChecker,
		auditSvc:    auditSvc,
	}
}

func (s *Service) CreateInventory(ctx context.Context, productID, branchID, quantity int64) (int64, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, "inventory.create")
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}

	product, err := s.productSvc.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			return 0, ErrProductNotFound
		}
		return 0, err
	}
	if !product.IsActive {
		return 0, ErrProductInactive
	}

	if quantity < 0 {
		return 0, &ValidationError{Field: "quantity", Message: "must be greater than or equal to 0"}
	}

	if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, branchID, true); err != nil {
		return 0, err
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

	inventory := &Inventory{
		ProductID: productID,
		BranchID:  branchID,
		Quantity:  quantity,
	}
	id, err := s.repo.CreateWithTx(ctx, tx, inventory)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, ErrInventoryConflict
		}
		return 0, err
	}

	if quantity > 0 {
		movement := &StockMovement{
			ProductID:     productID,
			BranchID:      branchID,
			MovementType:  MovementTypeIN,
			QuantityDelta: quantity,
			ActorUserID:   actorUserIDFromContext(ctx),
		}
		if _, err = s.repo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		auditMetadata := map[string]any{
			"branch_id":      branchID,
			"product_id":     productID,
			"movement_type":  MovementTypeIN,
			"quantity_delta": quantity,
		}
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "inventory.create",
			Resource:    "inventory",
			ResourceID:  &resourceID,
			Metadata:    auditMetadata,
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Inventory, error) {
	inventory, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInventoryNotFound
		}
		return nil, err
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, inventory.BranchID, true); err != nil {
		return nil, err
	}
	return inventory, nil
}

func (s *Service) List(ctx context.Context, branchID, productID *int64) ([]Inventory, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}

	if branchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *branchID, true); err != nil {
			return nil, err
		}
		return s.repo.List(ctx, branchID, productID)
	}

	branches, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return []Inventory{}, nil
	}

	var inventoryList []Inventory
	for _, b := range branches {
		items, err := s.repo.List(ctx, &b.ID, productID)
		if err != nil {
			return nil, err
		}
		inventoryList = append(inventoryList, items...)
	}
	return inventoryList, nil
}

func (s *Service) AdjustStock(ctx context.Context, inventoryID int64, movementType string, quantityDelta int64, referenceType *string, referenceID *int64) (int64, error) {
	inventory, err := s.repo.GetByID(ctx, inventoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInventoryNotFound
		}
		return 0, err
	}

	return s.adjustStock(ctx, inventory.ProductID, inventory.BranchID, movementType, quantityDelta, referenceType, referenceID)
}

func (s *Service) adjustStock(ctx context.Context, productID, branchID int64, movementType string, quantityDelta int64, referenceType *string, referenceID *int64) (int64, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return 0, ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, "inventory.adjust")
	if err != nil {
		return 0, err
	}
	if !allowed {
		return 0, ErrForbidden
	}

	product, err := s.productSvc.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			return 0, ErrProductNotFound
		}
		return 0, err
	}
	if !product.IsActive {
		return 0, ErrProductInactive
	}

	if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, branchID, true); err != nil {
		return 0, err
	}

	if quantityDelta == 0 {
		return 0, ErrInvalidQuantityDelta
	}

	switch movementType {
	case MovementTypeIN:
		if quantityDelta <= 0 {
			return 0, &ValidationError{Field: "quantity_delta", Message: "IN movement must have positive quantity_delta"}
		}
	case MovementTypeOUT:
		if quantityDelta >= 0 {
			return 0, &ValidationError{Field: "quantity_delta", Message: "OUT movement must have negative quantity_delta"}
		}
	case MovementTypeAdjustment:
		// any non-zero delta is allowed
	default:
		return 0, ErrInvalidMovementType
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

	inventory, err := s.repo.GetByProductAndBranchForUpdate(ctx, tx, productID, branchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInventoryNotFound
		}
		return 0, err
	}

	newQuantity := inventory.Quantity + quantityDelta
	if newQuantity < 0 {
		err = ErrInsufficientStock
		return 0, err
	}

	if err = s.repo.UpdateQuantityWithTx(ctx, tx, inventory.ID, newQuantity); err != nil {
		return 0, err
	}

	movement := &StockMovement{
		ProductID:     productID,
		BranchID:      branchID,
		MovementType:  movementType,
		QuantityDelta: quantityDelta,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		ActorUserID:   actorUserIDFromContext(ctx),
	}
	movementID, err := s.repo.CreateMovementWithTx(ctx, tx, movement)
	if err != nil {
		return 0, err
	}

	if s.auditSvc != nil {
		auditMetadata := map[string]any{
			"branch_id":      branchID,
			"product_id":     productID,
			"movement_type":  movementType,
			"quantity_delta": quantityDelta,
		}
		resourceID := fmt.Sprintf("%d", inventory.ID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserIDFromContext(ctx),
			Action:      "inventory.adjust",
			Resource:    "inventory",
			ResourceID:  &resourceID,
			Metadata:    auditMetadata,
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return movementID, nil
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
