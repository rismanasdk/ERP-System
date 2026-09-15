//go:build integration

package customers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/pkg/database"
)

func TestCustomerVersionedUpdateConcurrentIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	code := "VER-CU-" + suffix
	defer db.ExecContext(ctx, `DELETE FROM customers WHERE code = $1`, code)
	var id int64
	if err := db.QueryRowContext(ctx, `INSERT INTO customers (code, name, is_active) VALUES ($1, $2, true) RETURNING id`, code, "Version Customer").Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'customer' AND resource_id = $1`, fmt.Sprint(id))
	repo := NewRepository(db)
	service := NewService(repo, audit.NewService(audit.NewRepository(db)))
	item, err := service.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, name := range []string{"Customer A", "Customer B"} {
		candidate := *item
		candidate.Name = name
		wg.Add(1)
		go func(candidate Customer) { defer wg.Done(); <-start; results <- service.Update(ctx, &candidate) }(candidate)
	}
	close(start)
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for updateErr := range results {
		if updateErr == nil {
			successes++
		} else if errors.Is(updateErr, ErrCustomerVersionConflict) {
			conflicts++
		} else {
			t.Fatal(updateErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got %d/%d", successes, conflicts)
	}
	updated, err := service.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	stale := *item
	stale.Name = "Stale Customer"
	if err := service.Update(ctx, &stale); !errors.Is(err, ErrCustomerVersionConflict) {
		t.Fatalf("expected stale conflict, got %v", err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE resource = 'customer' AND resource_id = $1 AND action = 'customer.update'`, fmt.Sprint(id)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one update audit, got %d", auditCount)
	}
}
