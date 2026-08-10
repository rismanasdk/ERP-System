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
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/realtime"
	"erp-system/backend/internal/roles"
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
	refreshRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(userRepo, roleRepo, permRepo, refreshRepo, auditService)
	authHandler := auth.NewHandler(authService)

	authMiddleware := auth.NewMiddleware(authService)
	realtimeHub := realtime.NewHub()
	realtimeHandler := realtime.NewHandler(realtimeHub, authService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/health", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/auth/login", authHandler.Login).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh).Methods(http.MethodPost)
	router.Handle("/api/v1/users/protected", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(protectedUsersHandler)))).Methods(http.MethodGet)
	router.Handle("/api/v1/ws", authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket))).Methods(http.MethodGet)

	log.Printf("Starting backend on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func protectedUsersHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"message": "permission granted"})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"status": "ok"})
}
