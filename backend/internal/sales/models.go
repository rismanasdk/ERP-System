package sales

import "time"

type Sale struct {
	ID              int64     `json:"id"`
	BranchID        int64     `json:"branch_id"`
	CustomerID      int64     `json:"customer_id"`
	SaleNumber      string    `json:"sale_number"`
	Status          string    `json:"status"`
	TotalAmount     float64   `json:"total_amount"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedBy       int64     `json:"created_by"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
	PaymentStatus   string    `json:"payment_status,omitempty"`
	PaidAmount      float64   `json:"paid_amount,omitempty"`
	RemainingAmount float64   `json:"remaining_amount,omitempty"`
}

type SalesPayment struct {
	ID              int64     `json:"id"`
	SalesOrderID    int64     `json:"sales_order_id"`
	Amount          float64   `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
	ReferenceNumber *string   `json:"reference_number,omitempty"`
	PaidAt          time.Time `json:"paid_at"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedBy       int64     `json:"created_by"`
	CreatedByName   string    `json:"created_by_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type PaymentSummary struct {
	OrderTotal      float64 `json:"order_total"`
	PaidAmount      float64 `json:"paid_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	PaymentStatus   string  `json:"payment_status"`
}

type CreatePaymentInput struct {
	Amount          float64    `json:"amount"`
	PaymentMethod   string     `json:"payment_method"`
	ReferenceNumber *string    `json:"reference_number,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
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
