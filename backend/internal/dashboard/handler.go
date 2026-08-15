package dashboard

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"erp-system/backend/internal/branches"
	"erp-system/backend/pkg/response"
)

type DashboardService interface {
	GetSummary(ctx context.Context, branchID *int64) (*Summary, error)
}

type Handler struct {
	service DashboardService
}

func NewHandler(service DashboardService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	var branchID *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("branch_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
			return
		}
		branchID = &parsed
	}

	summary, err := h.service.GetSummary(r.Context(), branchID)
	if err != nil {
		handleServiceError(w, err, "failed to fetch dashboard summary")
		return
	}
	response.JSONOK(w, summary)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrAuthenticationRequired):
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
	case errors.Is(err, ErrForbidden):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
	case errors.Is(err, branches.ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, branches.ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, branches.ErrBranchInactive):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch is inactive"))
	default:
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
