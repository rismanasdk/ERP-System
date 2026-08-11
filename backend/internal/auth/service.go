package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/jwt"
	"erp-system/backend/pkg/logger"
	"erp-system/backend/pkg/password"
)

type Service struct {
	userRepo    *users.Repository
	roleRepo    *roles.Repository
	permRepo    *permissions.Repository
	refreshRepo *RefreshTokenRepository
	auditSvc    *audit.Service
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func NewService(userRepo *users.Repository, roleRepo *roles.Repository, permRepo *permissions.Repository, refreshRepo *RefreshTokenRepository, auditSvc *audit.Service) *Service {
	return &Service{userRepo: userRepo, roleRepo: roleRepo, permRepo: permRepo, refreshRepo: refreshRepo, auditSvc: auditSvc}
}

func (s *Service) Authenticate(ctx context.Context, email, passwordPlain string) (*users.User, []string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if err := password.Compare(user.PasswordHash, passwordPlain); err != nil {
		return nil, nil, ErrInvalidCredentials
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

	if s.auditSvc != nil {
		auditResourceID := fmt.Sprintf("%d", user.ID)
		if _, err = s.auditSvc.Record(ctx, audit.AuditLog{
			ActorUserID: &user.ID,
			Action:      "auth.login",
			Resource:    "user",
			ResourceID:  &auditResourceID,
			Metadata: map[string]any{
				"email": user.Email,
			},
		}); err != nil {
			logger.Error("failed to record auth.login audit: %v", err)
		}
	}

	return user, perms, nil
}

var ErrInvalidRefreshToken = errors.New("invalid refresh token")

func (s *Service) CreateRefreshToken(ctx context.Context, userID int64) (string, error) {
	rawToken, tokenHash, familyID, err := GenerateRefreshTokenData()
	if err != nil {
		return "", err
	}

	refreshToken := &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(RefreshTokenExpiry),
	}

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = s.refreshRepo.CreateWithTx(ctx, tx, refreshToken); err != nil {
		return "", err
	}

	if s.auditSvc != nil {
		var actorUserID *int64
		if userID, ok := UserIDFromContext(ctx); ok {
			actorUserID = &userID
		}
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "refresh_token.create",
			Resource:    "refresh_token",
			ResourceID:  &refreshToken.FamilyID,
			Metadata: map[string]any{
				"user_id": refreshToken.UserID,
			},
		}); err != nil {
			return "", err
		}
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return rawToken, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, rawRefreshToken string) (string, string, error) {
	hash := hashRefreshToken(rawRefreshToken)

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	refreshToken, err := s.refreshRepo.FindByHashWithTx(ctx, tx, hash)
	if err != nil {
		return "", "", err
	}
	if refreshToken == nil {
		return "", "", ErrInvalidRefreshToken
	}
	if refreshToken.RevokedAt != nil {
		if err = s.refreshRepo.RevokeFamilyWithTx(ctx, tx, refreshToken.FamilyID, time.Now()); err != nil {
			return "", "", err
		}
		if err = tx.Commit(); err != nil {
			return "", "", err
		}
		tx = nil
		return "", "", ErrInvalidRefreshToken
	}
	if refreshToken.ExpiresAt.Before(time.Now()) {
		return "", "", ErrInvalidRefreshToken
	}

	newRawToken, newTokenHash, _, err := GenerateRefreshTokenData()
	if err != nil {
		return "", "", err
	}
	newRefreshToken := &RefreshToken{
		UserID:    refreshToken.UserID,
		TokenHash: newTokenHash,
		FamilyID:  refreshToken.FamilyID,
		ExpiresAt: time.Now().Add(RefreshTokenExpiry),
	}

	if err = s.refreshRepo.RevokeWithTx(ctx, tx, hash, time.Now()); err != nil {
		return "", "", err
	}
	if _, err = s.refreshRepo.CreateWithTx(ctx, tx, newRefreshToken); err != nil {
		return "", "", err
	}

	if s.auditSvc != nil {
		var actorUserID *int64
		if userID, ok := UserIDFromContext(ctx); ok {
			actorUserID = &userID
		}
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "refresh_token.rotate",
			Resource:    "refresh_token",
			ResourceID:  &newRefreshToken.FamilyID,
			Metadata: map[string]any{
				"user_id": newRefreshToken.UserID,
			},
		}); err != nil {
			return "", "", err
		}
	}

	if err = tx.Commit(); err != nil {
		return "", "", err
	}
	tx = nil

	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrInvalidRefreshToken
		}
		return "", "", err
	}

	token, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", "", err
	}

	return token, newRawToken, nil
}

func (s *Service) CreateUser(ctx context.Context, user *users.User, initialRoleName string) (int64, error) {
	hashedPassword, err := password.Hash(user.PasswordHash)
	if err != nil {
		return 0, err
	}
	user.PasswordHash = hashedPassword

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	id, err := s.userRepo.CreateWithTx(ctx, tx, user)
	if err != nil {
		return 0, err
	}

	if initialRoleName != "" {
		role, err := s.roleRepo.GetByNameTx(ctx, tx, initialRoleName)
		if err != nil {
			return 0, err
		}
		if role == nil {
			return 0, errors.New("role does not exist")
		}
		if err := s.userRepo.AddRoleWithTx(ctx, tx, id, role.ID); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		var actorUserID *int64
		if userID, ok := UserIDFromContext(ctx); ok {
			actorUserID = &userID
		}
		auditResourceID := fmt.Sprintf("%d", id)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "user.create",
			Resource:    "user",
			ResourceID:  &auditResourceID,
			Metadata: map[string]any{
				"email":        user.Email,
				"name":         user.Name,
				"initial_role": initialRoleName,
			},
		}); err != nil {
			return 0, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Service) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	permissions, err := s.userRepo.GetPermissionNames(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) GetIdentity(ctx context.Context, userID int64) (*Identity, error) {
	roles, err := s.userRepo.GetRoleNames(ctx, userID)
	if err != nil {
		return nil, err
	}
	permissions, err := s.userRepo.GetPermissionNames(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Identity{
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}
