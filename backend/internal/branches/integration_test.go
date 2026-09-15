//go:build integration

package branches

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

func TestBranchVersionedUpdateConcurrentIntegration(t *testing.T) {
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
	code := "VER-BR-" + suffix
	defer db.ExecContext(ctx, `DELETE FROM branches WHERE code = $1`, code)
	var id int64
	if err := db.QueryRowContext(ctx, `INSERT INTO branches (name, code, is_active) VALUES ($1, $2, true) RETURNING id`, "Version Branch", code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'branch' AND resource_id = $1`, fmt.Sprint(id))
	repo := NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	service := NewService(repo, nil, auditSvc)
	item, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, name := range []string{"Branch A", "Branch B"} {
		candidate := *item
		candidate.Name = name
		wg.Add(1)
		go func(candidate Branch) { defer wg.Done(); <-start; results <- service.Update(ctx, &candidate) }(candidate)
	}
	close(start)
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for updateErr := range results {
		if updateErr == nil {
			successes++
		} else if errors.Is(updateErr, ErrBranchVersionConflict) {
			conflicts++
		} else {
			t.Fatal(updateErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got %d/%d", successes, conflicts)
	}
	updated, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	stale := *item
	stale.Name = "Stale Branch"
	if err := service.Update(ctx, &stale); !errors.Is(err, ErrBranchVersionConflict) {
		t.Fatalf("expected stale conflict, got %v", err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE resource = 'branch' AND resource_id = $1 AND action = 'branch.update'`, fmt.Sprint(id)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one update audit, got %d", auditCount)
	}
}
