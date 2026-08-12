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
	purchase                    *Purchase
	purchaseItems               []PurchaseItem
	updateStatusErr             error
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

func (r *fakePurchaseRepo) GetPurchaseByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Purchase, error) {
	if r.purchase == nil || r.purchase.ID != id {
		return nil, sql.ErrNoRows
	}
	return r.purchase, nil
}

func (r *fakePurchaseRepo) ListPurchaseItemsByPurchaseIDWithTx(ctx context.Context, tx *sql.Tx, purchaseID int64) ([]PurchaseItem, error) {
	if r.purchase == nil || r.purchase.ID != purchaseID {
		return nil, sql.ErrNoRows
	}
	return r.purchaseItems, nil
}

func (r *fakePurchaseRepo) UpdatePurchaseStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error {
	if r.purchase == nil || r.purchase.ID != id {
		return sql.ErrNoRows
	}
	r.purchase.Status = status
	if r.updateStatusErr != nil {
		return r.updateStatusErr
	}
	return nil
}

type fakeInventoryRepo struct {
	getInventoryErr error
	inventory       *inventory.Inventory
	created         bool
	updated         bool
	updatedCount    int
	movementCount   int
	movementCreated bool
	movementErr     error
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
	r.updatedCount++
	return nil
}

func (r *fakeInventoryRepo) CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *inventory.StockMovement) (int64, error) {
	if r.movementErr != nil {
		return 0, r.movementErr
	}
	r.movementCount++
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
	if invRepo.created {
		t.Fatal("expected inventory not to be created for draft purchase")
	}
	if invRepo.movementCount != 0 {
		t.Fatal("expected no stock movement for draft purchase")
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

func TestCreatePurchase_DraftDoesNotAffectInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &fakePurchaseRepo{db: db, nextPurchaseID: 42, nextPurchaseItemID: 100, supplier: &Supplier{ID: 3, IsActive: true}}
	invRepo := &fakeInventoryRepo{getInventoryErr: errors.New("should not be called")}
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
	if invRepo.created {
		t.Fatal("expected inventory not to be created for draft purchase")
	}
	if invRepo.movementCount != 0 {
		t.Fatal("expected no stock movement for draft purchase")
	}
	if auditSvc.lastLog.Action != "purchase.create" {
		t.Fatalf("expected audit action purchase.create, got %s", auditSvc.lastLog.Action)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreatePurchase_DraftDoesNotAffectExistingInventory(t *testing.T) {
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
	if invRepo.updated {
		t.Fatal("expected inventory not to be updated for draft purchase")
	}
	if invRepo.movementCreated {
		t.Fatal("expected no stock movement for draft purchase")
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

func TestCreatePurchase_IgnoresInventoryRepositoryDuringDraftCreation(t *testing.T) {
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
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	input := CreatePurchaseInput{
		BranchID:   1,
		SupplierID: 3,
		Items:      []CreatePurchaseItemInput{{ProductID: 5, Quantity: 5, UnitCost: 1000}},
	}

	_, err = service.CreatePurchase(ctx, input)
	if err != nil {
		t.Fatalf("expected no error during draft purchase creation, got %v", err)
	}
	if invRepo.created {
		t.Fatal("expected inventory not to be created for draft purchase")
	}
	if invRepo.movementCount != 0 {
		t.Fatal("expected no stock movement for draft purchase")
	}
	if auditSvc.lastLog.Action != "purchase.create" {
		t.Fatalf("expected audit action purchase.create, got %s", auditSvc.lastLog.Action)
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
	if invRepo.created {
		t.Fatal("expected inventory not to be created for draft purchase")
	}
	if invRepo.movementCount != 0 {
		t.Fatal("expected no stock movement for draft purchase")
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

func TestCompletePurchase_Success_Draft(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 5, UnitCost: 10.0, Subtotal: 50.0}}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	if err := service.CompletePurchase(ctx, 42); err != nil {
		t.Fatalf("expected complete success, got %v", err)
	}
	if !invRepo.created {
		t.Fatal("expected inventory to be created for draft completion")
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected one stock movement, got %d", invRepo.movementCount)
	}
	if repo.purchase.Status != PurchaseStatusCompleted {
		t.Fatalf("expected purchase status completed, got %s", repo.purchase.Status)
	}
	if service.auditSvc.(*fakeAuditService).lastLog.Action != "purchase.complete" {
		t.Fatalf("expected audit action purchase.complete, got %s", service.auditSvc.(*fakeAuditService).lastLog.Action)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_AlreadyCompletedFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, ErrPurchaseAlreadyCompleted) {
		t.Fatalf("expected ErrPurchaseAlreadyCompleted, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_AlreadyCancelledFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCancelled}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, ErrCannotCompleteCancelled) {
		t.Fatalf("expected ErrCannotCompleteCancelled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_Success_Draft(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	if err := service.CancelPurchase(ctx, 42); err != nil {
		t.Fatalf("expected cancel success, got %v", err)
	}
	if repo.purchase.Status != PurchaseStatusCancelled {
		t.Fatalf("expected purchase status cancelled, got %s", repo.purchase.Status)
	}
	if service.auditSvc.(*fakeAuditService).lastLog.Action != "purchase.cancel" {
		t.Fatalf("expected audit action purchase.cancel, got %s", service.auditSvc.(*fakeAuditService).lastLog.Action)
	}
	if service.auditSvc.(*fakeAuditService).lastLog.Metadata["status"] != PurchaseStatusCancelled {
		t.Fatalf("expected audit metadata status cancelled, got %v", service.auditSvc.(*fakeAuditService).lastLog.Metadata["status"])
	}
	if repo.purchase.Status != PurchaseStatusCancelled {
		t.Fatalf("expected purchase status cancelled, got %s", repo.purchase.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_Success_CompletedReversesInventory(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	if err := service.CancelPurchase(ctx, 42); err != nil {
		t.Fatalf("expected cancel success, got %v", err)
	}
	if !invRepo.updated {
		t.Fatal("expected inventory to be updated when cancelling completed purchase")
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected one stock movement when cancelling completed purchase, got %d", invRepo.movementCount)
	}
	if repo.purchase.Status != PurchaseStatusCancelled {
		t.Fatalf("expected purchase status cancelled, got %s", repo.purchase.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_AlreadyCancelledFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCancelled}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if !errors.Is(err, ErrPurchaseAlreadyCancelled) {
		t.Fatalf("expected ErrPurchaseAlreadyCancelled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_NonexistentPurchase(t *testing.T) {
	service, _, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 999)
	if !errors.Is(err, ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_NonexistentPurchase(t *testing.T) {
	service, _, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 999)
	if !errors.Is(err, ErrPurchaseNotFound) {
		t.Fatalf("expected ErrPurchaseNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_NoPurchaseItemsFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, ErrPurchaseHasNoItems) {
		t.Fatalf("expected ErrPurchaseHasNoItems, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_InventoryFailureRollsBack(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.getInventoryErr = errors.New("inventory unavailable")

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if err == nil {
		t.Fatal("expected error when inventory lookup fails")
	}
	if invRepo.movementCount != 0 {
		t.Fatal("expected no stock movement when inventory lookup fails")
	}
	if repo.purchase.Status != PurchaseStatusDraft {
		t.Fatalf("expected purchase status to remain draft, got %s", repo.purchase.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_MovementFailureRollsBack(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}
	invRepo.movementErr = errors.New("stock movement failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if err == nil {
		t.Fatal("expected error when movement creation fails")
	}
	if !invRepo.updated {
		t.Fatal("expected inventory update attempted before movement failure")
	}
	if repo.purchase.Status != PurchaseStatusDraft {
		t.Fatalf("expected purchase status to remain draft, got %s", repo.purchase.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_AuditFailureRollsBack(t *testing.T) {
	service, _, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	serviceRepo := service
	repo := serviceRepo.repo.(*fakePurchaseRepo)
	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}
	service.auditSvc = &fakeAuditService{err: errors.New("audit failed")}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if invRepo.movementCount != 1 {
		t.Fatal("expected movement attempted before audit failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_InaccessibleBranch(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 99, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchAccessDenied}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_InvalidTransitionFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: "UNKNOWN"}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, ErrInvalidPurchaseTransition) {
		t.Fatalf("expected ErrInvalidPurchaseTransition, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompletePurchase_RepeatedCompletionDoesNotDuplicateInventory(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusDraft}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	if err := service.CompletePurchase(ctx, 42); err != nil {
		t.Fatalf("expected first complete success, got %v", err)
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected one stock movement after first complete, got %d", invRepo.movementCount)
	}
	if repo.purchase.Status != PurchaseStatusCompleted {
		t.Fatalf("expected purchase status completed after first complete, got %s", repo.purchase.Status)
	}

	mock.ExpectBegin()
	mock.ExpectRollback()

	err := service.CompletePurchase(ctx, 42)
	if !errors.Is(err, ErrPurchaseAlreadyCompleted) {
		t.Fatalf("expected ErrPurchaseAlreadyCompleted on second complete, got %v", err)
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected no additional stock movement on second complete, got %d", invRepo.movementCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_CompletedInsufficientStockFails(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 15, UnitCost: 10.0, Subtotal: 150.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if repo.purchase.Status != PurchaseStatusCompleted {
		t.Fatalf("expected purchase status to remain completed, got %s", repo.purchase.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_MovementFailureRollsBack(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}
	invRepo.movementErr = errors.New("stock movement failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if err == nil {
		t.Fatal("expected error when movement creation fails")
	}
	if repo.purchase.Status != PurchaseStatusCompleted {
		t.Fatalf("expected purchase status to remain completed, got %s", repo.purchase.Status)
	}
	if !invRepo.updated {
		t.Fatal("expected inventory update attempted before movement failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_AuditFailureRollsBack(t *testing.T) {
	service, _, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo := service.repo.(*fakePurchaseRepo)
	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}
	service.auditSvc = &fakeAuditService{err: errors.New("audit failed")}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if invRepo.movementCount != 1 {
		t.Fatal("expected reversal movement attempted before audit failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_InaccessibleBranch(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 99, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	service.branchSvc = &fakeBranchService{err: branches.ErrBranchAccessDenied}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_InvalidTransitionFails(t *testing.T) {
	service, repo, _, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: "UNKNOWN"}

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	err := service.CancelPurchase(ctx, 42)
	if !errors.Is(err, ErrInvalidPurchaseTransition) {
		t.Fatalf("expected ErrInvalidPurchaseTransition, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelPurchase_RepeatedCancellationDoesNotDuplicateInventory(t *testing.T) {
	service, repo, invRepo, mock, cleanup := newServiceWithMocks(t)
	defer cleanup()

	repo.purchase = &Purchase{ID: 42, BranchID: 1, SupplierID: 3, PurchaseNumber: "PO-42", Status: PurchaseStatusCompleted}
	repo.purchaseItems = []PurchaseItem{{ID: 1, PurchaseID: 42, ProductID: 5, Quantity: 3, UnitCost: 10.0, Subtotal: 30.0}}
	invRepo.inventory = &inventory.Inventory{ID: 1, ProductID: 5, BranchID: 1, Quantity: 10}

	mock.ExpectBegin()
	mock.ExpectCommit()

	ctx := auth.ContextWithUserID(context.Background(), 7)
	if err := service.CancelPurchase(ctx, 42); err != nil {
		t.Fatalf("expected first cancel success, got %v", err)
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected one stock movement after first cancel, got %d", invRepo.movementCount)
	}
	if repo.purchase.Status != PurchaseStatusCancelled {
		t.Fatalf("expected purchase status cancelled after first cancel, got %s", repo.purchase.Status)
	}

	mock.ExpectBegin()
	mock.ExpectRollback()

	err := service.CancelPurchase(ctx, 42)
	if !errors.Is(err, ErrPurchaseAlreadyCancelled) {
		t.Fatalf("expected ErrPurchaseAlreadyCancelled on second cancel, got %v", err)
	}
	if invRepo.movementCount != 1 {
		t.Fatalf("expected no additional stock movement on second cancel, got %d", invRepo.movementCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
