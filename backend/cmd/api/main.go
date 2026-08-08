package main

import (
	"log"
	"net/http"
	"os"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file loaded")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/health", healthHandler).Methods(http.MethodGet)

	log.Printf("Starting backend on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSONOK(w, map[string]string{"status": "ok"})
}
