package roles

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByName(ctx context.Context, name string) (*Role, error) {
	role := &Role{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, description
        FROM roles
        WHERE name = $1
    `, name).Scan(&role.ID, &role.Name, &role.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return role, nil
}

func (r *Repository) GetByNameTx(ctx context.Context, tx *sql.Tx, name string) (*Role, error) {
	role := &Role{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, name, description
        FROM roles
        WHERE name = $1
    `, name).Scan(&role.ID, &role.Name, &role.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return role, nil
}
