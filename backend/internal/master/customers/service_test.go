package customers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"erp-system/backend/internal/audit"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCustomerService_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at FROM customers WHERE code = \$1`).
		WithArgs("CUST-001").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO customers").WithArgs("CUST-001", "Alpha", nil, nil, nil, nil, false).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))
	mock.ExpectQuery("INSERT INTO audit_logs").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectCommit()

	repo := NewRepository(db)
	service := NewService(repo, audit.NewService(audit.NewRepository(db)))

	id, err := service.Create(context.Background(), &Customer{Code: "CUST-001", Name: "Alpha"})
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

func TestCustomerService_Create_DuplicateCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at FROM customers WHERE code = \$1`).
		WithArgs("CUST-001").
		WillReturnRows(sqlmock.NewRows([]string{"id", "code", "name", "phone", "email", "address", "tax_id", "is_active", "created_at", "updated_at", "deleted_at"}).AddRow(int64(1), "CUST-001", "Alpha", nil, nil, nil, nil, true, time.Now(), time.Now(), nil))

	repo := NewRepository(db)
	service := NewService(repo, nil)

	_, err = service.Create(context.Background(), &Customer{Code: "CUST-001", Name: "Alpha"})
	if !errors.Is(err, ErrCustomerDuplicateCode) {
		t.Fatalf("expected ErrCustomerDuplicateCode, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCustomerService_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, code, name, phone, email, address, tax_id, is_active, created_at, updated_at, deleted_at FROM customers WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)

	repo := NewRepository(db)
	service := NewService(repo, nil)

	_, err = service.GetByID(context.Background(), 42)
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
