package branches

import (
	"context"
	"database/sql"
	"errors"
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

func (r *Repository) CreateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO branches (name, code, is_active)
        VALUES ($1, $2, $3)
        RETURNING id
    `, branch.Name, branch.Code, branch.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Branch, error) {
	branch := &Branch{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, code, is_active, created_at, updated_at
        FROM branches
        WHERE id = $1
    `, id).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Code,
		&branch.IsActive,
		&branch.CreatedAt,
		&branch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return branch, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Branch, error) {
	branch := &Branch{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, code, is_active, created_at, updated_at
        FROM branches
        WHERE code = $1
    `, code).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Code,
		&branch.IsActive,
		&branch.CreatedAt,
		&branch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return branch, nil
}

func (r *Repository) List(ctx context.Context, filter BranchFilter) ([]Branch, error) {
	query := `
        SELECT id, name, code, is_active, created_at, updated_at
        FROM branches
    `
	args := []any{}
	clauses := []string{}
	idx := 1

	if filter.Active != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *filter.Active)
		idx++
	}

	if len(clauses) > 0 {
		query += " WHERE " + clauses[0]
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []Branch
	for rows.Next() {
		var branch Branch
		if err := rows.Scan(
			&branch.ID,
			&branch.Name,
			&branch.Code,
			&branch.IsActive,
			&branch.CreatedAt,
			&branch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return branches, nil
}

func (r *Repository) ListAccessibleBranches(ctx context.Context, filter BranchFilter, userID int64) ([]Branch, error) {
	query := `
        SELECT b.id, b.name, b.code, b.is_active, b.created_at, b.updated_at
        FROM branches b
        JOIN user_branches ub ON ub.branch_id = b.id
        WHERE ub.user_id = $1
    `
	args := []any{userID}
	idx := 2

	if filter.Active != nil {
		query += fmt.Sprintf(" AND b.is_active = $%d", idx)
		args = append(args, *filter.Active)
		idx++
	}
	query += " ORDER BY b.name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []Branch
	for rows.Next() {
		var branch Branch
		if err := rows.Scan(
			&branch.ID,
			&branch.Name,
			&branch.Code,
			&branch.IsActive,
			&branch.CreatedAt,
			&branch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return branches, nil
}

func (r *Repository) UpdateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE branches
        SET name = $1, code = $2, is_active = $3, updated_at = NOW()
        WHERE id = $4
    `, branch.Name, branch.Code, branch.IsActive, branch.ID)
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

var ErrUserBranchDuplicate = errors.New("user branch assignment already exists")

func (r *Repository) AssignUserBranchWithTx(ctx context.Context, tx *sql.Tx, userID, branchID int64) error {
	res, err := tx.ExecContext(ctx, `
        INSERT INTO user_branches (user_id, branch_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `, userID, branchID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserBranchDuplicate
	}
	return nil
}

func (r *Repository) UserHasAccess(ctx context.Context, userID, branchID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
        SELECT EXISTS(
            SELECT 1
            FROM user_branches
            WHERE user_id = $1 AND branch_id = $2
        )
    `, userID, branchID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
