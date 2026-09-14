CREATE TABLE IF NOT EXISTS product_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT product_categories_name_non_empty CHECK (length(trim(name)) > 0)
);

ALTER TABLE products ADD COLUMN IF NOT EXISTS category_id BIGINT;

ALTER TABLE products
ADD COLUMN IF NOT EXISTS minimum_stock BIGINT NOT NULL DEFAULT 0;

ALTER TABLE products
ADD CONSTRAINT products_minimum_stock_non_negative CHECK (minimum_stock >= 0);

INSERT INTO
    product_categories (name)
SELECT DISTINCT
    trim(category)
FROM products
WHERE
    category IS NOT NULL
    AND trim(category) <> ''
ON CONFLICT (name) DO NOTHING;

UPDATE products p
SET
    category_id = c.id
FROM product_categories c
WHERE
    p.category_id IS NULL
    AND trim(p.category) = c.name;

ALTER TABLE products
ADD CONSTRAINT products_category_id_fkey FOREIGN KEY (category_id) REFERENCES product_categories (id);

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products (category_id);

INSERT INTO
    permissions (name, description)
VALUES (
        'product_categories.read',
        'Read product categories'
    ),
    (
        'product_categories.create',
        'Create product categories'
    ),
    (
        'product_categories.update',
        'Update product categories'
    ),
    (
        'product_categories.delete',
        'Delete product categories'
    )
ON CONFLICT (name) DO NOTHING;

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN permissions p
WHERE
    r.name = 'ADMIN'
    AND p.name LIKE 'product_categories.%'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO
    role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
    CROSS JOIN permissions p
WHERE
    r.name IN ('MANAGER', 'STAFF')
    AND p.name = 'product_categories.read'
ON CONFLICT (role_id, permission_id) DO NOTHING;