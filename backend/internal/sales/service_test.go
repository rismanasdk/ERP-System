package sales

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/products"

	"github.com/DATA-DOG/go-sqlmock"
)

type fakeSaleRepo struct {
	db            *sql.DB
	mu            sync.Mutex
	sales         map[int64]*Sale
	nextSaleID    int64
	nextItemID    int64
	itemsBySaleID map[int64][]SaleItem
}

func newFakeSaleRepo() *fakeSaleRepo {
	return &fakeSaleRepo{
		sales:         make(map[int64]*Sale),
		itemsBySaleID: make(map[int64][]SaleItem),
		nextSaleID:    1,
		nextItemID:    1,
	}
}

func (r *fakeSaleRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if r.db == nil {
		return nil, errors.New("db is not configured for this test")
	}
	return r.db.BeginTx(ctx, nil)
}

func (r *fakeSaleRepo) CreateSaleWithTx(ctx context.Context, tx *sql.Tx, sale *Sale) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sale.ID = r.nextSaleID
	r.nextSaleID++
	r.sales[sale.ID] = sale
	return sale.ID, nil
}

func (r *fakeSaleRepo) CreateSaleItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleItem) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item.ID = r.nextItemID
	r.nextItemID++
	r.itemsBySaleID[item.SaleID] = append(r.itemsBySaleID[item.SaleID], *item)
	return item.ID, nil
}

func (r *fakeSaleRepo) GetSaleByID(ctx context.Context, id int64) (*Sale, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sale, ok := r.sales[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	clone := *sale
	return &clone, nil
}

func (r *fakeSaleRepo) GetSaleByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Sale, error) {
	return r.GetSaleByID(ctx, id)
}

func (r *fakeSaleRepo) GetSaleByNumber(ctx context.Context, number string) (*Sale, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sale := range r.sales {
		if sale.SaleNumber == number {
			clone := *sale
			return &clone, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *fakeSaleRepo) ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]Sale, 0)
	for _, sale := range r.sales {
		if filter.BranchID != nil && sale.BranchID != *filter.BranchID {
			continue
		}
		result = append(result, *sale)
	}
	return result, nil
}

func (r *fakeSaleRepo) ListSaleItemsBySaleID(ctx context.Context, saleID int64) ([]SaleItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := append([]SaleItem(nil), r.itemsBySaleID[saleID]...)
	return items, nil
}

func (r *fakeSaleRepo) ListSaleItemsBySaleIDWithTx(ctx context.Context, tx *sql.Tx, saleID int64) ([]SaleItem, error) {
	return r.ListSaleItemsBySaleID(ctx, saleID)
}

func (r *fakeSaleRepo) UpdateSaleStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sale, ok := r.sales[id]
	if !ok {
		return sql.ErrNoRows
	}
	sale.Status = status
	sale.UpdatedAt = time.Now()
	return nil
}

type fakeInventoryRepo struct {
	mu          sync.Mutex
	quantities  map[int64]int64
	movements   []*inventory.StockMovement
	updateErr   error
	movementErr error
}

func newFakeInventoryRepo() *fakeInventoryRepo {
	return &fakeInventoryRepo{quantities: make(map[int64]int64)}
}

func (r *fakeInventoryRepo) GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*inventory.Inventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := productID*100000 + branchID
	if qty, ok := r.quantities[key]; ok {
		return &inventory.Inventory{ID: key, ProductID: productID, BranchID: branchID, Quantity: qty}, nil
	}
	return nil, sql.ErrNoRows
}

func (r *fakeInventoryRepo) UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	for k := range r.quantities {
		if k == id {
			r.quantities[k] = quantity
			return nil
		}
	}
	return nil
}

func (r *fakeInventoryRepo) CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *inventory.StockMovement) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.movementErr != nil {
		return 0, r.movementErr
	}
	r.movements = append(r.movements, movement)
	return 1, nil
}

type fakeProductService struct {
	products map[int64]*products.Product
	err      error
}

