package audit

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type mockAuditRepo struct {
	createCalled      bool
	createWithTxCalled bool
	lastAuditLog      AuditLog
	returnID          int64
	returnErr         error
}

func (m *mockAuditRepo) Create(ctx context.Context, auditLog AuditLog) (int64, error) {
	m.createCalled = true
	m.lastAuditLog = auditLog
	return m.returnID, m.returnErr
}

func (m *mockAuditRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, auditLog AuditLog) (int64, error) {
	m.createWithTxCalled = true
	m.lastAuditLog = auditLog
	return m.returnID, m.returnErr
}

func TestAuditService_Record(t *testing.T) {
	repo := &mockAuditRepo{returnID: 10}
	service := NewService(repo)
	auditLog := AuditLog{
		ActorUserID: nil,
		Action:      "user.login",
		Resource:    "sessions",
		ResourceID:  nil,
		Metadata:    map[string]any{"ip": "127.0.0.1"},
	}

	id, err := service.Record(context.Background(), auditLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 10 {
		t.Fatalf("expected id 10, got %d", id)
	}
	if !repo.createCalled {
		t.Fatal("expected Create to be called")
	}
	if repo.createWithTxCalled {
		t.Fatal("expected CreateWithTx not to be called")
	}
	if repo.lastAuditLog.Action != auditLog.Action || repo.lastAuditLog.Resource != auditLog.Resource {
		t.Fatalf("unexpected audit log passed through: %+v", repo.lastAuditLog)
	}
}

func TestAuditService_Record_Error(t *testing.T) {
	errExpected := errors.New("db error")
	repo := &mockAuditRepo{returnErr: errExpected}
	service := NewService(repo)
	auditLog := AuditLog{Action: "user.delete", Resource: "users"}

	_, err := service.Record(context.Background(), auditLog)
	if !errors.Is(err, errExpected) {
		t.Fatalf("expected error %v, got %v", errExpected, err)
	}
}

func TestAuditService_RecordWithTx(t *testing.T) {
	repo := &mockAuditRepo{returnID: 11}
	service := NewService(repo)
	auditLog := AuditLog{
		ActorUserID: nil,
		Action:      "user.create",
		Resource:    "users",
		ResourceID:  nil,
		Metadata:    map[string]any{"source": "admin"},
	}

	id, err := service.RecordWithTx(context.Background(), &sql.Tx{}, auditLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 11 {
		t.Fatalf("expected id 11, got %d", id)
	}
	if !repo.createWithTxCalled {
		t.Fatal("expected CreateWithTx to be called")
	}
	if repo.createCalled {
		t.Fatal("expected Create not to be called")
	}
	if repo.lastAuditLog.Metadata["source"] != "admin" {
		t.Fatalf("unexpected metadata: %+v", repo.lastAuditLog.Metadata)
	}
}

func TestAuditService_RecordWithTx_Error(t *testing.T) {
	errExpected := errors.New("tx error")
	repo := &mockAuditRepo{returnErr: errExpected}
	service := NewService(repo)
	auditLog := AuditLog{Action: "user.update", Resource: "users"}

	_, err := service.RecordWithTx(context.Background(), &sql.Tx{}, auditLog)
	if !errors.Is(err, errExpected) {
		t.Fatalf("expected error %v, got %v", errExpected, err)
	}
}

func TestAuditService_RecordDataIntegrity(t *testing.T) {
	repo := &mockAuditRepo{returnID: 12}
	service := NewService(repo)
	resourceID := "123"
	createdAt := time.Now()
	auditLog := AuditLog{
		ActorUserID: func() *int64 { v := int64(5); return &v }(),
		Action:      "resource.access",
		Resource:    "reports",
		ResourceID:  &resourceID,
		Metadata:    map[string]any{"ip": "127.0.0.1"},
		CreatedAt:   createdAt,
	}

	_, err := service.Record(context.Background(), auditLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastAuditLog.CreatedAt != createdAt {
		t.Fatalf("expected CreatedAt preserved, got %v", repo.lastAuditLog.CreatedAt)
	}
}
