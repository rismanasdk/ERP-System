package reporting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
)

type branchService interface {
	EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error
	ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error)
}

type RepositoryInterface interface {
	GetSalesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*SalesReport, error)
	GetPurchasesReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*PurchasesReport, error)
	GetInventoryReport(ctx context.Context, branchIDs []int64, productID *int64) (*InventoryReport, error)
	GetProfitReport(ctx context.Context, startDate, endDate *time.Time, branchIDs []int64) (*ProfitReport, error)
}

type Service struct {
	repo             RepositoryInterface
	branchSvc        branchService
	authChecker      auth.PermissionChecker
	identityProvider auth.IdentityProvider
}

func NewService(repo RepositoryInterface, branchSvc branchService, authChecker auth.PermissionChecker, identityProvider auth.IdentityProvider) *Service {
	return &Service{repo: repo, branchSvc: branchSvc, authChecker: authChecker, identityProvider: identityProvider}
}

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrForbidden              = errors.New("permission denied")
	ErrInvalidDateRange       = errors.New("invalid date range")
	ErrInvalidBranchID        = errors.New("invalid branch id")
	ErrInvalidProductID       = errors.New("invalid product id")
)

func (s *Service) GetSalesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*SalesReport, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if allowed, err := s.authChecker.HasPermission(ctx, userID, ReportReadPermission); err != nil {
		return nil, err
	} else if !allowed {
		return nil, ErrForbidden
	}
	startDate, endDate, err := parseDateRange(startDateRaw, endDateRaw)
	if err != nil {
		return nil, err
	}
	branchIDs, err := s.resolveBranchScope(ctx, userID, branchID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetSalesReport(ctx, startDate, endDate, branchIDs)
}

func (s *Service) GetPurchasesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*PurchasesReport, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if allowed, err := s.authChecker.HasPermission(ctx, userID, ReportReadPermission); err != nil {
		return nil, err
	} else if !allowed {
		return nil, ErrForbidden
	}
	startDate, endDate, err := parseDateRange(startDateRaw, endDateRaw)
	if err != nil {
		return nil, err
	}
	branchIDs, err := s.resolveBranchScope(ctx, userID, branchID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetPurchasesReport(ctx, startDate, endDate, branchIDs)
}

func (s *Service) GetInventoryReport(ctx context.Context, branchID *int64, productID *int64) (*InventoryReport, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if allowed, err := s.authChecker.HasPermission(ctx, userID, ReportReadPermission); err != nil {
		return nil, err
	} else if !allowed {
		return nil, ErrForbidden
	}
	if productID != nil && *productID <= 0 {
		return nil, ErrInvalidProductID
	}
	branchIDs, err := s.resolveBranchScope(ctx, userID, branchID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetInventoryReport(ctx, branchIDs, productID)
}

func (s *Service) GetProfitReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*ProfitReport, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}
	if allowed, err := s.authChecker.HasPermission(ctx, userID, ReportReadPermission); err != nil {
		return nil, err
	} else if !allowed {
		return nil, ErrForbidden
	}
	startDate, endDate, err := parseDateRange(startDateRaw, endDateRaw)
	if err != nil {
		return nil, err
	}
	branchIDs, err := s.resolveBranchScope(ctx, userID, branchID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetProfitReport(ctx, startDate, endDate, branchIDs)
}

func (s *Service) resolveBranchScope(ctx context.Context, userID int64, requestedBranch *int64) ([]int64, error) {
	if requestedBranch != nil {
		if *requestedBranch <= 0 {
			return nil, ErrInvalidBranchID
		}
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *requestedBranch, true); err != nil {
			return nil, err
		}
		return []int64{*requestedBranch}, nil
	}
	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isSuperAdmin {
		return nil, nil
	}
	accessible, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{Active: boolPtr(true)}, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(accessible))
	for _, branch := range accessible {
		ids = append(ids, branch.ID)
	}
	return ids, nil
}

func (s *Service) isSuperAdmin(ctx context.Context, userID int64) (bool, error) {
	if s.identityProvider == nil {
		return false, nil
	}
	identity, err := s.identityProvider.GetIdentity(ctx, userID)
	if err != nil {
		return false, err
	}
	if identity == nil {
		return false, nil
	}
	for _, role := range identity.Roles {
		if role == "SUPER_ADMIN" {
			return true, nil
		}
	}
	return false, nil
}

func parseDateRange(startDateRaw, endDateRaw string) (*time.Time, *time.Time, error) {
	if stringsTrim(startDateRaw) == "" || stringsTrim(endDateRaw) == "" {
		return nil, nil, ErrInvalidDateRange
	}
	start, err := time.Parse("2006-01-02", stringsTrim(startDateRaw))
	if err != nil {
		return nil, nil, ErrInvalidDateRange
	}
	end, err := time.Parse("2006-01-02", stringsTrim(endDateRaw))
	if err != nil {
		return nil, nil, ErrInvalidDateRange
	}
	if start.After(end) {
		return nil, nil, ErrInvalidDateRange
	}
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTC := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	return &startUTC, &endUTC, nil
}

func boolPtr(v bool) *bool {
	return &v
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}

func formatDateRangeError() error {
	return fmt.Errorf("%w: start_date and end_date are required and must be valid ISO dates", ErrInvalidDateRange)
}
