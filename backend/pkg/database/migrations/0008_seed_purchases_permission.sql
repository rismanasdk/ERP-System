INSERT INTO
    permissions (name, description)
SELECT 'purchases.create', 'Create purchases'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'purchases.create'
    );

INSERT INTO
    permissions (name, description)
SELECT 'purchases.read', 'Read purchases'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'purchases.read'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN ('purchases.create', 'purchases.read')
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;   