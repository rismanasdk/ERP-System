
INSERT INTO permissions (name, description)
SELECT 'test.read', 'Testing read'
WHERE NOT EXISTS (
    SELECT 1 FROM permissions WHERE name = 'test.read'
);

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = 'test.read'
WHERE r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

