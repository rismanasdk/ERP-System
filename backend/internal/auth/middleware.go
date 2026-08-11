package auth

import (
	"context"
	"net/http"
	"strings"

	"erp-system/backend/pkg/jwt"
	"erp-system/backend/pkg/response"
)

type contextKey string

const (
	userIDContextKey   contextKey = "userID"
	identityContextKey contextKey = "identity"
)

type Identity struct {
	UserID      int64    `json:"user_id"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok
}

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	identity, ok := ctx.Value(identityContextKey).(*Identity)
	return identity, ok
}

type IdentityProvider interface {
	GetIdentity(ctx context.Context, userID int64) (*Identity, error)
}

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID int64, permission string) (bool, error)
}

type Middleware struct {
	checker PermissionChecker
}

func NewMiddleware(checker PermissionChecker) *Middleware {
	return &Middleware{checker: checker}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
			return
		}

		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token"))
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok || userID == 0 {
				response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
				return
			}

			allowed, err := m.checker.HasPermission(r.Context(), userID, permission)
			if err != nil {
				response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error"))
				return
			}
			if !allowed {
				response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
