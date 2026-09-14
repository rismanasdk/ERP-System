package purchasing

import (
	"time"

	"erp-system/backend/internal/master/suppliers"
)

type Supplier = suppliers.Supplier
type SupplierFilter = suppliers.SupplierFilter

type Purchase struct {
	ID             int64          `json:"id"`
	BranchID       int64          `json:"branch_id"`
	SupplierID     int64          `json:"supplier_id"`
	PurchaseNumber string         `json:"purchase_number"`
	Status         string         `json:"status"`
	TotalAmount    float64        `json:"total_amount"`
	Notes          *string        `json:"notes,omitempty"`
	CreatedBy      int64          `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
	Items          []PurchaseItem `json:"items,omitempty"`
}

type PurchaseFilter struct {
	BranchID   *int64
	SupplierID *int64
	Status     *string
}

type PurchaseItem struct {
	ID               int64     `json:"id"`
	PurchaseID       int64     `json:"purchase_id"`
	ProductID        int64     `json:"product_id"`
	Quantity         int64     `json:"quantity"`
	UnitCost         float64   `json:"unit_cost"`
	Subtotal         float64   `json:"subtotal"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
	ReceivedQuantity int64     `json:"received_quantity"`
}

type PurchaseReceipt struct {
	ID             int64                 `json:"id"`
	PurchaseID     int64                 `json:"purchase_order_id"`
	BranchID       int64                 `json:"branch_id"`
	ReceivedBy     int64                 `json:"received_by"`
	ReceivedByName string                `json:"received_by_name,omitempty"`
	ReceivedAt     time.Time             `json:"received_at"`
	Notes          *string               `json:"notes,omitempty"`
	Items          []PurchaseReceiptItem `json:"items"`
}

type PurchaseReceiptItem struct {
	ID               int64 `json:"id"`
	ReceiptID        int64 `json:"purchase_receipt_id"`
	PurchaseItemID   int64 `json:"purchase_order_item_id"`
	ProductID        int64 `json:"product_id"`
	QuantityReceived int64 `json:"quantity_received"`
}
