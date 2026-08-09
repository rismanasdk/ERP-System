package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONOK_UsesConsistentSuccessEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONOK(recorder, map[string]string{"id": "1"})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %s", got)
	}

	if recorder.Body.String() != `{"data":{"id":"1"},"message":"success"}`+"\n" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestJSONError_UsesConsistentErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONError(recorder, http.StatusBadRequest, NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	if recorder.Body.String() != `{"error":{"code":"INVALID_REQUEST","message":"invalid request body"}}`+"\n" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestJSONError_DoesNotLeakInternalDetails(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONError(recorder, http.StatusInternalServerError, errors.New("sql: no rows in result set"))

	if recorder.Body.String() != `{"error":{"code":"Internal Server Error","message":"request failed"}}`+"\n" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
