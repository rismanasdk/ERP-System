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

func (r *Repository) List(ctx context.Context) ([]*Role, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description
		FROM roles
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *Repository) GetPermissionNamesByRoleID(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.name
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		perms = append(perms, name)
	}
	return perms, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Role, error) {
	role := &Role{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description
		FROM roles
		WHERE id = $1
	`, id).Scan(&role.ID, &role.Name, &role.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return role, nil
}
