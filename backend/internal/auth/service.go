package auth

import (
	"context"
	"errors"

	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/password"
)

type Service struct {
	userRepo *users.Repository
	roleRepo *roles.Repository
	permRepo *permissions.Repository
}

func NewService(userRepo *users.Repository, roleRepo *roles.Repository, permRepo *permissions.Repository) *Service {
	return &Service{userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo}
}

func (s *Service) Authenticate(ctx context.Context, email, passwordPlain string) (*users.User, []string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if err := password.Compare(user.PasswordHash, passwordPlain); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	roles, err := s.userRepo.GetRoleNames(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	user.RoleNames = roles

	perms, err := s.userRepo.GetPermissionNames(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, perms, nil
}

func (s *Service) CreateUser(ctx context.Context, user *users.User, initialRoleName string) (int64, error) {
	hashedPassword, err := password.Hash(user.PasswordHash)
	if err != nil {
		return 0, err
	}
	user.PasswordHash = hashedPassword

	id, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return 0, err
	}

	if initialRoleName != "" {
		role, err := s.roleRepo.GetByName(ctx, initialRoleName)
		if err != nil {
			return 0, err
		}
		if role == nil {
			return 0, errors.New("role does not exist")
		}
		if err := s.userRepo.AddRole(ctx, id, role.ID); err != nil {
			return 0, err
		}
	}
	return id, nil
}
