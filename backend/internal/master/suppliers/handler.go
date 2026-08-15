package suppliers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type SupplierService interface {
	List(ctx context.Context, filter SupplierFilter) ([]Supplier, error)
	GetByID(ctx context.Context, id int64) (*Supplier, error)
	Create(ctx context.Context, supplier *Supplier) (int64, error)
	Update(ctx context.Context, supplier *Supplier) error
	SoftDelete(ctx context.Context, id int64) error
}

type Handler struct {
	service SupplierService
}

func NewHandler(service SupplierService) *Handler {
	return &Handler{service: service}
}

type createSupplierRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
	Address  *string `json:"address,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type updateSupplierRequest = createSupplierRequest

type supplierResponse struct {
	Supplier *Supplier `json:"supplier"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("search")
	var search *string
	if q != "" {
		search = &q
	}
	var active *bool
	if raw := r.URL.Query().Get("active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid active query parameter"))
			return
		}
		active = &parsed
	}

	suppliers, err := h.service.List(r.Context(), SupplierFilter{Search: search, Active: active})
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list suppliers"))
		return
	}
	response.JSONOK(w, suppliers)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseSupplierID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid supplier id"))
		return
	}

	supplier, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch supplier")
		return
	}
	response.JSONOK(w, supplier)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	supplier := &Supplier{
		Code:     req.Code,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		IsActive: true,
	}
	if req.IsActive != nil {
		supplier.IsActive = *req.IsActive
	}

	id, err := h.service.Create(r.Context(), supplier)
	if err != nil {
		handleServiceError(w, err, "failed to create supplier")
		return
	}

	createdSupplier, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created supplier"))
		return
	}
	response.JSONOK(w, createdSupplier)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseSupplierID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid supplier id"))
		return
	}

	var req updateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	supplier := &Supplier{
		ID:       id,
		Code:     req.Code,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		IsActive: true,
	}
	if req.IsActive != nil {
		supplier.IsActive = *req.IsActive
	}

	if err := h.service.Update(r.Context(), supplier); err != nil {
		handleServiceError(w, err, "failed to update supplier")
		return
	}

	updatedSupplier, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated supplier"))
		return
	}
	response.JSONOK(w, updatedSupplier)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSupplierID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid supplier id"))
		return
	}

	if err := h.service.SoftDelete(r.Context(), id); err != nil {
		handleServiceError(w, err, "failed to delete supplier")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func parseSupplierID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing supplier id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrSupplierNotFound), errors.Is(err, ErrSupplierDeleted):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "supplier not found"))
	case errors.Is(err, ErrSupplierDuplicateCode):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "supplier code already exists"))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
