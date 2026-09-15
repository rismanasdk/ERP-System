package purchasing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) CreateSupplier(ctx context.Context, supplier *Supplier) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO suppliers (name, code, phone, email, address, is_active)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, supplier.Name, supplier.Code, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetSupplierByID(ctx context.Context, id int64) (*Supplier, error) {
	supplier := &Supplier{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
        WHERE id = $1
    `, id).Scan(
		&supplier.ID,
		&supplier.Name,
		&supplier.Code,
		&supplier.Phone,
		&supplier.Email,
		&supplier.Address,
		&supplier.IsActive,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (r *Repository) GetSupplierByCode(ctx context.Context, code string) (*Supplier, error) {
	supplier := &Supplier{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
        WHERE code = $1
    `, strings.TrimSpace(code)).Scan(
		&supplier.ID,
		&supplier.Name,
		&supplier.Code,
		&supplier.Phone,
		&supplier.Email,
		&supplier.Address,
		&supplier.IsActive,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return supplier, nil
}

func (r *Repository) ListSuppliers(ctx context.Context, filter SupplierFilter) ([]Supplier, error) {
	query := `
        SELECT id, name, code, phone, email, address, is_active, created_at, updated_at
        FROM suppliers
    `
	clauses := []string{}
	args := []any{}
	idx := 1

	if filter.Active != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *filter.Active)
		idx++
	}
	if filter.Search != nil && *filter.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", idx, idx+1))
		args = append(args, "%"+*filter.Search+"%", "%"+*filter.Search+"%")
		idx += 2
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []Supplier
	for rows.Next() {
		var supplier Supplier
		if err := rows.Scan(
			&supplier.ID,
			&supplier.Name,
			&supplier.Code,
			&supplier.Phone,
			&supplier.Email,
			&supplier.Address,
			&supplier.IsActive,
			&supplier.CreatedAt,
			&supplier.UpdatedAt,
		); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, supplier)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppliers, nil
}

func (r *Repository) UpdateSupplier(ctx context.Context, supplier *Supplier) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE suppliers
        SET name = $1, code = $2, phone = $3, email = $4, address = $5, is_active = $6, updated_at = NOW()
        WHERE id = $7
    `, supplier.Name, supplier.Code, supplier.Phone, supplier.Email, supplier.Address, supplier.IsActive, supplier.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreatePurchase(ctx context.Context, purchase *Purchase) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, purchase.BranchID, purchase.SupplierID, purchase.PurchaseNumber, purchase.Status, purchase.TotalAmount, purchase.Notes, purchase.CreatedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreatePurchaseWithTx(ctx context.Context, tx *sql.Tx, purchase *Purchase) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, purchase.BranchID, purchase.SupplierID, purchase.PurchaseNumber, purchase.Status, purchase.TotalAmount, purchase.Notes, purchase.CreatedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetPurchaseByID(ctx context.Context, id int64) (*Purchase, error) {
	purchase := &Purchase{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
        WHERE id = $1
    `, id).Scan(
		&purchase.ID,
		&purchase.BranchID,
		&purchase.SupplierID,
		&purchase.PurchaseNumber,
		&purchase.Status,
		&purchase.TotalAmount,
		&purchase.Notes,
		&purchase.CreatedBy,
		&purchase.CreatedAt,
		&purchase.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return purchase, nil
}

