INSERT INTO
    permissions (name, description)
SELECT 'suppliers.read', 'Read suppliers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'suppliers.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'suppliers.create', 'Create suppliers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'suppliers.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'suppliers.update', 'Update suppliers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'suppliers.update'
    );

INSERT INTO
    permissions (name, description)
SELECT 'suppliers.delete', 'Delete suppliers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'suppliers.delete'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN (
        'suppliers.read', 'suppliers.create', 'suppliers.update', 'suppliers.delete'
    )
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;