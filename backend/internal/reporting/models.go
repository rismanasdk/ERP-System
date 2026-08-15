package reporting

import "time"

const ReportReadPermission = "reports.read"

type SalesReportQuery struct {
	StartDate *time.Time
	EndDate   *time.Time
	BranchID  *int64
}

type PurchaseReportQuery struct {
	StartDate *time.Time
	EndDate   *time.Time
	BranchID  *int64
}

type InventoryReportQuery struct {
	BranchID  *int64
	ProductID *int64
}

type ProfitReportQuery struct {
	StartDate *time.Time
	EndDate   *time.Time
	BranchID  *int64
}

type DailySalesSummary struct {
	Date              string  `json:"date"`
	TotalSales        float64 `json:"total_sales"`
	TotalTransactions int64   `json:"total_transactions"`
	TotalItemsSold    int64   `json:"total_items_sold"`
	TotalRevenue      float64 `json:"total_revenue"`
}

type SalesReport struct {
	TotalSales          float64             `json:"total_sales"`
	TotalTransactions   int64               `json:"total_transactions"`
	TotalItemsSold      int64               `json:"total_items_sold"`
	TotalRevenue        float64             `json:"total_revenue"`
	TotalCancelledSales int64               `json:"total_cancelled_sales"`
	CancelledSalesValue float64             `json:"cancelled_sales_value"`
	DailySummary        []DailySalesSummary `json:"daily_summary,omitempty"`
}

type DailyPurchaseSummary struct {
	Date                string  `json:"date"`
	TotalPurchases      float64 `json:"total_purchases"`
	TotalTransactions   int64   `json:"total_transactions"`
	TotalPurchasedItems int64   `json:"total_purchased_items"`
	TotalPurchaseAmount float64 `json:"total_purchase_amount"`
	CompletedPurchases  int64   `json:"completed_purchases"`
	CancelledPurchases  int64   `json:"cancelled_purchases"`
}

type PurchasesReport struct {
	TotalPurchases            float64                `json:"total_purchases"`
	TotalPurchaseTransactions int64                  `json:"total_purchase_transactions"`
	TotalPurchasedItems       int64                  `json:"total_purchased_items"`
	TotalPurchaseAmount       float64                `json:"total_purchase_amount"`
	CompletedPurchases        int64                  `json:"completed_purchases"`
	CancelledPurchases        int64                  `json:"cancelled_purchases"`
	DailySummary              []DailyPurchaseSummary `json:"daily_summary,omitempty"`
}

type StockMovementSummary struct {
	InQuantity    int64 `json:"in_quantity"`
	OutQuantity   int64 `json:"out_quantity"`
	AdjustmentQty int64 `json:"adjustment_quantity"`
}

type InventoryReport struct {
	TotalInventoryRecords int64                `json:"total_inventory_records"`
	TotalQuantity         int64                `json:"total_quantity"`
	StockMovementSummary  StockMovementSummary `json:"stock_movement_summary"`
	LowStockProducts      []LowStockProduct    `json:"low_stock_products,omitempty"`
}

type LowStockProduct struct {
	ProductID int64  `json:"product_id"`
	BranchID  int64  `json:"branch_id"`
	Quantity  int64  `json:"quantity"`
	Name      string `json:"name,omitempty"`
}

type ProfitReport struct {
	Revenue        float64 `json:"revenue"`
	PurchasingCost float64 `json:"purchasing_cost"`
	GrossProfit    float64 `json:"gross_profit"`
}
