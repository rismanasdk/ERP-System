package roles

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-system/backend/internal/audit"

	"github.com/DATA-DOG/go-sqlmock"
)

type fakeAuditRecorder struct {
	err  error
	logs []audit.AuditLog
}

func (f *fakeAuditRecorder) RecordWithTx(_ context.Context, _ *sql.Tx, log audit.AuditLog) (int64, error) {
	f.logs = append(f.logs, log)
	return 1, f.err
}

func TestRoleMutationsRecordAuditsAndRollbackOnAuditFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditor := &fakeAuditRecorder{}
	repo := NewRepository(db, auditor)
	actor := int64(7)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO roles").WithArgs("Manager", "Operations").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id FROM permissions").WithArgs("sales.read").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(31))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(int64(21), int64(31)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if _, err := repo.Create(ctx, &Role{Name: "Manager", Description: "Operations"}, []string{"sales.read"}, &actor); err != nil {
		t.Fatalf("role create failed: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT name FROM roles").WithArgs(int64(21)).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Manager"))
	mock.ExpectExec("UPDATE roles").WithArgs("Manager 2", "Updated", int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id FROM permissions").WithArgs("inventory.read").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(32))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs(int64(21), int64(32)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Update(ctx, &Role{ID: 21, Name: "Manager 2", Description: "Updated"}, []string{"inventory.read"}, &actor); err != nil {
		t.Fatalf("role update failed: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT name FROM roles").WithArgs(int64(21)).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Manager 2"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_roles`).WithArgs(int64(21)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("DELETE FROM roles").WithArgs(int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Delete(ctx, 21, &actor); err != nil {
		t.Fatalf("role delete failed: %v", err)
	}

	if len(auditor.logs) != 3 || auditor.logs[0].Action != "role.create" || auditor.logs[1].Action != "role.update" || auditor.logs[2].Action != "role.delete" {
		t.Fatalf("unexpected role audit logs: %+v", auditor.logs)
	}
	for _, log := range auditor.logs {
		if log.ActorUserID == nil || *log.ActorUserID != actor || log.Resource != "role" || log.ResourceID == nil {
			t.Fatalf("missing role audit identity fields: %+v", log)
		}
	}

	auditor.err = errors.New("audit failed")
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO roles").WithArgs("Audited", "Failure").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(22))
	mock.ExpectExec("DELETE FROM role_permissions").WithArgs(int64(22)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()
	if _, err := repo.Create(ctx, &Role{Name: "Audited", Description: "Failure"}, nil, &actor); err == nil {
		t.Fatal("expected role audit failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
