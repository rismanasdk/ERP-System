package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

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
	ListMovements(ctx context.Context, branchID, productID *int64) ([]StockMovement, error)
	EnsureInventoryWithTx(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error)
	UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error
	CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *StockMovement) (int64, error)
	CreateTransferWithTx(ctx context.Context, tx *sql.Tx, transfer *StockTransfer) (int64, error)
	ListTransfers(ctx context.Context, branchIDs []int64) ([]StockTransfer, error)
	GetTransfer(ctx context.Context, id int64) (*StockTransfer, error)
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
	ErrSameTransferBranch     = errors.New("source and destination branches must be different")
	ErrTransferNotFound       = errors.New("stock transfer not found")
)

type ValidationError struct {
	Field   string
	Message string
}

type CreateTransferInput struct {
	SourceBranchID      int64   `json:"source_branch_id"`
	DestinationBranchID int64   `json:"destination_branch_id"`
	ProductID           int64   `json:"product_id"`
	Quantity            int64   `json:"quantity"`
	Notes               *string `json:"notes,omitempty"`
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

func (s *Service) ListMovements(ctx context.Context, branchID, productID *int64) ([]StockMovement, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}

	if branchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *branchID, true); err != nil {
			return nil, err
		}
		return s.repo.ListMovements(ctx, branchID, productID)
	}

	branches, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return []StockMovement{}, nil
	}

	var movements []StockMovement
	for _, b := range branches {
		items, err := s.repo.ListMovements(ctx, &b.ID, productID)
		if err != nil {
			return nil, err
		}
		movements = append(movements, items...)
	}
	return movements, nil
}

func (s *Service) CreateTransfer(ctx context.Context, input CreateTransferInput) (transferID int64, err error) {
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
	if input.SourceBranchID == input.DestinationBranchID {
		return 0, ErrSameTransferBranch
	}
	if input.Quantity <= 0 {
		return 0, &ValidationError{Field: "quantity", Message: "must be greater than zero"}
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, input.SourceBranchID, true); err != nil {
		return 0, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, input.DestinationBranchID, true); err != nil {
		return 0, err
	}
	product, err := s.productSvc.GetByID(ctx, input.ProductID)
	if err != nil {
		if errors.Is(err, products.ErrProductNotFound) {
			return 0, ErrProductNotFound
		}
		return 0, err
	}
	if !product.IsActive {
		return 0, ErrProductInactive
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

	branchIDs := []int64{input.SourceBranchID, input.DestinationBranchID}
	sort.Slice(branchIDs, func(i, j int) bool { return branchIDs[i] < branchIDs[j] })
	var source, destination *Inventory
	for _, branchID := range branchIDs {
		if branchID == input.SourceBranchID {
			source, err = s.repo.GetByProductAndBranchForUpdate(ctx, tx, input.ProductID, branchID)
		} else {
			destination, err = s.repo.EnsureInventoryWithTx(ctx, tx, input.ProductID, branchID)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) && branchID == input.SourceBranchID {
				return 0, ErrInventoryNotFound
			}
			return 0, err
		}
	}
	if source.Quantity < input.Quantity {
		return 0, ErrInsufficientStock
	}

	transfer := &StockTransfer{
		SourceBranchID:      input.SourceBranchID,
		DestinationBranchID: input.DestinationBranchID,
		ProductID:           input.ProductID,
		Quantity:            input.Quantity,
		Status:              "COMPLETED",
		Notes:               input.Notes,
		CreatedBy:           userID,
	}
	transferID, err = s.repo.CreateTransferWithTx(ctx, tx, transfer)
	if err != nil {
		return 0, err
	}
	referenceType := "stock_transfer"
	actorID := actorUserIDFromContext(ctx)
	if err = s.repo.UpdateQuantityWithTx(ctx, tx, source.ID, source.Quantity-input.Quantity); err != nil {
		return 0, err
	}
	if err = s.repo.UpdateQuantityWithTx(ctx, tx, destination.ID, destination.Quantity+input.Quantity); err != nil {
		return 0, err
	}
	for _, movement := range []*StockMovement{
		{ProductID: input.ProductID, BranchID: input.SourceBranchID, MovementType: "TRANSFER_OUT", QuantityDelta: -input.Quantity, ReferenceType: &referenceType, ReferenceID: &transferID, ActorUserID: actorID},
		{ProductID: input.ProductID, BranchID: input.DestinationBranchID, MovementType: "TRANSFER_IN", QuantityDelta: input.Quantity, ReferenceType: &referenceType, ReferenceID: &transferID, ActorUserID: actorID},
	} {
		if _, err = s.repo.CreateMovementWithTx(ctx, tx, movement); err != nil {
			return 0, err
		}
	}
	if s.auditSvc != nil {
		resourceID := fmt.Sprintf("%d", transferID)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorID,
			Action:      "inventory.transfer",
			Resource:    "stock_transfer",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"product_id": input.ProductID, "source_branch_id": input.SourceBranchID,
				"destination_branch_id": input.DestinationBranchID, "quantity": input.Quantity,
			},
		})
		if err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return transferID, nil
}

func (s *Service) ListTransfers(ctx context.Context) ([]StockTransfer, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, "inventory.read")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	branches, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{}, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(branches))
	for i, branch := range branches {
		ids[i] = branch.ID
	}
	return s.repo.ListTransfers(ctx, ids)
}

func (s *Service) GetTransfer(ctx context.Context, id int64) (*StockTransfer, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	allowed, err := s.authChecker.HasPermission(ctx, userID, "inventory.read")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	transfer, err := s.repo.GetTransfer(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTransferNotFound
		}
		return nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, transfer.SourceBranchID, true); err != nil {
		return nil, err
	}
	if err = s.branchSvc.EnsureUserHasAccess(ctx, userID, transfer.DestinationBranchID, true); err != nil {
		return nil, err
	}
	return transfer, nil
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
