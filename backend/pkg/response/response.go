package response

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func JSONOK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

func JSONError(w http.ResponseWriter, status int, err error) {
	JSON(w, status, ErrorResponse{Error: err.Error()})
}
