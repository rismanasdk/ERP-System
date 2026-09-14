INSERT INTO
    permissions (name, description)
SELECT 'purchases.receive', 'Receive goods for purchases'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'purchases.receive'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name = 'purchases.receive'
WHERE
    r.name IN (
        'SUPER_ADMIN',
        'ADMIN',
        'MANAGER'
    )
ON CONFLICT (role_id, permission_id) DO NOTHING;