package users

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/roles"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestService_Create_AssignsRolesAndBranchAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := NewRepository(db)
	roleRepo := roles.NewRepository(db)
	auditSvc := audit.NewService(audit.NewRepository(db))
	service := NewService(userRepo, roleRepo, auditSvc)

	newUser := &User{
		Email:        "alice@example.com",
		PasswordHash: "password123",
		Name:         "Alice",
	}
	roleNames := []string{"SUPER_ADMIN"}
	branchIDs := []int64{7, 9}

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE email = $1
    `)).WithArgs("alice@example.com").WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id
    `)).WithArgs("alice@example.com", sqlmock.AnyArg(), "Alice").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, description
        FROM roles
        WHERE name = $1
    `)).WithArgs("SUPER_ADMIN").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).AddRow(int64(1), "SUPER_ADMIN", "super admin"))
	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO user_roles (user_id, role_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `)).WithArgs(int64(42), int64(1)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT EXISTS (
            SELECT 1
            FROM branches
            WHERE id = $1
        )
    `)).WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO user_branches (user_id, branch_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `)).WithArgs(int64(42), int64(7)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT EXISTS (
            SELECT 1
            FROM branches
            WHERE id = $1
        )
    `)).WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO user_branches (user_id, branch_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `)).WithArgs(int64(42), int64(9)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).WithArgs(sqlmock.AnyArg(), "user.create", "user", "42", sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))
	mock.ExpectCommit()

	id, err := service.Create(context.Background(), newUser, roleNames, branchIDs)
	if err != nil {
		t.Fatalf("expected no error creating user, got %v", err)
	}
	if id != 42 {
		t.Fatalf("expected created user id 42, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
