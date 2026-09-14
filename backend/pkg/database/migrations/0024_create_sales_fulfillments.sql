ALTER TABLE sales
ADD COLUMN IF NOT EXISTS customer_id BIGINT REFERENCES customers (id);

ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_valid;

ALTER TABLE sales
ADD CONSTRAINT sales_status_valid CHECK (
    status IN (
        'DRAFT',
        'CONFIRMED',
        'PARTIALLY_FULFILLED',
        'FULFILLED',
        'COMPLETED',
        'CANCELLED'
    )
);

ALTER TABLE sale_items
ADD COLUMN IF NOT EXISTS fulfilled_quantity BIGINT NOT NULL DEFAULT 0;

ALTER TABLE sale_items
ADD CONSTRAINT sale_items_fulfilled_quantity_valid CHECK (
    fulfilled_quantity >= 0
    AND fulfilled_quantity <= quantity
);

CREATE TABLE IF NOT EXISTS sales_fulfillments (
    id BIGSERIAL PRIMARY KEY,
    sales_order_id BIGINT NOT NULL REFERENCES sales (id),
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    fulfilled_by BIGINT NOT NULL REFERENCES users (id),
    fulfilled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sales_fulfillment_items (
    id BIGSERIAL PRIMARY KEY,
    sales_fulfillment_id BIGINT NOT NULL REFERENCES sales_fulfillments (id) ON DELETE CASCADE,
    sales_order_item_id BIGINT NOT NULL REFERENCES sale_items (id),
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity_fulfilled BIGINT NOT NULL CHECK (quantity_fulfilled > 0)
);

CREATE INDEX IF NOT EXISTS idx_sales_customer_id ON sales (customer_id);

CREATE INDEX IF NOT EXISTS idx_sales_fulfillments_order_id ON sales_fulfillments (sales_order_id);

CREATE INDEX IF NOT EXISTS idx_sales_fulfillments_branch_id ON sales_fulfillments (branch_id);

CREATE INDEX IF NOT EXISTS idx_sales_fulfillments_fulfilled_by ON sales_fulfillments (fulfilled_by);

CREATE INDEX IF NOT EXISTS idx_sales_fulfillment_items_fulfillment_id ON sales_fulfillment_items (sales_fulfillment_id);

CREATE INDEX IF NOT EXISTS idx_sales_fulfillment_items_order_item_id ON sales_fulfillment_items (sales_order_item_id);