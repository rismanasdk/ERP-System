package reporting

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSalesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*SalesReport, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN total_amount ELSE 0 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END), 0),
			COALESCE(SUM(CASE WHEN status = 'CANCELLED' THEN total_amount ELSE 0 END), 0)
		FROM sales
	`
	args := []any{}
	clauses := []string{}
	if startDate != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *startDate)
	}
	if endDate != nil {
		clauses = append(clauses, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *endDate)
	}
	if len(branchIDs) > 0 {
		clauses = append(clauses, fmt.Sprintf("branch_id = ANY($%d)", len(args)+1))
		args = append(args, pq.Array(branchIDs))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	var report SalesReport
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&report.TotalSales,
		&report.TotalTransactions,
		&report.TotalCancelledSales,
		&report.CancelledSalesValue,
	); err != nil {
		return nil, err
	}
	report.TotalRevenue = report.TotalSales

	itemQuery := `
		SELECT COALESCE(SUM(si.quantity), 0)
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id
	`
	itemClauses := []string{}
	itemArgs := []any{}
	if startDate != nil {
		itemClauses = append(itemClauses, fmt.Sprintf("s.created_at >= $%d", len(itemArgs)+1))
		itemArgs = append(itemArgs, *startDate)
	}
	if endDate != nil {
		itemClauses = append(itemClauses, fmt.Sprintf("s.created_at < $%d", len(itemArgs)+1))
		itemArgs = append(itemArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		itemClauses = append(itemClauses, fmt.Sprintf("s.branch_id = ANY($%d)", len(itemArgs)+1))
		itemArgs = append(itemArgs, pq.Array(branchIDs))
	}
	if len(itemClauses) > 0 {
		itemQuery += " WHERE " + strings.Join(itemClauses, " AND ")
	}
	itemQuery += " AND s.status = 'COMPLETED'"
	if err := r.db.QueryRowContext(ctx, itemQuery, itemArgs...).Scan(&report.TotalItemsSold); err != nil {
		return nil, err
	}

	dailySalesQuery := `
		SELECT DATE(s.created_at) AS day, COUNT(*) AS transactions, COALESCE(SUM(s.total_amount), 0) AS sales_total
		FROM sales s
	`
	dailyClauses := []string{"s.status = 'COMPLETED'"}
	dailyArgs := []any{}
	if startDate != nil {
		dailyClauses = append(dailyClauses, fmt.Sprintf("s.created_at >= $%d", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, *startDate)
	}
	if endDate != nil {
		dailyClauses = append(dailyClauses, fmt.Sprintf("s.created_at < $%d", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		dailyClauses = append(dailyClauses, fmt.Sprintf("s.branch_id = ANY($%d)", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, pq.Array(branchIDs))
	}
	dailySalesQuery += " WHERE " + strings.Join(dailyClauses, " AND ")
	dailySalesQuery += " GROUP BY DATE(s.created_at) ORDER BY DATE(s.created_at) ASC"
	rows, err := r.db.QueryContext(ctx, dailySalesQuery, dailyArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dailyItemsQuery := `
		SELECT DATE(s.created_at) AS day, COALESCE(SUM(si.quantity), 0) AS items_sold
		FROM sales s
		JOIN sale_items si ON si.sale_id = s.id
	`
	dailyItemClauses := []string{"s.status = 'COMPLETED'"}
	dailyItemArgs := []any{}
	if startDate != nil {
		dailyItemClauses = append(dailyItemClauses, fmt.Sprintf("s.created_at >= $%d", len(dailyItemArgs)+1))
		dailyItemArgs = append(dailyItemArgs, *startDate)
	}
	if endDate != nil {
		dailyItemClauses = append(dailyItemClauses, fmt.Sprintf("s.created_at < $%d", len(dailyItemArgs)+1))
		dailyItemArgs = append(dailyItemArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		dailyItemClauses = append(dailyItemClauses, fmt.Sprintf("s.branch_id = ANY($%d)", len(dailyItemArgs)+1))
		dailyItemArgs = append(dailyItemArgs, pq.Array(branchIDs))
	}
	dailyItemsQuery += " WHERE " + strings.Join(dailyItemClauses, " AND ")
	dailyItemsQuery += " GROUP BY DATE(s.created_at) ORDER BY DATE(s.created_at) ASC"
	itemRows, err := r.db.QueryContext(ctx, dailyItemsQuery, dailyItemArgs...)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	itemsByDay := map[string]int64{}
	for itemRows.Next() {
		var day string
		var items int64
		if err := itemRows.Scan(&day, &items); err != nil {
			return nil, err
		}
		itemsByDay[day] = items
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for rows.Next() {
		var day string
		var transactions int64
		var salesTotal float64
		if err := rows.Scan(&day, &transactions, &salesTotal); err != nil {
			return nil, err
		}
		report.DailySummary = append(report.DailySummary, DailySalesSummary{
			Date:              day,
			TotalSales:        salesTotal,
			TotalTransactions: transactions,
			TotalItemsSold:    itemsByDay[day],
			TotalRevenue:      salesTotal,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &report, nil
}

func (r *Repository) GetPurchasesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*PurchasesReport, error) {
	baseQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN total_amount ELSE 0 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END), 0),
			COALESCE(SUM(CASE WHEN status = 'CANCELLED' THEN total_amount ELSE 0 END), 0)
		FROM purchases
	`
	args := []any{}
	clauses := []string{}
	if startDate != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)+1))
		args = append(args, *startDate)
	}
	if endDate != nil {
		clauses = append(clauses, fmt.Sprintf("created_at < $%d", len(args)+1))
		args = append(args, *endDate)
	}
	if len(branchIDs) > 0 {
		clauses = append(clauses, fmt.Sprintf("branch_id = ANY($%d)", len(args)+1))
		args = append(args, pq.Array(branchIDs))
	}
	if len(clauses) > 0 {
		baseQuery += " WHERE " + strings.Join(clauses, " AND ")
	}

	var report PurchasesReport
	if err := r.db.QueryRowContext(ctx, baseQuery, args...).Scan(
		&report.TotalPurchases,
		&report.CompletedPurchases,
		&report.CancelledPurchases,
		&report.TotalPurchaseAmount,
	); err != nil {
		return nil, err
	}
	report.TotalPurchaseTransactions = report.CompletedPurchases
	report.TotalPurchaseAmount = report.TotalPurchases

	itemQuery := `
		SELECT COALESCE(SUM(pi.quantity), 0)
		FROM purchase_items pi
		JOIN purchases p ON p.id = pi.purchase_id
	`
	itemClauses := []string{}
	itemArgs := []any{}
	if startDate != nil {
		itemClauses = append(itemClauses, fmt.Sprintf("p.created_at >= $%d", len(itemArgs)+1))
		itemArgs = append(itemArgs, *startDate)
	}
	if endDate != nil {
		itemClauses = append(itemClauses, fmt.Sprintf("p.created_at < $%d", len(itemArgs)+1))
		itemArgs = append(itemArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		itemClauses = append(itemClauses, fmt.Sprintf("p.branch_id = ANY($%d)", len(itemArgs)+1))
		itemArgs = append(itemArgs, pq.Array(branchIDs))
	}
	if len(itemClauses) > 0 {
		itemQuery += " WHERE " + strings.Join(itemClauses, " AND ")
	}
	itemQuery += " AND p.status = 'COMPLETED'"
	if err := r.db.QueryRowContext(ctx, itemQuery, itemArgs...).Scan(&report.TotalPurchasedItems); err != nil {
		return nil, err
	}

	dailyQuery := `
		SELECT DATE(p.created_at) AS day, COUNT(*) AS transactions, COALESCE(SUM(p.total_amount), 0) AS total_purchase_amount
		FROM purchases p
	`
	dailyClauses := []string{"p.status = 'COMPLETED'"}
	dailyArgs := []any{}
	if startDate != nil {
		dailyClauses = append(dailyClauses, fmt.Sprintf("p.created_at >= $%d", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, *startDate)
	}
	if endDate != nil {
		dailyClauses = append(dailyClauses, fmt.Sprintf("p.created_at < $%d", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		dailyClauses = append(dailyClauses, fmt.Sprintf("p.branch_id = ANY($%d)", len(dailyArgs)+1))
		dailyArgs = append(dailyArgs, pq.Array(branchIDs))
	}
	dailyQuery += " WHERE " + strings.Join(dailyClauses, " AND ")
	dailyQuery += " GROUP BY DATE(p.created_at) ORDER BY DATE(p.created_at) ASC"
	rows, err := r.db.QueryContext(ctx, dailyQuery, dailyArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemDailyQuery := `
		SELECT DATE(p.created_at) AS day, COALESCE(SUM(pi.quantity), 0) AS items_purchased
		FROM purchases p
		JOIN purchase_items pi ON pi.purchase_id = p.id
	`
	itemDailyClauses := []string{"p.status = 'COMPLETED'"}
	itemDailyArgs := []any{}
	if startDate != nil {
		itemDailyClauses = append(itemDailyClauses, fmt.Sprintf("p.created_at >= $%d", len(itemDailyArgs)+1))
		itemDailyArgs = append(itemDailyArgs, *startDate)
	}
	if endDate != nil {
		itemDailyClauses = append(itemDailyClauses, fmt.Sprintf("p.created_at < $%d", len(itemDailyArgs)+1))
		itemDailyArgs = append(itemDailyArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		itemDailyClauses = append(itemDailyClauses, fmt.Sprintf("p.branch_id = ANY($%d)", len(itemDailyArgs)+1))
		itemDailyArgs = append(itemDailyArgs, pq.Array(branchIDs))
	}
	itemDailyQuery += " WHERE " + strings.Join(itemDailyClauses, " AND ")
	itemDailyQuery += " GROUP BY DATE(p.created_at) ORDER BY DATE(p.created_at) ASC"
	itemRows, err := r.db.QueryContext(ctx, itemDailyQuery, itemDailyArgs...)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	itemsByDay := map[string]int64{}
	for itemRows.Next() {
		var day string
		var items int64
		if err := itemRows.Scan(&day, &items); err != nil {
			return nil, err
		}
		itemsByDay[day] = items
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for rows.Next() {
		var day string
		var transactions int64
		var amount float64
		if err := rows.Scan(&day, &transactions, &amount); err != nil {
			return nil, err
		}
		report.DailySummary = append(report.DailySummary, DailyPurchaseSummary{
			Date:                day,
			TotalPurchases:      amount,
			TotalTransactions:   transactions,
			TotalPurchasedItems: itemsByDay[day],
			TotalPurchaseAmount: amount,
			CompletedPurchases:  transactions,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &report, nil
}

func (r *Repository) GetInventoryReport(ctx context.Context, branchIDs []int64, productID *int64) (*InventoryReport, error) {
	query := `
		SELECT COUNT(*), COALESCE(SUM(quantity), 0)
		FROM inventory
	`
	args := []any{}
	clauses := []string{}
	if len(branchIDs) > 0 {
		clauses = append(clauses, fmt.Sprintf("branch_id = ANY($%d)", len(args)+1))
		args = append(args, pq.Array(branchIDs))
	}
	if productID != nil {
		clauses = append(clauses, fmt.Sprintf("product_id = $%d", len(args)+1))
		args = append(args, *productID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	var report InventoryReport
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&report.TotalInventoryRecords, &report.TotalQuantity); err != nil {
		return nil, err
	}

	movementQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN movement_type = 'IN' THEN ABS(quantity_delta) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN movement_type = 'OUT' THEN ABS(quantity_delta) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN movement_type = 'ADJUSTMENT' THEN ABS(quantity_delta) ELSE 0 END), 0)
		FROM stock_movements
	`
	movementArgs := []any{}
	movementClauses := []string{}
	if len(branchIDs) > 0 {
		movementClauses = append(movementClauses, fmt.Sprintf("branch_id = ANY($%d)", len(movementArgs)+1))
		movementArgs = append(movementArgs, pq.Array(branchIDs))
	}
	if productID != nil {
		movementClauses = append(movementClauses, fmt.Sprintf("product_id = $%d", len(movementArgs)+1))
		movementArgs = append(movementArgs, *productID)
	}
	if len(movementClauses) > 0 {
		movementQuery += " WHERE " + strings.Join(movementClauses, " AND ")
	}
	if err := r.db.QueryRowContext(ctx, movementQuery, movementArgs...).Scan(
		&report.StockMovementSummary.InQuantity,
		&report.StockMovementSummary.OutQuantity,
		&report.StockMovementSummary.AdjustmentQty,
	); err != nil {
		return nil, err
	}

	return &report, nil
}

func (r *Repository) GetProfitReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*ProfitReport, error) {
	report := &ProfitReport{}

	salesQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM sales
		WHERE status = 'COMPLETED'
	`
	salesArgs := []any{}
	salesClauses := []string{}
	if startDate != nil {
		salesClauses = append(salesClauses, fmt.Sprintf("created_at >= $%d", len(salesArgs)+1))
		salesArgs = append(salesArgs, *startDate)
	}
	if endDate != nil {
		salesClauses = append(salesClauses, fmt.Sprintf("created_at < $%d", len(salesArgs)+1))
		salesArgs = append(salesArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		salesClauses = append(salesClauses, fmt.Sprintf("branch_id = ANY($%d)", len(salesArgs)+1))
		salesArgs = append(salesArgs, pq.Array(branchIDs))
	}
	if len(salesClauses) > 0 {
		salesQuery += " AND " + strings.Join(salesClauses, " AND ")
	}
	if err := r.db.QueryRowContext(ctx, salesQuery, salesArgs...).Scan(&report.Revenue); err != nil {
		return nil, err
	}

	purchaseQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM purchases
		WHERE status = 'COMPLETED'
	`
	purchaseArgs := []any{}
	purchaseClauses := []string{}
	if startDate != nil {
		purchaseClauses = append(purchaseClauses, fmt.Sprintf("created_at >= $%d", len(purchaseArgs)+1))
		purchaseArgs = append(purchaseArgs, *startDate)
	}
	if endDate != nil {
		purchaseClauses = append(purchaseClauses, fmt.Sprintf("created_at < $%d", len(purchaseArgs)+1))
		purchaseArgs = append(purchaseArgs, *endDate)
	}
	if len(branchIDs) > 0 {
		purchaseClauses = append(purchaseClauses, fmt.Sprintf("branch_id = ANY($%d)", len(purchaseArgs)+1))
		purchaseArgs = append(purchaseArgs, pq.Array(branchIDs))
	}
	if len(purchaseClauses) > 0 {
		purchaseQuery += " AND " + strings.Join(purchaseClauses, " AND ")
	}
	if err := r.db.QueryRowContext(ctx, purchaseQuery, purchaseArgs...).Scan(&report.PurchasingCost); err != nil {
		return nil, err
	}

	report.GrossProfit = report.Revenue - report.PurchasingCost
	return report, nil
}
