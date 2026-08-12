package purchasing

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/products"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

type fakePurchaseRepo struct {
	db                          *sql.DB
	nextPurchaseID              int64
	nextPurchaseItemID          int64
	supplier                    *Supplier
	getSupplierErr              error
	createPurchaseErr           error
	createPurchaseErrorSequence []error
	createPurchaseAttempts      int
	createPurchaseItemErr       error
}

func (r *fakePurchaseRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *fakePurchaseRepo) CreatePurchaseWithTx(ctx context.Context, tx *sql.Tx, purchase *Purchase) (int64, error) {
	if r.createPurchaseAttempts < len(r.createPurchaseErrorSequence) {
		err := r.createPurchaseErrorSequence[r.createPurchaseAttempts]
		r.createPurchaseAttempts++
		if err != nil {
			return 0, err
		}
	}
	if r.createPurchaseErr != nil {
		return 0, r.createPurchaseErr
	}
	return r.nextPurchaseID, nil
}

func (r *fakePurchaseRepo) CreatePurchaseItemWithTx(ctx context.Context, tx *sql.Tx, item *PurchaseItem) (int64, error) {
	if r.createPurchaseItemErr != nil {
		return 0, r.createPurchaseItemErr
	}
	return r.nextPurchaseItemID, nil
}

func (r *fakePurchaseRepo) GetSupplierByID(ctx context.Context, id int64) (*Supplier, error) {
	if r.getSupplierErr != nil {
		return nil, r.getSupplierErr
	}
	if r.supplier == nil {
		return nil, sql.ErrNoRows
	}
	return r.supplier, nil
}

func (r *fakePurchaseRepo) GetPurchaseByID(ctx context.Context, id int64) (*Purchase, error) {
	return nil, sql.ErrNoRows
}

func (r *fakePurchaseRepo) ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error) {
	return nil, nil
}

type fakeInventoryRepo struct {
	getInventoryErr error
	inventory       *inventory.Inventory
	created         bool
	updated         bool
	movementCreated bool
}

func (r *fakeInventoryRepo) GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error) {
	if r.getInventoryErr != nil {
		return nil, r.getInventoryErr
	}
	if r.inventory == nil {
		return nil, sql.ErrNoRows
	}
	return r.inventory, nil
}

func (r *fakeInventoryRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, inv *inventory.Inventory) (int64, error) {
	r.created = true
	return 1, nil
}

func (r *fakeInventoryRepo) UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error {
	r.updated = true
	return nil
}

func (r *fakeInventoryRepo) CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *inventory.StockMovement) (int64, error) {
	r.movementCreated = true
	return 1, nil
}

type fakeProductService struct {
	product *products.Product
	err     error
}

func (f *fakeProductService) GetByID(ctx context.Context, id int64) (*products.Product, error) {
	return f.product, f.err
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

func newServiceWithMocks(t *testing.T) (*Service, *fakePurchaseRepo, *fakeInventoryRepo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100}
	invRepo := &fakeInventoryRepo{}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		&fakeAuditService{},
	)

	cleanup := func() {
		db.Close()
	}
	return service, repo, invRepo, mock, cleanup
}

func TestCreatePurchase_Success(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Notes:      nil,
		Items: []CreatePurchaseItemInput{
			{ProductID: 5, Quantity: 10, UnitCost: 12.5},
		},
	}

	id, err := service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 42 {
		t.Fatalf("expected purchase id 42, got %d", id)
	}
	if !invRepo.created {
		t.Fatal("expected inventory to be created")
	}
	if !invRepo.movementCreated {
		t.Fatal("expected stock movement to be created")
	}
	if service.auditSvc.(*fakeAuditService).lastLog.Action != "purchase.create" {
		t.Fatalf("expected audit action purchase.create, got %s", service.auditSvc.(*fakeAuditService).lastLog.Action)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_DuplicateProduct(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items: []CreatePurchaseItemInput{
			{ProductID: 5, Quantity: 3, UnitCost: 2.0},
			{ProductID: 5, Quantity: 4, UnitCost: 3.0},
		},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrDuplicateProduct) {
		t.Fatalf("expected ErrDuplicateProduct, got %v", err)
	}
}

func TestCreatePurchase_ProductNotFound(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.productSvc = &fakeProductService{err: products.ErrProductNotFound}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestCreatePurchase_Forbidden(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.authChecker = &fakeAuthChecker{allowed: false}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreatePurchase_ForbiddenBranchAccess(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchAccessDenied}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   99,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
}

func TestCreatePurchase_NonexistentBranch(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchNotFound}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   99,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, branches.ErrBranchNotFound) {
		t.Fatalf("expected ErrBranchNotFound, got %v", err)
	}
}

