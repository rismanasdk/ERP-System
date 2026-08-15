ALTER TABLE suppliers ADD COLUMN deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_suppliers_deleted_at ON suppliers (deleted_at);