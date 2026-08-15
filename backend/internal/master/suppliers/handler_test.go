package suppliers

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

type fakeSupplierService struct {
	createID  int64
	createErr error
	supplier  *Supplier
	listItems []Supplier
	listErr   error
	getErr    error
	updateErr error
	deleteErr error
}

func (s *fakeSupplierService) List(ctx context.Context, filter SupplierFilter) ([]Supplier, error) {
	return s.listItems, s.listErr
}

func (s *fakeSupplierService) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	return s.supplier, s.getErr
}

func (s *fakeSupplierService) Create(ctx context.Context, supplier *Supplier) (int64, error) {
	return s.createID, s.createErr
}

func (s *fakeSupplierService) Update(ctx context.Context, supplier *Supplier) error {
	return s.updateErr
}

func (s *fakeSupplierService) SoftDelete(ctx context.Context, id int64) error {
	return s.deleteErr
}

func TestSupplierHandler_Create_InvalidBody(t *testing.T) {
	handler := NewHandler(&fakeSupplierService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestSupplierHandler_Create_Success(t *testing.T) {
	handler := NewHandler(&fakeSupplierService{createID: 12, supplier: &Supplier{ID: 12, Code: "SUP-12", Name: "Alpha"}})
	body := bytes.NewBufferString(`{"code":"SUP-12","name":"Alpha","phone":"0812","email":"alpha@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppliers", body)
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("expected supplier payload")
	}
}

func TestSupplierHandler_Get_InvalidID(t *testing.T) {
	handler := NewHandler(&fakeSupplierService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid supplier id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}
