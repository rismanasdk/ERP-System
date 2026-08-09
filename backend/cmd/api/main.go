package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"erp-system/backend/internal/auth"
	"erp-system/backend/internal/permissions"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/database"
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

	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	permRepo := permissions.NewRepository(db)
	authService := auth.NewService(userRepo, roleRepo, permRepo)
	authHandler := auth.NewHandler(authService)

	authMiddleware := auth.NewMiddleware(authService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/health", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/auth/login", authHandler.Login).Methods(http.MethodPost)
	router.Handle("/api/v1/users/protected", authMiddleware.Authenticate(authMiddleware.RequirePermission("users.read")(http.HandlerFunc(protectedUsersHandler)))).Methods(http.MethodGet)

	log.Printf("Starting backend on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func authHandler(h *auth.Handler) http.HandlerFunc {
	return h.Login
}

func protectedUsersHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"message": "permission granted"})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"status": "ok"})
}
