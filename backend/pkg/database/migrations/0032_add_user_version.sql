ALTER TABLE users
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT users_version_positive CHECK (version >= 1);