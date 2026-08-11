package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/internal/users"
)

type fakeAuthService struct {
	user  *users.User
	perms []string
	err   error
}

func (s *fakeAuthService) Authenticate(ctx context.Context, email, passwordPlain string) (*users.User, []string, error) {
	return s.user, s.perms, s.err
}

func (s *fakeAuthService) CreateRefreshToken(ctx context.Context, userID int64) (string, error) {
	return "refresh-token", nil
}

func (s *fakeAuthService) RefreshAccessToken(ctx context.Context, rawRefreshToken string) (string, string, error) {
	return "", "", nil
}

func TestAuthHandler_LoginInvalidCredentials(t *testing.T) {
	authService := &fakeAuthService{err: ErrInvalidCredentials}
	handler := NewHandler(authService)

	reqBody := `{"email":"alice@example.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp map[string]map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"]["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %v", resp["error"]["code"])
	}
}

func TestAuthHandler_LoginMissingPassword(t *testing.T) {
	authService := &fakeAuthService{err: ErrInvalidCredentials}
	handler := NewHandler(authService)

	reqBody := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp map[string]map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"]["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %v", resp["error"]["code"])
	}
}
