CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(100) NOT NULL,
    barcode VARCHAR(100),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    unit VARCHAR(50),
    purchase_price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    selling_price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT products_sku_non_empty CHECK (length(trim(sku)) > 0),
    CONSTRAINT products_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT products_purchase_price_non_negative CHECK (purchase_price >= 0),
    CONSTRAINT products_selling_price_non_negative CHECK (selling_price >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku ON products (sku);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_barcode ON products (barcode)
WHERE
    barcode IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);

CREATE INDEX IF NOT EXISTS idx_products_is_active ON products (is_active);

CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

CREATE INDEX IF NOT EXISTS idx_products_category ON products (category);

CREATE INDEX IF NOT EXISTS idx_products_unit ON products (unit);