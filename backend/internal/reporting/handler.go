package reporting

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-system/backend/internal/branches"
	"erp-system/backend/pkg/response"
)

type ServiceInterface interface {
	GetSalesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*SalesReport, error)
	GetPurchasesReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*PurchasesReport, error)
	GetInventoryReport(ctx context.Context, branchID *int64, productID *int64) (*InventoryReport, error)
	GetProfitReport(ctx context.Context, startDateRaw, endDateRaw string, branchID *int64) (*ProfitReport, error)
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SalesReport(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	branchID, err := parseOptionalInt64(r.URL.Query().Get("branch_id"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
		return
	}
	report, err := h.service.GetSalesReport(r.Context(), startDate, endDate, branchID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.JSONOK(w, report)
}

func (h *Handler) PurchasesReport(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	branchID, err := parseOptionalInt64(r.URL.Query().Get("branch_id"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
		return
	}
	report, err := h.service.GetPurchasesReport(r.Context(), startDate, endDate, branchID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.JSONOK(w, report)
}

func (h *Handler) InventoryReport(w http.ResponseWriter, r *http.Request) {
	branchID, err := parseOptionalInt64(r.URL.Query().Get("branch_id"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
		return
	}
	productID, err := parseOptionalInt64(r.URL.Query().Get("product_id"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product_id"))
		return
	}
	report, err := h.service.GetInventoryReport(r.Context(), branchID, productID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.JSONOK(w, report)
}

func (h *Handler) ProfitReport(w http.ResponseWriter, r *http.Request) {
	startDate := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endDate := strings.TrimSpace(r.URL.Query().Get("end_date"))
	branchID, err := parseOptionalInt64(r.URL.Query().Get("branch_id"))
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
		return
	}
	report, err := h.service.GetProfitReport(r.Context(), startDate, endDate, branchID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	response.JSONOK(w, report)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAuthenticationRequired):
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
	case errors.Is(err, ErrForbidden):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
	case errors.Is(err, ErrInvalidDateRange):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid date range"))
	case errors.Is(err, ErrInvalidBranchID):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id"))
	case errors.Is(err, ErrInvalidProductID):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product_id"))
	case errors.Is(err, branches.ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, branches.ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, branches.ErrBranchInactive):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch is inactive"))
	default:
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error"))
	}
}

func parseOptionalInt64(raw string) (*int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, err
	}
	return &value, nil
}

func parseDate(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	utc := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	return &utc, nil
}
