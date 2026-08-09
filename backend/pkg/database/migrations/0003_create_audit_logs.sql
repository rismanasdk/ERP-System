CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id INT REFERENCES users (id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs (actor_user_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);

CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at);