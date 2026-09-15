//go:build integration

package users

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/roles"
	"erp-system/backend/pkg/database"
)

func TestUserVersionedUpdateConcurrentIntegration(t *testing.T) {
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
	actorEmail := "m2-user-actor-" + suffix + "@example.com"
	targetEmail := "m2-user-target-" + suffix + "@example.com"
	roleInitial := "M2_USER_INITIAL_" + suffix
	roleA := "M2_USER_A_" + suffix
	roleB := "M2_USER_B_" + suffix
	branchInitial := "M2 User Initial " + suffix
	branchA := "M2 User A " + suffix
	branchB := "M2 User B " + suffix

	var actorID, targetID int64
	var initialBranchID, branchAID, branchBID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`, actorEmail, "hash", "M2 actor").Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id`, targetEmail, "hash", "M2 target").Scan(&targetID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM roles WHERE name = 'SUPER_ADMIN'`).Scan(&initialBranchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, actorID, initialBranchID); err != nil {
		t.Fatal(err)
	}

	roleIDs := make(map[string]int64)
	for _, roleName := range []string{roleInitial, roleA, roleB} {
		var roleID int64
		if err := db.QueryRowContext(ctx, `INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id`, roleName, "M2 integration role").Scan(&roleID); err != nil {
			t.Fatal(err)
		}
		roleIDs[roleName] = roleID
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`, branchInitial, "M2-UI-"+suffix).Scan(&initialBranchID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`, branchA, "M2-UA-"+suffix).Scan(&branchAID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`, branchB, "M2-UB-"+suffix).Scan(&branchBID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, targetID, roleIDs[roleInitial]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO user_branches (user_id, branch_id) VALUES ($1, $2)`, targetID, initialBranchID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'user' AND resource_id IN ($1, $2)`, fmt.Sprint(actorID), fmt.Sprint(targetID))
		_, _ = db.ExecContext(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, actorID, targetID)
		_, _ = db.ExecContext(ctx, `DELETE FROM roles WHERE id IN ($1, $2, $3)`, roleIDs[roleInitial], roleIDs[roleA], roleIDs[roleB])
		_, _ = db.ExecContext(ctx, `DELETE FROM branches WHERE id IN ($1, $2, $3)`, initialBranchID, branchAID, branchBID)
	}()

	service := NewService(NewRepository(db), roles.NewRepository(db), audit.NewService(audit.NewRepository(db)))
	updateContext := context.WithValue(ctx, "userID", actorID)
	initial, err := service.GetByID(updateContext, targetID)
	if err != nil {
		t.Fatal(err)
	}
	initialVersion := initial.Version
	results := make(chan error, 2)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	updates := []User{
		{ID: targetID, Version: initialVersion, Email: targetEmail, Name: "M2 winner A", IsActive: false, RoleNames: []string{roleA}, BranchIDs: []int64{branchAID}},
		{ID: targetID, Version: initialVersion, Email: targetEmail, Name: "M2 winner B", IsActive: true, RoleNames: []string{roleB}, BranchIDs: []int64{branchBID}},
	}
	for _, candidate := range updates {
		waitGroup.Add(1)
		go func(candidate User) {
			defer waitGroup.Done()
			<-start
			results <- service.Update(updateContext, &candidate, candidate.RoleNames, candidate.BranchIDs)
		}(candidate)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes, conflicts := 0, 0
	for updateErr := range results {
		if updateErr == nil {
			successes++
		} else if errors.Is(updateErr, ErrUserVersionConflict) {
			conflicts++
		} else {
			t.Fatal(updateErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}

	finalUser, err := service.GetByID(updateContext, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if finalUser.Version != initialVersion+1 {
		t.Fatalf("expected version %d, got %d", initialVersion+1, finalUser.Version)
	}
	if finalUser.Name != updates[0].Name && finalUser.Name != updates[1].Name {
		t.Fatalf("unexpected final name %q", finalUser.Name)
	}
	if len(finalUser.RoleNames) != 1 || (finalUser.RoleNames[0] != roleA && finalUser.RoleNames[0] != roleB) {
		t.Fatalf("unexpected final roles %#v", finalUser.RoleNames)
	}
	if len(finalUser.BranchIDs) != 1 || (finalUser.BranchIDs[0] != branchAID && finalUser.BranchIDs[0] != branchBID) {
		t.Fatalf("unexpected final branches %#v", finalUser.BranchIDs)
	}

	stale := *initial
	stale.Name = "M2 stale update"
	stale.IsActive = !finalUser.IsActive
	stale.RoleNames = []string{roleInitial}
	stale.BranchIDs = []int64{initialBranchID}
	if err := service.Update(updateContext, &stale, stale.RoleNames, stale.BranchIDs); !errors.Is(err, ErrUserVersionConflict) {
		t.Fatalf("expected stale conflict, got %v", err)
	}
	afterStale, err := service.GetByID(updateContext, targetID)
	if err != nil {
		t.Fatal(err)
	}
	if afterStale.Name != finalUser.Name || afterStale.IsActive != finalUser.IsActive || !reflect.DeepEqual(afterStale.RoleNames, finalUser.RoleNames) || !reflect.DeepEqual(afterStale.BranchIDs, finalUser.BranchIDs) {
		t.Fatalf("stale update modified user: before=%+v after=%+v", finalUser, afterStale)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE resource = 'user' AND resource_id = $1 AND action = 'user.update'`, fmt.Sprint(targetID)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one successful update audit, got %d", auditCount)
	}
}
