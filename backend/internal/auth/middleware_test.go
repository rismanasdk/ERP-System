package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"erp-system/backend/pkg/jwt"
)

type fakeChecker struct {
	allowed map[string]bool
}

func (f *fakeChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return f.allowed[permission], nil
}

func TestAuthenticateMiddleware(t *testing.T) {
	h := NewMiddleware(&fakeChecker{})
	handler := h.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", res.Code)
	}
}

func TestAuthenticateMiddlewareExpiredToken(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}
	token, err := jwt.GenerateToken(1, -time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	h := NewMiddleware(&fakeChecker{})
	handler := h.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", res.Code)
	}
}

func TestRequirePermissionMiddleware(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}
	validToken, err := jwt.GenerateToken(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	check := &fakeChecker{allowed: map[string]bool{"users.read": true}}
	h := NewMiddleware(check)

	handler := h.Authenticate(h.RequirePermission("users.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token with permission, got %d", res.Code)
	}
}

func TestRequirePermissionMiddlewareForbidden(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}
	validToken, err := jwt.GenerateToken(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	check := &fakeChecker{allowed: map[string]bool{"users.read": false}}
	h := NewMiddleware(check)

	handler := h.Authenticate(h.RequirePermission("users.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for valid token without permission, got %d", res.Code)
	}
}
