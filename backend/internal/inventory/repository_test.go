package inventory

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE id = $1
    `)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "branch_id", "quantity", "created_at", "updated_at"}).AddRow(
			int64(1), int64(2), int64(3), int64(100), now, now,
		))

	inv, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil || inv.ID != 1 || inv.Quantity != 100 {
		t.Fatalf("unexpected inventory row: %+v", inv)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_GetByProductAndBranchForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE product_id = $1 AND branch_id = $2
        FOR UPDATE
    `)).
		WithArgs(int64(2), int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "branch_id", "quantity", "created_at", "updated_at"}).AddRow(
			int64(5), int64(2), int64(3), int64(20), time.Now(), time.Now(),
		))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	inv, err := repo.GetByProductAndBranchForUpdate(context.Background(), tx, 2, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil || inv.ProductID != 2 || inv.BranchID != 3 {
		t.Fatalf("unexpected inventory row: %+v", inv)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_ListFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	branchID := int64(10)
	productID := int64(20)
	now := time.Now().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
 WHERE branch_id = $1 AND product_id = $2 ORDER BY id ASC`)).
		WithArgs(branchID, productID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "branch_id", "quantity", "created_at", "updated_at"}).AddRow(
			int64(7), productID, branchID, int64(50), now, now,
		))

	items, err := repo.List(context.Background(), &branchID, &productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].BranchID != branchID || items[0].ProductID != productID {
		t.Fatalf("unexpected inventory list: %+v", items)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_CreateWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO inventory (product_id, branch_id, quantity)
        VALUES ($1, $2, $3)
        RETURNING id
    `)).
		WithArgs(int64(11), int64(22), int64(33)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	id, err := repo.CreateWithTx(context.Background(), tx, &Inventory{ProductID: 11, BranchID: 22, Quantity: 33})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Fatalf("expected id 99, got %d", id)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_UpdateQuantityWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE inventory
        SET quantity = $1, updated_at = NOW()
        WHERE id = $2
    `)).
		WithArgs(int64(200), int64(5)).
		WillReturnResult(driver.RowsAffected(1))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	if err := repo.UpdateQuantityWithTx(context.Background(), tx, 5, 200); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_CreateMovementWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO stock_movements (
            product_id,
            branch_id,
            movement_type,
            quantity_delta,
            reference_type,
            reference_id,
            actor_user_id,
            metadata
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `)).
		WithArgs(int64(2), int64(3), "adjustment", int64(15), nil, nil, nil, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(101)))
	mock.ExpectCommit()

	tx, err := repo.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	movement := &StockMovement{
		ProductID:     2,
		BranchID:      3,
		MovementType:  "adjustment",
		QuantityDelta: 15,
		Metadata:      map[string]any{"reason": "correction"},
	}
	id, err := repo.CreateMovementWithTx(context.Background(), tx, movement)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 101 {
		t.Fatalf("expected id 101, got %d", id)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit tx: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
