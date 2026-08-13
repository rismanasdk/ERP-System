//go:build integration
// +build integration

package purchasing

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "sync"
    "testing"
    "time"

    "erp-system/backend/internal/auth"
    "erp-system/backend/internal/branches"
    "erp-system/backend/internal/inventory"
    "erp-system/backend/internal/master/products"
    "erp-system/backend/internal/permissions"
    "erp-system/backend/internal/roles"
    "erp-system/backend/internal/users"
    "erp-system/backend/pkg/database"
    "github.com/lib/pq"
)

// This integration test requires a real Postgres instance and will be skipped
// when DATABASE_URL is not set. Run with `go test -tags=integration`.
func TestPurchaseCompleteConcurrentIntegration(t *testing.T) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        t.Skip("DATABASE_URL not set; skipping integration test")
    }

    db, err := database.Connect(dsn)
    if err != nil {
        t.Fatalf("failed to connect to database: %v", err)
    }
    defer db.Close()

    // repositories
    userRepo := users.NewRepository(db)
    roleRepo := roles.NewRepository(db)
    permRepo := permissions.NewRepository(db)
    branchRepo := branches.NewRepository(db)
    prodRepo := products.NewRepository(db)
    purchaseRepo := NewRepository(db)
    invRepo := inventory.NewRepository(db)

    // services
    authSvc := auth.NewService(userRepo, roleRepo, permRepo, nil, nil)
    branchSvc := branches.NewService(branchRepo, authSvc, nil)
    productSvc := prodRepo // implements GetByID
    // purchase service with real repos; no audit service
    purchaseSvc := NewService(purchaseRepo, invRepo, productSvc, branchSvc, authSvc, nil)

    // create test fixtures (branch, product, supplier, user)
    randSuffix := fmt.Sprintf("itest_%d", time.Now().UnixNano())
    branchCode := "BR-" + randSuffix
    productSKU := "SKU-" + randSuffix
    supplierCode := "SUP-" + randSuffix
    userEmail := randSuffix + "@example.local"

    // create or get branch
    var branchID int64
    err = db.QueryRowContext(context.Background(), `INSERT INTO branches (name, code, is_active) VALUES ($1,$2,true) ON CONFLICT (code) DO NOTHING RETURNING id`, "Integration Branch", branchCode).Scan(&branchID)
    if err != nil {
        // if no row returned, fetch existing
        if err == sql.ErrNoRows {
            if err = db.QueryRowContext(context.Background(), `SELECT id FROM branches WHERE code = $1`, branchCode).Scan(&branchID); err != nil {
                t.Fatalf("failed to select existing branch: %v", err)
            }
        } else {
            t.Fatalf("failed to create branch: %v", err)
        }
    }

    // create or get product
    var productID int64
    err = db.QueryRowContext(context.Background(), `INSERT INTO products (sku, barcode, name, purchase_price, selling_price, is_active) VALUES ($1,$2,$3,0,0,true) ON CONFLICT (sku) DO NOTHING RETURNING id`, productSKU, nil, "Integration Product").Scan(&productID)
    if err != nil {
        if err == sql.ErrNoRows {
            if err = db.QueryRowContext(context.Background(), `SELECT id FROM products WHERE sku = $1`, productSKU).Scan(&productID); err != nil {
                t.Fatalf("failed to select existing product: %v", err)
            }
        } else {
            t.Fatalf("failed to create product: %v", err)
        }
    }

    // create or get supplier
    var supplierID int64
    err = db.QueryRowContext(context.Background(), `INSERT INTO suppliers (name, code, is_active) VALUES ($1,$2,true) ON CONFLICT (code) DO NOTHING RETURNING id`, "Integration Supplier", supplierCode).Scan(&supplierID)
    if err != nil {
        if err == sql.ErrNoRows {
            if err = db.QueryRowContext(context.Background(), `SELECT id FROM suppliers WHERE code = $1`, supplierCode).Scan(&supplierID); err != nil {
                t.Fatalf("failed to select existing supplier: %v", err)
            }
        } else {
            t.Fatalf("failed to create supplier: %v", err)
        }
    }

    // create or get user (handle possible pkey/sequence inconsistencies)
    var userID int64
    err = db.QueryRowContext(context.Background(), `INSERT INTO users (email, password_hash, name) VALUES ($1,$2,$3) ON CONFLICT (email) DO NOTHING RETURNING id`, userEmail, "noop", "integration").Scan(&userID)
    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
            // duplicate primary key (sequence issue). Try to select by email
            if err = db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE email = $1`, userEmail).Scan(&userID); err != nil {
                t.Fatalf("failed to find existing user after 23505: %v", err)
            }
        } else if err == sql.ErrNoRows {
            // no returned id, select existing
            if err = db.QueryRowContext(context.Background(), `SELECT id FROM users WHERE email = $1`, userEmail).Scan(&userID); err != nil {
                t.Fatalf("failed to select existing user: %v", err)
            }
        } else {
            t.Fatalf("failed to create user: %v", err)
        }
    }

    // assign SUPER_ADMIN role to user so it has all permissions
    role, err := roleRepo.GetByName(context.Background(), "SUPER_ADMIN")
    if err != nil || role == nil {
        t.Fatalf("failed to find SUPER_ADMIN role: %v", err)
    }
    if err := userRepo.AddRole(context.Background(), userID, role.ID); err != nil {
        t.Fatalf("failed to assign role: %v", err)
    }

    // give user access to branch
    // use a tx to insert into user_branches
    if _, err := db.ExecContext(context.Background(), `INSERT INTO user_branches (user_id, branch_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, userID, branchID); err != nil {
        t.Fatalf("failed to assign user to branch: %v", err)
    }

    // Ensure no inventory exists for product+branch
    if _, err := db.ExecContext(context.Background(), `DELETE FROM inventory WHERE product_id = $1 AND branch_id = $2`, productID, branchID); err != nil {
        t.Fatalf("failed to delete existing inventory: %v", err)
    }

    // run multiple iterations to increase confidence
    iterations := 3
    for it := 0; it < iterations; it++ {
        t.Logf("iteration %d", it+1)

        ctx := auth.ContextWithUserID(context.Background(), userID)

        // create two draft purchases via service
        p1Input := CreatePurchaseInput{BranchID: branchID, SupplierID: supplierID, Items: []CreatePurchaseItemInput{{ProductID: productID, Quantity: 5, UnitCost: 1.0}}}
        p2Input := CreatePurchaseInput{BranchID: branchID, SupplierID: supplierID, Items: []CreatePurchaseItemInput{{ProductID: productID, Quantity: 7, UnitCost: 1.0}}}

        p1ID, err := purchaseSvc.CreatePurchase(ctx, p1Input)
        if err != nil {
            t.Fatalf("CreatePurchase p1 failed: %v", err)
        }
        p2ID, err := purchaseSvc.CreatePurchase(ctx, p2Input)
        if err != nil {
            t.Fatalf("CreatePurchase p2 failed: %v", err)
        }

        // ensure inventory row does not exist
        var invCount int
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM inventory WHERE product_id = $1 AND branch_id = $2`, productID, branchID).Scan(&invCount); err != nil {
            t.Fatalf("count inventory failed: %v", err)
        }
        if invCount != 0 {
            // cleanup then fail
            t.Fatalf("expected no inventory row before completion, found %d", invCount)
        }

        // run two concurrent completes
        var wg sync.WaitGroup
        start := make(chan struct{})
        var errs [2]error

        wg.Add(2)
        go func() {
            defer wg.Done()
            <-start
            errs[0] = purchaseSvc.CompletePurchase(ctx, p1ID)
        }()
        go func() {
            defer wg.Done()
            <-start
            errs[1] = purchaseSvc.CompletePurchase(ctx, p2ID)
        }()

        // release both goroutines at same time
        close(start)
        wg.Wait()

        if errs[0] != nil {
            t.Fatalf("complete p1 failed: %v", errs[0])
        }
        if errs[1] != nil {
            t.Fatalf("complete p2 failed: %v", errs[1])
        }

        // assertions
        var status1, status2 string
        if err := db.QueryRowContext(context.Background(), `SELECT status FROM purchases WHERE id = $1`, p1ID).Scan(&status1); err != nil {
            t.Fatalf("select p1 status: %v", err)
        }
        if err := db.QueryRowContext(context.Background(), `SELECT status FROM purchases WHERE id = $1`, p2ID).Scan(&status2); err != nil {
            t.Fatalf("select p2 status: %v", err)
        }
        if status1 != PurchaseStatusCompleted || status2 != PurchaseStatusCompleted {
            t.Fatalf("expected both purchases completed, got %s and %s", status1, status2)
        }

        // exactly one inventory row, quantity equals sum
        var invRows int
        var qty int64
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*), COALESCE(SUM(quantity),0) FROM inventory WHERE product_id = $1 AND branch_id = $2`, productID, branchID).Scan(&invRows, &qty); err != nil {
            t.Fatalf("inventory count/qty query failed: %v", err)
        }
        if invRows != 1 {
            t.Fatalf("expected exactly one inventory row, got %d", invRows)
        }
        if qty != 5+7 {
            t.Fatalf("expected inventory quantity %d, got %d", 5+7, qty)
        }

        // two IN stock movements referencing purchases
        var movementCount int
        if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM stock_movements WHERE product_id = $1 AND branch_id = $2 AND movement_type = 'IN' AND reference_type = 'purchase' AND reference_id IN ($3,$4)`, productID, branchID, p1ID, p2ID).Scan(&movementCount); err != nil {
            t.Fatalf("movement count query failed: %v", err)
        }
        if movementCount != 2 {
            t.Fatalf("expected 2 IN movements, got %d", movementCount)
        }

        // cleanup: remove created purchases, items, movements, inventory for next iteration
        if _, err := db.ExecContext(context.Background(), `DELETE FROM stock_movements WHERE reference_type = 'purchase' AND reference_id IN ($1,$2)`, p1ID, p2ID); err != nil {
            t.Fatalf("cleanup movements: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `DELETE FROM purchase_items WHERE purchase_id IN ($1,$2)`, p1ID, p2ID); err != nil {
            t.Fatalf("cleanup items: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `DELETE FROM purchases WHERE id IN ($1,$2)`, p1ID, p2ID); err != nil {
            t.Fatalf("cleanup purchases: %v", err)
        }
        if _, err := db.ExecContext(context.Background(), `DELETE FROM inventory WHERE product_id = $1 AND branch_id = $2`, productID, branchID); err != nil {
            t.Fatalf("cleanup inventory: %v", err)
        }
    }

    // final cleanup of created fixtures
    if _, err := db.ExecContext(context.Background(), `DELETE FROM user_branches WHERE user_id = $1 AND branch_id = $2`, userID, branchID); err != nil {
        t.Logf("warning cleanup user_branches: %v", err)
    }
    if _, err := db.ExecContext(context.Background(), `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
        t.Logf("warning cleanup user_roles: %v", err)
    }
    if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
        t.Logf("warning cleanup users: %v", err)
    }
    if _, err := db.ExecContext(context.Background(), `DELETE FROM suppliers WHERE id = $1`, supplierID); err != nil {
        t.Logf("warning cleanup suppliers: %v", err)
    }
    if _, err := db.ExecContext(context.Background(), `DELETE FROM products WHERE id = $1`, productID); err != nil {
        t.Logf("warning cleanup products: %v", err)
    }
    if _, err := db.ExecContext(context.Background(), `DELETE FROM branches WHERE id = $1`, branchID); err != nil {
        t.Logf("warning cleanup branches: %v", err)
    }
}
