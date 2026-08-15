package suppliers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"erp-system/backend/internal/audit"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSupplierService_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at FROM suppliers WHERE code = \\$1 AND deleted_at IS NULL").
		WithArgs("SUP-001").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO suppliers").WithArgs("SUP-001", "Alpha", nil, nil, nil, false).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	mock.ExpectQuery("INSERT INTO audit_logs").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectCommit()

	repo := NewRepository(db)
	service := NewService(repo, audit.NewService(audit.NewRepository(db)))

	id, err := service.Create(context.Background(), &Supplier{Code: "SUP-001", Name: "Alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Fatalf("expected id 99, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierService_Create_DuplicateCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at FROM suppliers WHERE code = \\$1").
		WithArgs("SUP-001").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "phone", "email", "address", "is_active", "created_at", "updated_at", "deleted_at"}).AddRow(int64(1), "SUP-001", "Alpha", nil, nil, nil, true, time.Now(), time.Now(), nil))

	repo := NewRepository(db)
	service := NewService(repo, nil)

	_, err = service.Create(context.Background(), &Supplier{Code: "SUP-001", Name: "Alpha"})
	if !errors.Is(err, ErrSupplierDuplicateCode) {
		t.Fatalf("expected ErrSupplierDuplicateCode, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSupplierService_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT id, code, name, phone, email, address, is_active, created_at, updated_at, deleted_at FROM suppliers WHERE id = \\$1 AND deleted_at IS NULL").
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	service := NewService(repo, nil)

	_, err = service.GetByID(context.Background(), 42)
	if !errors.Is(err, ErrSupplierNotFound) {
		t.Fatalf("expected ErrSupplierNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
