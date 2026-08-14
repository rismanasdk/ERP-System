INSERT INTO
    permissions (name, description)
SELECT 'sales.complete', 'Complete sales'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'sales.complete'
    );

INSERT INTO
    permissions (name, description)
SELECT 'sales.cancel', 'Cancel sales'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'sales.cancel'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN (
        'sales.complete', 'sales.cancel'
    )
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;
