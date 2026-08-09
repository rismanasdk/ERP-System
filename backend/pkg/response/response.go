package response

import (
	"encoding/json"
	"errors"
	"net/http"
)

type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func JSONOK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, SuccessResponse{Data: data, Message: "success"})
}

func JSONError(w http.ResponseWriter, status int, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		JSON(w, status, ErrorResponse{Error: ErrorDetail{Code: apiErr.Code, Message: apiErr.Message}})
		return
	}

	var statusErr interface{ StatusCode() int }
	if errors.As(err, &statusErr) {
		JSON(w, status, ErrorResponse{Error: ErrorDetail{Code: http.StatusText(status), Message: err.Error()}})
		return
	}

	JSON(w, status, ErrorResponse{Error: ErrorDetail{Code: http.StatusText(status), Message: "request failed"}})
}
