//go:build integration

package suppliers

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

func TestSupplierVersionedUpdateConcurrentIntegration(t *testing.T) {
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
	code := "VER-SU-" + suffix
	defer db.ExecContext(ctx, `DELETE FROM suppliers WHERE code = $1`, code)
	var id int64
	if err := db.QueryRowContext(ctx, `INSERT INTO suppliers (code, name, is_active) VALUES ($1, $2, true) RETURNING id`, code, "Version Supplier").Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'supplier' AND resource_id = $1`, fmt.Sprint(id))
	repo := NewRepository(db)
	service := NewService(repo, audit.NewService(audit.NewRepository(db)))
	item, err := service.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, name := range []string{"Supplier A", "Supplier B"} {
		candidate := *item
		candidate.Name = name
		wg.Add(1)
		go func(candidate Supplier) { defer wg.Done(); <-start; results <- service.Update(ctx, &candidate) }(candidate)
	}
	close(start)
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for updateErr := range results {
		if updateErr == nil {
			successes++
		} else if errors.Is(updateErr, ErrSupplierVersionConflict) {
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
	stale.Name = "Stale Supplier"
	if err := service.Update(ctx, &stale); !errors.Is(err, ErrSupplierVersionConflict) {
		t.Fatalf("expected stale conflict, got %v", err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE resource = 'supplier' AND resource_id = $1 AND action = 'supplier.update'`, fmt.Sprint(id)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one update audit, got %d", auditCount)
	}
}
