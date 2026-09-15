DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM sales_fulfillment_items sfi
        JOIN sales_fulfillments sf ON sf.id = sfi.sales_fulfillment_id
        JOIN sale_items si ON si.id = sfi.sales_order_item_id
        WHERE si.sale_id <> sf.sales_order_id OR si.product_id <> sfi.product_id
    ) THEN
        RAISE EXCEPTION 'existing sales fulfillment items reference a different sales order';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM purchase_receipt_items pri
        JOIN purchase_receipts pr ON pr.id = pri.purchase_receipt_id
        JOIN purchase_items pi ON pi.id = pri.purchase_order_item_id
        WHERE pi.purchase_id <> pr.purchase_order_id OR pi.product_id <> pri.product_id
    ) THEN
        RAISE EXCEPTION 'existing purchase receipt items reference a different purchase order';
    END IF;
END $$;

ALTER TABLE sale_items
ADD CONSTRAINT sale_items_sale_id_id_product_key UNIQUE (sale_id, id, product_id);

ALTER TABLE sales_fulfillments
ADD CONSTRAINT sales_fulfillments_id_order_key UNIQUE (id, sales_order_id);

ALTER TABLE sales_fulfillment_items
ADD COLUMN sales_order_id BIGINT;

UPDATE sales_fulfillment_items sfi
SET
    sales_order_id = sf.sales_order_id
FROM sales_fulfillments sf
WHERE
    sf.id = sfi.sales_fulfillment_id;

ALTER TABLE sales_fulfillment_items
ALTER COLUMN sales_order_id
SET NOT NULL;

ALTER TABLE sales_fulfillment_items
ADD CONSTRAINT sales_fulfillment_items_fulfillment_order_fk FOREIGN KEY (
    sales_fulfillment_id,
    sales_order_id
) REFERENCES sales_fulfillments (id, sales_order_id),
ADD CONSTRAINT sales_fulfillment_items_order_item_product_fk FOREIGN KEY (
    sales_order_id,
    sales_order_item_id,
    product_id
) REFERENCES sale_items (sale_id, id, product_id);

ALTER TABLE purchase_items
ADD CONSTRAINT purchase_items_purchase_id_id_product_key UNIQUE (purchase_id, id, product_id);

ALTER TABLE purchase_receipts
ADD CONSTRAINT purchase_receipts_id_order_key UNIQUE (id, purchase_order_id);

ALTER TABLE purchase_receipt_items
ADD COLUMN purchase_order_id BIGINT;

UPDATE purchase_receipt_items pri
SET
    purchase_order_id = pr.purchase_order_id
FROM purchase_receipts pr
WHERE
    pr.id = pri.purchase_receipt_id;

ALTER TABLE purchase_receipt_items
ALTER COLUMN purchase_order_id
SET NOT NULL;

ALTER TABLE purchase_receipt_items
ADD CONSTRAINT purchase_receipt_items_receipt_order_fk FOREIGN KEY (
    purchase_receipt_id,
    purchase_order_id
) REFERENCES purchase_receipts (id, purchase_order_id),
ADD CONSTRAINT purchase_receipt_items_order_item_product_fk FOREIGN KEY (
    purchase_order_id,
    purchase_order_item_id,
    product_id
) REFERENCES purchase_items (purchase_id, id, product_id);