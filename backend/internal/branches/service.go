package branches

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
)

type repository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) (int64, error)
	GetByID(ctx context.Context, id int64) (*Branch, error)
	GetByCode(ctx context.Context, code string) (*Branch, error)
	List(ctx context.Context, filter BranchFilter) ([]Branch, error)
	ListAccessibleBranches(ctx context.Context, filter BranchFilter, userID int64) ([]Branch, error)
	UpdateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) error
	AssignUserBranchWithTx(ctx context.Context, tx *sql.Tx, userID, branchID int64) error
	UserHasAccess(ctx context.Context, userID, branchID int64) (bool, error)
}

type Service struct {
	repo             repository
	identityProvider auth.IdentityProvider
	auditSvc         *audit.Service
}

func NewService(repo repository, identityProvider auth.IdentityProvider, auditSvc *audit.Service) *Service {
	return &Service{repo: repo, identityProvider: identityProvider, auditSvc: auditSvc}
}

var (
	ErrBranchNotFound      = errors.New("branch not found")
	ErrBranchCodeDuplicate = errors.New("branch code already exists")
	ErrBranchNameRequired  = errors.New("branch name is required")
	ErrBranchCodeRequired  = errors.New("branch code is required")
	ErrBranchAccessDenied  = errors.New("branch access denied")
	ErrBranchInactive      = errors.New("branch is inactive")
)

func (s *Service) Create(ctx context.Context, branch *Branch) (int64, error) {
	branch.Name = strings.TrimSpace(branch.Name)
	branch.Code = strings.TrimSpace(branch.Code)

	if branch.Name == "" {
		return 0, ErrBranchNameRequired
	}
	if branch.Code == "" {
		return 0, ErrBranchCodeRequired
	}

	if existing, err := s.repo.GetByCode(ctx, branch.Code); err == nil && existing != nil {
		return 0, ErrBranchCodeDuplicate
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	id, err := s.repo.CreateWithTx(ctx, tx, branch)
	if err != nil {
		return 0, err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", id)
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      "branch.create",
			Resource:    "branch",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": branch.Code,
			},
		})
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) List(ctx context.Context, filter BranchFilter) ([]Branch, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, errors.New("missing user id")
	}

	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin {
		return s.repo.List(ctx, filter)
	}
	return s.repo.ListAccessibleBranches(ctx, filter, userID)
}

func (s *Service) ListAccessibleBranches(ctx context.Context, filter BranchFilter, userID int64) ([]Branch, error) {
	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin {
		return s.repo.List(ctx, filter)
	}
	return s.repo.ListAccessibleBranches(ctx, filter, userID)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Branch, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, errors.New("missing user id")
	}

	branch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}

	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin {
		return branch, nil
	}

	allowed, err := s.repo.UserHasAccess(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrBranchAccessDenied
	}
	return branch, nil
}

func (s *Service) isSuperAdmin(ctx context.Context, userID int64) (bool, error) {
	if s.identityProvider == nil {
		return false, nil
	}
	identity, err := s.identityProvider.GetIdentity(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, role := range identity.Roles {
		if role == "SUPER_ADMIN" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Update(ctx context.Context, branch *Branch) error {
	existing, err := s.repo.GetByID(ctx, branch.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBranchNotFound
		}
		return err
	}

	branch.Name = strings.TrimSpace(branch.Name)
	branch.Code = strings.TrimSpace(branch.Code)

	if branch.Name == "" {
		return ErrBranchNameRequired
	}
	if branch.Code == "" {
		return ErrBranchCodeRequired
	}

	if existing.Code != branch.Code {
		if other, err := s.repo.GetByCode(ctx, branch.Code); err == nil && other != nil && other.ID != branch.ID {
			return ErrBranchCodeDuplicate
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err := s.repo.UpdateWithTx(ctx, tx, branch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBranchNotFound
		}
		return err
	}

	if s.auditSvc != nil {
		actorUserID := actorUserIDFromContext(ctx)
		resourceID := fmt.Sprintf("%d", branch.ID)
		action := "branch.update"
		if !branch.IsActive && existing.IsActive {
			action = "branch.deactivate"
		}
		_, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: actorUserID,
			Action:      action,
			Resource:    "branch",
			ResourceID:  &resourceID,
			Metadata: map[string]any{
				"code": branch.Code,
			},
		})
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error {
	branch, err := s.repo.GetByID(ctx, branchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBranchNotFound
		}
		return err
	}
	if requireActive && !branch.IsActive {
		return ErrBranchInactive
	}
	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if isSuperAdmin {
		return nil
	}
	allowed, err := s.repo.UserHasAccess(ctx, userID, branchID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrBranchAccessDenied
	}
	return nil
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}
