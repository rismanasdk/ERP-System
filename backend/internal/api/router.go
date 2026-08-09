package api

import (
	"net/http"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"

	"github.com/gorilla/mux"
)

func NewRouter(authService *auth.Service, userRepo *users.Repository, roleRepo *roles.Repository, permRepo *permissions.Repository) http.Handler {
	r := mux.NewRouter()

	api := r.PathPrefix("/api/v1").Subrouter()

	authHandler := auth.NewHandler(authService)
	authMiddleware := auth.NewMiddleware(authService)

	api.Path("/auth/login").HandlerFunc(authHandler.Login).Methods(http.MethodPost)
	api.Path("/auth/refresh").HandlerFunc(authHandler.Refresh).Methods(http.MethodPost)

	api.Path("/users/protected").Handler(authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"message":"permission granted"},"message":"success"}`))
	}))))

	api.Path("/roles").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":null,"message":"roles endpoint placeholder"}`))
	}).Methods(http.MethodGet)

	api.Path("/permissions").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":null,"message":"permissions endpoint placeholder"}`))
	}).Methods(http.MethodGet)

	return r
}
