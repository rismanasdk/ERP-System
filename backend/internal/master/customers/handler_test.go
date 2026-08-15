package customers

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

type fakeCustomerService struct {
	createID  int64
	createErr error
	customer  *Customer
	listItems []Customer
	listErr   error
	getErr    error
	updateErr error
	deleteErr error
}

func (s *fakeCustomerService) List(ctx context.Context, filter CustomerFilter) ([]Customer, error) {
	return s.listItems, s.listErr
}

func (s *fakeCustomerService) GetByID(ctx context.Context, id int64) (*Customer, error) {
	return s.customer, s.getErr
}

func (s *fakeCustomerService) Create(ctx context.Context, customer *Customer) (int64, error) {
	return s.createID, s.createErr
}

func (s *fakeCustomerService) Update(ctx context.Context, customer *Customer) error {
	return s.updateErr
}

func (s *fakeCustomerService) SoftDelete(ctx context.Context, id int64) error {
	return s.deleteErr
}

func TestCustomerHandler_Create_InvalidBody(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestCustomerHandler_Create_Success(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{createID: 12, customer: &Customer{ID: 12, Code: "CUST-12", Name: "Alpha"}})
	body := bytes.NewBufferString(`{"code":"CUST-12","name":"Alpha","phone":"0812","email":"alpha@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", body)
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
		t.Fatal("expected customer payload")
	}
}

func TestCustomerHandler_Get_InvalidID(t *testing.T) {
	handler := NewHandler(&fakeCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid customer id"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}
