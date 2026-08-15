INSERT INTO
    permissions (name, description)
SELECT 'dashboard.read', 'Read dashboard summary'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'dashboard.read'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name = 'dashboard.read'
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;