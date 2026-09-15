//go:build integration
// +build integration

package roles

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/pkg/database"
)

func TestRoleMutationsAuditIntegration(t *testing.T) {
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
	roleName := fmt.Sprintf("Audit Role %d", time.Now().UnixNano())
	auditor := audit.NewService(audit.NewRepository(db))
	repo := NewRepository(db, auditor)
	actorID := int64(1)

	roleID, err := repo.Create(ctx, &Role{Name: roleName, Description: "audit test"}, nil, &actorID)
	if err != nil {
		t.Fatalf("role create failed: %v", err)
	}
	if err := repo.Update(ctx, &Role{ID: roleID, Name: roleName + " Updated", Description: "updated"}, nil, &actorID); err != nil {
		t.Fatalf("role update failed: %v", err)
	}
	if err := repo.Delete(ctx, roleID, &actorID); err != nil {
		t.Fatalf("role delete failed: %v", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT action, actor_user_id, resource, resource_id FROM audit_logs WHERE resource = 'role' AND resource_id = $1 ORDER BY id`, fmt.Sprintf("%d", roleID))
	if err != nil {
		t.Fatalf("failed to query role audits: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action, resource, resourceID string
		var actor int64
		if err := rows.Scan(&action, &actor, &resource, &resourceID); err != nil {
			t.Fatalf("failed to scan role audit: %v", err)
		}
		if actor != actorID || resource != "role" || resourceID != fmt.Sprintf("%d", roleID) {
			t.Fatalf("unexpected role audit identity: actor=%d resource=%s resource_id=%s", actor, resource, resourceID)
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed reading role audits: %v", err)
	}
	if len(actions) != 3 || actions[0] != "role.create" || actions[1] != "role.update" || actions[2] != "role.delete" {
		t.Fatalf("unexpected role audit actions: %v", actions)
	}
	_, _ = db.ExecContext(ctx, `DELETE FROM audit_logs WHERE resource = 'role' AND resource_id = $1`, fmt.Sprintf("%d", roleID))
}
