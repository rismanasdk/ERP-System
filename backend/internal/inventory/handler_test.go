package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type fakeInventoryService struct {
	createID      int64
	createErr     error
	adjustID      int64
	adjustErr     error
	listItems     []Inventory
	listErr       error
	lastBranchID  *int64
	lastProductID *int64
	getItem       *Inventory
	getErr        error
}

func (s *fakeInventoryService) CreateInventory(ctx context.Context, productID, branchID, quantity int64) (int64, error) {
	return s.createID, s.createErr
}

func (s *fakeInventoryService) AdjustStock(ctx context.Context, inventoryID int64, movementType string, quantityDelta int64, referenceType *string, referenceID *int64) (int64, error) {
	return s.adjustID, s.adjustErr
}

func (s *fakeInventoryService) List(ctx context.Context, branchID, productID *int64) ([]Inventory, error) {
	s.lastBranchID = branchID
	s.lastProductID = productID
	return s.listItems, s.listErr
}

func (s *fakeInventoryService) GetByID(ctx context.Context, id int64) (*Inventory, error) {
	return s.getItem, s.getErr
}

func TestInventoryHandler_Create_InvalidBody(t *testing.T) {
	service := &fakeInventoryService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestInventoryHandler_Create_Success(t *testing.T) {
	service := &fakeInventoryService{createID: 123}
	handler := NewHandler(service)

	body := bytes.NewBufferString(`{"product_id":1,"branch_id":2,"quantity":10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory", body)
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if id, ok := data["id"].(float64); !ok || int64(id) != 123 {
		t.Fatalf("expected id 123, got %v", data["id"])
	}
}

func TestInventoryHandler_Adjust_InvalidInventoryID(t *testing.T) {
	service := &fakeInventoryService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/abc/adjust", bytes.NewBufferString(`{"movement_type":"IN","quantity_delta":5}`))
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Adjust(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid inventory id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestInventoryHandler_Adjust_Success(t *testing.T) {
	service := &fakeInventoryService{adjustID: 456}
	handler := NewHandler(service)

	body := bytes.NewBufferString(`{"movement_type":"IN","quantity_delta":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/1/adjust", body)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handler.Adjust(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if movementID, ok := data["movement_id"].(float64); !ok || int64(movementID) != 456 {
		t.Fatalf("expected movement_id 456, got %v", data["movement_id"])
	}
}

func TestInventoryHandler_List_InvalidBranchIDQuery(t *testing.T) {
	service := &fakeInventoryService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory?branch_id=abc", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid branch_id query parameter"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestInventoryHandler_Get_InvalidInventoryID(t *testing.T) {
	service := &fakeInventoryService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid inventory id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestInventoryHandler_Get_Success(t *testing.T) {
	service := &fakeInventoryService{getItem: &Inventory{ID: 7, ProductID: 1, BranchID: 2, Quantity: 12}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/7", nil)
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
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if id, ok := data["id"].(float64); !ok || int64(id) != 7 {
		t.Fatalf("expected id 7, got %v", data["id"])
	}
}
