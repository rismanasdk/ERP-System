package sales

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
	service SaleService
}

type SaleService interface {
	CreateSale(ctx context.Context, input CreateSaleInput) (int64, error)
	ListSales(ctx context.Context, filter SaleFilter) ([]Sale, error)
	GetSale(ctx context.Context, id int64) (*Sale, []SaleItem, error)
	CompleteSale(ctx context.Context, saleID int64) error
	CancelSale(ctx context.Context, saleID int64) error
	ConfirmSale(ctx context.Context, saleID int64) error
	FulfillSale(ctx context.Context, saleID int64, input FulfillSaleInput) (int64, error)
	ListFulfillments(ctx context.Context, saleID int64) ([]SaleFulfillment, error)
	CreatePayment(ctx context.Context, saleID int64, input CreatePaymentInput) (int64, error)
	ListPayments(ctx context.Context, saleID int64) ([]SalesPayment, error)
	GetPaymentSummary(ctx context.Context, saleID int64) (*PaymentSummary, error)
}

func (h *Handler) Payments(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	payments, err := h.service.ListPayments(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to list payments")
		return
	}
	response.JSONOK(w, payments)
}

func (h *Handler) PaymentSummary(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	summary, err := h.service.GetPaymentSummary(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to fetch payment summary")
		return
	}
	response.JSONOK(w, summary)
}

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	var input CreatePaymentInput
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}
	paymentID, err := h.service.CreatePayment(r.Context(), id, input)
	if err != nil {
		h.handleServiceError(w, err, "failed to create payment")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]int64{"id": paymentID})
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	if err = h.service.ConfirmSale(r.Context(), id); err != nil {
		h.handleServiceError(w, err, "failed to confirm sale")
		return
	}
	response.JSONOK(w, map[string]any{"id": id, "status": SaleStatusConfirmed})
}

func (h *Handler) Fulfill(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	var input FulfillSaleInput
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}
	fulfillmentID, err := h.service.FulfillSale(r.Context(), id, input)
	if err != nil {
		h.handleServiceError(w, err, "failed to fulfill sale")
		return
	}
	response.JSONOK(w, map[string]int64{"id": fulfillmentID})
}

func (h *Handler) Fulfillments(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		h.handleServiceError(w, err, "invalid sale id")
		return
	}
	items, err := h.service.ListFulfillments(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to list fulfillments")
		return
	}
	response.JSONOK(w, items)
}

func NewHandler(service SaleService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSaleInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	if req.BranchID <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch_id must be a positive integer"))
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
		if item.UnitPrice < 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "unit_price must be greater than or equal to 0"))
			return
		}
	}

	id, err := h.service.CreateSale(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err, "failed to create sale")
		return
	}

	sale, _, err := h.service.GetSale(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to fetch created sale")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"id": id, "sale_number": sale.SaleNumber})
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

	items, err := h.service.ListSales(r.Context(), SaleFilter{BranchID: branchID})
	if err != nil {
		h.handleServiceError(w, err, "failed to list sales")
		return
	}
	response.JSONOK(w, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid sale id"))
		return
	}

	sale, _, err := h.service.GetSale(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err, "failed to fetch sale")
		return
	}
	response.JSONOK(w, sale)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid sale id"))
		return
	}

	if err := h.service.CompleteSale(r.Context(), id); err != nil {
		h.handleServiceError(w, err, "failed to complete sale")
		return
	}

	response.JSONOK(w, map[string]any{"id": id, "status": SaleStatusCompleted})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseSaleID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid sale id"))
		return
	}

	if err := h.service.CancelSale(r.Context(), id); err != nil {
		h.handleServiceError(w, err, "failed to cancel sale")
		return
	}

	response.JSONOK(w, map[string]any{"id": id, "status": SaleStatusCancelled})
}

func parseSaleID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing sale id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrAuthenticationRequired):
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
	case errors.Is(err, ErrSaleNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "sale not found"))
	case errors.Is(err, ErrForbidden):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
	case errors.Is(err, ErrProductNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", err.Error()))
	case errors.Is(err, ErrProductInactive), errors.Is(err, ErrInventoryNotFound):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, branches.ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, branches.ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, branches.ErrBranchInactive):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch is inactive"))
	case errors.Is(err, ErrSaleAlreadyCompleted), errors.Is(err, ErrSaleAlreadyCancelled), errors.Is(err, ErrSaleHasNoItems), errors.Is(err, ErrInsufficientStock), errors.Is(err, ErrInvalidSaleTransition):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrSaleNotConfirmed), errors.Is(err, ErrFulfillmentExceedsRemaining):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrPaymentSaleNotPayable), errors.Is(err, ErrPaymentOverpayment), errors.Is(err, ErrInvalidPaymentMethod), errors.Is(err, ErrInvalidPaymentAmount):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
