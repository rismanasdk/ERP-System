package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/jwt"
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

func TestAuthHandler_LoginReturnsPermissions(t *testing.T) {
	// Prepare fake auth service that returns a user and permissions
	fake := &fakeAuthService{
		user:  &users.User{ID: 7, Email: "mgr@example.com", Name: "Manager", RoleNames: []string{"MANAGER"}},
		perms: []string{"products.read", "inventory.read", "sales.read", "reports.read"},
		err:   nil,
	}
	// ensure JWT secret configured for token generation
	_ = jwt.Configure("test-secret")
	handler := NewHandler(fake)

	reqBody := `{"email":"mgr@example.com","password":"password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope in response")
	}
	userObj, ok := data["user"].(map[string]any)
	if !ok {
		t.Fatalf("user object missing in response data")
	}
	permsIface, ok := userObj["permissions"].([]any)
	if !ok {
		t.Fatalf("permissions field missing or not array")
	}
	// convert to strings
	var got []string
	for _, p := range permsIface {
		if s, ok := p.(string); ok {
			got = append(got, s)
		}
	}
	if len(got) != len(fake.perms) {
		t.Fatalf("expected %d permissions, got %d: %v", len(fake.perms), len(got), got)
	}
}
