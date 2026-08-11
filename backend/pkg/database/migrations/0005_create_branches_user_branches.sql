CREATE TABLE IF NOT EXISTS branches (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT branches_name_non_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT branches_code_non_empty CHECK (length(trim(code)) > 0)
);

CREATE TABLE IF NOT EXISTS user_branches (
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    branch_id BIGINT NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, branch_id)
);

CREATE INDEX IF NOT EXISTS idx_user_branches_branch_id ON user_branches (branch_id);