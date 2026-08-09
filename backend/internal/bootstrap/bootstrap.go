package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"erp-system/backend/internal/users"
)

const SuperAdminRole = "SUPER_ADMIN"

// UserLookup is used to verify whether an email already exists before bootstrapping.
type UserLookup interface {
	GetByEmail(ctx context.Context, email string) (*users.User, error)
}

// UserCreator creates a user and assigns an initial role.
type UserCreator interface {
	CreateUser(ctx context.Context, user *users.User, initialRoleName string) (int64, error)
}

// BootstrapAdmin creates the initial SUPER_ADMIN user when no user exists.
// It uses the existing CreateUser service to ensure password hashing and role assignment.
func BootstrapAdmin(ctx context.Context, getter UserLookup, creator UserCreator, email, name, password string) (int64, error) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	password = strings.TrimSpace(password)

	if email == "" {
		return 0, errors.New("bootstrap admin email is required")
	}
	if name == "" {
		return 0, errors.New("bootstrap admin name is required")
	}
	if password == "" {
		return 0, errors.New("bootstrap admin password is required")
	}

	_, err := getter.GetByEmail(ctx, email)
	if err == nil {
		return 0, fmt.Errorf("user with email %q already exists", email)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	user := &users.User{
		Email:        email,
		Name:         name,
		PasswordHash: password,
	}

	return creator.CreateUser(ctx, user, SuperAdminRole)
}
