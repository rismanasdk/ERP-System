package sales

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaleRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `)).
		WithArgs(int64(1), "S-100", "DRAFT", float64(125.50), nil, int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(11)))

	id, err := repo.CreateSale(context.Background(), &Sale{BranchID: 1, SaleNumber: "S-100", Status: "DRAFT", TotalAmount: 125.50, CreatedBy: 3})
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

func TestSaleRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
    `)).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "sale_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(11), int64(1), "S-100", "DRAFT", float64(125.50), nil, int64(3), now, now,
		))

	sale, err := repo.GetSaleByID(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale == nil || sale.ID != 11 || sale.SaleNumber != "S-100" {
		t.Fatalf("unexpected sale: %+v", sale)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
    `)).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetSaleByID(context.Background(), 999); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleRepository_GetByIDForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
        FOR UPDATE
    `)).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "sale_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(12), int64(2), "S-200", "DRAFT", float64(300.00), nil, int64(4), now, now,
		))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	sale, err := repo.GetSaleByIDForUpdate(context.Background(), tx, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale == nil || sale.ID != 12 || sale.SaleNumber != "S-200" {
		t.Fatalf("unexpected sale: %+v", sale)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleRepository_GetByNumber(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE sale_number = $1
    `)).
		WithArgs("S-100").
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "sale_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(13), int64(1), "S-100", "COMPLETED", float64(150.00), nil, int64(5), now, now,
		))

	sale, err := repo.GetSaleByNumber(context.Background(), "S-100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sale == nil || sale.ID != 13 || sale.SaleNumber != "S-100" {
		t.Fatalf("unexpected sale: %+v", sale)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE sale_number = $1
    `)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetSaleByNumber(context.Background(), "missing"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)
	branchID := int64(2)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
 WHERE branch_id = $1 ORDER BY id ASC`)).
		WithArgs(branchID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "sale_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(20), int64(2), "S-201", "DRAFT", float64(80.00), nil, int64(6), now, now,
		))

	sales, err := repo.ListSales(context.Background(), SaleFilter{BranchID: &branchID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sales) != 1 || sales[0].BranchID != branchID || sales[0].SaleNumber != "S-201" {
		t.Fatalf("unexpected sales list for branch filter: %+v", sales)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
         ORDER BY id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "sale_number", "status", "total_amount", "notes", "created_by", "created_at", "updated_at"}).AddRow(
			int64(21), int64(1), "S-301", "COMPLETED", float64(100.00), nil, int64(7), now, now,
		).AddRow(
			int64(22), int64(2), "S-302", "DRAFT", float64(55.00), nil, int64(8), now, now,
		))

	sales, err = repo.ListSales(context.Background(), SaleFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sales) != 2 {
		t.Fatalf("expected 2 sales without branch filter, got %d: %+v", len(sales), sales)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE sales
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `)).
		WithArgs("COMPLETED", int64(14)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateSaleStatus(context.Background(), 14, "COMPLETED"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE sales
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `)).
		WithArgs("CANCELLED", int64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.UpdateSaleStatus(context.Background(), 999, "CANCELLED"); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleItemRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(11), int64(7), int64(2), float64(50.00), float64(100.00)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)))

	id, err := repo.CreateSaleItem(context.Background(), &SaleItem{SaleID: 11, ProductID: 7, Quantity: 2, UnitPrice: 50.00, Subtotal: 100.00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 101 {
		t.Fatalf("expected id 101, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleItemRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE id = $1
    `)).
		WithArgs(int64(101)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "sale_id", "product_id", "quantity", "unit_price", "subtotal", "created_at", "updated_at"}).AddRow(
			int64(101), int64(11), int64(7), int64(2), float64(50.00), float64(100.00), now, now,
		))

	item, err := repo.GetSaleItemByID(context.Background(), 101)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil || item.ID != 101 || item.SaleID != 11 {
		t.Fatalf("unexpected sale item: %+v", item)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE id = $1
    `)).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	if _, err := repo.GetSaleItemByID(context.Background(), 999); err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaleItemRepository_ListBySaleID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE sale_id = $1
        ORDER BY id ASC
    `)).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "sale_id", "product_id", "quantity", "unit_price", "subtotal", "created_at", "updated_at"}).AddRow(
			int64(101), int64(11), int64(7), int64(2), float64(50.00), float64(100.00), now, now,
		).AddRow(
			int64(102), int64(11), int64(8), int64(1), float64(60.00), float64(60.00), now, now,
		))

	items, err := repo.ListSaleItemsBySaleID(context.Background(), 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].SaleID != 11 || items[1].ProductID != 8 {
		t.Fatalf("unexpected sale-item list: %+v", items)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransaction_CreateSaleAndItem_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `)).
		WithArgs(int64(1), "S-500", "DRAFT", float64(200.00), nil, int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(50)))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(50), int64(9), int64(4), float64(50.00), float64(200.00)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(150)))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	saleID, err := repo.CreateSaleWithTx(context.Background(), tx, &Sale{BranchID: 1, SaleNumber: "S-500", Status: "DRAFT", TotalAmount: 200.00, CreatedBy: 5})
	if err != nil {
		t.Fatalf("unexpected error creating sale: %v", err)
	}
	if saleID != 50 {
		t.Fatalf("expected sale ID 50, got %d", saleID)
	}

	itemID, err := repo.CreateSaleItemWithTx(context.Background(), tx, &SaleItem{SaleID: saleID, ProductID: 9, Quantity: 4, UnitPrice: 50.00, Subtotal: 200.00})
	if err != nil {
		t.Fatalf("unexpected error creating sale item: %v", err)
	}
	if itemID != 150 {
		t.Fatalf("expected item ID 150, got %d", itemID)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestTransaction_CreateSaleAndItem_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `)).
		WithArgs(int64(2), "S-600", "DRAFT", float64(75.00), nil, int64(6)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(60)))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(int64(60), int64(10), int64(3), float64(25.00), float64(75.00)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(160)))
	mock.ExpectRollback()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	saleID, err := repo.CreateSaleWithTx(context.Background(), tx, &Sale{BranchID: 2, SaleNumber: "S-600", Status: "DRAFT", TotalAmount: 75.00, CreatedBy: 6})
	if err != nil {
		t.Fatalf("unexpected error creating sale: %v", err)
	}
	if _, err := repo.CreateSaleItemWithTx(context.Background(), tx, &SaleItem{SaleID: saleID, ProductID: 10, Quantity: 3, UnitPrice: 25.00, Subtotal: 75.00}); err != nil {
		t.Fatalf("unexpected error creating sale item: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback tx: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
