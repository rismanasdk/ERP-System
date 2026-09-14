package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Inventory, error) {
	inventory := &Inventory{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE id = $1
    `, id).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.BranchID,
		&inventory.Quantity,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inventory, nil
}

func (r *Repository) GetByProductAndBranch(ctx context.Context, productID, branchID int64) (*Inventory, error) {
	inventory := &Inventory{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE product_id = $1 AND branch_id = $2
    `, productID, branchID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.BranchID,
		&inventory.Quantity,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inventory, nil
}

func (r *Repository) GetByProductAndBranchForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error) {
	inventory := &Inventory{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE product_id = $1 AND branch_id = $2
        FOR UPDATE
    `, productID, branchID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.BranchID,
		&inventory.Quantity,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inventory, nil
}

func (r *Repository) EnsureInventoryWithTx(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error) {
	_, err := tx.ExecContext(ctx, `
        INSERT INTO inventory (product_id, branch_id, quantity)
        VALUES ($1, $2, 0)
        ON CONFLICT (product_id, branch_id) DO NOTHING
    `, productID, branchID)
	if err != nil {
		return nil, err
	}
	return getInventoryForUpdate(ctx, tx, productID, branchID)
}

func getInventoryForUpdate(ctx context.Context, tx *sql.Tx, productID, branchID int64) (*Inventory, error) {
	inventory := &Inventory{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
        WHERE product_id = $1 AND branch_id = $2
        FOR UPDATE
    `, productID, branchID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.BranchID,
		&inventory.Quantity,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inventory, nil
}

func (r *Repository) List(ctx context.Context, branchID, productID *int64) ([]Inventory, error) {
	query := `
        SELECT id, product_id, branch_id, quantity, created_at, updated_at
        FROM inventory
    `
	args := []any{}
	clauses := []string{}
	idx := 1

	if branchID != nil {
		clauses = append(clauses, `branch_id = $`+itoa(idx))
		args = append(args, *branchID)
		idx++
	}
	if productID != nil {
		clauses = append(clauses, `product_id = $`+itoa(idx))
		args = append(args, *productID)
		idx++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inventoryList []Inventory
	for rows.Next() {
		var inventory Inventory
		if err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.BranchID,
			&inventory.Quantity,
			&inventory.CreatedAt,
			&inventory.UpdatedAt,
		); err != nil {
			return nil, err
		}
		inventoryList = append(inventoryList, inventory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return inventoryList, nil
}

func (r *Repository) ListMovements(ctx context.Context, branchID, productID *int64) ([]StockMovement, error) {
	query := `
		SELECT sm.id, sm.product_id, sm.branch_id, sm.movement_type, sm.quantity_delta, sm.reference_type, sm.reference_id, sm.actor_user_id, u.name, sm.metadata, sm.created_at
		FROM stock_movements sm
		LEFT JOIN users u ON u.id = sm.actor_user_id
    `
	args := []any{}
	clauses := []string{}
	idx := 1

	if branchID != nil {
		clauses = append(clauses, `sm.branch_id = $`+itoa(idx))
		args = append(args, *branchID)
		idx++
	}
	if productID != nil {
		clauses = append(clauses, `sm.product_id = $`+itoa(idx))
		args = append(args, *productID)
		idx++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []StockMovement
	for rows.Next() {
		var movement StockMovement
		var metadataJSON []byte
		if err := rows.Scan(
			&movement.ID,
			&movement.ProductID,
			&movement.BranchID,
			&movement.MovementType,
			&movement.QuantityDelta,
			&movement.ReferenceType,
			&movement.ReferenceID,
			&movement.ActorUserID,
			&movement.ActorUserName,
			&metadataJSON,
			&movement.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
			if err := json.Unmarshal(metadataJSON, &movement.Metadata); err != nil {
				return nil, err
			}
		}
		movements = append(movements, movement)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return movements, nil
}

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, inventory *Inventory) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO inventory (product_id, branch_id, quantity)
        VALUES ($1, $2, $3)
        RETURNING id
    `, inventory.ProductID, inventory.BranchID, inventory.Quantity).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) UpdateQuantityWithTx(ctx context.Context, tx *sql.Tx, id, quantity int64) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE inventory
        SET quantity = $1, updated_at = NOW()
        WHERE id = $2
    `, quantity, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateMovementWithTx(ctx context.Context, tx *sql.Tx, movement *StockMovement) (int64, error) {
	var id int64
	metadataJSON, err := json.Marshal(movement.Metadata)
	if err != nil {
		return 0, err
	}
	err = tx.QueryRowContext(ctx, `
        INSERT INTO stock_movements (
            product_id,
            branch_id,
            movement_type,
            quantity_delta,
            reference_type,
            reference_id,
            actor_user_id,
            metadata
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `, movement.ProductID, movement.BranchID, movement.MovementType, movement.QuantityDelta, movement.ReferenceType, movement.ReferenceID, movement.ActorUserID, metadataJSON).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateTransferWithTx(ctx context.Context, tx *sql.Tx, transfer *StockTransfer) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO stock_transfers (
            source_branch_id, destination_branch_id, product_id, quantity,
            status, notes, created_by, completed_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
        RETURNING id, created_at, completed_at
    `, transfer.SourceBranchID, transfer.DestinationBranchID, transfer.ProductID, transfer.Quantity,
		transfer.Status, transfer.Notes, transfer.CreatedBy).Scan(&id, &transfer.CreatedAt, &transfer.CompletedAt)
	if err != nil {
		return 0, err
	}
	transfer.ID = id
	return id, nil
}

func (r *Repository) ListTransfers(ctx context.Context, branchIDs []int64) ([]StockTransfer, error) {
	if len(branchIDs) == 0 {
		return []StockTransfer{}, nil
	}
	placeholders := make([]string, len(branchIDs))
	args := make([]any, len(branchIDs))
	for i, id := range branchIDs {
		placeholders[i] = "$" + itoa(i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT st.id, st.source_branch_id, sb.name, st.destination_branch_id, db.name,
		       st.product_id, st.quantity, st.status, st.notes, st.created_by, COALESCE(u.name, ''),
               st.created_at, st.completed_at
        FROM stock_transfers st
        JOIN branches sb ON sb.id = st.source_branch_id
        JOIN branches db ON db.id = st.destination_branch_id
        LEFT JOIN users u ON u.id = st.created_by
        WHERE st.source_branch_id IN (%s) AND st.destination_branch_id IN (%s)
        ORDER BY st.created_at DESC, st.id DESC
    `, strings.Join(placeholders, ", "), strings.Join(placeholders, ", "))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTransferRows(rows)
}

func (r *Repository) GetTransfer(ctx context.Context, id int64) (*StockTransfer, error) {
	query := `
		SELECT st.id, st.source_branch_id, sb.name, st.destination_branch_id, db.name,
		       st.product_id, st.quantity, st.status, st.notes, st.created_by, COALESCE(u.name, ''),
               st.created_at, st.completed_at
        FROM stock_transfers st
        JOIN branches sb ON sb.id = st.source_branch_id
        JOIN branches db ON db.id = st.destination_branch_id
        LEFT JOIN users u ON u.id = st.created_by
        WHERE st.id = $1
    `
	transfer := StockTransfer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transfer.ID, &transfer.SourceBranchID, &transfer.SourceBranchName,
		&transfer.DestinationBranchID, &transfer.DestinationBranchName,
		&transfer.ProductID, &transfer.Quantity, &transfer.Status, &transfer.Notes,
		&transfer.CreatedBy, &transfer.CreatedByName, &transfer.CreatedAt, &transfer.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &transfer, nil
}

func scanTransferRows(rows *sql.Rows) ([]StockTransfer, error) {
	transfers := []StockTransfer{}
	for rows.Next() {
		var transfer StockTransfer
		if err := rows.Scan(
			&transfer.ID, &transfer.SourceBranchID, &transfer.SourceBranchName,
			&transfer.DestinationBranchID, &transfer.DestinationBranchName,
			&transfer.ProductID, &transfer.Quantity, &transfer.Status, &transfer.Notes,
			&transfer.CreatedBy, &transfer.CreatedByName, &transfer.CreatedAt, &transfer.CompletedAt,
		); err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transfers, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
