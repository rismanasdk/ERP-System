CREATE TABLE IF NOT EXISTS stock_transfers (
    id BIGSERIAL PRIMARY KEY,
    source_branch_id BIGINT NOT NULL REFERENCES branches (id),
    destination_branch_id BIGINT NOT NULL REFERENCES branches (id),
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED',
    notes TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT stock_transfers_different_branches CHECK (
        source_branch_id <> destination_branch_id
    ),
    CONSTRAINT stock_transfers_status_check CHECK (status IN ('COMPLETED'))
);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_source_branch_id ON stock_transfers (source_branch_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_destination_branch_id ON stock_transfers (destination_branch_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_product_id ON stock_transfers (product_id);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_created_by ON stock_transfers (created_by);

CREATE INDEX IF NOT EXISTS idx_stock_transfers_created_at ON stock_transfers (created_at);