package inventory

import "time"

type Inventory struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	BranchID  int64     `json:"branch_id"`
	Quantity  int64     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StockMovement struct {
	ID            int64          `json:"id"`
	ProductID     int64          `json:"product_id"`
	BranchID      int64          `json:"branch_id"`
	MovementType  string         `json:"movement_type"`
	QuantityDelta int64          `json:"quantity_delta"`
	ReferenceType *string        `json:"reference_type,omitempty"`
	ReferenceID   *int64         `json:"reference_id,omitempty"`
	ActorUserID   *int64         `json:"actor_user_id,omitempty"`
	ActorUserName *string        `json:"actor_user_name,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type StockTransfer struct {
	ID                    int64     `json:"id"`
	SourceBranchID        int64     `json:"source_branch_id"`
	SourceBranchName      string    `json:"source_branch_name,omitempty"`
	DestinationBranchID   int64     `json:"destination_branch_id"`
	DestinationBranchName string    `json:"destination_branch_name,omitempty"`
	ProductID             int64     `json:"product_id"`
	Quantity              int64     `json:"quantity"`
	Status                string    `json:"status"`
	Notes                 *string   `json:"notes,omitempty"`
	CreatedBy             int64     `json:"created_by"`
	CreatedByName         string    `json:"created_by_name,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	CompletedAt           time.Time `json:"completed_at"`
}

func NewInventory(productID, branchID, quantity int64) *Inventory {
	return &Inventory{
		ProductID: productID,
		BranchID:  branchID,
		Quantity:  quantity,
	}
}

func NewStockMovement(productID, branchID int64, movementType string, quantityDelta int64) *StockMovement {
	return &StockMovement{
		ProductID:     productID,
		BranchID:      branchID,
		MovementType:  movementType,
		QuantityDelta: quantityDelta,
	}
}

func PtrString(value string) *string {
	return &value
}

func PtrInt64(value int64) *int64 {
	return &value
}
