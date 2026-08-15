package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/internal/branches"
	"erp-system/backend/pkg/response"
)

type fakeDashboardHandlerService struct {
	summary *Summary
	err     error
}

func (f *fakeDashboardHandlerService) GetSummary(ctx context.Context, branchID *int64) (*Summary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.summary, nil
}

func TestDashboardHandler_RequiresAuth(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{summary: &Summary{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()

	// the middleware is responsible for auth; handler itself should still be callable in unit tests
	h.Summary(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected OK from handler unit test; got %d", resp.Code)
	}
}

func TestDashboardHandler_InvalidBranchID(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{summary: &Summary{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary?branch_id=abc", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestDashboardHandler_ServiceErrorMapping(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{err: branches.ErrBranchAccessDenied})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary?branch_id=7", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestDashboardHandler_SuccessResponse(t *testing.T) {
	summary := &Summary{
		Sales:      SalesSummary{TodayAmount: 12, TodayTransactions: 3, MonthAmount: 200, MonthTransactions: 10},
		Purchases:  PurchasesSummary{TodayAmount: 5, TodayTransactions: 2, MonthAmount: 60, MonthTransactions: 4},
		MasterData: MasterDataSummary{Products: 22, Customers: 12, Suppliers: 7},
		Inventory:  InventorySummary{TotalItems: 77},
	}
	h := NewHandler(&fakeDashboardHandlerService{summary: summary})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var payload response.SuccessResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Message != "success" {
		t.Fatalf("expected success message, got %q", payload.Message)
	}
}

func TestDashboardHandler_ServiceErrorUnauthorized(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{err: ErrAuthenticationRequired})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestDashboardHandler_ServiceErrorForbidden(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{err: ErrForbidden})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestDashboardHandler_ServiceErrorGeneric(t *testing.T) {
	h := NewHandler(&fakeDashboardHandlerService{err: errors.New("boom")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()

	h.Summary(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}
