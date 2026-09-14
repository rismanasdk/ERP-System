CREATE TABLE IF NOT EXISTS sales_payments (
    id BIGSERIAL PRIMARY KEY,
    sales_order_id BIGINT NOT NULL REFERENCES sales (id),
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(30) NOT NULL,
    reference_number VARCHAR(150),
    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_payments_method_valid CHECK (
        payment_method IN (
            'CASH',
            'BANK_TRANSFER',
            'OTHER',
            'QRIS'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_sales_payments_sales_order_id ON sales_payments (sales_order_id);

CREATE INDEX IF NOT EXISTS idx_sales_payments_created_by ON sales_payments (created_by);

CREATE INDEX IF NOT EXISTS idx_sales_payments_paid_at ON sales_payments (paid_at);