package sales

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SaleRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateSale(ctx context.Context, sale *Sale) (int64, error)
	CreateSaleWithTx(ctx context.Context, tx *sql.Tx, sale *Sale) (int64, error)
	GetSaleByID(ctx context.Context, id int64) (*Sale, error)
	GetSaleByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Sale, error)
	GetSaleByNumber(ctx context.Context, number string) (*Sale, error)
	ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error)
	UpdateSaleStatus(ctx context.Context, id int64, status string) error
	CreateSaleItem(ctx context.Context, item *SaleItem) (int64, error)
	CreateSaleItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleItem) (int64, error)
	GetSaleItemByID(ctx context.Context, id int64) (*SaleItem, error)
	ListSaleItemsBySaleID(ctx context.Context, saleID int64) ([]SaleItem, error)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) CreateSale(ctx context.Context, sale *Sale) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, sale.BranchID, sale.SaleNumber, sale.Status, sale.TotalAmount, sale.Notes, sale.CreatedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateSaleWithTx(ctx context.Context, tx *sql.Tx, sale *Sale) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, sale.BranchID, sale.SaleNumber, sale.Status, sale.TotalAmount, sale.Notes, sale.CreatedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetSaleByID(ctx context.Context, id int64) (*Sale, error) {
	sale := &Sale{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
    `, id).Scan(
		&sale.ID,
		&sale.BranchID,
		&sale.SaleNumber,
		&sale.Status,
		&sale.TotalAmount,
		&sale.Notes,
		&sale.CreatedBy,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sale, nil
}

func (r *Repository) GetSaleByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Sale, error) {
	sale := &Sale{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
        FOR UPDATE
    `, id).Scan(
		&sale.ID,
		&sale.BranchID,
		&sale.SaleNumber,
		&sale.Status,
		&sale.TotalAmount,
		&sale.Notes,
		&sale.CreatedBy,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sale, nil
}

func (r *Repository) GetSaleByNumber(ctx context.Context, number string) (*Sale, error) {
	sale := &Sale{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE sale_number = $1
    `, strings.TrimSpace(number)).Scan(
		&sale.ID,
		&sale.BranchID,
		&sale.SaleNumber,
		&sale.Status,
		&sale.TotalAmount,
		&sale.Notes,
		&sale.CreatedBy,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sale, nil
}

func (r *Repository) ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error) {
	query := `
        SELECT id, branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
    `
	clauses := []string{}
	args := []any{}
	idx := 1

	if filter.BranchID != nil {
		clauses = append(clauses, fmt.Sprintf("branch_id = $%d", idx))
		args = append(args, *filter.BranchID)
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

	var sales []Sale
	for rows.Next() {
		var sale Sale
		if err := rows.Scan(
			&sale.ID,
			&sale.BranchID,
			&sale.SaleNumber,
			&sale.Status,
			&sale.TotalAmount,
			&sale.Notes,
			&sale.CreatedBy,
			&sale.CreatedAt,
			&sale.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sales = append(sales, sale)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sales, nil
}

func (r *Repository) UpdateSaleStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE sales
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, status, id)
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

func (r *Repository) UpdateSaleStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE sales
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, status, id)
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

func (r *Repository) CreateSaleItem(ctx context.Context, item *SaleItem) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateSaleItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleItem) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetSaleItemByID(ctx context.Context, id int64) (*SaleItem, error) {
	item := &SaleItem{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE id = $1
    `, id).Scan(
		&item.ID,
		&item.SaleID,
		&item.ProductID,
		&item.Quantity,
		&item.UnitPrice,
		&item.Subtotal,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) ListSaleItemsBySaleID(ctx context.Context, saleID int64) ([]SaleItem, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE sale_id = $1
        ORDER BY id ASC
    `, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SaleItem
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) ListSaleItemsBySaleIDWithTx(ctx context.Context, tx *sql.Tx, saleID int64) ([]SaleItem, error) {
	rows, err := tx.QueryContext(ctx, `
        SELECT id, sale_id, product_id, quantity, unit_price, subtotal, created_at, updated_at
        FROM sale_items
        WHERE sale_id = $1
        ORDER BY id ASC
    `, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SaleItem
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
