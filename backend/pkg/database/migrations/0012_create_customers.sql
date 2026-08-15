CREATE TABLE IF NOT EXISTS customers (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(50),
    email VARCHAR(150),
    address TEXT,
    tax_id VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT customers_code_non_empty CHECK (length(trim(code)) > 0),
    CONSTRAINT customers_name_non_empty CHECK (length(trim(name)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_code ON customers (code);

CREATE INDEX IF NOT EXISTS idx_customers_name ON customers (name);

CREATE INDEX IF NOT EXISTS idx_customers_is_active ON customers (is_active);

CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers (deleted_at);