package products

import (
	"context"
	"database/sql"
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

func (r *Repository) Create(ctx context.Context, product *Product) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO products (sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `, product.SKU, product.Barcode, product.Name, product.Description, product.Category, product.Unit, product.PurchasePrice, product.SellingPrice, product.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, product *Product) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO products (sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `, product.SKU, product.Barcode, product.Name, product.Description, product.Category, product.Unit, product.PurchasePrice, product.SellingPrice, product.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Product, error) {
	product := &Product{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active, created_at, updated_at, deleted_at
        FROM products
        WHERE id = $1 AND deleted_at IS NULL
    `, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Barcode,
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Unit,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *Repository) GetByIDIncludeDeleted(ctx context.Context, id int64) (*Product, error) {
	product := &Product{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active, created_at, updated_at, deleted_at
        FROM products
        WHERE id = $1
    `, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Barcode,
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Unit,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *Repository) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	return r.getByColumn(ctx, "sku", strings.TrimSpace(sku))
}

func (r *Repository) GetByBarcode(ctx context.Context, barcode string) (*Product, error) {
	return r.getByColumn(ctx, "barcode", strings.TrimSpace(barcode))
}

func (r *Repository) getByColumn(ctx context.Context, column, value string) (*Product, error) {
	if value == "" {
		return nil, sql.ErrNoRows
	}

	query := fmt.Sprintf(`
        SELECT id, sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active, created_at, updated_at, deleted_at
        FROM products
        WHERE %s = $1
    `, column)

	product := &Product{}
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&product.ID,
		&product.SKU,
		&product.Barcode,
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Unit,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *Repository) List(ctx context.Context, filter ProductFilter) ([]Product, error) {
	query := `
        SELECT id, sku, barcode, name, description, category, unit, purchase_price, selling_price, is_active, created_at, updated_at, deleted_at
        FROM products
        WHERE deleted_at IS NULL
    `
	args := []any{}
	clauses := []string{}
	idx := 1

	if filter.Active != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *filter.Active)
		idx++
	}
	if filter.Search != nil && *filter.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(sku ILIKE $%d OR name ILIKE $%d)", idx, idx+1))
		args = append(args, "%"+*filter.Search+"%", "%"+*filter.Search+"%")
		idx += 2
	}

	if len(clauses) > 0 {
		query += " AND " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Barcode,
			&product.Name,
			&product.Description,
			&product.Category,
			&product.Unit,
			&product.PurchasePrice,
			&product.SellingPrice,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.DeletedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *Repository) Update(ctx context.Context, product *Product) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE products
        SET sku = $1, barcode = $2, name = $3, description = $4, category = $5, unit = $6, purchase_price = $7, selling_price = $8, is_active = $9, updated_at = NOW()
        WHERE id = $10 AND deleted_at IS NULL
    `, product.SKU, product.Barcode, product.Name, product.Description, product.Category, product.Unit, product.PurchasePrice, product.SellingPrice, product.IsActive, product.ID)
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

func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, product *Product) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE products
        SET sku = $1, barcode = $2, name = $3, description = $4, category = $5, unit = $6, purchase_price = $7, selling_price = $8, is_active = $9, updated_at = NOW()
        WHERE id = $10 AND deleted_at IS NULL
    `, product.SKU, product.Barcode, product.Name, product.Description, product.Category, product.Unit, product.PurchasePrice, product.SellingPrice, product.IsActive, product.ID)
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

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE products
        SET deleted_at = NOW(), is_active = false, updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `, id)
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

func (r *Repository) SoftDeleteWithTx(ctx context.Context, tx *sql.Tx, id int64) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE products
        SET deleted_at = NOW(), is_active = false, updated_at = NOW()
        WHERE id = $1 AND deleted_at IS NULL
    `, id)
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
