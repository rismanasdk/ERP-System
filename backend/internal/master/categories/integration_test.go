//go:build integration
// +build integration

package categories

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/pkg/database"
)

func TestCreateConcurrentCaseInsensitiveDuplicateIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	name := fmt.Sprintf("Concurrent Category %d", time.Now().UnixNano())
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM product_categories WHERE LOWER(BTRIM(name)) = LOWER(BTRIM($1))`, name)
	}()

	service := NewService(NewRepository(db), audit.NewService(audit.NewRepository(db)))
	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, categoryName := range []string{name, "concurrent category " + name[len("Concurrent Category "):]} {
		waitGroup.Add(1)
		go func(categoryName string) {
			defer waitGroup.Done()
			<-start
			_, createErr := service.Create(context.Background(), &Category{Name: categoryName})
			results <- createErr
		}(categoryName)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var successCount, duplicateCount int
	for createErr := range results {
		if createErr == nil {
			successCount++
		} else if createErr == ErrDuplicate {
			duplicateCount++
		} else {
			t.Fatalf("unexpected create error: %v", createErr)
		}
	}
	if successCount != 1 || duplicateCount != 1 {
		t.Fatalf("expected one success and one duplicate, got success=%d duplicate=%d", successCount, duplicateCount)
	}

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM product_categories WHERE LOWER(BTRIM(name)) = LOWER(BTRIM($1))`, name).Scan(&count); err != nil {
		t.Fatalf("failed to count created categories: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one persisted category, got %d", count)
	}
	var auditCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM audit_logs WHERE action = 'category.create' AND resource = 'product_category' AND resource_id = (SELECT id::text FROM product_categories WHERE LOWER(BTRIM(name)) = LOWER(BTRIM($1)))`, name).Scan(&auditCount); err != nil {
		t.Fatalf("failed to count category audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one category audit log, got %d", auditCount)
	}
}

func TestCategoryVersionedUpdateConcurrentIntegration(t *testing.T) {
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
	name := "Version Category " + suffix
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'product_category' AND resource_id = (SELECT id::text FROM product_categories WHERE name = $1)`, name)
		_, _ = db.ExecContext(ctx, `DELETE FROM product_categories WHERE name = $1`, name)
	}()

	var id int64
	if err := db.QueryRowContext(ctx, `INSERT INTO product_categories (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("create category fixture: %v", err)
	}
	service := NewService(NewRepository(db), audit.NewService(audit.NewRepository(db)))
	category, err := service.Get(ctx, id)
	if err != nil {
		t.Fatalf("load category fixture: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, description := range []string{"Description A", "Description B"} {
		candidate := *category
		candidate.Description = &description
		waitGroup.Add(1)
		go func(candidate Category) {
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
		} else if updateErr == ErrVersionConflict {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent update error: %v", updateErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}

	updated, err := service.Get(ctx, id)
	if err != nil {
		t.Fatalf("reload category fixture: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2 after one successful update, got %d", updated.Version)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action = 'category.update' AND resource_id = $1`, fmt.Sprintf("%d", id)).Scan(&auditCount); err != nil {
		t.Fatalf("count category audits: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one category.update audit, got %d", auditCount)
	}
}
