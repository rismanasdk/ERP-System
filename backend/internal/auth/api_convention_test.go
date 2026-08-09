package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/response"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAuthHandler_MalformedRequestUsesStructuredError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewHandler(NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db)))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{invalid`))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestAuthHandler_RefreshMissingTokenUsesStructuredError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewHandler(NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db)))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{"refresh_token":""}`))
	w := httptest.NewRecorder()

	handler.Refresh(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"INVALID_REQUEST","message":"refresh_token is required"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestAuthMiddleware_UnauthorizedUsesStructuredError(t *testing.T) {
	middleware := NewMiddleware(&fakePermissionChecker{})
	handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.JSONOK(w, map[string]string{"ok": "true"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/protected", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if got := w.Body.String(); got != `{"error":{"code":"UNAUTHORIZED","message":"authentication required"}}`+"\n" {
		t.Fatalf("unexpected body: %s", got)
	}
}

type fakePermissionChecker struct{}

func (f *fakePermissionChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return false, nil
}
