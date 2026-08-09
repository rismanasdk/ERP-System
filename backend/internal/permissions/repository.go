package permissions

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

func (r *Repository) GetByName(ctx context.Context, name string) (*Permission, error) {
	perm := &Permission{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, description
        FROM permissions
        WHERE name = $1
    `, name).Scan(&perm.ID, &perm.Name, &perm.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return perm, nil
}