func TestCreatePurchase_InactiveBranch(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchInactive}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   99,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, branches.ErrBranchInactive) {
		t.Fatalf("expected ErrBranchInactive, got %v", err)
	}
}

func TestCreatePurchase_SuperAdminAccess(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.branchSvc = &fakeBranchService{err: nil}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 1)
	input := CreatePurchaseInput{
		BranchID:   2,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 2, UnitCost: 10.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error for super admin access, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_NonexistentSupplier(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.getSupplierErr = sql.ErrNoRows

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrSupplierNotFound) {
		t.Fatalf("expected ErrSupplierNotFound, got %v", err)
	}
}

func TestCreatePurchase_InactiveSupplier(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: false}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrSupplierInactive) {
		t.Fatalf("expected ErrSupplierInactive, got %v", err)
	}
}

func TestCreatePurchase_InactiveProduct(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	service.productSvc = &fakeProductService{product: &products.Product{ID: 5, IsActive: false}}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if !errors.Is(err, ErrProductInactive) {
		t.Fatalf("expected ErrProductInactive, got %v", err)
	}
}

func TestCreatePurchase_InvalidQuantity(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 0, UnitCost: 2.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError for invalid quantity, got %v", err)
	}
}

func TestCreatePurchase_InvalidUnitCost(t *testing.T) {
	service, repo, _, _, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: -1.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError for invalid unit cost, got %v", err)
	}
}

func TestCreatePurchase_Success_NewInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}}
	invRepo := &fakeInventoryRepo{}
	auditSvc := &fakeAuditService{}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	id, err := service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 42 {
		t.Fatalf("expected purchase id 42, got %d", id)
	}
	if !invRepo.created {
		t.Fatal("expected new inventory to be created")
	}
	if !invRepo.movementCreated {
		t.Fatal("expected stock movement to be created")
	}
	if auditSvc.lastLog.Action != "purchase.create" {
		t.Fatalf("expected audit action purchase.create, got %s", auditSvc.lastLog.Action)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_Success_ExistingInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}}
	invRepo := &fakeInventoryRepo{inventory: &inventory.Inventory{ID: 10, ProductID: 5, BranchID: 1, Quantity: 10}}
	auditSvc := &fakeAuditService{}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	id, err := service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != 42 {
		t.Fatalf("expected purchase id 42, got %d", id)
	}
	if !invRepo.updated {
		t.Fatal("expected inventory to be updated")
	}
	if !invRepo.movementCreated {
		t.Fatal("expected stock movement to be created")
	}
	if auditSvc.lastLog.Action != "purchase.create" {
		t.Fatalf("expected audit action purchase.create, got %s", auditSvc.lastLog.Action)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_PurchaseItemFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}, createPurchaseItemErr: errors.New("purchase item insert failed")}
	invRepo := &fakeInventoryRepo{}
	auditSvc := &fakeAuditService{}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	_, err = service.CreatePurchase(ctx, input)
	if err == nil {
		t.Fatal("expected error when purchase item creation fails")
	}
	if invRepo.created {
		t.Fatal("expected inventory not to be created when purchase item fails")
	}
	if invRepo.movementCreated {
		t.Fatal("expected no stock movement when purchase item fails")
	}
	if auditSvc.lastLog.Action != "" {
		t.Fatal("expected no audit log when purchase item fails")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_InventoryFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}}
	invRepo := &fakeInventoryRepo{getInventoryErr: errors.New("inventory lookup failed")}
	auditSvc := &fakeAuditService{}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	_, err = service.CreatePurchase(ctx, input)
	if err == nil {
		t.Fatal("expected error when inventory lookup fails")
	}
	if invRepo.movementCreated {
		t.Fatal("expected no stock movement when inventory lookup fails")
	}
	if auditSvc.lastLog.Action != "" {
		t.Fatal("expected no audit log when inventory lookup fails")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_AuditFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}}
	invRepo := &fakeInventoryRepo{}
	auditSvc := &fakeAuditService{err: errors.New("audit failure")}
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{product: &products.Product{ID: 5, IsActive: true}},
		&fakeBranchService{},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	_, err = service.CreatePurchase(ctx, input)
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if !invRepo.created {
		t.Fatal("expected inventory to be created before audit fails")
	}
	if !invRepo.movementCreated {
		t.Fatal("expected stock movement to be created before audit fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_PurchaseNumberRetry(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.supplier = &Supplier{ID: 3, IsActive: true}
	repo.createPurchaseErrorSequence = []error{&pq.Error{Code: "23505"}}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 1, UnitCost: 10.0}},
	}

	_, err := service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error after purchase number retry, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
