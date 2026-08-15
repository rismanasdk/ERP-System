package dashboard

import (
	"context"
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSalesSummary(ctx context.Context, branchID *int64) (*SalesSummary, error) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= $1 AND created_at < $2 THEN total_amount ELSE 0 END), 0) AS today_amount,
			COALESCE(COUNT(CASE WHEN created_at >= $1 AND created_at < $2 THEN 1 END), 0) AS today_transactions,
			COALESCE(SUM(CASE WHEN created_at >= $3 AND created_at < $4 THEN total_amount ELSE 0 END), 0) AS month_amount,
			COALESCE(COUNT(CASE WHEN created_at >= $3 AND created_at < $4 THEN 1 END), 0) AS month_transactions
		FROM sales
		WHERE status = 'COMPLETED'`
	args := []any{startOfToday, endOfToday, startOfMonth, endOfMonth}

	if branchID == nil {
		query += " AND branch_id IN (SELECT id FROM branches WHERE is_active = true)"
	} else {
		query += " AND branch_id = $5"
		args = append(args, *branchID)
	}

	var summary SalesSummary
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TodayAmount,
		&summary.TodayTransactions,
		&summary.MonthAmount,
		&summary.MonthTransactions,
	); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *Repository) GetPurchasesSummary(ctx context.Context, branchID *int64) (*PurchasesSummary, error) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= $1 AND created_at < $2 THEN total_amount ELSE 0 END), 0) AS today_amount,
			COALESCE(COUNT(CASE WHEN created_at >= $1 AND created_at < $2 THEN 1 END), 0) AS today_transactions,
			COALESCE(SUM(CASE WHEN created_at >= $3 AND created_at < $4 THEN total_amount ELSE 0 END), 0) AS month_amount,
			COALESCE(COUNT(CASE WHEN created_at >= $3 AND created_at < $4 THEN 1 END), 0) AS month_transactions
		FROM purchases
		WHERE status = 'COMPLETED'`
	args := []any{startOfToday, endOfToday, startOfMonth, endOfMonth}

	if branchID == nil {
		query += " AND branch_id IN (SELECT id FROM branches WHERE is_active = true)"
	} else {
		query += " AND branch_id = $5"
		args = append(args, *branchID)
	}

	var summary PurchasesSummary
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TodayAmount,
		&summary.TodayTransactions,
		&summary.MonthAmount,
		&summary.MonthTransactions,
	); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *Repository) GetActiveProductsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM products
		WHERE is_active = true AND deleted_at IS NULL
	`).Scan(&count)
	return count, err
}

func (r *Repository) GetActiveCustomersCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM customers
		WHERE is_active = true AND deleted_at IS NULL
	`).Scan(&count)
	return count, err
}

func (r *Repository) GetActiveSuppliersCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM suppliers
		WHERE is_active = true AND deleted_at IS NULL
	`).Scan(&count)
	return count, err
}

func (r *Repository) GetInventoryCount(ctx context.Context, branchID *int64) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM inventory i`
	args := []any{}

	if branchID == nil {
		query += ` WHERE EXISTS (
			SELECT 1
			FROM branches b
			WHERE b.id = i.branch_id AND b.is_active = true
		)`
	} else {
		query += ` WHERE branch_id = $1`
		args = append(args, *branchID)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}
