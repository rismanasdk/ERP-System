ALTER TABLE products
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT products_version_positive CHECK (version >= 1);

ALTER TABLE product_categories
ADD COLUMN version BIGINT NOT NULL DEFAULT 1,
ADD CONSTRAINT product_categories_version_positive CHECK (version >= 1);