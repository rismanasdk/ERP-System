package bootstrap

import (
	"context"
	"database/sql"
	"testing"

	"erp-system/backend/internal/users"
)

type fakeUserLookup struct {
	user *users.User
	err  error
}

func (f *fakeUserLookup) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	return f.user, f.err
}

type fakeUserCreator struct {
	user *users.User
	role string
	id   int64
	err  error
}

func (f *fakeUserCreator) CreateUser(ctx context.Context, user *users.User, initialRoleName string) (int64, error) {
	f.user = user
	f.role = initialRoleName
	return f.id, f.err
}

func TestBootstrapAdmin_Success(t *testing.T) {
	getter := &fakeUserLookup{err: sql.ErrNoRows}
	creator := &fakeUserCreator{id: 123}

	id, err := BootstrapAdmin(context.Background(), getter, creator, "admin@example.com", "Admin User", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected id 123, got %d", id)
	}
	if creator.user == nil {
		t.Fatal("expected user to be created")
	}
	if creator.user.Email != "admin@example.com" {
		t.Fatalf("expected email admin@example.com, got %s", creator.user.Email)
	}
	if creator.user.Name != "Admin User" {
		t.Fatalf("expected name Admin User, got %s", creator.user.Name)
	}
	if creator.role != SuperAdminRole {
		t.Fatalf("expected role %s, got %s", SuperAdminRole, creator.role)
	}
}

func TestBootstrapAdmin_DuplicateEmail(t *testing.T) {
	getter := &fakeUserLookup{user: &users.User{Email: "admin@example.com"}}
	creator := &fakeUserCreator{}

	_, err := BootstrapAdmin(context.Background(), getter, creator, "admin@example.com", "Admin User", "password123")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if err.Error() != "user with email \"admin@example.com\" already exists" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBootstrapAdmin_InvalidInput(t *testing.T) {
	getter := &fakeUserLookup{err: sql.ErrNoRows}
	creator := &fakeUserCreator{}

	_, err := BootstrapAdmin(context.Background(), getter, creator, "", "Admin User", "password123")
	if err == nil || err.Error() != "bootstrap admin email is required" {
		t.Fatalf("expected email required error, got %v", err)
	}

	_, err = BootstrapAdmin(context.Background(), getter, creator, "admin@example.com", "", "password123")
	if err == nil || err.Error() != "bootstrap admin name is required" {
		t.Fatalf("expected name required error, got %v", err)
	}

	_, err = BootstrapAdmin(context.Background(), getter, creator, "admin@example.com", "Admin User", "")
	if err == nil || err.Error() != "bootstrap admin password is required" {
		t.Fatalf("expected password required error, got %v", err)
	}
}
