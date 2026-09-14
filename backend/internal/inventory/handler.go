package inventory

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

type InventoryService interface {
	CreateInventory(ctx context.Context, productID, branchID, quantity int64) (int64, error)
	AdjustStock(ctx context.Context, inventoryID int64, movementType string, quantityDelta int64, referenceType *string, referenceID *int64) (int64, error)
	List(ctx context.Context, branchID, productID *int64) ([]Inventory, error)
	ListMovements(ctx context.Context, branchID, productID *int64) ([]StockMovement, error)
	GetByID(ctx context.Context, id int64) (*Inventory, error)
	CreateTransfer(ctx context.Context, input CreateTransferInput) (int64, error)
	ListTransfers(ctx context.Context) ([]StockTransfer, error)
	GetTransfer(ctx context.Context, id int64) (*StockTransfer, error)
}

func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var input CreateTransferInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}
	if input.SourceBranchID <= 0 || input.DestinationBranchID <= 0 || input.ProductID <= 0 || input.Quantity <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "source_branch_id, destination_branch_id, product_id, and quantity must be positive"))
		return
	}
	id, err := h.service.CreateTransfer(r.Context(), input)
	if err != nil {
		handleServiceError(w, err, "failed to create stock transfer")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListTransfers(r.Context())
	if err != nil {
		handleServiceError(w, err, "failed to list stock transfers")
		return
	}
	response.JSONOK(w, items)
}

func (h *Handler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	id, err := parseInventoryID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid transfer id"))
		return
	}
	transfer, err := h.service.GetTransfer(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch stock transfer")
		return
	}
	response.JSONOK(w, transfer)
}

type Handler struct {
	service InventoryService
}

func NewHandler(service InventoryService) *Handler {
	return &Handler{service: service}
}

type createInventoryRequest struct {
	ProductID int64 `json:"product_id"`
	BranchID  int64 `json:"branch_id"`
	Quantity  int64 `json:"quantity"`
}

type adjustInventoryRequest struct {
	MovementType  string  `json:"movement_type"`
	QuantityDelta int64   `json:"quantity_delta"`
	ReferenceType *string `json:"reference_type,omitempty"`
	ReferenceID   *int64  `json:"reference_id,omitempty"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	if req.ProductID <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "product_id must be a positive integer"))
		return
	}
	if req.BranchID <= 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch_id must be a positive integer"))
		return
	}
	if req.Quantity < 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "quantity must be greater than or equal to 0"))
		return
	}

	id, err := h.service.CreateInventory(r.Context(), req.ProductID, req.BranchID, req.Quantity)
	if err != nil {
		handleServiceError(w, err, "failed to create inventory")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	inventoryID, err := parseInventoryID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid inventory id"))
		return
	}

	var req adjustInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	if req.MovementType == "" {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "movement_type is required"))
		return
	}
	if req.QuantityDelta == 0 {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "quantity_delta must not be zero"))
		return
	}

	movementID, err := h.service.AdjustStock(r.Context(), inventoryID, req.MovementType, req.QuantityDelta, req.ReferenceType, req.ReferenceID)
	if err != nil {
		handleServiceError(w, err, "failed to adjust inventory")
		return
	}
	response.JSONOK(w, map[string]int64{"movement_id": movementID})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var branchID *int64
	var productID *int64

	if raw := r.URL.Query().Get("branch_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id query parameter"))
			return
		}
		branchID = &parsed
	}
	if raw := r.URL.Query().Get("product_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product_id query parameter"))
			return
		}
		productID = &parsed
	}

	items, err := h.service.List(r.Context(), branchID, productID)
	if err != nil {
		handleServiceError(w, err, "failed to list inventory")
		return
	}
	response.JSONOK(w, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseInventoryID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid inventory id"))
		return
	}

	inventory, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch inventory")
		return
	}
	response.JSONOK(w, inventory)
}

func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	var branchID *int64
	var productID *int64

	if raw := r.URL.Query().Get("branch_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch_id query parameter"))
			return
		}
		branchID = &parsed
	}
	if raw := r.URL.Query().Get("product_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product_id query parameter"))
			return
		}
		productID = &parsed
	}

	items, err := h.service.ListMovements(r.Context(), branchID, productID)
	if err != nil {
		handleServiceError(w, err, "failed to fetch inventory movements")
		return
	}
	response.JSONOK(w, items)
}

func parseInventoryID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing inventory id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrAuthenticationRequired):
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
	case errors.Is(err, ErrInventoryNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "inventory not found"))
	case errors.Is(err, ErrForbidden):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "permission denied"))
	case errors.Is(err, ErrProductNotFound), errors.Is(err, ErrProductInactive):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", err.Error()))
	case errors.Is(err, ErrInvalidMovementType), errors.Is(err, ErrInvalidQuantityDelta):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrInsufficientStock):
		response.JSONError(w, http.StatusConflict, response.NewAPIError(http.StatusConflict, "CONFLICT", "insufficient stock"))
	case errors.Is(err, branches.ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, branches.ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, branches.ErrBranchInactive):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch is inactive"))
	case errors.Is(err, ErrInventoryConflict):
		response.JSONError(w, http.StatusConflict, response.NewAPIError(http.StatusConflict, "CONFLICT", "inventory already exists for product and branch"))
	case errors.Is(err, ErrSameTransferBranch):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrTransferNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "stock transfer not found"))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
