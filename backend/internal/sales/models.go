package sales

import "time"

type Sale struct {
	ID          int64     `json:"id"`
	BranchID    int64     `json:"branch_id"`
	SaleNumber  string    `json:"sale_number"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type SaleFilter struct {
	BranchID *int64
}

type SaleItem struct {
	ID        int64     `json:"id"`
	SaleID    int64     `json:"sale_id"`
	ProductID int64     `json:"product_id"`
	Quantity  int64     `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Subtotal  float64   `json:"subtotal"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
