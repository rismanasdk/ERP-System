package categories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]Category, error) {
	query := `SELECT c.id, c.name, c.description, c.is_active, COUNT(p.id), c.version, c.created_at, c.updated_at FROM product_categories c LEFT JOIN products p ON p.category_id = c.id AND p.deleted_at IS NULL`
	args := []any{}
	clauses := []string{}
	idx := 1
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		clauses = append(clauses, fmt.Sprintf("c.name ILIKE $%d", idx))
		args = append(args, "%"+strings.TrimSpace(*filter.Search)+"%")
		idx++
	}
	if filter.Active != nil {
		clauses = append(clauses, fmt.Sprintf("c.is_active = $%d", idx))
		args = append(args, *filter.Active)
		idx++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " GROUP BY c.id ORDER BY c.name ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Category
	for rows.Next() {
		var item Category
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.IsActive, &item.ProductCount, &item.Version, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Category, error) {
	var item Category
	err := r.db.QueryRowContext(ctx, `SELECT c.id, c.name, c.description, c.is_active, COUNT(p.id), c.version, c.created_at, c.updated_at FROM product_categories c LEFT JOIN products p ON p.category_id = c.id AND p.deleted_at IS NULL WHERE c.id = $1 GROUP BY c.id`, id).Scan(&item.ID, &item.Name, &item.Description, &item.IsActive, &item.ProductCount, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*Category, error) {
	var item Category
	err := r.db.QueryRowContext(ctx, `SELECT id, name, description, is_active, 0, version, created_at, updated_at FROM product_categories WHERE LOWER(name) = LOWER($1)`, strings.TrimSpace(name)).Scan(&item.ID, &item.Name, &item.Description, &item.IsActive, &item.ProductCount, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) Create(ctx context.Context, item *Category) (int64, error) {
	return r.CreateWithTx(ctx, nil, item)
}
func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, item *Category) (int64, error) {
	var id int64
	query := `INSERT INTO product_categories (name, description, is_active) VALUES ($1, $2, $3) RETURNING id`
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, item.Name, item.Description, item.IsActive)
	} else {
		row = r.db.QueryRowContext(ctx, query, item.Name, item.Description, item.IsActive)
	}
	err := row.Scan(&id)
	return id, err
}
func (r *Repository) Update(ctx context.Context, item *Category) error {
	return r.UpdateWithTx(ctx, nil, item)
}
func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, item *Category) error {
	query := `UPDATE product_categories SET name = $1, description = $2, is_active = $3, version = version + 1, updated_at = NOW() WHERE id = $4 AND version = $5`
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, query, item.Name, item.Description, item.IsActive, item.ID, item.Version)
	} else {
		res, err = r.db.ExecContext(ctx, query, item.Name, item.Description, item.IsActive, item.ID, item.Version)
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (r *Repository) GetVersionStateWithTx(ctx context.Context, tx *sql.Tx, id int64) (version int64, err error) {
	err = tx.QueryRowContext(ctx, `SELECT version FROM product_categories WHERE id = $1 FOR UPDATE`, id).Scan(&version)
	return
}
func (r *Repository) Delete(ctx context.Context, id int64) error {
	return r.DeleteWithTx(ctx, nil, id)
}
func (r *Repository) DeleteWithTx(ctx context.Context, tx *sql.Tx, id int64) error {
	query := `DELETE FROM product_categories WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM products WHERE category_id = $1 AND deleted_at IS NULL)`
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, query, id)
	} else {
		res, err = r.db.ExecContext(ctx, query, id)
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return err
}
func (r *Repository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM product_categories WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}
