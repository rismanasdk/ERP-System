package branches

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/pkg/response"
)

type fakeBranchService struct {
	branches []Branch
	branch   *Branch
	err      error
}

func (s *fakeBranchService) List(ctx context.Context, filter BranchFilter) ([]Branch, error) {
	return s.branches, s.err
}

func (s *fakeBranchService) GetByID(ctx context.Context, id int64) (*Branch, error) {
	return s.branch, s.err
}

func (s *fakeBranchService) Create(ctx context.Context, branch *Branch) (int64, error) {
	return 1, s.err
}

func (s *fakeBranchService) Update(ctx context.Context, branch *Branch) error {
	return s.err
}

func TestBranchHandler_List(t *testing.T) {
	service := &fakeBranchService{branches: []Branch{{ID: 1, Name: "HQ", Code: "HQ"}}}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/branches", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp response.SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestBranchHandler_Create_InvalidBody(t *testing.T) {
	service := &fakeBranchService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/branches", bytes.NewBufferString("invalid"))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
