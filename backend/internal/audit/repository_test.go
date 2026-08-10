package audit

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAuditRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	actorID := int64(42)
	resourceID := "resource-1"
	auditLog := AuditLog{
		ActorUserID: &actorID,
		Action:      "user.updated",
		Resource:    "users",
		ResourceID:  &resourceID,
		Metadata:    map[string]any{"ip": "127.0.0.1"},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).
		WithArgs(actorID, auditLog.Action, auditLog.Resource, resourceID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(123))

	id, err := repo.Create(context.Background(), auditLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected id 123, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAuditRepository_ListFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)
	actorID := int64(42)
	action := "user.created"
	resource := "users"
	resourceID := "resource-1"
	after := time.Now().Add(-24 * time.Hour)
	before := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, actor_user_id, action, resource, resource_id, metadata, created_at
        FROM audit_logs
     WHERE actor_user_id = $1 AND action = $2 AND resource = $3 AND resource_id = $4 AND created_at >= $5 AND created_at <= $6 ORDER BY created_at DESC`)).
		WithArgs(actorID, action, resource, resourceID, after, before).
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_user_id", "action", "resource", "resource_id", "metadata", "created_at"}).
			AddRow(123, actorID, action, resource, resourceID, `{"ip":"127.0.0.1"}`, time.Now()))

	logs, err := repo.List(context.Background(), AuditLogFilter{
		ActorUserID:   &actorID,
		Action:        &action,
		Resource:      &resource,
		ResourceID:    &resourceID,
		CreatedAfter:  &after,
		CreatedBefore: &before,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ID != 123 || logs[0].ActorUserID == nil || *logs[0].ActorUserID != actorID {
		t.Fatalf("unexpected log actor_user_id: %+v", logs[0])
	}
	if logs[0].Action != action || logs[0].Resource != resource {
		t.Fatalf("unexpected log values: %+v", logs[0])
	}
	if logs[0].ResourceID == nil || *logs[0].ResourceID != resourceID {
		t.Fatalf("unexpected resource_id: %+v", logs[0].ResourceID)
	}
	if logs[0].Metadata == nil || logs[0].Metadata["ip"] != "127.0.0.1" {
		t.Fatalf("unexpected metadata: %+v", logs[0].Metadata)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAuditRepository_ListHandlesNullables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, actor_user_id, action, resource, resource_id, metadata, created_at
        FROM audit_logs
    ORDER BY created_at DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "actor_user_id", "action", "resource", "resource_id", "metadata", "created_at"}).
			AddRow(124, nil, "system.task", "jobs", nil, nil, time.Now()))

	logs, err := repo.List(context.Background(), AuditLogFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ActorUserID != nil {
		t.Fatalf("expected nil actor_user_id, got %v", logs[0].ActorUserID)
	}
	if logs[0].ResourceID != nil {
		t.Fatalf("expected nil resource_id, got %v", logs[0].ResourceID)
	}
	if logs[0].Metadata != nil {
		t.Fatalf("expected nil metadata, got %v", logs[0].Metadata)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
