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
	CreateFulfillmentWithTx(ctx context.Context, tx *sql.Tx, fulfillment *SaleFulfillment) (int64, error)
	CreateFulfillmentItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleFulfillmentItem) (int64, error)
	UpdateSaleItemFulfilledWithTx(ctx context.Context, tx *sql.Tx, itemID, fulfilled int64) error
	ListFulfillments(ctx context.Context, saleID int64) ([]SaleFulfillment, error)
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
	var row *sql.Row
	if sale.CustomerID > 0 {
		row = tx.QueryRowContext(ctx, `
		INSERT INTO sales (customer_id, branch_id, sale_number, status, total_amount, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, sale.CustomerID, sale.BranchID, sale.SaleNumber, sale.Status, sale.TotalAmount, sale.Notes, sale.CreatedBy)
	} else {
		row = tx.QueryRowContext(ctx, `
        INSERT INTO sales (branch_id, sale_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
	`, sale.BranchID, sale.SaleNumber, sale.Status, sale.TotalAmount, sale.Notes, sale.CreatedBy)
	}
	err := row.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetSaleByID(ctx context.Context, id int64) (*Sale, error) {
	sale := &Sale{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(customer_id, 0), branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
    `, id).Scan(
		&sale.ID,
		&sale.CustomerID,
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
		SELECT id, COALESCE(customer_id, 0), branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE id = $1
        FOR UPDATE
    `, id).Scan(
		&sale.ID,
		&sale.CustomerID,
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
		SELECT id, COALESCE(customer_id, 0), branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM sales
        WHERE sale_number = $1
    `, strings.TrimSpace(number)).Scan(
		&sale.ID,
		&sale.CustomerID,
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
		SELECT id, COALESCE(customer_id, 0), branch_id, sale_number, status, total_amount, notes, created_by, created_at, updated_at
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
			&sale.CustomerID,
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
		SELECT id, sale_id, product_id, quantity, unit_price, subtotal, fulfilled_quantity, created_at, updated_at
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
			&item.FulfilledQuantity,
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
	SELECT id, sale_id, product_id, quantity, unit_price, subtotal, fulfilled_quantity, created_at, updated_at
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
			&item.FulfilledQuantity,
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

func (r *Repository) CreateFulfillmentWithTx(ctx context.Context, tx *sql.Tx, fulfillment *SaleFulfillment) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `INSERT INTO sales_fulfillments (sales_order_id, branch_id, fulfilled_by, notes) VALUES ($1, $2, $3, $4) RETURNING id, fulfilled_at`, fulfillment.SaleID, fulfillment.BranchID, fulfillment.FulfilledBy, fulfillment.Notes).Scan(&id, &fulfillment.FulfilledAt)
	return id, err
}

func (r *Repository) CreateFulfillmentItemWithTx(ctx context.Context, tx *sql.Tx, item *SaleFulfillmentItem) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `INSERT INTO sales_fulfillment_items (sales_fulfillment_id, sales_order_item_id, product_id, quantity_fulfilled) VALUES ($1, $2, $3, $4) RETURNING id`, item.FulfillmentID, item.SaleItemID, item.ProductID, item.QuantityFulfilled).Scan(&id)
	return id, err
}

func (r *Repository) UpdateSaleItemFulfilledWithTx(ctx context.Context, tx *sql.Tx, itemID, fulfilled int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE sale_items SET fulfilled_quantity = $1, updated_at = NOW() WHERE id = $2`, fulfilled, itemID)
	return err
}

func (r *Repository) ListFulfillments(ctx context.Context, saleID int64) ([]SaleFulfillment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT sf.id, sf.sales_order_id, sf.branch_id, sf.fulfilled_by, COALESCE(u.name, ''), sf.fulfilled_at, sf.notes, sfi.id, sfi.sales_order_item_id, sfi.product_id, sfi.quantity_fulfilled FROM sales_fulfillments sf LEFT JOIN users u ON u.id = sf.fulfilled_by LEFT JOIN sales_fulfillment_items sfi ON sfi.sales_fulfillment_id = sf.id WHERE sf.sales_order_id = $1 ORDER BY sf.id ASC, sfi.id ASC`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SaleFulfillment{}
	byID := map[int64]*SaleFulfillment{}
	for rows.Next() {
		var f SaleFulfillment
		var itemID, saleItemID, productID, quantity sql.NullInt64
		if err := rows.Scan(&f.ID, &f.SaleID, &f.BranchID, &f.FulfilledBy, &f.FulfilledByName, &f.FulfilledAt, &f.Notes, &itemID, &saleItemID, &productID, &quantity); err != nil {
			return nil, err
		}
		if old := byID[f.ID]; old != nil {
			f = *old
		}
		if itemID.Valid {
			f.Items = append(f.Items, SaleFulfillmentItem{ID: itemID.Int64, FulfillmentID: f.ID, SaleItemID: saleItemID.Int64, ProductID: productID.Int64, QuantityFulfilled: quantity.Int64})
		}
		byID[f.ID] = &f
	}
	for _, f := range byID {
		result = append(result, *f)
	}
	return result, rows.Err()
}
