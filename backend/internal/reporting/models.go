package reporting

import "time"

const ReportReadPermission = "reports.read"

type SalesReportQuery struct {
	StartDate  *time.Time
	EndDate    *time.Time
	BranchID   *int64
	CustomerID *int64
	Status     *string
}

type PurchaseReportQuery struct {
	StartDate  *time.Time
	EndDate    *time.Time
	BranchID   *int64
	SupplierID *int64
	Status     *string
}

type InventoryReportQuery struct {
	BranchID    *int64
	ProductID   *int64
	CategoryID  *int64
	StockStatus *string
}

type PaymentReportQuery struct {
	StartDate     *time.Time
	EndDate       *time.Time
	BranchID      *int64
	PaymentMethod *string
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

type SalesReportRow struct {
	ID            int64     `json:"id"`
	OrderNumber   string    `json:"order_number"`
	Date          time.Time `json:"date"`
	CustomerName  string    `json:"customer_name,omitempty"`
	BranchName    string    `json:"branch_name,omitempty"`
	Status        string    `json:"status"`
	TotalAmount   float64   `json:"total_amount"`
	CreatedBy     int64     `json:"created_by,omitempty"`
	CreatedByName string    `json:"created_by_name,omitempty"`
}

type SalesReport struct {
	TotalSales           float64             `json:"total_sales"`
	TotalOrders          int64               `json:"total_orders"`
	TotalOrderedQuantity int64               `json:"total_ordered_quantity"`
	TotalTransactions    int64               `json:"total_transactions"`
	TotalItemsSold       int64               `json:"total_items_sold"`
	TotalRevenue         float64             `json:"total_revenue"`
	TotalCancelledSales  int64               `json:"total_cancelled_sales"`
	CancelledSalesValue  float64             `json:"cancelled_sales_value"`
	TotalFulfilledAmount float64             `json:"total_fulfilled_amount"`
	StatusBreakdown      map[string]int64    `json:"status_breakdown,omitempty"`
	Rows                 []SalesReportRow    `json:"rows,omitempty"`
	DailySummary         []DailySalesSummary `json:"daily_summary,omitempty"`
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

type PurchaseReportRow struct {
	ID            int64     `json:"id"`
	OrderNumber   string    `json:"order_number"`
	Date          time.Time `json:"date"`
	SupplierName  string    `json:"supplier_name,omitempty"`
	BranchName    string    `json:"branch_name,omitempty"`
	Status        string    `json:"status"`
	TotalAmount   float64   `json:"total_amount"`
	CreatedBy     int64     `json:"created_by,omitempty"`
	CreatedByName string    `json:"created_by_name,omitempty"`
}

type PurchasesReport struct {
	TotalPurchases            float64                `json:"total_purchases"`
	TotalPurchaseOrders       int64                  `json:"total_purchase_orders"`
	TotalPurchaseTransactions int64                  `json:"total_purchase_transactions"`
	TotalPurchasedItems       int64                  `json:"total_purchased_items"`
	TotalPurchaseAmount       float64                `json:"total_purchase_amount"`
	CompletedPurchases        int64                  `json:"completed_purchases"`
	CancelledPurchases        int64                  `json:"cancelled_purchases"`
	TotalReceivedAmount       float64                `json:"total_received_amount"`
	StatusBreakdown           map[string]int64       `json:"status_breakdown,omitempty"`
	Rows                      []PurchaseReportRow    `json:"rows,omitempty"`
	DailySummary              []DailyPurchaseSummary `json:"daily_summary,omitempty"`
}

type StockMovementSummary struct {
	InQuantity    int64 `json:"in_quantity"`
	OutQuantity   int64 `json:"out_quantity"`
	AdjustmentQty int64 `json:"adjustment_quantity"`
}

type InventoryReportRow struct {
	BranchID     int64  `json:"branch_id"`
	BranchName   string `json:"branch_name,omitempty"`
	ProductID    int64  `json:"product_id"`
	ProductName  string `json:"product_name,omitempty"`
	SKU          string `json:"sku,omitempty"`
	CategoryName string `json:"category_name,omitempty"`
	CurrentStock int64  `json:"current_stock"`
	MinimumStock int64  `json:"minimum_stock"`
	StockStatus  string `json:"stock_status"`
}

type InventoryReport struct {
	TotalInventoryRecords int64                `json:"total_inventory_records"`
	TotalQuantity         int64                `json:"total_quantity"`
	StockMovementSummary  StockMovementSummary `json:"stock_movement_summary"`
	LowStockProducts      []LowStockProduct    `json:"low_stock_products,omitempty"`
	Rows                  []InventoryReportRow `json:"rows,omitempty"`
	StockStatusBreakdown  map[string]int64     `json:"stock_status_breakdown,omitempty"`
}

type LowStockProduct struct {
	ProductID int64  `json:"product_id"`
	BranchID  int64  `json:"branch_id"`
	Quantity  int64  `json:"quantity"`
	Name      string `json:"name,omitempty"`
}

type PaymentReportRow struct {
	ID              int64     `json:"id"`
	SaleNumber      string    `json:"sale_number,omitempty"`
	CustomerName    string    `json:"customer_name,omitempty"`
	BranchName      string    `json:"branch_name,omitempty"`
	Amount          float64   `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
	ReferenceNumber string    `json:"reference_number,omitempty"`
	PaidAt          time.Time `json:"paid_at"`
	CreatedBy       int64     `json:"created_by,omitempty"`
	CreatedByName   string    `json:"created_by_name,omitempty"`
}

type PaymentReport struct {
	TotalPayments        int64              `json:"total_payments"`
	TotalPaymentAmount   float64            `json:"total_payment_amount"`
	MethodBreakdown      map[string]int64   `json:"method_breakdown,omitempty"`
	MethodTotalBreakdown map[string]float64 `json:"method_total_breakdown,omitempty"`
	Rows                 []PaymentReportRow `json:"rows,omitempty"`
	StatusSummary        map[string]int64   `json:"status_summary,omitempty"`
}

type ProfitReport struct {
	Revenue        float64 `json:"revenue"`
	PurchasingCost float64 `json:"purchasing_cost"`
	GrossProfit    float64 `json:"gross_profit"`
}
