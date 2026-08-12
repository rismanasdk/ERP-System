package purchasing

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

type fakePurchasingService struct {
	createID     int64
	createErr    error
	createdInput CreatePurchaseInput
	listItems    []Purchase
	listErr      error
	lastFilter   PurchaseFilter
	purchase     *Purchase
	getErr       error
}

func (s *fakePurchasingService) CreatePurchase(ctx context.Context, input CreatePurchaseInput) (int64, error) {
	s.createdInput = input
	return s.createID, s.createErr
}

func (s *fakePurchasingService) ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error) {
	s.lastFilter = filter
	return s.listItems, s.listErr
}

func (s *fakePurchasingService) GetPurchase(ctx context.Context, id int64) (*Purchase, error) {
	return s.purchase, s.getErr
}

func TestPurchaseHandler_Create_InvalidBody(t *testing.T) {
	service := &fakePurchasingService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestPurchaseHandler_Create_Success(t *testing.T) {
	service := &fakePurchasingService{createID: 10, purchase: &Purchase{ID: 10, PurchaseNumber: "PO-123"}}
	handler := NewHandler(service)

	body := bytes.NewBufferString(`{"branch_id":1,"supplier_id":2,"items":[{"product_id":5,"quantity":3,"unit_cost":100.0}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchases", body)
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
	if number, ok := resp["purchase_number"].(string); !ok || number != "PO-123" {
		t.Fatalf("expected purchase_number PO-123, got %v", resp["purchase_number"])
	}
}

func TestPurchaseHandler_List_InvalidBranchIDQuery(t *testing.T) {
	service := &fakePurchasingService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases?branch_id=abc", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid branch_id query parameter"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestPurchaseHandler_List_Success(t *testing.T) {
	service := &fakePurchasingService{listItems: []Purchase{{ID: 5, PurchaseNumber: "PO-001"}}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases?branch_id=1", nil)
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
		t.Fatalf("expected one purchase item, got %v", resp.Data)
	}
}

func TestPurchaseHandler_Get_InvalidPurchaseID(t *testing.T) {
	service := &fakePurchasingService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid purchase id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestPurchaseHandler_Get_Success(t *testing.T) {
	service := &fakePurchasingService{purchase: &Purchase{ID: 7, PurchaseNumber: "PO-007"}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/7", nil)
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

func TestPurchaseHandler_Get_NotFound(t *testing.T) {
	service := &fakePurchasingService{getErr: ErrPurchaseNotFound}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases/7", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "7"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"NOT_FOUND","message":"purchase not found"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestPurchaseHandler_List_BranchAccessDenied(t *testing.T) {
	service := &fakePurchasingService{listErr: branches.ErrBranchAccessDenied}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchases?branch_id=1", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}
