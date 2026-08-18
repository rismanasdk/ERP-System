package main

// hot reload test

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/branches"
	"erp-system/backend/internal/dashboard"
	"erp-system/backend/internal/inventory"
	"erp-system/backend/internal/master/customers"
	"erp-system/backend/internal/master/products"
	"erp-system/backend/internal/master/suppliers"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/purchasing"
	"erp-system/backend/internal/realtime"
	"erp-system/backend/internal/reporting"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/sales"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/database"
	"erp-system/backend/pkg/jwt"
	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("POSTGRES_HOST"),
			os.Getenv("POSTGRES_PORT"),
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("POSTGRES_DB"),
		)
	}

	db, err := database.Connect(connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable must be set and non-empty")
	}
	if err := jwt.Configure(jwtSecret); err != nil {
		log.Fatal(err)
	}

	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	auditService := audit.NewService(auditRepo)
	roleHandler := roles.NewHandler(roleRepo)
	refreshRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(userRepo, roleRepo, permRepo, refreshRepo, auditService)
	authHandler := auth.NewHandler(authService)
	userService := users.NewService(userRepo, roleRepo, auditService)
	userHandler := users.NewHandler(userService)

	branchRepo := branches.NewRepository(db)
	branchService := branches.NewService(branchRepo, authService, auditService)
	branchHandler := branches.NewHandler(branchService)

	dashboardRepo := dashboard.NewRepository(db)
	dashboardSvc := dashboard.NewService(dashboardRepo, branchService, authService, authService)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)

	reportingRepo := reporting.NewRepository(db)
	reportingSvc := reporting.NewService(reportingRepo, branchService, authService, authService)
	reportingHandler := reporting.NewHandler(reportingSvc)

	productRepo := products.NewRepository(db)
	productService := products.NewService(productRepo, auditService)
	productHandler := products.NewHandler(productService)

	customerRepo := customers.NewRepository(db)
	customerService := customers.NewService(customerRepo, auditService)
	customerHandler := customers.NewHandler(customerService)

	supplierRepo := suppliers.NewRepository(db)
	supplierService := suppliers.NewService(supplierRepo, auditService)
	supplierHandler := suppliers.NewHandler(supplierService)

	inventoryRepo := inventory.NewRepository(db)
	inventoryService := inventory.NewService(inventoryRepo, productService, branchService, authService, auditService)
	inventoryHandler := inventory.NewHandler(inventoryService)

	purchaseRepo := purchasing.NewRepository(db)
	purchaseService := purchasing.NewService(purchaseRepo, inventoryRepo, productService, branchService, authService, auditService)
	purchaseHandler := purchasing.NewHandler(purchaseService)

	saleRepo := sales.NewRepository(db)
	saleService := sales.NewService(saleRepo, inventoryRepo, productService, branchService, authService, auditService)
	saleHandler := sales.NewHandler(saleService)

	authMiddleware := auth.NewMiddleware(authService)
	realtimeHub := realtime.NewHub()
	realtimeHandler := realtime.NewHandler(realtimeHub, authService)

	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.PathPrefix("/api/v1").HandlerFunc(corsPreflightHandler).Methods(http.MethodOptions)
	router.HandleFunc("/api/v1/health", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/auth/login", authHandler.Login).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh).Methods(http.MethodPost)
	router.Handle("/api/v1/users/protected", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(protectedUsersHandler)))).Methods(http.MethodGet)
	router.Handle("/api/v1/users", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(userHandler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/users/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(userHandler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/users", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.create")(http.HandlerFunc(userHandler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/users/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.update")(http.HandlerFunc(userHandler.Update)))).Methods(http.MethodPut)
	router.Handle("/api/v1/users/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.delete")(http.HandlerFunc(userHandler.Delete)))).Methods(http.MethodDelete)
	router.Handle("/api/v1/ws", authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket))).Methods(http.MethodGet)
	router.Handle("/api/v1/roles", authMiddleware.Authenticate(authMiddleware.RequirePermission("roles.read")(http.HandlerFunc(roleHandler.List)))).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/roles", corsPreflightHandler).Methods(http.MethodOptions)
	router.Handle("/api/v1/roles/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("roles.read")(http.HandlerFunc(roleHandler.Get)))).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/roles/{id}", corsPreflightHandler).Methods(http.MethodOptions)
	router.Handle("/api/v1/branches", authMiddleware.Authenticate(authMiddleware.RequirePermission("inventory.read")(http.HandlerFunc(branchHandler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/branches/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("inventory.read")(http.HandlerFunc(branchHandler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/branches", authMiddleware.Authenticate(authMiddleware.RequirePermission("inventory.create")(http.HandlerFunc(branchHandler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/branches/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("inventory.adjust")(http.HandlerFunc(branchHandler.Update)))).Methods(http.MethodPut)
	router.Handle("/api/v1/dashboard/summary", authMiddleware.Authenticate(authMiddleware.RequirePermission(dashboard.DashboardReadPermission)(http.HandlerFunc(dashboardHandler.Summary)))).Methods(http.MethodGet)
	router.Handle("/api/v1/reports/sales", authMiddleware.Authenticate(authMiddleware.RequirePermission(reporting.ReportReadPermission)(http.HandlerFunc(reportingHandler.SalesReport)))).Methods(http.MethodGet)
	router.Handle("/api/v1/reports/purchases", authMiddleware.Authenticate(authMiddleware.RequirePermission(reporting.ReportReadPermission)(http.HandlerFunc(reportingHandler.PurchasesReport)))).Methods(http.MethodGet)
	router.Handle("/api/v1/reports/inventory", authMiddleware.Authenticate(authMiddleware.RequirePermission(reporting.ReportReadPermission)(http.HandlerFunc(reportingHandler.InventoryReport)))).Methods(http.MethodGet)
	router.Handle("/api/v1/reports/profit", authMiddleware.Authenticate(authMiddleware.RequirePermission(reporting.ReportReadPermission)(http.HandlerFunc(reportingHandler.ProfitReport)))).Methods(http.MethodGet)
	registerInventoryRoutes(router, authMiddleware, inventoryHandler)
	registerPurchasingRoutes(router, authMiddleware, purchaseHandler)
	registerSalesRoutes(router, authMiddleware, saleHandler)
	registerCustomerRoutes(router, authMiddleware, customerHandler)
	registerSupplierRoutes(router, authMiddleware, supplierHandler)
	router.Handle("/api/v1/products", authMiddleware.Authenticate(authMiddleware.RequirePermission("products.read")(http.HandlerFunc(productHandler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/products/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("products.read")(http.HandlerFunc(productHandler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/products", authMiddleware.Authenticate(authMiddleware.RequirePermission("products.create")(http.HandlerFunc(productHandler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/products/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("products.update")(http.HandlerFunc(productHandler.Update)))).Methods(http.MethodPut)
	router.Handle("/api/v1/products/{id}", authMiddleware.Authenticate(authMiddleware.RequirePermission("products.delete")(http.HandlerFunc(productHandler.Delete)))).Methods(http.MethodDelete)

	log.Printf("Starting backend on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func protectedUsersHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"message": "permission granted"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Set("Vary", "Origin")
		}

		next.ServeHTTP(w, r)
	})
}

func corsPreflightHandler(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" && isAllowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.Header().Set("Vary", "Origin")
	}
	w.WriteHeader(http.StatusNoContent)
}

func isAllowedOrigin(origin string) bool {
	allowed := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://192.168.0.9:5173",
	}
	if configured := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); configured != "" {
		for _, part := range strings.Split(configured, ",") {
			allowed = append(allowed, strings.TrimSpace(part))
		}
	}
	for _, candidate := range allowed {
		if candidate != "" && candidate == origin {
			return true
		}
	}
	return false
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"status": "ok"})
}

func registerInventoryRoutes(router *mux.Router, middleware *auth.Middleware, handler *inventory.Handler) {
	router.Handle("/api/v1/inventory", middleware.Authenticate(middleware.RequirePermission("inventory.read")(http.HandlerFunc(handler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/inventory/{id}", middleware.Authenticate(middleware.RequirePermission("inventory.read")(http.HandlerFunc(handler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/inventory", middleware.Authenticate(middleware.RequirePermission("inventory.create")(http.HandlerFunc(handler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/inventory/{id}/adjust", middleware.Authenticate(middleware.RequirePermission("inventory.adjust")(http.HandlerFunc(handler.Adjust)))).Methods(http.MethodPost)
}

func registerPurchasingRoutes(router *mux.Router, middleware *auth.Middleware, handler *purchasing.Handler) {
	router.Handle("/api/v1/purchases", middleware.Authenticate(middleware.RequirePermission("purchases.create")(http.HandlerFunc(handler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/purchases", middleware.Authenticate(middleware.RequirePermission("purchases.read")(http.HandlerFunc(handler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/purchases/{id}", middleware.Authenticate(middleware.RequirePermission("purchases.read")(http.HandlerFunc(handler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/purchases/{id}/complete", middleware.Authenticate(middleware.RequirePermission(purchasing.PurchaseCompletePermission)(http.HandlerFunc(handler.Complete)))).Methods(http.MethodPost)
	router.Handle("/api/v1/purchases/{id}/cancel", middleware.Authenticate(middleware.RequirePermission(purchasing.PurchaseCancelPermission)(http.HandlerFunc(handler.Cancel)))).Methods(http.MethodPost)
}

func registerSalesRoutes(router *mux.Router, middleware *auth.Middleware, handler *sales.Handler) {
	router.Handle("/api/v1/sales", middleware.Authenticate(middleware.RequirePermission("sales.create")(http.HandlerFunc(handler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/sales", middleware.Authenticate(middleware.RequirePermission("sales.read")(http.HandlerFunc(handler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/sales/{id}", middleware.Authenticate(middleware.RequirePermission("sales.read")(http.HandlerFunc(handler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/sales/{id}/complete", middleware.Authenticate(middleware.RequirePermission(sales.SaleCompletePermission)(http.HandlerFunc(handler.Complete)))).Methods(http.MethodPost)
	router.Handle("/api/v1/sales/{id}/cancel", middleware.Authenticate(middleware.RequirePermission(sales.SaleCancelPermission)(http.HandlerFunc(handler.Cancel)))).Methods(http.MethodPost)
}

func registerCustomerRoutes(router *mux.Router, middleware *auth.Middleware, handler *customers.Handler) {
	router.Handle("/api/v1/customers", middleware.Authenticate(middleware.RequirePermission("customers.read")(http.HandlerFunc(handler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/customers/{id}", middleware.Authenticate(middleware.RequirePermission("customers.read")(http.HandlerFunc(handler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/customers", middleware.Authenticate(middleware.RequirePermission("customers.create")(http.HandlerFunc(handler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/customers/{id}", middleware.Authenticate(middleware.RequirePermission("customers.update")(http.HandlerFunc(handler.Update)))).Methods(http.MethodPut)
	router.Handle("/api/v1/customers/{id}", middleware.Authenticate(middleware.RequirePermission("customers.delete")(http.HandlerFunc(handler.Delete)))).Methods(http.MethodDelete)
}

func registerSupplierRoutes(router *mux.Router, middleware *auth.Middleware, handler *suppliers.Handler) {
	router.Handle("/api/v1/suppliers", middleware.Authenticate(middleware.RequirePermission("suppliers.read")(http.HandlerFunc(handler.List)))).Methods(http.MethodGet)
	router.Handle("/api/v1/suppliers/{id}", middleware.Authenticate(middleware.RequirePermission("suppliers.read")(http.HandlerFunc(handler.Get)))).Methods(http.MethodGet)
	router.Handle("/api/v1/suppliers", middleware.Authenticate(middleware.RequirePermission("suppliers.create")(http.HandlerFunc(handler.Create)))).Methods(http.MethodPost)
	router.Handle("/api/v1/suppliers/{id}", middleware.Authenticate(middleware.RequirePermission("suppliers.update")(http.HandlerFunc(handler.Update)))).Methods(http.MethodPut)
	router.Handle("/api/v1/suppliers/{id}", middleware.Authenticate(middleware.RequirePermission("suppliers.delete")(http.HandlerFunc(handler.Delete)))).Methods(http.MethodDelete)
}
