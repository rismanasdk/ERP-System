ALTER TABLE branches
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT branches_version_positive CHECK (version >= 1);

ALTER TABLE customers
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT customers_version_positive CHECK (version >= 1);

ALTER TABLE suppliers
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT suppliers_version_positive CHECK (version >= 1);