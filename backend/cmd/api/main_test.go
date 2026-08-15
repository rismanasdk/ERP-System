package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/pkg/jwt"

	"github.com/gorilla/mux"
)

type inventoryRouteChecker struct {
	permissions map[int64]map[string]bool
}

func (c *inventoryRouteChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return c.permissions[userID][permission], nil
}

type inventoryRouteService struct {
	items []inventory.Inventory
}

func (s *inventoryRouteService) CreateInventory(context.Context, int64, int64, int64) (int64, error) {
	return 1, nil
}

func (s *inventoryRouteService) AdjustStock(context.Context, int64, string, int64, *string, *int64) (int64, error) {
	return 1, nil
}

func (s *inventoryRouteService) List(ctx context.Context, branchID, productID *int64) ([]inventory.Inventory, error) {
	userID, _ := auth.UserIDFromContext(ctx)
	if branchID != nil && userID != 99 && *branchID != 1 {
		return nil, branches.ErrBranchAccessDenied
	}
	if userID == 99 {
		return s.items, nil
	}
	var accessible []inventory.Inventory
	for _, item := range s.items {
		if item.BranchID == 1 && (branchID == nil || *branchID == item.BranchID) {
			accessible = append(accessible, item)
		}
	}
	return accessible, nil
}

func (s *inventoryRouteService) GetByID(ctx context.Context, id int64) (*inventory.Inventory, error) {
	for _, item := range s.items {
		if item.ID == id {
			userID, _ := auth.UserIDFromContext(ctx)
			if userID != 99 && item.BranchID != 1 {
				return nil, branches.ErrBranchAccessDenied
			}
			return &item, nil
		}
	}
	return nil, inventory.ErrInventoryNotFound
}

func newInventoryTestRouter(service *inventoryRouteService, checker *inventoryRouteChecker) http.Handler {
	router := mux.NewRouter()
	middleware := auth.NewMiddleware(checker)
	registerInventoryRoutes(router, middleware, inventory.NewHandler(service))
	return router
}

func inventoryToken(t *testing.T, userID int64) string {
	t.Helper()
	if err := jwt.Configure("inventory-route-test-secret"); err != nil {
		t.Fatal(err)
	}
	token, err := jwt.GenerateToken(userID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestInventoryRoutes_RequireAuthentication(t *testing.T) {
	router := newInventoryTestRouter(&inventoryRouteService{}, &inventoryRouteChecker{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", res.Code)
	}
}

func TestInventoryRoutes_RequireEndpointPermission(t *testing.T) {
	checker := &inventoryRouteChecker{permissions: map[int64]map[string]bool{
		10: {"inventory.read": true},
	}}
	router := newInventoryTestRouter(&inventoryRouteService{}, checker)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory", nil)
	req.Header.Set("Authorization", "Bearer "+inventoryToken(t, 10))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without inventory.create permission, got %d", res.Code)
	}
}

func TestInventoryRoutes_IsolateBranchInventory(t *testing.T) {
	service := &inventoryRouteService{items: []inventory.Inventory{
		{ID: 1, ProductID: 101, BranchID: 1, Quantity: 10},
		{ID: 2, ProductID: 202, BranchID: 2, Quantity: 20},
	}}
	checker := &inventoryRouteChecker{permissions: map[int64]map[string]bool{
		10: {"inventory.read": true},
		99: {"inventory.read": true},
	}}
	router := newInventoryTestRouter(service, checker)

	tests := []struct {
		name        string
		userID      int64
		expectItems int
	}{
		{name: "branch user", userID: 10, expectItems: 1},
		{name: "super admin", userID: 99, expectItems: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
			req.Header.Set("Authorization", "Bearer "+inventoryToken(t, test.userID))
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", res.Code)
			}
			var body struct {
				Data []inventory.Inventory `json:"data"`
			}
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(body.Data) != test.expectItems {
				t.Fatalf("expected %d inventory items, got %d", test.expectItems, len(body.Data))
			}
		})
	}
}

func TestInventoryRoutes_DenyUnauthorizedBranch(t *testing.T) {
	checker := &inventoryRouteChecker{permissions: map[int64]map[string]bool{
		10: {"inventory.read": true},
	}}
	router := newInventoryTestRouter(&inventoryRouteService{}, checker)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory?branch_id=2", nil)
	req.Header.Set("Authorization", "Bearer "+inventoryToken(t, 10))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unauthorized branch, got %d", res.Code)
	}
	var body map[string]map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN error, got %v", body)
	}
}

func TestCORSPreflightAllowsFrontendOrigins(t *testing.T) {
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.PathPrefix("/api/v1").HandlerFunc(corsPreflightHandler).Methods(http.MethodOptions)
	router.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://192.168.0.9:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://192.168.0.9:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin to echo frontend origin, got %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods header to be present")
	}
}
