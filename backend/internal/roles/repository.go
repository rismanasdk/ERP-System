package roles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-system/backend/internal/audit"
)

var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrProtectedRole      = errors.New("the SUPER_ADMIN role is protected")
	ErrRoleHasUsers       = errors.New("role is assigned to users")
	ErrPermissionNotFound = errors.New("one or more permissions do not exist")
)

type Repository struct {
	db       *sql.DB
	auditSvc auditRecorder
}

type auditRecorder interface {
	RecordWithTx(context.Context, *sql.Tx, audit.AuditLog) (int64, error)
}

func NewRepository(db *sql.DB, auditSvcs ...auditRecorder) *Repository {
	var auditSvc auditRecorder
	if len(auditSvcs) > 0 {
		auditSvc = auditSvcs[0]
	}
	return &Repository{db: db, auditSvc: auditSvc}
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
		SELECT r.id, r.name, r.description, COUNT(ur.user_id)
		FROM roles r
		LEFT JOIN user_roles ur ON ur.role_id = r.id
		GROUP BY r.id, r.name, r.description
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.UserCount); err != nil {
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

func (r *Repository) Create(ctx context.Context, role *Role, permissionNames []string, actorUserIDs ...*int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var id int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO roles (name, description)
		VALUES ($1, $2)
		RETURNING id
	`, role.Name, role.Description).Scan(&id); err != nil {
		return 0, err
	}
	if err := replacePermissions(ctx, tx, id, permissionNames); err != nil {
		return 0, err
	}
	if err := r.recordAudit(ctx, tx, audit.AuditLog{
		ActorUserID: firstActorUserID(actorUserIDs),
		Action:      "role.create",
		Resource:    "role",
		ResourceID:  stringID(id),
		Metadata:    roleAuditMetadata(role, permissionNames),
	}); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *Repository) Update(ctx context.Context, role *Role, permissionNames []string, actorUserIDs ...*int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM roles WHERE id = $1 FOR UPDATE`, role.ID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoleNotFound
	}
	if err != nil {
		return err
	}
	if name == "SUPER_ADMIN" {
		return ErrProtectedRole
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE roles SET name = $1, description = $2, updated_at = NOW() WHERE id = $3
	`, role.Name, role.Description, role.ID); err != nil {
		return err
	}
	if err := replacePermissions(ctx, tx, role.ID, permissionNames); err != nil {
		return err
	}
	if err := r.recordAudit(ctx, tx, audit.AuditLog{
		ActorUserID: firstActorUserID(actorUserIDs),
		Action:      "role.update",
		Resource:    "role",
		ResourceID:  stringID(role.ID),
		Metadata:    roleAuditMetadata(role, permissionNames),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) Delete(ctx context.Context, id int64, actorUserIDs ...*int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM roles WHERE id = $1 FOR UPDATE`, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoleNotFound
	}
	if err != nil {
		return err
	}
	if name == "SUPER_ADMIN" {
		return ErrProtectedRole
	}
	var userCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_roles WHERE role_id = $1`, id).Scan(&userCount); err != nil {
		return err
	}
	if userCount > 0 {
		return ErrRoleHasUsers
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM roles WHERE id = $1`, id); err != nil {
		return err
	}
	if err := r.recordAudit(ctx, tx, audit.AuditLog{
		ActorUserID: firstActorUserID(actorUserIDs),
		Action:      "role.delete",
		Resource:    "role",
		ResourceID:  stringID(id),
		Metadata:    map[string]any{"name": name},
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) recordAudit(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) error {
	if r.auditSvc == nil {
		return nil
	}
	_, err := r.auditSvc.RecordWithTx(ctx, tx, auditLog)
	return err
}

func firstActorUserID(actorUserIDs []*int64) *int64 {
	if len(actorUserIDs) == 0 {
		return nil
	}
	return actorUserIDs[0]
}

func stringID(id int64) *string {
	value := fmt.Sprintf("%d", id)
	return &value
}

func roleAuditMetadata(role *Role, permissionNames []string) map[string]any {
	return map[string]any{
		"name":        role.Name,
		"description": role.Description,
		"permissions": permissionNames,
	}
}

func replacePermissions(ctx context.Context, tx *sql.Tx, roleID int64, permissionNames []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	for _, name := range permissionNames {
		var permissionID int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM permissions WHERE name = $1`, name).Scan(&permissionID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrPermissionNotFound, name)
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
		`, roleID, permissionID); err != nil {
			return err
		}
	}
	return nil
}