func (f *fakeProductService) GetByID(ctx context.Context, id int64) (*products.Product, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.products[id]
	if !ok {
		return nil, products.ErrProductNotFound
	}
	return p, nil
}

type fakeBranchService struct {
	err             error
	allowedBranches []branches.Branch
}

func (f *fakeBranchService) EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error {
	if f.err != nil {
		return f.err
	}
	for _, b := range f.allowedBranches {
		if b.ID == branchID {
			if requireActive && !b.IsActive {
				return branches.ErrBranchInactive
			}
			return nil
		}
	}
	return branches.ErrBranchAccessDenied
}

func (f *fakeBranchService) ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error) {
	return f.allowedBranches, f.err
}

type fakeAuthChecker struct {
	allowed bool
	err     error
}

func (f *fakeAuthChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return f.allowed, f.err
}

type fakeAuditService struct {
	err     error
	lastLog audit.AuditLog
}

func (f *fakeAuditService) RecordWithTx(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) (int64, error) {
	f.lastLog = auditLog
	return 1, f.err
}

func newSalesServiceWithDB(t *testing.T, repo *fakeSaleRepo, invRepo *fakeInventoryRepo, auditSvc auditService) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	repo.db = db
	service := NewService(
		repo,
		invRepo,
		&fakeProductService{products: map[int64]*products.Product{1: {ID: 1, IsActive: true}}},
		&fakeBranchService{allowedBranches: []branches.Branch{{ID: 3, IsActive: true}}},
		&fakeAuthChecker{allowed: true},
		auditSvc,
	)
	return service, mock
}

