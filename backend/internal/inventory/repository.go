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

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
