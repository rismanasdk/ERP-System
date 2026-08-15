INSERT INTO
    permissions (name, description)
SELECT 'customers.read', 'Read customers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'customers.read'
    );

INSERT INTO
    permissions (name, description)
SELECT 'customers.create', 'Create customers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'customers.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'customers.update', 'Update customers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'customers.update'
    );

INSERT INTO
    permissions (name, description)
SELECT 'customers.delete', 'Delete customers'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'customers.delete'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN (
        'customers.read', 'customers.create', 'customers.update', 'customers.delete'
    )
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;