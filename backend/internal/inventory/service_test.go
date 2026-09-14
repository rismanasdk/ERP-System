package inventory

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/master/products"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

type fakeInventoryRepo struct {
	db                *sql.DB
	createErr         error
	getInventory      *Inventory
	getErr            error
	updateErr         error
	movementErr       error
	listItems         []Inventory
	listItemsByBranch map[int64][]Inventory
	listErr           error
	updatedQuantities map[int64]int64
	movements         []StockMovement
	transfer          *StockTransfer
}

func (r *fakeInventoryRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *fakeInventoryRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, inventory *Inventory) (int64, error) {
	if r.createErr != nil {
		return 0, r.createErr
	}
	return 1, nil
}

func (r *fakeInventoryRepo) GetByID(ctx context.Context, id int64) (*Inventory, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.getInventory == nil {
		return nil, sql.ErrNoRows
	}
	return r.getInventory, nil
}

func (r *fakeInventoryRepo) GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.getInventory == nil {
		return nil, sql.ErrNoRows
	}
	return r.getInventory, nil
}

func (r *fakeInventoryRepo) List(ctx context.Context, branchID, productID *int64) ([]Inventory, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	if branchID == nil {
		if len(r.listItemsByBranch) > 0 {
			var items []Inventory
			for _, branchItems := range r.listItemsByBranch {
				items = append(items, branchItems...)
			}
			return items, nil
		}
		return r.listItems, nil
	}
	if items, ok := r.listItemsByBranch[*branchID]; ok {
		return items, nil
	}
	return nil, nil
}

func (r *fakeInventoryRepo) ListMovements(ctx context.Context, branchID, productID *int64) ([]StockMovement, error) {
	return []StockMovement{{ID: 1, ProductID: 101, BranchID: 1, MovementType: MovementTypeIN, QuantityDelta: 5}}, nil
}

func (r *fakeInventoryRepo) UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error {
	if r.updatedQuantities == nil {
		r.updatedQuantities = map[int64]int64{}
	}
	r.updatedQuantities[id] = quantity
	return r.updateErr
}

func (r *fakeInventoryRepo) CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *StockMovement) (int64, error) {
	if r.movementErr != nil {
		return 0, r.movementErr
	}
	r.movements = append(r.movements, *movement)
	return 1, nil
}

func (r *fakeInventoryRepo) EnsureInventoryWithTx(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return &Inventory{ID: 2, ProductID: productID, BranchID: branchID}, nil
}

func (r *fakeInventoryRepo) CreateTransferWithTx(ctx context.Context, tx *sql.Tx, transfer *StockTransfer) (int64, error) {
	transfer.ID = 1
	r.transfer = transfer
	return transfer.ID, nil
}

func (r *fakeInventoryRepo) ListTransfers(ctx context.Context, branchIDs []int64) ([]StockTransfer, error) {
	return nil, nil
}

func (r *fakeInventoryRepo) GetTransfer(ctx context.Context, id int64) (*StockTransfer, error) {
	return nil, sql.ErrNoRows
}

type fakeProductService struct {
	product *products.Product
	err     error
}

func (f *fakeProductService) GetByID(ctx context.Context, id int64) (*products.Product, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.product, nil
}

type fakeBranchService struct {
	err      error
	branches []branches.Branch
	listErr  error
}

func (f *fakeBranchService) EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error {
	return f.err
}

func (f *fakeBranchService) ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error) {
	return f.branches, f.listErr
}

type fakeAuthChecker struct {
	allowed bool
	err     error
}

func (f *fakeAuthChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return f.allowed, f.err
}

type fakeAuditService struct {
	lastLog audit.AuditLog
	err     error
}

func (f *fakeAuditService) RecordWithTx(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) (int64, error) {
	f.lastLog = auditLog
	return 1, f.err
}

func newServiceWithTx(t *testing.T) (*Service, *fakeInventoryRepo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	repo := &fakeInventoryRepo{db: db}
	service := NewService(repo, &fakeProductService{}, &fakeBranchService{}, &fakeAuthChecker{allowed: true}, &fakeAuditService{})

	cleanup := func() {
		db.Close()
	}
	return service, repo, mock, cleanup
}

