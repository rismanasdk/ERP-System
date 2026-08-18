package users

import (
	"context"
	"database/sql"
	"fmt"
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

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, user *User) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id
    `, user.Email, user.PasswordHash, user.Name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, user *User) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE users
        SET email = $1, password_hash = $2, name = $3, updated_at = NOW()
        WHERE id = $4
    `, user.Email, user.PasswordHash, user.Name, user.ID)
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

func (r *Repository) AddRoleWithTx(ctx context.Context, tx *sql.Tx, userID, roleID int64) error {
	_, err := tx.ExecContext(ctx, `
        INSERT INTO user_roles (user_id, role_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `, userID, roleID)
	return err
}

func (r *Repository) DeleteRolesWithTx(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) AddBranchWithTx(ctx context.Context, tx *sql.Tx, userID, branchID int64) error {
	_, err := tx.ExecContext(ctx, `
        INSERT INTO user_branches (user_id, branch_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `, userID, branchID)
	return err
}

func (r *Repository) DeleteBranchesWithTx(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM user_branches WHERE user_id = $1`, userID)
	return err
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

func (r *Repository) List(ctx context.Context, filter UserFilter) ([]User, error) {
	query := `
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
    `
	args := []any{}
	clauses := []string{}
	if filter.Search != nil {
		pattern := fmt.Sprintf("%%%s%%", *filter.Search)
		clauses = append(clauses, "(LOWER(email) LIKE LOWER($1) OR LOWER(name) LIKE LOWER($1))")
		args = append(args, pattern)
	}
	if len(clauses) > 0 {
		query += " WHERE " + clauses[0]
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *Repository) ListByBranch(ctx context.Context, branchID int64) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT u.id, u.email, u.password_hash, u.name, u.created_at, u.updated_at
        FROM users u
        JOIN user_branches ub ON ub.user_id = u.id
        WHERE ub.branch_id = $1
        ORDER BY u.created_at DESC
    `, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
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

func (r *Repository) GetBranchIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT branch_id
        FROM user_branches
        WHERE user_id = $1
        ORDER BY branch_id ASC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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

func (r *Repository) branchExists(ctx context.Context, branchID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM branches
            WHERE id = $1
        )
    `, branchID).Scan(&exists)
	return exists, err
}
