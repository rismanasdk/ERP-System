INSERT INTO
    permissions (name, description)
SELECT 'purchases.complete', 'Complete purchases'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'purchases.complete'
    );

INSERT INTO
    permissions (name, description)
SELECT 'purchases.cancel', 'Cancel purchases'
WHERE
    NOT EXISTS (
        SELECT 1
        FROM permissions
        WHERE
            name = 'purchases.cancel'
    );

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    JOIN permissions p ON p.name IN (
        'purchases.complete', 'purchases.cancel'
    )
WHERE
    r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;