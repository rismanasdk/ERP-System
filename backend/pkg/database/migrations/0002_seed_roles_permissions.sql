INSERT INTO
    roles (name, description)
SELECT 'SUPER_ADMIN', 'Initial system super admin role'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM roles
        WHERE
            name = 'SUPER_ADMIN'
    );

INSERT INTO
    permissions (name, description)
SELECT 'users.read', 'Read user accounts'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'users.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'users.create', 'Create user accounts'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'users.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'users.update', 'Update user accounts'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'users.update'
    );

INSERT INTO
    permissions (name, description)
SELECT 'users.delete', 'Delete user accounts'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'users.delete'
    );

INSERT INTO
    permissions (name, description)
SELECT 'roles.read', 'Read roles'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'roles.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'roles.create', 'Create roles'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'roles.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'roles.update', 'Update roles'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'roles.update'
    );

INSERT INTO
    permissions (name, description)
SELECT 'roles.delete', 'Delete roles'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'roles.delete'
    );

INSERT INTO
    permissions (name, description)
SELECT 'permissions.read', 'Read permissions'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'permissions.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'permissions.create', 'Create permissions'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'permissions.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'products.read', 'Read products'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'products.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'products.create', 'Create products'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'products.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'products.update', 'Update products'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'products.update'
    );

INSERT INTO
    permissions (name, description)
SELECT 'products.delete', 'Delete products'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'products.delete'
    );

INSERT INTO
    permissions (name, description)
SELECT 'inventory.read', 'Read inventory'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'inventory.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'inventory.create', 'Create inventory'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'inventory.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'inventory.adjust', 'Adjust inventory'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'inventory.adjust'
    );

INSERT INTO
    permissions (name, description)
SELECT 'sales.read', 'Read sales'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'sales.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'sales.create', 'Create sales'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'sales.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'reports.read', 'Read reports'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'reports.read'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN permissions p
WHERE
    r.name = 'SUPER_ADMIN' ON CONFLICT (role_id, permission_id) DO NOTHING;