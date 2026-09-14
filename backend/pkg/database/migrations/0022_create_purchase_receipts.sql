ALTER TABLE purchases
DROP CONSTRAINT IF EXISTS purchases_status_valid;

ALTER TABLE purchases
ADD CONSTRAINT purchases_status_valid CHECK (
    status IN (
        'DRAFT',
        'ORDERED',
        'PARTIALLY_RECEIVED',
        'RECEIVED',
        'COMPLETED',
        'CANCELLED'
    )
);

ALTER TABLE purchase_items
ADD COLUMN IF NOT EXISTS received_quantity BIGINT NOT NULL DEFAULT 0;

ALTER TABLE purchase_items
ADD CONSTRAINT purchase_items_received_quantity_valid CHECK (
    received_quantity >= 0
    AND received_quantity <= quantity
);

CREATE TABLE IF NOT EXISTS purchase_receipts (
    id BIGSERIAL PRIMARY KEY,
    purchase_order_id BIGINT NOT NULL REFERENCES purchases (id),
    branch_id BIGINT NOT NULL REFERENCES branches (id),
    received_by BIGINT NOT NULL REFERENCES users (id),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_receipt_items (
    id BIGSERIAL PRIMARY KEY,
    purchase_receipt_id BIGINT NOT NULL REFERENCES purchase_receipts (id) ON DELETE CASCADE,
    purchase_order_item_id BIGINT NOT NULL REFERENCES purchase_items (id),
    product_id BIGINT NOT NULL REFERENCES products (id),
    quantity_received BIGINT NOT NULL CHECK (quantity_received > 0)
);

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_purchase_order_id ON purchase_receipts (purchase_order_id);

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_branch_id ON purchase_receipts (branch_id);

CREATE INDEX IF NOT EXISTS idx_purchase_receipts_received_by ON purchase_receipts (received_by);

CREATE INDEX IF NOT EXISTS idx_purchase_receipt_items_receipt_id ON purchase_receipt_items (purchase_receipt_id);

CREATE INDEX IF NOT EXISTS idx_purchase_receipt_items_order_item_id ON purchase_receipt_items (purchase_order_item_id);