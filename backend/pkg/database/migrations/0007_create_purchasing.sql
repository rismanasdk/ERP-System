CREATE TABLE IF NOT EXISTS suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    phone VARCHAR(50),
    email VARCHAR(150),
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT suppliers_name_not_empty CHECK (trim(name) <> ''),
    CONSTRAINT suppliers_code_not_empty CHECK (trim(code) <> '')
);

CREATE TABLE IF NOT EXISTS purchases (
    id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    supplier_id BIGINT NOT NULL REFERENCES suppliers (id),
    purchase_number VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL,
    total_amount NUMERIC(20, 2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchases_total_amount_non_negative CHECK (total_amount >= 0),
    CONSTRAINT purchases_status_valid CHECK (
        status IN (
            'DRAFT',
            'COMPLETED',
            'CANCELLED'
        )
    )
);

CREATE TABLE IF NOT EXISTS purchase_items (
    id BIGSERIAL PRIMARY KEY,
    purchase_id BIGINT NOT NULL REFERENCES purchases (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity BIGINT NOT NULL,
    unit_cost NUMERIC(20, 2) NOT NULL,
    subtotal NUMERIC(20, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_items_quantity_positive CHECK (quantity > 0),
    CONSTRAINT purchase_items_unit_cost_non_negative CHECK (unit_cost >= 0),
    CONSTRAINT purchase_items_subtotal_non_negative CHECK (subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_purchases_branch_id ON purchases (branch_id);

CREATE INDEX IF NOT EXISTS idx_purchases_supplier_id ON purchases (supplier_id);

CREATE INDEX IF NOT EXISTS idx_purchases_created_by ON purchases (created_by);

CREATE INDEX IF NOT EXISTS idx_purchases_status ON purchases (status);

CREATE INDEX IF NOT EXISTS idx_purchase_items_purchase_id ON purchase_items (purchase_id);

CREATE INDEX IF NOT EXISTS idx_purchase_items_product_id ON purchase_items (product_id);