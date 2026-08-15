package suppliers

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

func (r *Repository) Create(ctx context.Context, supplier *Supplier) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO suppliers (code, name, phone, email, address, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, supplier.Code, supplier.Name, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, supplier *Supplier) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO suppliers (code, name, phone, email, address, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, supplier.Code, supplier.Name, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	supplier := &Supplier{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at
		FROM suppliers
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&supplier.ID,
		&supplier.Code,
		&supplier.Name,
		&supplier.Phone,
		&supplier.Email,
		&supplier.Address,
		&supplier.IsActive,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
		&supplier.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (r *Repository) GetByIDIncludeDeleted(ctx context.Context, id int64) (*Supplier, error) {
	supplier := &Supplier{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at
		FROM suppliers
		WHERE id = $1
	`, id).Scan(
		&supplier.ID,
		&supplier.Code,
		&supplier.Name,
		&supplier.Phone,
		&supplier.Email,
		&supplier.Address,
		&supplier.IsActive,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
		&supplier.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	return r.getByColumn(ctx, "code", strings.TrimSpace(code), false)
}

func (r *Repository) GetByCodeIncludeDeleted(ctx context.Context, code string) (*Supplier, error) {
	return r.getByColumn(ctx, "code", strings.TrimSpace(code), true)
}

func (r *Repository) getByColumn(ctx context.Context, column, value string, includeDeleted bool) (*Supplier, error) {
	if value == "" {
		return nil, sql.ErrNoRows
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at
		FROM suppliers
		WHERE %s = $1`, column)
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}

	supplier := &Supplier{}
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&supplier.ID,
		&supplier.Code,
		&supplier.Name,
		&supplier.Phone,
		&supplier.Email,
		&supplier.Address,
		&supplier.IsActive,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
		&supplier.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (r *Repository) List(ctx context.Context, filter SupplierFilter) ([]Supplier, error) {
	query := `
		SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at
		FROM suppliers
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
		clauses = append(clauses, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", idx, idx))
		args = append(args, "%"+*filter.Search+"%")
		idx++
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

	suppliers := []Supplier{}
	for rows.Next() {
		var supplier Supplier
		if err := rows.Scan(
			&supplier.ID,
			&supplier.Code,
			&supplier.Name,
			&supplier.Phone,
			&supplier.Email,
			&supplier.Address,
			&supplier.IsActive,
			&supplier.CreatedAt,
			&supplier.UpdatedAt,
			&supplier.DeletedAt,
		); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, supplier)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppliers, nil
}

func (r *Repository) Update(ctx context.Context, supplier *Supplier) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE suppliers
		SET code = $1, name = $2, phone = $3, email = $4, address = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL
	`, supplier.Code, supplier.Name, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive, supplier.ID)
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

func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, supplier *Supplier) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE suppliers
		SET code = $1, name = $2, phone = $3, email = $4, address = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL
	`, supplier.Code, supplier.Name, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive, supplier.ID)
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
		UPDATE suppliers
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
		UPDATE suppliers
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
