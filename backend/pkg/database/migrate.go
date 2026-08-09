package database

import (
	"database/sql"
	"errors"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    family_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
`

var defaultPermissions = []struct {
	name        string
	description string
}{
	{name: "users.read", description: "Read user accounts"},
	{name: "users.create", description: "Create user accounts"},
	{name: "users.update", description: "Update user accounts"},
	{name: "users.delete", description: "Delete user accounts"},
	{name: "roles.read", description: "Read roles"},
	{name: "roles.create", description: "Create roles"},
	{name: "roles.update", description: "Update roles"},
	{name: "roles.delete", description: "Delete roles"},
	{name: "permissions.read", description: "Read permissions"},
	{name: "permissions.create", description: "Create permissions"},
	{name: "products.read", description: "Read products"},
	{name: "products.create", description: "Create products"},
	{name: "products.update", description: "Update products"},
	{name: "products.delete", description: "Delete products"},
	{name: "inventory.read", description: "Read inventory"},
	{name: "inventory.create", description: "Create inventory"},
	{name: "inventory.adjust", description: "Adjust inventory"},
	{name: "sales.read", description: "Read sales"},
	{name: "sales.create", description: "Create sales"},
	{name: "reports.read", description: "Read reports"},
}

const superAdminRoleName = "SUPER_ADMIN"

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedPermissions(db); err != nil {
		return err
	}
	return seedSuperAdminPermissions(db)
}

func seedRoles(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO roles (name, description)
        SELECT $1, $2
        WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = $1)`,
		superAdminRoleName,
		"Initial system super admin role",
	)
	return err
}

func seedPermissions(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, p := range defaultPermissions {
		_, err = tx.Exec(`INSERT INTO permissions (name, description)
            SELECT $1, $2
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE name = $1)`,
			p.name,
			p.description,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func seedSuperAdminPermissions(db *sql.DB) error {
	result, err := db.Exec(`
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = $1
  AND NOT EXISTS (
        SELECT 1 FROM role_permissions rp
        WHERE rp.role_id = r.id
          AND rp.permission_id = p.id
    )
`, superAdminRoleName)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return nil
	}
	return nil
}

func GetSuperAdminRoleID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM roles WHERE name = $1`, superAdminRoleName).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}