func TestCreateInventory_Success(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.createErr = nil

	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	id, err := service.CreateInventory(ctx, 2, 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id 1, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateTransfer_MovesStockAndSharesReference(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 11, ProductID: 2, BranchID: 3, Quantity: 50}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}
	mock.ExpectBegin()
	mock.ExpectCommit()

	transferID, err := service.CreateTransfer(auth.ContextWithUserID(context.Background(), 10), CreateTransferInput{
		SourceBranchID: 3, DestinationBranchID: 4, ProductID: 2, Quantity: 30,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if transferID != 1 || repo.transfer == nil || repo.transfer.Status != "COMPLETED" {
		t.Fatalf("unexpected transfer: %+v", repo.transfer)
	}
	if repo.updatedQuantities[11] != 20 || repo.updatedQuantities[2] != 30 {
		t.Fatalf("unexpected quantities: %+v", repo.updatedQuantities)
	}
	if len(repo.movements) != 2 || repo.movements[0].QuantityDelta != -30 || repo.movements[1].QuantityDelta != 30 {
		t.Fatalf("unexpected movements: %+v", repo.movements)
	}
	if repo.movements[0].ReferenceID == nil || repo.movements[1].ReferenceID == nil || *repo.movements[0].ReferenceID != transferID || *repo.movements[1].ReferenceID != transferID {
		t.Fatalf("movements do not share transfer reference: %+v", repo.movements)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateTransfer_RejectsSameBranchAndInsufficientStock(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()
	repo.getInventory = &Inventory{ID: 11, ProductID: 2, BranchID: 3, Quantity: 5}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{}
	service.authChecker = &fakeAuthChecker{allowed: true}
	ctx := auth.ContextWithUserID(context.Background(), 10)

	_, err := service.CreateTransfer(ctx, CreateTransferInput{SourceBranchID: 3, DestinationBranchID: 3, ProductID: 2, Quantity: 1})
	if !errors.Is(err, ErrSameTransferBranch) {
		t.Fatalf("expected same branch error, got %v", err)
	}
	mock.ExpectBegin()
	mock.ExpectRollback()
	_, err = service.CreateTransfer(ctx, CreateTransferInput{SourceBranchID: 3, DestinationBranchID: 4, ProductID: 2, Quantity: 6})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}
}

func TestCreateInventory_NonexistentProduct(t *testing.T) {
	service, _, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	service.productSvc = &fakeProductService{err: products.ErrProductNotFound}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.CreateInventory(ctx, 99, 1, 5)
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestCreateInventory_InactiveProduct(t *testing.T) {
	service, _, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: false}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.CreateInventory(ctx, 2, 1, 5)
	if !errors.Is(err, ErrProductInactive) {
		t.Fatalf("expected ErrProductInactive, got %v", err)
	}
}

func TestCreateInventory_NonexistentBranch(t *testing.T) {
	service, _, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchNotFound}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.CreateInventory(ctx, 2, 99, 5)
	if !errors.Is(err, branches.ErrBranchNotFound) {
		t.Fatalf("expected ErrBranchNotFound, got %v", err)
	}
}

func TestCreateInventory_ForbiddenBranchAccess(t *testing.T) {
	service, _, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchAccessDenied}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.CreateInventory(ctx, 2, 3, 5)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
}

func TestCreateInventory_DuplicateInventoryConflict(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.createErr = &pq.Error{Code: "23505"}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.CreateInventory(ctx, 2, 3, 5)
	if !errors.Is(err, ErrInventoryConflict) {
		t.Fatalf("expected ErrInventoryConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAdjustStock_PositiveIncreasesQuantity(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	id, err := service.AdjustStock(ctx, 1, MovementTypeIN, 5, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 1 {
		t.Fatalf("expected movement id 1, got %d", id)
	}
}

func TestAdjustStock_NegativeWithinStockSucceeds(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeOUT, -3, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAdjustStock_ZeroResultSucceeds(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeOUT, -10, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAdjustStock_ExceedingStockFails(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeOUT, -11, nil, nil)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAdjustStock_ZeroDeltaValidationError(t *testing.T) {
	service, repo, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeAdjustment, 0, nil, nil)
	if !errors.Is(err, ErrInvalidQuantityDelta) {
		t.Fatalf("expected ErrInvalidQuantityDelta, got %v", err)
	}
}

func TestAdjustStock_InvalidMovementType(t *testing.T) {
	service, repo, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, "PURCHASE", 5, nil, nil)
	if !errors.Is(err, ErrInvalidMovementType) {
		t.Fatalf("expected ErrInvalidMovementType, got %v", err)
	}
}

func TestAdjustStock_InvalidINDelta(t *testing.T) {
	service, repo, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeIN, -5, nil, nil)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestAdjustStock_InvalidOUTDelta(t *testing.T) {
	service, repo, _, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeOUT, 5, nil, nil)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestAdjustStock_MovementInsertFailureRollsBack(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	repo.movementErr = errors.New("insert failed")
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeIN, 5, nil, nil)
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("expected insert failed error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAdjustStock_AuditInsertFailureRollsBack(t *testing.T) {
	service, repo, mock, cleanup := newServiceWithTx(t)
	defer cleanup()

	repo.getInventory = &Inventory{ID: 1, ProductID: 2, BranchID: 3, Quantity: 10}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 2, IsActive: true}}
	service.branchSvc = &fakeBranchService{err: nil}
	service.authChecker = &fakeAuthChecker{allowed: true}
	service.auditSvc = &fakeAuditService{err: errors.New("audit failed")}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 10)
	_, err := service.AdjustStock(ctx, 1, MovementTypeIN, 5, nil, nil)
	if err == nil || err.Error() != "audit failed" {
		t.Fatalf("expected audit failed error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
