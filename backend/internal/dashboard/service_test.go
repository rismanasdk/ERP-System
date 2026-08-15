package dashboard

import (
	"context"
	"errors"
	"testing"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
)

type fakeDashboardRepo struct {
	sales          *SalesSummary
	purchases      *PurchasesSummary
	err            error
	productCount   int64
	customerCount  int64
	supplierCount  int64
	inventoryCount int64
}

func (f *fakeDashboardRepo) GetSalesSummary(ctx context.Context, branchID *int64) (*SalesSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sales, nil
}

func (f *fakeDashboardRepo) GetPurchasesSummary(ctx context.Context, branchID *int64) (*PurchasesSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.purchases, nil
}

func (f *fakeDashboardRepo) GetActiveProductsCount(ctx context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.productCount, nil
}

func (f *fakeDashboardRepo) GetActiveCustomersCount(ctx context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.customerCount, nil
}

func (f *fakeDashboardRepo) GetActiveSuppliersCount(ctx context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.supplierCount, nil
}

func (f *fakeDashboardRepo) GetInventoryCount(ctx context.Context, branchID *int64) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.inventoryCount, nil
}

type fakeDashboardBranchService struct {
	err      error
	branches []branches.Branch
}

func (f *fakeDashboardBranchService) EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeDashboardBranchService) ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error) {
	return f.branches, nil
}

type fakeDashboardIdentityProvider struct{ roles []string }

func (f *fakeDashboardIdentityProvider) GetIdentity(ctx context.Context, userID int64) (*auth.Identity, error) {
	return &auth.Identity{UserID: userID, Roles: f.roles}, nil
}

type fakeDashboardPermissionChecker struct {
	allowed bool
	err     error
}

func (f *fakeDashboardPermissionChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return f.allowed, f.err
}

func TestDashboardService_Unauthenticated(t *testing.T) {
	svc := NewService(&fakeDashboardRepo{sales: &SalesSummary{}}, &fakeDashboardBranchService{}, &fakeDashboardPermissionChecker{allowed: true}, nil)
	_, err := svc.GetSummary(context.Background(), nil)
	if !errors.Is(err, ErrAuthenticationRequired) {
		t.Fatalf("expected ErrAuthenticationRequired, got %v", err)
	}
}

func TestDashboardService_MissingPermission(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	svc := NewService(&fakeDashboardRepo{sales: &SalesSummary{}}, &fakeDashboardBranchService{}, &fakeDashboardPermissionChecker{allowed: false}, nil)
	_, err := svc.GetSummary(ctx, nil)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDashboardService_NormalUserOwnBranch(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	repo := &fakeDashboardRepo{
		sales:        &SalesSummary{TodayAmount: 10, TodayTransactions: 2, MonthAmount: 100, MonthTransactions: 5},
		purchases:    &PurchasesSummary{TodayAmount: 7, TodayTransactions: 1, MonthAmount: 75, MonthTransactions: 4},
		productCount: 12, customerCount: 8, supplierCount: 7, inventoryCount: 33,
	}
	branchSvc := &fakeDashboardBranchService{branches: []branches.Branch{{ID: 5, IsActive: true}}}
	svc := NewService(repo, branchSvc, &fakeDashboardPermissionChecker{allowed: true}, &fakeDashboardIdentityProvider{})
	res, err := svc.GetSummary(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Sales.TodayAmount != 10 || res.Inventory.TotalItems != 33 {
		t.Fatalf("unexpected summary payload: %+v", res)
	}
}

func TestDashboardService_NormalUserBlockedOtherBranch(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	repo := &fakeDashboardRepo{sales: &SalesSummary{}, purchases: &PurchasesSummary{}}
	branchSvc := &fakeDashboardBranchService{err: branches.ErrBranchAccessDenied}
	svc := NewService(repo, branchSvc, &fakeDashboardPermissionChecker{allowed: true}, &fakeDashboardIdentityProvider{})
	branchID := int64(99)
	_, err := svc.GetSummary(ctx, &branchID)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected ErrBranchAccessDenied, got %v", err)
	}
}

func TestDashboardService_SuperAdmin_NoBranch(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 99)
	repo := &fakeDashboardRepo{sales: &SalesSummary{}, purchases: &PurchasesSummary{}, productCount: 4, customerCount: 5, supplierCount: 6, inventoryCount: 40}
	svc := NewService(repo, &fakeDashboardBranchService{}, &fakeDashboardPermissionChecker{allowed: true}, &fakeDashboardIdentityProvider{roles: []string{"SUPER_ADMIN"}})
	res, err := svc.GetSummary(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.MasterData.Products != 4 || res.Inventory.TotalItems != 40 {
		t.Fatalf("unexpected summary for super admin: %+v", res)
	}
}

func TestDashboardService_SuperAdmin_RequestedBranch(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 99)
	repo := &fakeDashboardRepo{sales: &SalesSummary{TodayTransactions: 1}, purchases: &PurchasesSummary{MonthTransactions: 2}, productCount: 1, customerCount: 2, supplierCount: 3, inventoryCount: 7}
	branchSvc := &fakeDashboardBranchService{}
	svc := NewService(repo, branchSvc, &fakeDashboardPermissionChecker{allowed: true}, &fakeDashboardIdentityProvider{roles: []string{"SUPER_ADMIN"}})
	branchID := int64(5)
	res, err := svc.GetSummary(ctx, &branchID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Sales.TodayTransactions != 1 || res.Purchases.MonthTransactions != 2 {
		t.Fatalf("unexpected summary for requested branch: %+v", res)
	}
}

func TestDashboardService_RepositoryError(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	repo := &fakeDashboardRepo{err: errors.New("db boom")}
	svc := NewService(repo, &fakeDashboardBranchService{branches: []branches.Branch{{ID: 2, IsActive: true}}}, &fakeDashboardPermissionChecker{allowed: true}, &fakeDashboardIdentityProvider{})
	_, err := svc.GetSummary(ctx, nil)
	if err == nil || err.Error() != "db boom" {
		t.Fatalf("expected repo error, got %v", err)
	}
}
