package users

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

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE email = $1
    `, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE id = $1
    `, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) Create(ctx context.Context, user *User) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id
    `, user.Email, user.PasswordHash, user.Name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetRoleNames(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT r.name
        FROM roles r
        JOIN user_roles ur ON ur.role_id = r.id
        WHERE ur.user_id = $1
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}

func (r *Repository) GetPermissionNames(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT DISTINCT p.name
        FROM permissions p
        JOIN role_permissions rp ON rp.permission_id = p.id
        JOIN user_roles ur ON ur.role_id = rp.role_id
        WHERE ur.user_id = $1
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		permissions = append(permissions, name)
	}
	return permissions, rows.Err()
}

func (r *Repository) AddRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO user_roles (user_id, role_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `, userID, roleID)
	return err
}