func TestCreateSale_Success(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 7)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	mock.ExpectBegin()
	mock.ExpectCommit()

	id, err := service.CreateSale(ctx, CreateSaleInput{BranchID: 3, Items: []CreateSaleItemInput{{ProductID: 1, Quantity: 2, UnitPrice: 25.5}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected sale id > 0")
	}
	if sale, err := repo.GetSaleByID(ctx, id); err != nil || sale.Status != SaleStatusDraft || sale.TotalAmount != 51.0 {
		t.Fatalf("unexpected saved sale: %+v err=%v", sale, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateSale_AuditFailureRollsBack(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 7)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	auditSvc := &fakeAuditService{err: errors.New("audit fail")}
	service, mock := newSalesServiceWithDB(t, repo, invRepo, auditSvc)
	mock.ExpectBegin()
	mock.ExpectRollback()

	_, err := service.CreateSale(ctx, CreateSaleInput{BranchID: 3, Items: []CreateSaleItemInput{{ProductID: 1, Quantity: 1, UnitPrice: 10}}})
	if err == nil || !errors.Is(err, auditSvc.err) {
		t.Fatalf("expected audit error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetSale_Success(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 42)
	repo := newFakeSaleRepo()
	repo.sales[10] = &Sale{ID: 10, BranchID: 4, SaleNumber: "SALE-1", Status: SaleStatusDraft, TotalAmount: 30, CreatedBy: 42}
	repo.itemsBySaleID[10] = []SaleItem{{ID: 100, SaleID: 10, ProductID: 1, Quantity: 2, UnitPrice: 15, Subtotal: 30}}
	service := NewService(repo, newFakeInventoryRepo(), &fakeProductService{products: map[int64]*products.Product{1: {ID: 1, IsActive: true}}}, &fakeBranchService{allowedBranches: []branches.Branch{{ID: 4, IsActive: true}}}, &fakeAuthChecker{allowed: true}, &fakeAuditService{})

	sale, items, err := service.GetSale(ctx, 10)
	if err != nil || sale == nil || len(items) != 1 {
		t.Fatalf("unexpected sale or items: sale=%+v items=%+v err=%v", sale, items, err)
	}
}

func TestCompleteSale_DraftSuccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[20] = &Sale{ID: 20, BranchID: 3, SaleNumber: "SALE-20", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[20] = []SaleItem{{ID: 200, SaleID: 20, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 10
	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := service.CompleteSale(ctx, 20); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.sales[20].Status != SaleStatusCompleted {
		t.Fatalf("sale status should be COMPLETED")
	}
	if len(invRepo.movements) != 1 {
		t.Fatalf("expected 1 OUT stock movement, got %d", len(invRepo.movements))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompleteSale_InsufficientStock(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[23] = &Sale{ID: 23, BranchID: 3, SaleNumber: "SALE-23", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[23] = []SaleItem{{ID: 203, SaleID: 23, ProductID: 1, Quantity: 5, UnitPrice: 20, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 4
	mock.ExpectBegin()
	mock.ExpectRollback()

	if err := service.CompleteSale(ctx, 23); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelSale_DraftSuccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 6)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[30] = &Sale{ID: 30, BranchID: 3, SaleNumber: "SALE-30", Status: SaleStatusDraft, TotalAmount: 50, CreatedBy: 6}
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE sales\s+SET status = \$1, updated_at = NOW\(\)\s+WHERE id = \$2`).
		WithArgs(SaleStatusCancelled, int64(30)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := service.CancelSale(ctx, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelSale_CompletedRestoresInventory(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 6)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[31] = &Sale{ID: 31, BranchID: 3, SaleNumber: "SALE-31", Status: SaleStatusCompleted, TotalAmount: 100, CreatedBy: 6}
	repo.itemsBySaleID[31] = []SaleItem{{ID: 301, SaleID: 31, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 8
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE sales\s+SET status = \$1, updated_at = NOW\(\)\s+WHERE id = \$2`).
		WithArgs(SaleStatusCancelled, int64(31)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := service.CancelSale(ctx, 31); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.movements) != 1 {
		t.Fatalf("expected one reversal movement, got %d", len(invRepo.movements))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompleteSale_InventoryUpdateFailureRollsBack(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	invRepo.updateErr = errors.New("inventory update failed")
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[40] = &Sale{ID: 40, BranchID: 3, SaleNumber: "SALE-40", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[40] = []SaleItem{{ID: 400, SaleID: 40, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 10
	mock.ExpectBegin()
	mock.ExpectRollback()

	if err := service.CompleteSale(ctx, 40); !errors.Is(err, invRepo.updateErr) {
		t.Fatalf("expected inventory update error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompleteSale_StockMovementFailureRollsBack(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	invRepo.movementErr = errors.New("movement failed")
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[41] = &Sale{ID: 41, BranchID: 3, SaleNumber: "SALE-41", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[41] = []SaleItem{{ID: 401, SaleID: 41, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 10
	mock.ExpectBegin()
	mock.ExpectRollback()

	if err := service.CompleteSale(ctx, 41); !errors.Is(err, invRepo.movementErr) {
		t.Fatalf("expected movement error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompleteSale_AuditFailureRollsBack(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	auditSvc := &fakeAuditService{err: errors.New("audit fail")}
	service, mock := newSalesServiceWithDB(t, repo, invRepo, auditSvc)
	repo.sales[42] = &Sale{ID: 42, BranchID: 3, SaleNumber: "SALE-42", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[42] = []SaleItem{{ID: 402, SaleID: 42, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 10
	mock.ExpectBegin()
	mock.ExpectRollback()

	if err := service.CompleteSale(ctx, 42); !errors.Is(err, auditSvc.err) {
		t.Fatalf("expected audit error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelSale_AuditFailureRollsBack(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 6)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	auditSvc := &fakeAuditService{err: errors.New("audit fail")}
	service, mock := newSalesServiceWithDB(t, repo, invRepo, auditSvc)
	repo.sales[43] = &Sale{ID: 43, BranchID: 3, SaleNumber: "SALE-43", Status: SaleStatusCompleted, TotalAmount: 100, CreatedBy: 6}
	repo.itemsBySaleID[43] = []SaleItem{{ID: 403, SaleID: 43, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 8
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE sales\s+SET status = \$1, updated_at = NOW\(\)\s+WHERE id = \$2`).
		WithArgs(SaleStatusCancelled, int64(43)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	if err := service.CancelSale(ctx, 43); !errors.Is(err, auditSvc.err) {
		t.Fatalf("expected audit error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCompleteSale_DuplicateCompletionPrevention(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 5)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[44] = &Sale{ID: 44, BranchID: 3, SaleNumber: "SALE-44", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 5}
	repo.itemsBySaleID[44] = []SaleItem{{ID: 404, SaleID: 44, ProductID: 1, Quantity: 2, UnitPrice: 50, Subtotal: 100}}
	invRepo.quantities[1*100000+3] = 10
	mock.ExpectBegin()
	mock.ExpectCommit()
	if err := service.CompleteSale(ctx, 44); err != nil {
		t.Fatalf("unexpected error on first completion: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := service.CompleteSale(ctx, 44); !errors.Is(err, ErrSaleAlreadyCompleted) {
		t.Fatalf("expected duplicate completion error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCancelSale_DuplicateCancellationPrevention(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 6)
	repo := newFakeSaleRepo()
	invRepo := newFakeInventoryRepo()
	service, mock := newSalesServiceWithDB(t, repo, invRepo, &fakeAuditService{})
	repo.sales[45] = &Sale{ID: 45, BranchID: 3, SaleNumber: "SALE-45", Status: SaleStatusDraft, TotalAmount: 100, CreatedBy: 6}
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE sales\s+SET status = \$1, updated_at = NOW\(\)\s+WHERE id = \$2`).
		WithArgs(SaleStatusCancelled, int64(45)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := service.CancelSale(ctx, 45); err != nil {
		t.Fatalf("unexpected first cancel error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	repo.sales[45].Status = SaleStatusCancelled
	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := service.CancelSale(ctx, 45); !errors.Is(err, ErrSaleAlreadyCancelled) {
		t.Fatalf("expected duplicate cancellation error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetSale_ForbiddenBranchAccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 7)
	repo := newFakeSaleRepo()
	repo.sales[46] = &Sale{ID: 46, BranchID: 99, SaleNumber: "SALE-46", Status: SaleStatusDraft, TotalAmount: 50, CreatedBy: 7}
	service := NewService(repo, newFakeInventoryRepo(), &fakeProductService{products: map[int64]*products.Product{}}, &fakeBranchService{allowedBranches: []branches.Branch{}}, &fakeAuthChecker{allowed: true}, &fakeAuditService{})

	if _, _, err := service.GetSale(ctx, 46); !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected branch access denied, got %v", err)
	}
}

func TestListSales_ForbiddenBranchAccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 7)
	repo := newFakeSaleRepo()
	service := NewService(repo, newFakeInventoryRepo(), &fakeProductService{products: map[int64]*products.Product{}}, &fakeBranchService{err: branches.ErrBranchAccessDenied}, &fakeAuthChecker{allowed: true}, &fakeAuditService{})

	if _, err := service.ListSales(ctx, SaleFilter{BranchID: int64Ptr(9)}); !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected branch access denied, got %v", err)
	}
}

func TestListSales_MultipleAccessibleBranches(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 7)
	repo := newFakeSaleRepo()
	repo.sales[50] = &Sale{ID: 50, BranchID: 10, SaleNumber: "SALE-50", Status: SaleStatusDraft, TotalAmount: 10, CreatedBy: 7}
	repo.sales[51] = &Sale{ID: 51, BranchID: 11, SaleNumber: "SALE-51", Status: SaleStatusDraft, TotalAmount: 20, CreatedBy: 7}
	service := NewService(repo, newFakeInventoryRepo(), &fakeProductService{products: map[int64]*products.Product{}}, &fakeBranchService{allowedBranches: []branches.Branch{{ID: 10, IsActive: true}, {ID: 11, IsActive: true}}}, &fakeAuthChecker{allowed: true}, &fakeAuditService{})

	salesList, err := service.ListSales(ctx, SaleFilter{})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(salesList) != 2 {
		t.Fatalf("expected 2 sales across accessible branches, got %d", len(salesList))
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
