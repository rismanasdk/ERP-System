package purchasing

import "time"

type Supplier struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Address   *string   `json:"address,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type SupplierFilter struct {
	Active *bool
	Search *string
}

type Purchase struct {
	ID             int64     `json:"id"`
	BranchID       int64     `json:"branch_id"`
	SupplierID     int64     `json:"supplier_id"`
	PurchaseNumber string    `json:"purchase_number"`
	Status         string    `json:"status"`
	TotalAmount    float64   `json:"total_amount"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedBy      int64     `json:"created_by"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type PurchaseFilter struct {
	BranchID   *int64
	SupplierID *int64
	Status     *string
}

type PurchaseItem struct {
	ID         int64     `json:"id"`
	PurchaseID int64     `json:"purchase_id"`
	ProductID  int64     `json:"product_id"`
	Quantity   int64     `json:"quantity"`
	UnitCost   float64   `json:"unit_cost"`
	Subtotal   float64   `json:"subtotal"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}
