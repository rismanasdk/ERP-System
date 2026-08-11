CREATE TABLE IF NOT EXISTS inventory (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products (id),
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    quantity BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT inventory_unique_product_branch UNIQUE (product_id, branch_id),
    CONSTRAINT inventory_quantity_non_negative CHECK (quantity >= 0)
);

CREATE INDEX IF NOT EXISTS idx_inventory_branch_id ON inventory (branch_id);

CREATE INDEX IF NOT EXISTS idx_inventory_product_id ON inventory (product_id);

CREATE TABLE IF NOT EXISTS stock_movements (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products (id),
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    movement_type VARCHAR(50) NOT NULL,
    quantity_delta BIGINT NOT NULL,
    reference_type VARCHAR(100),
    reference_id BIGINT,
    actor_user_id BIGINT REFERENCES users (id),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_branch_id ON stock_movements (branch_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id ON stock_movements (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_movements_created_at ON stock_movements (created_at);