package dashboard

import "time"

const DashboardReadPermission = "dashboard.read"

type Summary struct {
	Sales      SalesSummary      `json:"sales"`
	Purchases  PurchasesSummary  `json:"purchases"`
	MasterData MasterDataSummary `json:"master_data"`
	Inventory  InventorySummary  `json:"inventory"`
}

type SalesSummary struct {
	TodayAmount       float64 `json:"today_amount"`
	TodayTransactions int64   `json:"today_transactions"`
	MonthAmount       float64 `json:"month_amount"`
	MonthTransactions int64   `json:"month_transactions"`
}

type PurchasesSummary struct {
	TodayAmount       float64 `json:"today_amount"`
	TodayTransactions int64   `json:"today_transactions"`
	MonthAmount       float64 `json:"month_amount"`
	MonthTransactions int64   `json:"month_transactions"`
}

type MasterDataSummary struct {
	Products  int64 `json:"products"`
	Customers int64 `json:"customers"`
	Suppliers int64 `json:"suppliers"`
}

type InventorySummary struct {
	TotalItems int64 `json:"total_items"`
}

type BranchScope struct {
	BranchID *int64
}

type SalesPeriod struct {
	TodayAmount       float64
	TodayTransactions int64
	MonthAmount       float64
	MonthTransactions int64
}

type PurchasePeriod struct {
	TodayAmount       float64
	TodayTransactions int64
	MonthAmount       float64
	MonthTransactions int64
}

type BranchSummary struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
