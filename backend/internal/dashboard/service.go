package dashboard

import (
	"context"
	"errors"
	"fmt"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
)

type branchService interface {
	EnsureUserHasAccess(ctx context.Context, userID, branchID int64, requireActive bool) error
	ListAccessibleBranches(ctx context.Context, filter branches.BranchFilter, userID int64) ([]branches.Branch, error)
}

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrForbidden              = errors.New("permission denied")
)

type RepositoryInterface interface {
	GetSalesSummary(ctx context.Context, branchID *int64) (*SalesSummary, error)
	GetPurchasesSummary(ctx context.Context, branchID *int64) (*PurchasesSummary, error)
	GetActiveProductsCount(ctx context.Context) (int64, error)
	GetActiveCustomersCount(ctx context.Context) (int64, error)
	GetActiveSuppliersCount(ctx context.Context) (int64, error)
	GetInventoryCount(ctx context.Context, branchID *int64) (int64, error)
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

func (s *Service) GetSummary(ctx context.Context, requestedBranchID *int64) (*Summary, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok || userID == 0 {
		return nil, ErrAuthenticationRequired
	}

	allowed, err := s.authChecker.HasPermission(ctx, userID, DashboardReadPermission)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}

	isSuperAdmin, err := s.isSuperAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}

	var branchID *int64
	if requestedBranchID != nil {
		if err := s.branchSvc.EnsureUserHasAccess(ctx, userID, *requestedBranchID, true); err != nil {
			return nil, err
		}
		branchID = requestedBranchID
	} else if !isSuperAdmin {
		branchID, err = s.defaultUserBranchID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	sales, err := s.repo.GetSalesSummary(ctx, branchID)
	if err != nil {
		return nil, err
	}
	purchases, err := s.repo.GetPurchasesSummary(ctx, branchID)
	if err != nil {
		return nil, err
	}
	productCount, err := s.repo.GetActiveProductsCount(ctx)
	if err != nil {
		return nil, err
	}
	customerCount, err := s.repo.GetActiveCustomersCount(ctx)
	if err != nil {
		return nil, err
	}
	supplierCount, err := s.repo.GetActiveSuppliersCount(ctx)
	if err != nil {
		return nil, err
	}
	inventoryCount, err := s.repo.GetInventoryCount(ctx, branchID)
	if err != nil {
		return nil, err
	}

	return &Summary{
		Sales: SalesSummary{
			TodayAmount:       sales.TodayAmount,
			TodayTransactions: sales.TodayTransactions,
			MonthAmount:       sales.MonthAmount,
			MonthTransactions: sales.MonthTransactions,
		},
		Purchases: PurchasesSummary{
			TodayAmount:       purchases.TodayAmount,
			TodayTransactions: purchases.TodayTransactions,
			MonthAmount:       purchases.MonthAmount,
			MonthTransactions: purchases.MonthTransactions,
		},
		MasterData: MasterDataSummary{
			Products:  productCount,
			Customers: customerCount,
			Suppliers: supplierCount,
		},
		Inventory: InventorySummary{TotalItems: inventoryCount},
	}, nil
}

func boolPtr(v bool) *bool {
	return &v
}

func (s *Service) defaultUserBranchID(ctx context.Context, userID int64) (*int64, error) {
	accessible, err := s.branchSvc.ListAccessibleBranches(ctx, branches.BranchFilter{Active: boolPtr(true)}, userID)
	if err != nil {
		return nil, err
	}
	if len(accessible) == 0 {
		return nil, nil
	}
	id := accessible[0].ID
	return &id, nil
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

func int64Ptr(v int64) *int64 {
	return &v
}

func actorUserIDFromContext(ctx context.Context) *int64 {
	if userID, ok := auth.UserIDFromContext(ctx); ok {
		return &userID
	}
	return nil
}

func (s *Service) String() string {
	return fmt.Sprintf("dashboard service")
}
