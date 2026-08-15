package reporting

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

type fakeReportingHandlerService struct {
	salesReport     *SalesReport
	purchasesReport *PurchasesReport
	inventoryReport *InventoryReport
	profitReport    *ProfitReport
	err             error
}

func (f *fakeReportingHandlerService) GetSalesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*SalesReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.salesReport, nil
}

func (f *fakeReportingHandlerService) GetPurchasesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*PurchasesReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.purchasesReport, nil
}

func (f *fakeReportingHandlerService) GetInventoryReport(ctx context.Context, branchID *int64, productID *int64) (*InventoryReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.inventoryReport, nil
}

func (f *fakeReportingHandlerService) GetProfitReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*ProfitReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.profitReport, nil
}

func TestReportingHandler_InvalidBranchID(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{salesReport: &SalesReport{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31&branch_id=bad", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestReportingHandler_Unauthorized(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{err: ErrAuthenticationRequired})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestReportingHandler_Forbidden(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{err: ErrForbidden})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestReportingHandler_Success(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{salesReport: &SalesReport{TotalSales: 125, TotalTransactions: 2, TotalItemsSold: 6, TotalRevenue: 125}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
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

func TestReportingHandler_EmptyReport(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{salesReport: &SalesReport{}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestReportingHandler_BranchAccessDenied(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{err: branches.ErrBranchAccessDenied})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31&branch_id=9", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.Code)
	}
}

func TestReportingHandler_InvalidDateRangeMapping(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{err: ErrInvalidDateRange})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-31&end_date=2026-01-01", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestReportingHandler_GenericErrorMapping(t *testing.T) {
	h := NewHandler(&fakeReportingHandlerService{err: errors.New("db boom")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales?start_date=2026-01-01&end_date=2026-01-31", nil)
	resp := httptest.NewRecorder()
	h.SalesReport(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}