func (r *Repository) GetPurchaseByNumber(ctx context.Context, number string) (*Purchase, error) {
	purchase := &Purchase{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
        WHERE purchase_number = $1
    `, strings.TrimSpace(number)).Scan(
		&purchase.ID,
		&purchase.BranchID,
		&purchase.SupplierID,
		&purchase.PurchaseNumber,
		&purchase.Status,
		&purchase.TotalAmount,
		&purchase.Notes,
		&purchase.CreatedBy,
		&purchase.CreatedAt,
		&purchase.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return purchase, nil
}

func (r *Repository) ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error) {
	query := `
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
    `
	clauses := []string{}
	args := []any{}
	idx := 1

	if filter.BranchID != nil {
		clauses = append(clauses, fmt.Sprintf("branch_id = $%d", idx))
		args = append(args, *filter.BranchID)
		idx++
	}
	if filter.SupplierID != nil {
		clauses = append(clauses, fmt.Sprintf("supplier_id = $%d", idx))
		args = append(args, *filter.SupplierID)
		idx++
	}
	if filter.Status != nil {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, *filter.Status)
		idx++
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var purchases []Purchase
	for rows.Next() {
		var purchase Purchase
		if err := rows.Scan(
			&purchase.ID,
			&purchase.BranchID,
			&purchase.SupplierID,
			&purchase.PurchaseNumber,
			&purchase.Status,
			&purchase.TotalAmount,
			&purchase.Notes,
			&purchase.CreatedBy,
			&purchase.CreatedAt,
			&purchase.UpdatedAt,
		); err != nil {
			return nil, err
		}
		purchases = append(purchases, purchase)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return purchases, nil
}

func (r *Repository) UpdatePurchaseStatus(ctx context.Context, id int64, status string) error {
	res, err := r.db.ExecContext(ctx, `
        UPDATE purchases
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreatePurchaseItem(ctx context.Context, item *PurchaseItem) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, item.PurchaseID, item.ProductID, item.Quantity, item.UnitCost, item.Subtotal).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) CreatePurchaseItemWithTx(ctx context.Context, tx *sql.Tx, item *PurchaseItem) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, item.PurchaseID, item.ProductID, item.Quantity, item.UnitCost, item.Subtotal).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetPurchaseItemByID(ctx context.Context, id int64) (*PurchaseItem, error) {
	item := &PurchaseItem{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, received_quantity, created_at, updated_at
        FROM purchase_items
        WHERE id = $1
    `, id).Scan(
		&item.ID,
		&item.PurchaseID,
		&item.ProductID,
		&item.Quantity,
		&item.UnitCost,
		&item.Subtotal,
		&item.ReceivedQuantity,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) ListPurchaseItemsByPurchaseID(ctx context.Context, purchaseID int64) ([]PurchaseItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, received_quantity, created_at, updated_at
        FROM purchase_items
        WHERE purchase_id = $1
        ORDER BY id ASC
    `, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PurchaseItem
	for rows.Next() {
		var item PurchaseItem
		if err := rows.Scan(
			&item.ID,
			&item.PurchaseID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitCost,
			&item.Subtotal,
			&item.ReceivedQuantity,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) GetPurchaseByIDForUpdate(ctx context.Context, tx *sql.Tx, id int64) (*Purchase, error) {
	purchase := &Purchase{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, branch_id, supplier_id, purchase_number, status, total_amount, notes, created_by, created_at, updated_at
        FROM purchases
        WHERE id = $1
        FOR UPDATE
    `, id).Scan(
		&purchase.ID,
		&purchase.BranchID,
		&purchase.SupplierID,
		&purchase.PurchaseNumber,
		&purchase.Status,
		&purchase.TotalAmount,
		&purchase.Notes,
		&purchase.CreatedBy,
		&purchase.CreatedAt,
		&purchase.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return purchase, nil
}

func (r *Repository) ListPurchaseItemsByPurchaseIDWithTx(ctx context.Context, tx *sql.Tx, purchaseID int64) ([]PurchaseItem, error) {
	rows, err := tx.QueryContext(ctx, `
        SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, created_at, updated_at
        FROM purchase_items
        WHERE purchase_id = $1
        ORDER BY id ASC
    `, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PurchaseItem
	for rows.Next() {
		var item PurchaseItem
		if err := rows.Scan(
			&item.ID,
			&item.PurchaseID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitCost,
			&item.Subtotal,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) ListPurchaseItemsForReceiveWithTx(ctx context.Context, tx *sql.Tx, purchaseID int64) ([]PurchaseItem, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, purchase_id, product_id, quantity, unit_cost, subtotal, received_quantity, created_at, updated_at FROM purchase_items WHERE purchase_id = $1 ORDER BY id ASC FOR UPDATE`, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PurchaseItem
	for rows.Next() {
		var item PurchaseItem
		if err := rows.Scan(&item.ID, &item.PurchaseID, &item.ProductID, &item.Quantity, &item.UnitCost, &item.Subtotal, &item.ReceivedQuantity, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpdatePurchaseStatusWithTx(ctx context.Context, tx *sql.Tx, id int64, status string) error {
	res, err := tx.ExecContext(ctx, `
        UPDATE purchases
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `, status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateReceiptWithTx(ctx context.Context, tx *sql.Tx, receipt *PurchaseReceipt) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO purchase_receipts (purchase_order_id, branch_id, received_by, notes)
        VALUES ($1, $2, $3, $4)
        RETURNING id, received_at
    `, receipt.PurchaseID, receipt.BranchID, receipt.ReceivedBy, receipt.Notes).Scan(&id, &receipt.ReceivedAt)
	return id, err
}

func (r *Repository) CreateReceiptItemWithTx(ctx context.Context, tx *sql.Tx, item *PurchaseReceiptItem) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO purchase_receipt_items (purchase_receipt_id, purchase_order_id, purchase_order_item_id, product_id, quantity_received)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, item.ReceiptID, item.PurchaseOrderID, item.PurchaseItemID, item.ProductID, item.QuantityReceived).Scan(&id)
	return id, err
}

func (r *Repository) UpdatePurchaseItemReceivedWithTx(ctx context.Context, tx *sql.Tx, itemID, received int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE purchase_items SET received_quantity = $1, updated_at = NOW() WHERE id = $2`, received, itemID)
	return err
}

func (r *Repository) ListReceipts(ctx context.Context, purchaseID int64) ([]PurchaseReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT pr.id, pr.purchase_order_id, pr.branch_id, pr.received_by, COALESCE(u.name, ''), pr.received_at, pr.notes,
               pri.id, pri.purchase_order_item_id, pri.product_id, pri.quantity_received
        FROM purchase_receipts pr
        LEFT JOIN users u ON u.id = pr.received_by
        LEFT JOIN purchase_receipt_items pri ON pri.purchase_receipt_id = pr.id
        WHERE pr.purchase_order_id = $1 ORDER BY pr.id ASC, pri.id ASC`, purchaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []PurchaseReceipt
	byID := map[int64]*PurchaseReceipt{}
	for rows.Next() {
		var receipt PurchaseReceipt
		var itemID, itemOrderID, itemProductID, itemQuantity sql.NullInt64
		if err := rows.Scan(&receipt.ID, &receipt.PurchaseID, &receipt.BranchID, &receipt.ReceivedBy, &receipt.ReceivedByName, &receipt.ReceivedAt, &receipt.Notes, &itemID, &itemOrderID, &itemProductID, &itemQuantity); err != nil {
			return nil, err
		}
		if existing, ok := byID[receipt.ID]; ok {
			receipt = *existing
		}
		if itemID.Valid {
			receipt.Items = append(receipt.Items, PurchaseReceiptItem{ID: itemID.Int64, ReceiptID: receipt.ID, PurchaseItemID: itemOrderID.Int64, ProductID: itemProductID.Int64, QuantityReceived: itemQuantity.Int64})
		}
		byID[receipt.ID] = &receipt
	}
	for _, receipt := range byID {
		receipts = append(receipts, *receipt)
	}
	return receipts, rows.Err()
}
