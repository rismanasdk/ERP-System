//go:build integration
// +build integration

package reporting

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/master/products"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/database"
)

func TestReportingIntegration_RealPostgresReports(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	prefix := fmt.Sprintf("rpt_%d", time.Now().UnixNano())
	branchCode := prefix + "_branch"
	productSKU := prefix + "_sku"
	userEmail := prefix + "@example.local"

	if _, err := db.ExecContext(ctx, `INSERT INTO branches (name, code, is_active) VALUES ($1, $2, true) ON CONFLICT (code) DO NOTHING`, "Reporting Branch", branchCode); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	var branchID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM branches WHERE code = $1`, branchCode).Scan(&branchID); err != nil {
		t.Fatalf("failed to fetch branch id: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO products (sku, barcode, name, purchase_price, selling_price, is_active) VALUES ($1, $2, $3, 10, 20, true) ON CONFLICT (sku) DO NOTHING`, productSKU, prefix, "Reporting Product"); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	var productID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM products WHERE sku = $1`, productSKU).Scan(&productID); err != nil {
		t.Fatalf("failed to fetch product id: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) ON CONFLICT (email) DO NOTHING`, userEmail, "hashed", "Reporting User"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	var userID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, userEmail).Scan(&userID); err != nil {
		t.Fatalf("failed to fetch user id: %v", err)
	}

	roleRepo := roles.NewRepository(db)
	userRepo := users.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	authSvc := auth.NewService(userRepo, roleRepo, permRepo, nil, nil)
	branchSvc := branches.NewService(branches.NewRepository(db), authSvc, nil)
	productSvc := products.NewRepository(db)
	reportingRepo := NewRepository(db)
	reportingSvc := NewService(reportingRepo, branchSvc, authSvc, authSvc)

	if _, err := db.ExecContext(ctx, `INSERT INTO user_branches (user_id, branch_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, branchID); err != nil {
		t.Fatalf("failed to assign user to branch: %v", err)
	}
	if err := userRepo.AddRole(ctx, userID, 1); err != nil {
		// ignore if default role not present; this test is mainly for report generation
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO inventory (product_id, branch_id, quantity) VALUES ($1, $2, 12) ON CONFLICT (product_id, branch_id) DO UPDATE SET quantity = inventory.quantity + EXCLUDED.quantity`, productID, branchID); err != nil {
		t.Fatalf("failed to set inventory: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO stock_movements (product_id, branch_id, movement_type, quantity_delta) VALUES ($1, $2, 'IN', 10), ($1, $2, 'OUT', -5), ($1, $2, 'ADJUSTMENT', 2)`, productID, branchID); err != nil {
		t.Fatalf("failed to insert stock movements: %v", err)
	}

	saleNumber := prefix + "_sale"
	if _, err := db.ExecContext(ctx, `INSERT INTO sales (branch_id, sale_number, status, total_amount, created_by) VALUES ($1, $2, 'COMPLETED', 250.00, $3)`, branchID, saleNumber, userID); err != nil {
		t.Fatalf("failed to insert sale: %v", err)
	}
	var saleID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM sales WHERE sale_number = $1`, saleNumber).Scan(&saleID); err != nil {
		t.Fatalf("failed to fetch sale id: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES ($1, $2, 5, 50, 250)`, saleID, productID, 5, 50.0, 250.0); err != nil {
		t.Fatalf("failed to insert sale item: %v", err)
	}

	purchaseNumber := prefix + "_purchase"
	if _, err := db.ExecContext(ctx, `INSERT INTO purchases (branch_id, supplier_id, purchase_number, status, total_amount, created_by) VALUES ($1, $2, $3, 'COMPLETED', 90.00, $4)`, branchID, 1, purchaseNumber, userID); err != nil {
		t.Fatalf("failed to insert purchase: %v", err)
	}
	var purchaseID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM purchases WHERE purchase_number = $1`, purchaseNumber).Scan(&purchaseID); err != nil {
		t.Fatalf("failed to fetch purchase id: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal) VALUES ($1, $2, 3, 30, 90)`, purchaseID, productID, 3, 30.0, 90.0); err != nil {
		t.Fatalf("failed to insert purchase item: %v", err)
	}

	ctx = auth.ContextWithUserID(ctx, userID)
	if _, err := productSvc.GetByID(ctx, productID); err != nil {
		t.Fatalf("product lookup failed: %v", err)
	}

	start := time.Now().AddDate(0, 0, -1)
	end := time.Now().AddDate(0, 0, 1)
	branchScope := &branchID
	if _, err := reportingSvc.GetSalesReport(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"), branchScope); err != nil {
		t.Fatalf("sales report failed: %v", err)
	}
	if _, err := reportingSvc.GetPurchasesReport(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"), branchScope); err != nil {
		t.Fatalf("purchases report failed: %v", err)
	}
	inventoryReport, err := reportingSvc.GetInventoryReport(ctx, branchScope, &productID)
	if err != nil {
		t.Fatalf("inventory report failed: %v", err)
	}
	if inventoryReport.TotalQuantity == 0 {
		t.Fatalf("expected inventory quantity > 0")
	}
	if _, err := reportingSvc.GetProfitReport(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"), branchScope); err != nil {
		t.Fatalf("profit report failed: %v", err)
	}
}
