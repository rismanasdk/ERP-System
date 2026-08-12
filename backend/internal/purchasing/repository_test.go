package purchasing

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSupplierRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO suppliers (name, code, phone, email, address, is_active)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `)).
		WithArgs("Acme", "ACME", nil, nil, nil, true).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	id, err := repo.CreateSupplier(context.Background(), &Supplier{Name: "Acme", Code: "ACME", IsActive: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 10 {
		t.Fatalf("expected id 10, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
        WHERE id = $1
    `)).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "phone", "email", "address", "is_active", "created_at", "updated_at"}).AddRow(
			int64(5), "Acme", "ACME", nil, nil, nil, true, now, now,
		))

	supplier, err := repo.GetSupplierByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if supplier == nil || supplier.ID != 5 || supplier.Code != "ACME" {
		t.Fatalf("unexpected supplier: %+v", supplier)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierRepository_GetByCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
        WHERE code = $1
    `)).
		WithArgs("ACME").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "phone", "email", "address", "is_active", "created_at", "updated_at"}).AddRow(
			int64(6), "Acme", "ACME", nil, nil, nil, true, now, now,
		))

	supplier, err := repo.GetSupplierByCode(context.Background(), "ACME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if supplier == nil || supplier.ID != 6 || supplier.Code != "ACME" {
		t.Fatalf("unexpected supplier: %+v", supplier)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)
	active := true
	search := "Acme"

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
     WHERE is_active = $1 AND (name ILIKE $2 OR code ILIKE $3) ORDER BY name ASC`)).
		WithArgs(active, "%Acme%", "%Acme%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "phone", "email", "address", "is_active", "created_at", "updated_at"}).AddRow(
			int64(7), "Acme", "ACME", nil, nil, nil, true, now, now,
		))

	suppliers, err := repo.ListSuppliers(context.Background(), SupplierFilter{Active: &active, Search: &search})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(suppliers) != 1 || suppliers[0].ID != 7 {
		t.Fatalf("unexpected suppliers: %+v", suppliers)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE suppliers
        SET name = $1, code = $2, phone = $3, email = $4, address = $5, is_active = $6, updated_at = NOW()
        WHERE id = $7
    `)).
		WithArgs("Acme", "ACME", nil, nil, nil, true, int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateSupplier(context.Background(), &Supplier{ID: 8, Name: "Acme", Code: "ACME", IsActive: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `)).
		WithArgs(int64(1), int64(2), "PO-100", "DRAFT", float64(100.00), nil, int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(11)))

	id, err := repo.CreatePurchase(context.Background(), &Purchase{BranchID: 1, SupplierID: 2, PurchaseNumber: "PO-100", Status: "DRAFT", TotalAmount: 100.00, CreatedBy: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 11 {
		t.Fatalf("expected id 11, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
        WHERE id = $1
    `)).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "supplier_id", "purchase_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(11), int64(1), int64(2), "PO-100", "DRAFT", float64(100.00), nil, int64(3), now, now,
		))

	purchase, err := repo.GetPurchaseByID(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if purchase == nil || purchase.ID != 11 || purchase.PurchaseNumber != "PO-100" {
		t.Fatalf("unexpected purchase: %+v", purchase)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseRepository_GetByNumber(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
        WHERE purchase_number = $1
    `)).
		WithArgs("PO-100").
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "supplier_id", "purchase_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(12), int64(1), int64(2), "PO-100", "DRAFT", float64(100.00), nil, int64(3), now, now,
		))

	purchase, err := repo.GetPurchaseByNumber(context.Background(), "PO-100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if purchase == nil || purchase.ID != 12 || purchase.PurchaseNumber != "PO-100" {
		t.Fatalf("unexpected purchase: %+v", purchase)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)
	branchID := int64(1)
	supplierID := int64(2)
	status := "DRAFT"

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
     WHERE branch_id = $1 AND supplier_id = $2 AND status = $3 ORDER BY id ASC`)).
		WithArgs(branchID, supplierID, status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "supplier_id", "purchase_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(13), branchID, supplierID, "PO-101", status, float64(150.00), nil, int64(3), now, now,
		))

	purchases, err := repo.ListPurchases(context.Background(), PurchaseFilter{BranchID: &branchID, SupplierID: &supplierID, Status: &status})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(purchases) != 1 || purchases[0].ID != 13 || purchases[0].BranchID != 1 {
		t.Fatalf("unexpected purchases: %+v", purchases)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE purchases
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `)).
		WithArgs("COMPLETED", int64(14)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdatePurchaseStatus(context.Background(), 14, "COMPLETED"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseItemRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(11), int64(20), int64(5), float64(10.00), float64(50.00)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(21)))

	id, err := repo.CreatePurchaseItem(context.Background(), &PurchaseItem{PurchaseID: 11, ProductID: 20, Quantity: 5, UnitCost: 10.00, Subtotal: 50.00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 21 {
		t.Fatalf("expected id 21, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseItemRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, created_at, updated_at
        FROM purchase_items
        WHERE id = $1
    `)).
		WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "purchase_id", "product_id", "quantity", "unit_cost", "subtotal", "created_at", "updated_at"}).AddRow(
			int64(21), int64(11), int64(20), int64(5), float64(10.00), float64(50.00), now, now,
		))

	item, err := repo.GetPurchaseItemByID(context.Background(), 21)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil || item.ID != 21 || item.PurchaseID != 11 {
		t.Fatalf("unexpected purchase item: %+v", item)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPurchaseItemRepository_ListByPurchaseID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, created_at, updated_at
        FROM purchase_items
        WHERE purchase_id = $1
        ORDER BY id ASC
    `)).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "purchase_id", "product_id", "quantity", "unit_cost", "subtotal", "created_at", "updated_at"}).AddRow(
			int64(21), int64(11), int64(20), int64(5), float64(10.00), float64(50.00), now, now,
		))

	items, err := repo.ListPurchaseItemsByPurchaseID(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].PurchaseID != 11 {
		t.Fatalf("unexpected purchase items: %+v", items)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransaction_CreatePurchaseAndItem_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `)).
		WithArgs(int64(1), int64(2), "PO-200", "DRAFT", float64(100.00), nil, int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(31)))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(31), int64(20), int64(2), float64(25.00), float64(50.00)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(41)))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	purchaseID, err := repo.CreatePurchaseWithTx(context.Background(), tx, &Purchase{BranchID: 1, SupplierID: 2, PurchaseNumber: "PO-200", Status: "DRAFT", TotalAmount: 100.00, CreatedBy: 3})
	if err != nil {
		t.Fatalf("unexpected error creating purchase: %v", err)
	}
	if purchaseID != 31 {
		t.Fatalf("expected purchase id 31, got %d", purchaseID)
	}

	itemID, err := repo.CreatePurchaseItemWithTx(context.Background(), tx, &PurchaseItem{PurchaseID: purchaseID, ProductID: 20, Quantity: 2, UnitCost: 25.00, Subtotal: 50.00})
	if err != nil {
		t.Fatalf("unexpected error creating purchase item: %v", err)
	}
	if itemID != 41 {
		t.Fatalf("expected item id 41, got %d", itemID)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransaction_CreatePurchaseAndItem_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `)).
		WithArgs(int64(1), int64(2), "PO-300", "DRAFT", float64(120.00), nil, int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(32)))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(32), int64(20), int64(0), float64(25.00), float64(0.00)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	_, err = repo.CreatePurchaseWithTx(context.Background(), tx, &Purchase{BranchID: 1, SupplierID: 2, PurchaseNumber: "PO-300", Status: "DRAFT", TotalAmount: 120.00, CreatedBy: 3})
	if err != nil {
		t.Fatalf("unexpected error creating purchase: %v", err)
	}

	_, err = repo.CreatePurchaseItemWithTx(context.Background(), tx, &PurchaseItem{PurchaseID: 32, ProductID: 20, Quantity: 0, UnitCost: 25.00, Subtotal: 0.00})
	if err == nil {
		t.Fatalf("expected error creating invalid purchase item")
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		t.Fatalf("failed to rollback tx: %v", rollbackErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
