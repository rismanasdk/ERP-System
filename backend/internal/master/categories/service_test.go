package categories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

type fakeAuditRecorder struct {
	err  error
	logs []audit.AuditLog
}

func (f *fakeAuditRecorder) RecordWithTx(_ context.Context, _ *sql.Tx, log audit.AuditLog) (int64, error) {
	f.logs = append(f.logs, log)
	return 1, f.err
}

func TestCreate_DuplicateCategoryNameReturnsConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(NewRepository(db))
	mock.ExpectQuery("SELECT id, name, description").WithArgs("hardware").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "is_active", "product_count", "version", "created_at", "updated_at"}).AddRow(1, "Hardware", nil, true, 0, int64(1), time.Now(), time.Now()),
	)

	_, err = service.Create(context.Background(), &Category{Name: "hardware"})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestCreate_DatabaseUniqueViolationReturnsConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(NewRepository(db))
	mock.ExpectQuery("SELECT id, name, description").WithArgs("hardware").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO product_categories").WithArgs("hardware", nil, false).WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	_, err = service.Create(context.Background(), &Category{Name: "hardware"})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error from database constraint, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdate_DatabaseUniqueViolationReturnsConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewService(NewRepository(db))
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(int64(2)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "description", "is_active", "product_count", "version", "created_at", "updated_at"}).AddRow(2, "Office", nil, true, 0, int64(1), time.Now(), time.Now()),
	)
	mock.ExpectQuery("SELECT id, name, description").WithArgs("hardware").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE product_categories").WithArgs("hardware", nil, false, int64(2), int64(1)).WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	err = service.Update(context.Background(), &Category{ID: 2, Name: "hardware", Version: 1})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error from database constraint, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreate_Update_Delete_RecordTransactionalAudits(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditor := &fakeAuditRecorder{}
	service := NewService(NewRepository(db), auditor)
	ctx := auth.ContextWithUserID(context.Background(), 9)

	mock.ExpectQuery("SELECT id, name, description").WithArgs("Hardware").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO product_categories").WithArgs("Hardware", nil, true).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))
	mock.ExpectCommit()
	if _, err := service.Create(ctx, &Category{Name: "Hardware", IsActive: true}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "is_active", "product_count", "version", "created_at", "updated_at"}).AddRow(11, "Hardware", nil, true, 0, int64(1), time.Now(), time.Now()))
	mock.ExpectQuery("SELECT id, name, description").WithArgs("Office").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE product_categories").WithArgs("Office", nil, false, int64(11), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := service.Update(ctx, &Category{ID: 11, Name: "Office", Version: 1}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "is_active", "product_count", "version", "created_at", "updated_at"}).AddRow(11, "Office", nil, false, 0, int64(2), time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM product_categories").WithArgs(int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := service.Delete(ctx, 11); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if len(auditor.logs) != 3 || auditor.logs[0].Action != "category.create" || auditor.logs[1].Action != "category.update" || auditor.logs[2].Action != "category.delete" {
		t.Fatalf("unexpected audit logs: %+v", auditor.logs)
	}
	for _, log := range auditor.logs {
		if log.ActorUserID == nil || *log.ActorUserID != 9 || log.Resource != "product_category" || log.ResourceID == nil {
			t.Fatalf("missing audit identity fields: %+v", log)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreate_AuditFailureRollsBackCategory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	auditor := &fakeAuditRecorder{err: errors.New("audit failed")}
	service := NewService(NewRepository(db), auditor)
	mock.ExpectQuery("SELECT id, name, description").WithArgs("Hardware").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO product_categories").WithArgs("Hardware", nil, true).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))
	mock.ExpectRollback()

	if _, err := service.Create(auth.ContextWithUserID(context.Background(), 9), &Category{Name: "Hardware", IsActive: true}); err == nil {
		t.Fatal("expected audit failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
