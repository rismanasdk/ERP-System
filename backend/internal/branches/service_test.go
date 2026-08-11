package branches

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-system/backend/internal/auth"
)

type fakeBranchRepo struct {
	branches        []Branch
	branch          *Branch
	accessible      []Branch
	allowedBranchID int64
	accessAllowed   bool
	err             error
}

func (r *fakeBranchRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return nil, errors.New("not implemented")
}
func (r *fakeBranchRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) (int64, error) {
	return 0, errors.New("not implemented")
}
func (r *fakeBranchRepo) GetByID(ctx context.Context, id int64) (*Branch, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.branch == nil || r.branch.ID != id {
		return nil, sql.ErrNoRows
	}
	return r.branch, nil
}
func (r *fakeBranchRepo) GetByCode(ctx context.Context, code string) (*Branch, error) {
	return nil, errors.New("not implemented")
}
func (r *fakeBranchRepo) List(ctx context.Context, filter BranchFilter) ([]Branch, error) {
	return r.branches, r.err
}
func (r *fakeBranchRepo) ListAccessibleBranches(ctx context.Context, filter BranchFilter, userID int64) ([]Branch, error) {
	return r.accessible, r.err
}
func (r *fakeBranchRepo) UpdateWithTx(ctx context.Context, tx *sql.Tx, branch *Branch) error {
	return errors.New("not implemented")
}
func (r *fakeBranchRepo) AssignUserBranchWithTx(ctx context.Context, tx *sql.Tx, userID, branchID int64) error {
	return errors.New("not implemented")
}
func (r *fakeBranchRepo) UserHasAccess(ctx context.Context, userID, branchID int64) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	return r.accessAllowed && branchID == r.allowedBranchID, nil
}

type fakeIdentityProvider struct {
	identity *auth.Identity
	err      error
}

func (f *fakeIdentityProvider) GetIdentity(ctx context.Context, userID int64) (*auth.Identity, error) {
	return f.identity, f.err
}

func TestService_List_UsesAccessibleBranchesForNonSuperAdmin(t *testing.T) {
	repo := &fakeBranchRepo{
		accessible: []Branch{{ID: 2, Name: "Branch 2", Code: "B2"}},
	}
	service := NewService(repo, &fakeIdentityProvider{identity: &auth.Identity{Roles: []string{"user"}}}, nil)

	ctx := auth.ContextWithUserID(context.Background(), int64(1))
	branches, err := service.List(ctx, BranchFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(branches) != 1 || branches[0].ID != 2 {
		t.Fatalf("expected accessible branch returned, got %+v", branches)
	}
}

func TestService_List_UsesAllBranchesForSuperAdmin(t *testing.T) {
	repo := &fakeBranchRepo{
		branches: []Branch{{ID: 1, Name: "HQ", Code: "HQ"}, {ID: 2, Name: "Branch 2", Code: "B2"}},
	}
	service := NewService(repo, &fakeIdentityProvider{identity: &auth.Identity{Roles: []string{"SUPER_ADMIN"}}}, nil)

	ctx := auth.ContextWithUserID(context.Background(), int64(1))
	branches, err := service.List(ctx, BranchFilter{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected all branches for super admin, got %+v", branches)
	}
}

func TestService_GetByID_RequiresAccessForNonSuperAdmin(t *testing.T) {
	branch := &Branch{ID: 2, Name: "Branch 2", Code: "B2"}
	repo := &fakeBranchRepo{branch: branch, allowedBranchID: 1, accessAllowed: false}
	service := NewService(repo, &fakeIdentityProvider{identity: &auth.Identity{Roles: []string{"user"}}}, nil)

	ctx := auth.ContextWithUserID(context.Background(), int64(42))
	_, err := service.GetByID(ctx, 2)
	if !errors.Is(err, ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
}

func TestService_GetByID_AllowsSuperAdmin(t *testing.T) {
	branch := &Branch{ID: 2, Name: "Branch 2", Code: "B2"}
	repo := &fakeBranchRepo{branch: branch, accessAllowed: false}
	service := NewService(repo, &fakeIdentityProvider{identity: &auth.Identity{Roles: []string{"SUPER_ADMIN"}}}, nil)

	ctx := auth.ContextWithUserID(context.Background(), int64(42))
	result, err := service.GetByID(ctx, 2)
	if err != nil {
		t.Fatalf("expected no error for super admin, got %v", err)
	}
	if result == nil || result.ID != 2 {
		t.Fatalf("expected branch result, got %+v", result)
	}
}
