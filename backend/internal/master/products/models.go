package products

import "time"

type Product struct {
	ID            int64      `json:"id"`
	SKU           string     `json:"sku"`
	Barcode       *string    `json:"barcode,omitempty"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	Category      *string    `json:"category,omitempty"`
	Unit          *string    `json:"unit,omitempty"`
	PurchasePrice float64    `json:"purchase_price"`
	SellingPrice  float64    `json:"selling_price"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type ProductFilter struct {
	Search *string
	Active *bool
}
