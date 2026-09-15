//go:build integration
// +build integration

package products

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/master/categories"
	"erp-system/backend/pkg/database"
)

func TestProductVersionedUpdateConcurrentIntegration(t *testing.T) {
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
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	categoryName := "Version Category " + suffix
	sku := "VERSION-" + suffix
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'product' AND metadata->>'sku' = $1`, sku)
		_, _ = db.ExecContext(ctx, `DELETE FROM products WHERE sku = $1`, sku)
		_, _ = db.ExecContext(ctx, `DELETE FROM product_categories WHERE name = $1`, categoryName)
	}()

	var categoryID, productID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO product_categories (name) VALUES ($1) RETURNING id`, categoryName).Scan(&categoryID); err != nil {
		t.Fatalf("create category fixture: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO products (sku, name, category_id) VALUES ($1, $2, $3) RETURNING id`, sku, "Version Product", categoryID).Scan(&productID); err != nil {
		t.Fatalf("create product fixture: %v", err)
	}

	service := NewService(NewRepository(db), audit.NewService(audit.NewRepository(db)), categories.NewRepository(db))
	product, err := service.GetByID(ctx, productID)
	if err != nil {
		t.Fatalf("load product fixture: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, name := range []string{"Product Update A", "Product Update B"} {
		candidate := *product
		candidate.Name = name
		waitGroup.Add(1)
		go func(candidate Product) {
			defer waitGroup.Done()
			<-start
			results <- service.Update(ctx, &candidate)
		}(candidate)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var successes, conflicts int
	for updateErr := range results {
		if updateErr == nil {
			successes++
		} else if updateErr == ErrProductVersionConflict {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent update error: %v", updateErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}

	updated, err := service.GetByID(ctx, productID)
	if err != nil {
		t.Fatalf("reload product fixture: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2 after one successful update, got %d", updated.Version)
	}
}
