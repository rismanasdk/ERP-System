package customers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type CustomerService interface {
	List(ctx context.Context, filter CustomerFilter) ([]Customer, error)
	GetByID(ctx context.Context, id int64) (*Customer, error)
	Create(ctx context.Context, customer *Customer) (int64, error)
	Update(ctx context.Context, customer *Customer) error
	SoftDelete(ctx context.Context, id int64) error
}

type Handler struct {
	service CustomerService
}

func NewHandler(service CustomerService) *Handler {
	return &Handler{service: service}
}

type createCustomerRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
	Address  *string `json:"address,omitempty"`
	TaxID    *string `json:"tax_id,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type updateCustomerRequest = createCustomerRequest

type customerResponse struct {
	Customer *Customer `json:"customer"`
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

	customers, err := h.service.List(r.Context(), CustomerFilter{Search: search, Active: active})
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list customers"))
		return
	}
	response.JSONOK(w, customers)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseCustomerID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid customer id"))
		return
	}

	customer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch customer")
		return
	}
	response.JSONOK(w, customer)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	customer := &Customer{
		Code:     req.Code,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		TaxID:    req.TaxID,
		IsActive: true,
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	id, err := h.service.Create(r.Context(), customer)
	if err != nil {
		handleServiceError(w, err, "failed to create customer")
		return
	}

	createdCustomer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created customer"))
		return
	}
	response.JSONOK(w, createdCustomer)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseCustomerID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid customer id"))
		return
	}

	var req updateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	customer := &Customer{
		ID:       id,
		Code:     req.Code,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		TaxID:    req.TaxID,
		IsActive: true,
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	if err := h.service.Update(r.Context(), customer); err != nil {
		handleServiceError(w, err, "failed to update customer")
		return
	}

	updatedCustomer, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated customer"))
		return
	}
	response.JSONOK(w, updatedCustomer)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCustomerID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid customer id"))
		return
	}

	if err := h.service.SoftDelete(r.Context(), id); err != nil {
		handleServiceError(w, err, "failed to delete customer")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func parseCustomerID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing customer id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrCustomerNotFound), errors.Is(err, ErrCustomerDeleted):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "customer not found"))
	case errors.Is(err, ErrCustomerDuplicateCode):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "customer code already exists"))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
