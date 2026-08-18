INSERT INTO
    roles (name, description)
SELECT 'ADMIN', 'Administrative user'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM roles
        WHERE
            name = 'ADMIN'
    );

INSERT INTO
    roles (name, description)
SELECT 'MANAGER', 'Operational manager'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM roles
        WHERE
            name = 'MANAGER'
    );

INSERT INTO
    roles (name, description)
SELECT 'KASIR', 'Cashier / Sales'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM roles
        WHERE
            name = 'KASIR'
    );

INSERT INTO
    roles (name, description)
SELECT 'STAFF', 'Operational staff'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM roles
        WHERE
            name = 'STAFF'
    );
WITH
    admin_perms AS (
        SELECT id
        FROM permissions
        WHERE
            name IN (
                'users.read',
                'users.create',
                'users.update',
                'users.delete',
                'roles.read',
                'roles.create',
                'roles.update',
                'roles.delete',
                'products.read',
                'products.create',
                'products.update',
                'products.delete',
                'inventory.read',
                'inventory.create',
                'inventory.adjust',
                'reports.read'
            )
    )
INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN admin_perms p
WHERE
    r.name = 'ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- MANAGER: monitoring and reports + product/inventory read
WITH
    manager_perms AS (
        SELECT id
        FROM permissions
        WHERE
            name IN (
                'products.read',
                'inventory.read',
                'sales.read',
                'reports.read',
                'dashboard.read'
            )
    )
INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN manager_perms p
WHERE
    r.name = 'MANAGER'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- KASIR: sales create/read and inventory read
WITH
    kasir_perms AS (
        SELECT id
        FROM permissions
        WHERE
            name IN (
                'sales.create',
                'sales.read',
                'inventory.read',
                'products.read'
            )
    )
INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN kasir_perms p
WHERE
    r.name = 'KASIR'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- STAFF: limited operational permissions (products/inventory read)
WITH
    staff_perms AS (
        SELECT id
        FROM permissions
        WHERE
            name IN (
                'products.read',
                'inventory.read'
            )
    )
INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN staff_perms p
WHERE
    r.name = 'STAFF'
ON CONFLICT (role_id, permission_id) DO NOTHING;