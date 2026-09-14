package sales

import "time"

type Sale struct {
	ID          int64     `json:"id"`
	BranchID    int64     `json:"branch_id"`
	CustomerID  int64     `json:"customer_id"`
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
	ID                int64     `json:"id"`
	SaleID            int64     `json:"sale_id"`
	ProductID         int64     `json:"product_id"`
	Quantity          int64     `json:"quantity"`
	UnitPrice         float64   `json:"unit_price"`
	Subtotal          float64   `json:"subtotal"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	FulfilledQuantity int64     `json:"fulfilled_quantity"`
}

type SaleFulfillment struct {
	ID              int64                 `json:"id"`
	SaleID          int64                 `json:"sales_order_id"`
	BranchID        int64                 `json:"branch_id"`
	FulfilledBy     int64                 `json:"fulfilled_by"`
	FulfilledByName string                `json:"fulfilled_by_name,omitempty"`
	FulfilledAt     time.Time             `json:"fulfilled_at"`
	Notes           *string               `json:"notes,omitempty"`
	Items           []SaleFulfillmentItem `json:"items"`
}

type SaleFulfillmentItem struct {
	ID                int64 `json:"id"`
	FulfillmentID     int64 `json:"sales_fulfillment_id"`
	SaleItemID        int64 `json:"sales_order_item_id"`
	ProductID         int64 `json:"product_id"`
	QuantityFulfilled int64 `json:"quantity_fulfilled"`
}
