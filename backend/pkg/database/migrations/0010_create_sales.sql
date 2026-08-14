CREATE TABLE IF NOT EXISTS sales (
    id BIGSERIAL PRIMARY KEY,
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    sale_number VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL,
    total_amount NUMERIC(20, 2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_total_amount_non_negative CHECK (total_amount >= 0),
    CONSTRAINT sales_status_valid CHECK (
        status IN (
            'DRAFT',
            'COMPLETED',
            'CANCELLED'
        )
    )
);

CREATE TABLE IF NOT EXISTS sale_items (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES sales (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity BIGINT NOT NULL,
    unit_price NUMERIC(20, 2) NOT NULL,
    subtotal NUMERIC(20, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sale_items_quantity_positive CHECK (quantity > 0),
    CONSTRAINT sale_items_unit_price_non_negative CHECK (unit_price >= 0),
    CONSTRAINT sale_items_subtotal_non_negative CHECK (subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS idx_sales_branch_id ON sales (branch_id);

CREATE INDEX IF NOT EXISTS idx_sales_created_by ON sales (created_by);

CREATE INDEX IF NOT EXISTS idx_sales_status ON sales (status);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items (sale_id);

CREATE INDEX IF NOT EXISTS idx_sale_items_product_id ON sale_items (product_id);