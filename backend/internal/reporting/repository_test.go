package reporting

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRepository_GetSalesReport_AggregatesByDateAndBranch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	branchIDs := []int64{5, 7}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT
			COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN total_amount ELSE 0 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END), 0),
			COALESCE(SUM(CASE WHEN status = 'CANCELLED' THEN total_amount ELSE 0 END), 0)
		FROM sales
	 WHERE created_at >= $1 AND created_at < $2 AND branch_id = ANY($3)`)).
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"sum", "count", "cancelled_count", "cancelled_value"}).AddRow(150.00, 2, 1, 10.00))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(SUM(si.quantity), 0)
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id
	 WHERE s.created_at >= $1 AND s.created_at < $2 AND s.branch_id = ANY($3) AND s.status = 'COMPLETED'`)).
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(12)))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DATE(s.created_at) AS day, COUNT(*) AS transactions, COALESCE(SUM(s.total_amount), 0) AS sales_total
		FROM sales s
	 WHERE s.status = 'COMPLETED' AND s.created_at >= $1 AND s.created_at < $2 AND s.branch_id = ANY($3) GROUP BY DATE(s.created_at) ORDER BY DATE(s.created_at) ASC`)).
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"day", "transactions", "sales_total"}).AddRow("2026-01-15", 2, 150.00))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DATE(s.created_at) AS day, COALESCE(SUM(si.quantity), 0) AS items_sold
		FROM sales s
		JOIN sale_items si ON si.sale_id = s.id
	 WHERE s.status = 'COMPLETED' AND s.created_at >= $1 AND s.created_at < $2 AND s.branch_id = ANY($3) GROUP BY DATE(s.created_at) ORDER BY DATE(s.created_at) ASC`)).
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"day", "items_sold"}).AddRow("2026-01-15", 12))

	_, err = repo.GetSalesReport(context.Background(), &start, &end, branchIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRepository_GetInventoryReport_AggregatesStockMovements(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	branchIDs := []int64{3}
	productID := int64(8)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*), COALESCE(SUM(quantity), 0)
		FROM inventory
	 WHERE branch_id = ANY($1) AND product_id = $2`)).
		WithArgs(sqlmock.AnyArg(), productID).
		WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(5, 50))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT
			COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN ABS(quantity_delta) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN movement_type = 'OUT' THEN ABS(quantity_delta) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN movement_type = 'ADJUSTMENT' THEN ABS(quantity_delta) ELSE 0 END), 0)
		FROM stock_movements
	 WHERE branch_id = ANY($1) AND product_id = $2`)).
		WithArgs(sqlmock.AnyArg(), productID).
		WillReturnRows(sqlmock.NewRows([]string{"in_count", "out_count", "adj_count"}).AddRow(10, 8, 2))

	_, err = repo.GetInventoryReport(context.Background(), branchIDs, &productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
