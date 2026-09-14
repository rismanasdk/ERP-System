INSERT INTO
    permissions (name, description)
SELECT permission.name, permission.description
FROM (
        VALUES (
                'sales.payment.read', 'Read sales payments'
            ), (
                'sales.payment.create', 'Create sales payments'
            )
    ) AS permission (name, description)
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            permissions.name = permission.name
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN (
        'sales.payment.read', 'sales.payment.create'
    )
WHERE
    r.name IN (
        'SUPER_ADMIN',
        'ADMIN',
        'MANAGER'
    )
ON CONFLICT (role_id, permission_id) DO NOTHING;