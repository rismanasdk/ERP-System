//go:build integration
// +build integration

package sales

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/products"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/database"
)

func TestSalesCompleteConcurrentIntegration(t *testing.T) {
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
	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	branchRepo := branches.NewRepository(db)
	productRepo := products.NewRepository(db)
	saleRepo := NewRepository(db)
	invRepo := inventory.NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	authSvc := auth.NewService(userRepo, roleRepo, permRepo, nil, auditSvc)
	branchSvc := branches.NewService(branchRepo, authSvc, auditSvc)
	productSvc := productRepo
	svc := NewService(saleRepo, invRepo, productSvc, branchSvc, authSvc, auditSvc)

	randSuffix := fmt.Sprintf("sales_%d", time.Now().UnixNano())
	branchCode := "BR-" + randSuffix
	productSKU := "SKU-" + randSuffix
	userEmail := randSuffix + "@example.local"

	branchID, err := ensureBranch(db, branchCode)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	productID, err := ensureProduct(db, productSKU)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	userID, err := ensureUser(db, userEmail)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := ensureRoleAccess(db, userRepo, roleRepo, userID, branchID); err != nil {
		t.Fatalf("failed to grant role/branch access: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO inventory (product_id, branch_id, quantity) VALUES ($1, $2, $3) ON CONFLICT (product_id, branch_id) DO UPDATE SET quantity = EXCLUDED.quantity`, productID, branchID, 5); err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}

	firstSaleID, err := svc.CreateSale(auth.ContextWithUserID(ctx, userID), CreateSaleInput{BranchID: branchID, Items: []CreateSaleItemInput{{ProductID: productID, Quantity: 4, UnitPrice: 10}}})
	if err != nil {
		t.Fatalf("failed to create first sale: %v", err)
	}
	secondSaleID, err := svc.CreateSale(auth.ContextWithUserID(ctx, userID), CreateSaleInput{BranchID: branchID, Items: []CreateSaleItemInput{{ProductID: productID, Quantity: 4, UnitPrice: 10}}})
	if err != nil {
		t.Fatalf("failed to create second sale: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, saleID := range []int64{firstSaleID, secondSaleID} {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			results <- svc.CompleteSale(auth.ContextWithUserID(ctx, userID), id)
		}(saleID)
	}
	close(start)
	wg.Wait()
	close(results)

	var okCount int
	for err := range results {
		if err == nil {
			okCount++
		}
	}
	if okCount != 1 {
		t.Fatalf("expected exactly one sale to complete under stock contention, got %d successes", okCount)
	}

	var qty int64
	if err := db.QueryRowContext(ctx, `SELECT quantity FROM inventory WHERE product_id = $1 AND branch_id = $2`, productID, branchID).Scan(&qty); err != nil {
		t.Fatalf("failed to read inventory quantity: %v", err)
	}
	if qty != 1 {
		t.Fatalf("expected final inventory quantity 1 after one successful completion, got %d", qty)
	}
}

func TestSalesCompleteSameSaleConcurrentIntegration(t *testing.T) {
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
	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	branchRepo := branches.NewRepository(db)
	productRepo := products.NewRepository(db)
	saleRepo := NewRepository(db)
	invRepo := inventory.NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	authSvc := auth.NewService(userRepo, roleRepo, permRepo, nil, auditSvc)
	branchSvc := branches.NewService(branchRepo, authSvc, auditSvc)
	productSvc := productRepo
	svc := NewService(saleRepo, invRepo, productSvc, branchSvc, authSvc, auditSvc)

	randSuffix := fmt.Sprintf("sales_dup_%d", time.Now().UnixNano())
	branchCode := "BR-" + randSuffix
	productSKU := "SKU-" + randSuffix
	userEmail := randSuffix + "@example.local"

	branchID, err := ensureBranch(db, branchCode)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	productID, err := ensureProduct(db, productSKU)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	userID, err := ensureUser(db, userEmail)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := ensureRoleAccess(db, userRepo, roleRepo, userID, branchID); err != nil {
		t.Fatalf("failed to grant role/branch access: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inventory (product_id, branch_id, quantity) VALUES ($1, $2, $3) ON CONFLICT (product_id, branch_id) DO UPDATE SET quantity = EXCLUDED.quantity`, productID, branchID, 10); err != nil {
		t.Fatalf("failed to seed inventory: %v", err)
	}

	saleID, err := svc.CreateSale(auth.ContextWithUserID(ctx, userID), CreateSaleInput{BranchID: branchID, Items: []CreateSaleItemInput{{ProductID: productID, Quantity: 2, UnitPrice: 10}}})
	if err != nil {
		t.Fatalf("failed to create sale: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			results <- svc.CompleteSale(auth.ContextWithUserID(ctx, userID), saleID)
		}()
	}
	close(start)

	var successCount int
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}
	if successCount != 1 {
		t.Fatalf("expected exactly one successful completion for the same sale, got %d", successCount)
	}

	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM sales WHERE id = $1`, saleID).Scan(&status); err != nil {
		t.Fatalf("failed to query sale status: %v", err)
	}
	if status != SaleStatusCompleted {
		t.Fatalf("expected sale status COMPLETED, got %s", status)
	}
}

func ensureBranch(db *sql.DB, code string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(context.Background(), `INSERT INTO branches (name, code, is_active) VALUES ($1, $2, true) ON CONFLICT (code) DO NOTHING RETURNING id`, "Sales Branch", code).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, db.QueryRowContext(context.Background(), `SELECT id FROM branches WHERE code = $1`, code).Scan(&id)
		}
		return 0, err
	}
	return id, nil
}

func ensureProduct(db *sql.DB, sku string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(context.Background(), `INSERT INTO products (sku, barcode, name, purchase_price, selling_price, is_active) VALUES ($1, NULL, $2, 0, 0, true) ON CONFLICT (sku) DO NOTHING RETURNING id`, sku, "Sales Product").Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, db.QueryRowContext(context.Background(), `SELECT id FROM products WHERE sku = $1`, sku).Scan(&id)
		}
		return 0, err
	}
	return id, nil
}

func ensureUser(db *sql.DB, email string) (int64, error) {
	var id int64
	if err := db.QueryRowContext(context.Background(), `INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) ON CONFLICT (email) DO NOTHING RETURNING id`, email, "hashed-pass", "Sales User").Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
		}
		return 0, err
	}
	return id, nil
}

func ensureRoleAccess(db *sql.DB, userRepo *users.Repository, roleRepo *roles.Repository, userID, branchID int64) error {
	role, err := roleRepo.GetByName(context.Background(), "SUPER_ADMIN")
	if err != nil || role == nil {
		return fmt.Errorf("failed to load SUPER_ADMIN role: %w", err)
	}
	if err := userRepo.AddRole(context.Background(), userID, role.ID); err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO user_branches (user_id, branch_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, branchID)
	return err
}
