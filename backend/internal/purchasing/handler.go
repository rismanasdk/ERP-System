package purchasing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/internal/branches"
	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type Handler struct {
	service PurchasingService
}

type PurchasingService interface {
	CreatePurchase(ctx context.Context, input CreatePurchaseInput) (int64, error)
	ListPurchases(ctx context.Context, filter PurchaseFilter) ([]Purchase, error)
	GetPurchase(ctx context.Context, id int64) (*Purchase, error)
}

func NewHandler(service PurchasingService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePurchaseInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	if req.BranchID <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch_id must be a positive integer"))
		return
	}
	if req.SupplierID <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "supplier_id must be a positive integer"))
		return
	}
	if len(req.Items) == 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "items must contain at least one item"))
		return
	}

	for _, item := range req.Items {
		if item.ProductID <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "product_id must be a positive integer"))
			return
		}
		if item.Quantity <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "quantity must be greater than zero"))
			return
		}
		if item.UnitCost < 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "unit_cost must be greater than or equal to 0"))
			return
		}
	}

	id, err := h.service.CreatePurchase(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err, "failed to create purchase")
		return
	}

	purchase, err := h.service.GetPurchase(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to fetch created purchase")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"id": id, "purchase_number": purchase.PurchaseNumber})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var branchID *int64
	if raw := r.URL.Query().Get("branch_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id query parameter"))
			return
		}
		branchID = &parsed
	}

	items, err := h.service.ListPurchases(r.Context(), PurchaseFilter{BranchID: branchID})
	if err != nil {
		h.handleServiceError(w, err, "failed to list purchases")
		return
	}
	response.JSONOK(w, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parsePurchaseID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid purchase id"))
		return
	}

	purchase, err := h.service.GetPurchase(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to fetch purchase")
		return
	}
	response.JSONOK(w, purchase)
}

func parsePurchaseID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing purchase id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrAuthenticationRequired):
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
	case errors.Is(err, ErrPurchaseNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "purchase not found"))
	case errors.Is(err, ErrForbidden):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
	case errors.Is(err, ErrSupplierNotFound), errors.Is(err, ErrProductNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", err.Error()))
	case errors.Is(err, ErrSupplierInactive), errors.Is(err, ErrProductInactive), errors.Is(err, ErrDuplicateProduct):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, branches.ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, branches.ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, branches.ErrBranchInactive):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch is inactive"))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
