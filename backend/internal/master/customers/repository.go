package customers

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

func (r *Repository) Create(ctx context.Context, customer *Customer) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO customers (code, name, phone, email, address, tax_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, customer.Code, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, customer *Customer) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO customers (code, name, phone, email, address, tax_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, customer.Code, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Customer, error) {
	customer := &Customer{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&customer.ID,
		&customer.Code,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.Address,
		&customer.TaxID,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
		&customer.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return customer, nil
}

func (r *Repository) GetByIDIncludeDeleted(ctx context.Context, id int64) (*Customer, error) {
	customer := &Customer{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1
	`, id).Scan(
		&customer.ID,
		&customer.Code,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.Address,
		&customer.TaxID,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
		&customer.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return customer, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Customer, error) {
	return r.getByColumn(ctx, "code", strings.TrimSpace(code))
}

func (r *Repository) getByColumn(ctx context.Context, column, value string) (*Customer, error) {
	if value == "" {
		return nil, sql.ErrNoRows
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at
		FROM customers
		WHERE %s = $1
	`, column)

	customer := &Customer{}
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&customer.ID,
		&customer.Code,
		&customer.Name,
		&customer.Phone,
		&customer.Email,
		&customer.Address,
		&customer.TaxID,
		&customer.IsActive,
		&customer.CreatedAt,
		&customer.UpdatedAt,
		&customer.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return customer, nil
}

func (r *Repository) List(ctx context.Context, filter CustomerFilter) ([]Customer, error) {
	query := `
		SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at
		FROM customers
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

	customers := []Customer{}
	for rows.Next() {
		var customer Customer
		if err := rows.Scan(
			&customer.ID,
			&customer.Code,
			&customer.Name,
			&customer.Phone,
			&customer.Email,
			&customer.Address,
			&customer.TaxID,
			&customer.IsActive,
			&customer.CreatedAt,
			&customer.UpdatedAt,
			&customer.DeletedAt,
		); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return customers, nil
}

func (r *Repository) Update(ctx context.Context, customer *Customer) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE customers
		SET code = $1, name = $2, phone = $3, email = $4, address = $5, tax_id = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8 AND deleted_at IS NULL
	`, customer.Code, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.IsActive, customer.ID)
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

func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, customer *Customer) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE customers
		SET code = $1, name = $2, phone = $3, email = $4, address = $5, tax_id = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8 AND deleted_at IS NULL
	`, customer.Code, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.IsActive, customer.ID)
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
		UPDATE customers
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
		UPDATE customers
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
