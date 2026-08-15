package reporting

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
)

type fakeReportingRepo struct {
	salesReport     *SalesReport
	purchasesReport *PurchasesReport
	inventoryReport *InventoryReport
	profitReport    *ProfitReport
	err             error
}

func (f *fakeReportingRepo) GetSalesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*SalesReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.salesReport, nil
}

func (f *fakeReportingRepo) GetPurchasesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*PurchasesReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.purchasesReport, nil
}

func (f *fakeReportingRepo) GetInventoryReport(ctx context.Context, branchIDs []int64, productID *int64) (*InventoryReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.inventoryReport, nil
}

func (f *fakeReportingRepo) GetProfitReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*ProfitReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.profitReport, nil
}

type fakeReportingBranchService struct {
	branches []branches.Branch
	err      error
}

func (f *fakeReportingBranchService) EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeReportingBranchService) ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error) {
	return f.branches, nil
}

type fakeReportingPermissionChecker struct {
	allowed bool
	err     error
}

func (f *fakeReportingPermissionChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return f.allowed, f.err
}

type fakeReportingIdentityProvider struct{ roles []string }

func (f *fakeReportingIdentityProvider) GetIdentity(ctx context.Context, userID int64) (*auth.Identity, error) {
	return &auth.Identity{UserID: userID, Roles: f.roles}, nil
}

func TestReportingService_Unauthenticated(t *testing.T) {
	svc := NewService(&fakeReportingRepo{salesReport: &SalesReport{}}, &fakeReportingBranchService{}, &fakeReportingPermissionChecker{allowed: true}, nil)
	_, err := svc.GetSalesReport(context.Background(), "2026-01-01", "2026-01-31", nil)
	if !errors.Is(err, ErrAuthenticationRequired) {
		t.Fatalf("expected authentication required, got %v", err)
	}
}

func TestReportingService_MissingPermission(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	svc := NewService(&fakeReportingRepo{salesReport: &SalesReport{}}, &fakeReportingBranchService{}, &fakeReportingPermissionChecker{allowed: false}, nil)
	_, err := svc.GetSalesReport(ctx, "2026-01-01", "2026-01-31", nil)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestReportingService_SuperAdminAccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 99)
	repo := &fakeReportingRepo{salesReport: &SalesReport{TotalSales: 200}}
	svc := NewService(repo, &fakeReportingBranchService{}, &fakeReportingPermissionChecker{allowed: true}, &fakeReportingIdentityProvider{roles: []string{"SUPER_ADMIN"}})
	report, err := svc.GetSalesReport(ctx, "2026-01-01", "2026-01-31", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalSales != 200 {
		t.Fatalf("expected total sales = 200, got %v", report.TotalSales)
	}
}

func TestReportingService_UserOwnBranch(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	repo := &fakeReportingRepo{salesReport: &SalesReport{TotalSales: 75}}
	branchSvc := &fakeReportingBranchService{branches: []branches.Branch{{ID: 5, IsActive: true}}}
	svc := NewService(repo, branchSvc, &fakeReportingPermissionChecker{allowed: true}, &fakeReportingIdentityProvider{})
	report, err := svc.GetSalesReport(ctx, "2026-01-01", "2026-01-31", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalSales != 75 {
		t.Fatalf("expected 75, got %v", report.TotalSales)
	}
}

func TestReportingService_ForbiddenBranchAccess(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	svc := NewService(&fakeReportingRepo{salesReport: &SalesReport{}}, &fakeReportingBranchService{err: branches.ErrBranchAccessDenied}, &fakeReportingPermissionChecker{allowed: true}, &fakeReportingIdentityProvider{})
	branchID := int64(99)
	_, err := svc.GetSalesReport(ctx, "2026-01-01", "2026-01-31", &branchID)
	if !errors.Is(err, branches.ErrBranchAccessDenied) {
		t.Fatalf("expected branch access denied, got %v", err)
	}
}

func TestReportingService_InvalidDateRange(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	svc := NewService(&fakeReportingRepo{salesReport: &SalesReport{}}, &fakeReportingBranchService{branches: []branches.Branch{{ID: 5}}}, &fakeReportingPermissionChecker{allowed: true}, &fakeReportingIdentityProvider{})
	_, err := svc.GetSalesReport(ctx, "2026-01-31", "2026-01-01", nil)
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("expected invalid date range, got %v", err)
	}
}

func TestReportingService_ValidRangeAndOptionalFilters(t *testing.T) {
	ctx := auth.ContextWithUserID(context.Background(), 10)
	repo := &fakeReportingRepo{inventoryReport: &InventoryReport{TotalInventoryRecords: 6, TotalQuantity: 42}}
	svc := NewService(repo, &fakeReportingBranchService{branches: []branches.Branch{{ID: 5, IsActive: true}}}, &fakeReportingPermissionChecker{allowed: true}, &fakeReportingIdentityProvider{})
	branchID := int64(5)
	productID := int64(7)
	report, err := svc.GetInventoryReport(ctx, &branchID, &productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalInventoryRecords != 6 || report.TotalQuantity != 42 {
		t.Fatalf("unexpected inventory report: %+v", report)
	}
}
