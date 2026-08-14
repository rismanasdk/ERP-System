package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/internal/branches"
	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type fakeSalesService struct {
	createID     int64
	createErr    error
	createdInput CreateSaleInput
	listItems    []Sale
	listErr      error
	sale         *Sale
	getErr       error
	completeErr  error
	cancelErr    error
}

func (s *fakeSalesService) CreateSale(ctx context.Context, input CreateSaleInput) (int64, error) {
	s.createdInput = input
	return s.createID, s.createErr
}

func (s *fakeSalesService) ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error) {
	return s.listItems, s.listErr
}

func (s *fakeSalesService) GetSale(ctx context.Context, id int64) (*Sale, []SaleItem, error) {
	return s.sale, nil, s.getErr
}

func (s *fakeSalesService) CompleteSale(ctx context.Context, saleID int64) error {
	return s.completeErr
}

func (s *fakeSalesService) CancelSale(ctx context.Context, saleID int64) error {
	return s.cancelErr
}

func TestSaleHandler_Create_InvalidBody(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSaleHandler_Create_Success(t *testing.T) {
	service := &fakeSalesService{createID: 10, sale: &Sale{ID: 10, SaleNumber: "SALE-123"}}
	handler := NewHandler(service)

	body := bytes.NewBufferString(`{"branch_id":1,"items":[{"product_id":5,"quantity":3,"unit_price":100.0}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales", body)
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if id, ok := resp["id"].(float64); !ok || int64(id) != 10 {
		t.Fatalf("expected id 10, got %v", resp["id"])
	}
	if number, ok := resp["sale_number"].(string); !ok || number != "SALE-123" {
		t.Fatalf("expected sale_number SALE-123, got %v", resp["sale_number"])
	}
}

func TestSaleHandler_List_InvalidBranchIDQuery(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales?branch_id=abc", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid branch_id query parameter"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSaleHandler_List_Success(t *testing.T) {
	service := &fakeSalesService{listItems: []Sale{{ID: 5, SaleNumber: "SALE-001"}}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales?branch_id=1", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	items, ok := resp.Data.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one sale item, got %v", resp.Data)
	}
}

func TestSaleHandler_Get_InvalidSaleID(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid sale id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSaleHandler_Get_Success(t *testing.T) {
	service := &fakeSalesService{sale: &Sale{ID: 7, SaleNumber: "SALE-007"}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales/7", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if id, ok := data["id"].(float64); !ok || int64(id) != 7 {
		t.Fatalf("expected id 7, got %v", data["id"])
	}
}

func TestSaleHandler_Get_NotFound(t *testing.T) {
	service := &fakeSalesService{getErr: ErrSaleNotFound}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales/7", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"NOT_FOUND","message":"sale not found"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSaleHandler_List_BranchAccessDenied(t *testing.T) {
	service := &fakeSalesService{listErr: branches.ErrBranchAccessDenied}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sales?branch_id=1", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSaleHandler_Complete_InvalidSaleID(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/abc/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid sale id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSaleHandler_Complete_Success(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if id, ok := data["id"].(float64); !ok || int64(id) != 7 {
		t.Fatalf("expected id 7, got %v", data["id"])
	}
	if status, ok := data["status"].(string); !ok || status != SaleStatusCompleted {
		t.Fatalf("expected status %s, got %v", SaleStatusCompleted, data["status"])
	}
}

func TestSaleHandler_Complete_Forbidden(t *testing.T) {
	service := &fakeSalesService{completeErr: ErrForbidden}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSaleHandler_Complete_Unauthorized(t *testing.T) {
	service := &fakeSalesService{completeErr: ErrAuthenticationRequired}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestSaleHandler_Cancel_InvalidSaleID(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/abc/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSaleHandler_Cancel_Success(t *testing.T) {
	service := &fakeSalesService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if id, ok := data["id"].(float64); !ok || int64(id) != 7 {
		t.Fatalf("expected id 7, got %v", data["id"])
	}
	if status, ok := data["status"].(string); !ok || status != SaleStatusCancelled {
		t.Fatalf("expected status %s, got %v", SaleStatusCancelled, data["status"])
	}
}

func TestSaleHandler_Cancel_Forbidden(t *testing.T) {
	service := &fakeSalesService{cancelErr: ErrForbidden}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSaleHandler_Cancel_Unauthorized(t *testing.T) {
	service := &fakeSalesService{cancelErr: ErrAuthenticationRequired}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestSaleHandler_Complete_InvalidTransition(t *testing.T) {
	service := &fakeSalesService{completeErr: ErrInvalidSaleTransition}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSaleHandler_Cancel_InvalidTransition(t *testing.T) {
	service := &fakeSalesService{cancelErr: ErrInvalidSaleTransition}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSaleHandler_Complete_BranchAccessDenied(t *testing.T) {
	service := &fakeSalesService{completeErr: branches.ErrBranchAccessDenied}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/complete", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Complete(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSaleHandler_Cancel_BranchAccessDenied(t *testing.T) {
	service := &fakeSalesService{cancelErr: branches.ErrBranchAccessDenied}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/7/cancel", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Cancel(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}
