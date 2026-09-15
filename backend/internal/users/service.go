package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/roles"
	"erp-system/backend/pkg/password"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserEmailRequired  = errors.New("email is required")
	ErrUserNameRequired   = errors.New("name is required")
	ErrUserPasswordNeeded = errors.New("password is required")
	ErrUserDuplicateEmail = errors.New("email already exists")
	ErrUserRoleNotFound   = errors.New("role not found")
	ErrUserBranchNotFound = errors.New("branch not found")
	ErrUserAccessDenied   = errors.New("user access denied")
	ErrUserSelfDelete     = errors.New("cannot delete the authenticated user")
	ErrProtectedUser      = errors.New("the SUPER_ADMIN user is protected")
	ErrUserPrivilege      = errors.New("insufficient privilege for requested role or branch")
)

type Service struct {
	repo     *Repository
	roleRepo *roles.Repository
	auditSvc *audit.Service
}

func NewService(repo *Repository, roleRepo *roles.Repository, auditSvc *audit.Service) *Service {
	return &Service{repo: repo, roleRepo: roleRepo, auditSvc: auditSvc}
}

func (s *Service) List(ctx context.Context, filter UserFilter) ([]User, error) {
	actorID, scoped := actorIDFromContext(ctx)
	var users []User
	var err error
	if scoped && !s.actorIsSuperAdmin(ctx, actorID) {
		users, err = s.repo.ListAccessible(ctx, filter, actorID)
	} else {
		users, err = s.repo.List(ctx, filter)
	}
	if err != nil {
		return nil, err
	}
	for i := range users {
		if scoped {
			users[i].IsActive, err = s.repo.GetActiveStatus(ctx, users[i].ID)
			if err != nil {
				return nil, err
			}
		}
		if users[i].RoleNames, err = s.repo.GetRoleNames(ctx, users[i].ID); err != nil {
			return nil, err
		}
		if users[i].BranchIDs, err = s.repo.GetBranchIDs(ctx, users[i].ID); err != nil {
			return nil, err
		}
	}
	return users, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if actorID, scoped := actorIDFromContext(ctx); scoped && !s.actorIsSuperAdmin(ctx, actorID) {
		allowed, accessErr := s.repo.HasBranchAccess(ctx, actorID, id)
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			return nil, ErrUserAccessDenied
		}
	}
	if _, scoped := actorIDFromContext(ctx); scoped {
		user.IsActive, err = s.repo.GetActiveStatus(ctx, user.ID)
		if err != nil {
			return nil, err
		}
	}
	user.RoleNames, err = s.repo.GetRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.BranchIDs, err = s.repo.GetBranchIDs(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) Create(ctx context.Context, user *User, roleNames []string, branchIDs []int64) (int64, error) {
	if err := validateUser(user); err != nil {
		return 0, err
	}
	if user.PasswordHash == "" {
		return 0, ErrUserPasswordNeeded
	}
	if actorID, scoped := actorIDFromContext(ctx); scoped && !s.actorIsSuperAdmin(ctx, actorID) {
		if err := s.validateRoleAssignments(ctx, actorID, roleNames); err != nil {
			return 0, err
		}
		if err := s.validateBranchAssignments(ctx, actorID, branchIDs); err != nil {
			return 0, err
		}
	}

	if existing, err := s.repo.GetByEmail(ctx, user.Email); err == nil && existing != nil {
		return 0, ErrUserDuplicateEmail
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	hashedPassword, err := password.Hash(user.PasswordHash)
	if err != nil {
		return 0, err
	}
	user.PasswordHash = hashedPassword
	user.Email = strings.TrimSpace(user.Email)
	user.Name = strings.TrimSpace(user.Name)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	id, err := s.repo.CreateWithTx(ctx, tx, user)
	if err != nil {
		return 0, err
	}

	for _, roleName := range roleNames {
		trimmed := strings.TrimSpace(roleName)
		if trimmed == "" {
			continue
		}
		role, err := s.roleRepo.GetByNameTx(ctx, tx, trimmed)
		if err != nil {
			return 0, err
		}
		if role == nil {
			return 0, fmt.Errorf("%w: %s", ErrUserRoleNotFound, trimmed)
		}
		if err := s.repo.AddRoleWithTx(ctx, tx, id, role.ID); err != nil {
			return 0, err
		}
	}

	for _, branchID := range branchIDs {
		if branchID <= 0 {
			continue
		}
		exists, err := s.repo.branchExists(ctx, branchID)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, fmt.Errorf("%w: %d", ErrUserBranchNotFound, branchID)
		}
		if err := s.repo.AddBranchWithTx(ctx, tx, id, branchID); err != nil {
			return 0, err
		}
	}
	if _, scoped := actorIDFromContext(ctx); scoped {
		if err = s.repo.SetActiveStatusWithTx(ctx, tx, id, user.IsActive); err != nil {
			return 0, err
		}
	}

	if s.auditSvc != nil {
		auditResourceID := fmt.Sprintf("%d", id)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: userActorIDFromContext(ctx),
			Action:      "user.create",
			Resource:    "user",
			ResourceID:  &auditResourceID,
			Metadata: map[string]any{
				"email": user.Email,
				"name":  user.Name,
			},
		}); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) Update(ctx context.Context, user *User, roleNames []string, branchIDs []int64) error {
	if user == nil || user.ID == 0 {
		return ErrUserNotFound
	}
	if err := validateUser(user); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if actorID, scoped := actorIDFromContext(ctx); scoped && !s.actorIsSuperAdmin(ctx, actorID) {
		allowed, accessErr := s.repo.HasBranchAccess(ctx, actorID, user.ID)
		if accessErr != nil {
			return accessErr
		}
		if !allowed {
			return ErrUserAccessDenied
		}
		if err := s.validateRoleAssignments(ctx, actorID, roleNames); err != nil {
			return err
		}
		if err := s.validateBranchAssignments(ctx, actorID, branchIDs); err != nil {
			return err
		}
	}
	if existing.Email != user.Email {
		if other, err := s.repo.GetByEmail(ctx, user.Email); err == nil && other != nil && other.ID != user.ID {
			return ErrUserDuplicateEmail
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	if user.PasswordHash != "" {
		hashedPassword, err := password.Hash(user.PasswordHash)
		if err != nil {
			return err
		}
		user.PasswordHash = hashedPassword
	} else {
		user.PasswordHash = existing.PasswordHash
	}
	user.Email = strings.TrimSpace(user.Email)
	user.Name = strings.TrimSpace(user.Name)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err := s.repo.UpdateWithTx(ctx, tx, user); err != nil {
		return err
	}

	if err := s.repo.DeleteRolesWithTx(ctx, tx, user.ID); err != nil {
		return err
	}
	for _, roleName := range roleNames {
		trimmed := strings.TrimSpace(roleName)
		if trimmed == "" {
			continue
		}
		role, err := s.roleRepo.GetByNameTx(ctx, tx, trimmed)
		if err != nil {
			return err
		}
		if role == nil {
			return fmt.Errorf("%w: %s", ErrUserRoleNotFound, trimmed)
		}
		if err := s.repo.AddRoleWithTx(ctx, tx, user.ID, role.ID); err != nil {
			return err
		}
	}

	if err := s.repo.DeleteBranchesWithTx(ctx, tx, user.ID); err != nil {
		return err
	}
	for _, branchID := range branchIDs {
		if branchID <= 0 {
			continue
		}
		exists, err := s.repo.branchExists(ctx, branchID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %d", ErrUserBranchNotFound, branchID)
		}
		if err := s.repo.AddBranchWithTx(ctx, tx, user.ID, branchID); err != nil {
			return err
		}
	}
	if _, scoped := actorIDFromContext(ctx); scoped {
		if err = s.repo.SetActiveStatusWithTx(ctx, tx, user.ID, user.IsActive); err != nil {
			return err
		}
	}

	if s.auditSvc != nil {
		auditResourceID := fmt.Sprintf("%d", user.ID)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: userActorIDFromContext(ctx),
			Action:      "user.update",
			Resource:    "user",
			ResourceID:  &auditResourceID,
			Metadata: map[string]any{
				"email": user.Email,
				"name":  user.Name,
			},
		}); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) ListByBranch(ctx context.Context, branchID int64) ([]User, error) {
	if branchID <= 0 {
		return nil, ErrUserBranchNotFound
	}
	usersList, err := s.repo.ListByBranch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	for i := range usersList {
		usersList[i].RoleNames, err = s.repo.GetRoleNames(ctx, usersList[i].ID)
		if err != nil {
			return nil, err
		}
		usersList[i].BranchIDs, err = s.repo.GetBranchIDs(ctx, usersList[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return usersList, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrUserNotFound
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if actorID, scoped := actorIDFromContext(ctx); scoped {
		if actorID == id {
			return ErrUserSelfDelete
		}
		if !s.actorIsSuperAdmin(ctx, actorID) {
			allowed, accessErr := s.repo.HasBranchAccess(ctx, actorID, id)
			if accessErr != nil {
				return accessErr
			}
			if !allowed {
				return ErrUserAccessDenied
			}
		}
	}
	if rolesForUser, err := s.repo.GetRoleNames(ctx, id); err != nil {
		return err
	} else if containsRole(rolesForUser, "SUPER_ADMIN") {
		return ErrProtectedUser
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

	if _, err = tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM user_branches WHERE user_id = $1`, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}

	if s.auditSvc != nil {
		auditResourceID := fmt.Sprintf("%d", id)
		if _, err = s.auditSvc.RecordWithTx(ctx, tx, audit.AuditLog{
			ActorUserID: userActorIDFromContext(ctx),
			Action:      "user.delete",
			Resource:    "user",
			ResourceID:  &auditResourceID,
			Metadata:    map[string]any{},
		}); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func validateUser(user *User) error {
	if user == nil {
		return ErrUserNotFound
	}
	user.Email = strings.TrimSpace(user.Email)
	user.Name = strings.TrimSpace(user.Name)
	if user.Email == "" {
		return ErrUserEmailRequired
	}
	if user.Name == "" {
		return ErrUserNameRequired
	}
	return nil
}

func (s *Service) validateBranchAssignments(ctx context.Context, actorID int64, branchIDs []int64) error {
	if len(branchIDs) == 0 {
		return ErrUserPrivilege
	}
	for _, branchID := range branchIDs {
		if branchID <= 0 {
			continue
		}
		allowed, err := s.repo.HasBranchAccess(ctx, actorID, branchID)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrUserPrivilege
		}
	}
	return nil
}

func (s *Service) validateRoleAssignments(ctx context.Context, actorID int64, roleNames []string) error {
	actorPermissions, err := s.repo.GetPermissionNames(ctx, actorID)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool, len(actorPermissions))
	for _, permission := range actorPermissions {
		allowed[permission] = true
	}
	for _, roleName := range roleNames {
		role, err := s.roleRepo.GetByName(ctx, strings.TrimSpace(roleName))
		if err != nil {
			return err
		}
		if role == nil {
			return fmt.Errorf("%w: %s", ErrUserRoleNotFound, roleName)
		}
		if role.Name == "SUPER_ADMIN" {
			return ErrUserPrivilege
		}
		permissions, err := s.roleRepo.GetPermissionNamesByRoleID(ctx, role.ID)
		if err != nil {
			return err
		}
		for _, permission := range permissions {
			if !allowed[permission] {
				return ErrUserPrivilege
			}
		}
	}
	return nil
}

func (s *Service) actorIsSuperAdmin(ctx context.Context, actorID int64) bool {
	rolesForUser, err := s.repo.GetRoleNames(ctx, actorID)
	return err == nil && containsRole(rolesForUser, "SUPER_ADMIN")
}

func containsRole(roleNames []string, target string) bool {
	for _, roleName := range roleNames {
		if strings.EqualFold(strings.TrimSpace(roleName), target) {
			return true
		}
	}
	return false
}

func userActorIDFromContext(ctx context.Context) *int64 {
	var userID int64
	if value, ok := contextValueUserID(ctx); ok {
		userID = value
		return &userID
	}
	return nil
}

func contextValueUserID(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	if value, ok := ctx.Value("userID").(int64); ok {
		return value, true
	}
	return 0, false
}

func actorIDFromContext(ctx context.Context) (int64, bool) {
	return contextValueUserID(ctx)
}

func normalizeUser(user *User) {
	if user == nil {
		return
	}
	user.Email = strings.TrimSpace(user.Email)
	user.Name = strings.TrimSpace(user.Name)
	if strings.TrimSpace(user.PasswordHash) == "" {
		user.PasswordHash = ""
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Time{}
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Time{}
	}
}
