package products

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type ProductService interface {
	List(ctx context.Context, filter ProductFilter) ([]Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	Create(ctx context.Context, product *Product) (int64, error)
	Update(ctx context.Context, product *Product) error
	SoftDelete(ctx context.Context, id int64) error
}

type Handler struct {
	service ProductService
}

func NewHandler(service ProductService) *Handler {
	return &Handler{service: service}
}

type createProductRequest struct {
	SKU           string  `json:"sku"`
	Barcode       *string `json:"barcode,omitempty"`
	Name          string  `json:"name"`
	Description   *string `json:"description,omitempty"`
	Category      *string `json:"category,omitempty"`
	Unit          *string `json:"unit,omitempty"`
	PurchasePrice float64 `json:"purchase_price"`
	SellingPrice  float64 `json:"selling_price"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

type updateProductRequest = createProductRequest

type productResponse struct {
	Product *Product `json:"product"`
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

	products, err := h.service.List(r.Context(), ProductFilter{Search: search, Active: active})
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list products"))
		return
	}
	response.JSONOK(w, products)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseProductID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product id"))
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch product")
		return
	}
	response.JSONOK(w, product)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	product := &Product{
		SKU:           req.SKU,
		Barcode:       req.Barcode,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SellingPrice:  req.SellingPrice,
		IsActive:      true,
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	id, err := h.service.Create(r.Context(), product)
	if err != nil {
		handleServiceError(w, err, "failed to create product")
		return
	}

	createdProduct, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created product"))
		return
	}
	response.JSONOK(w, createdProduct)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseProductID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product id"))
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	product := &Product{
		ID:            id,
		SKU:           req.SKU,
		Barcode:       req.Barcode,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SellingPrice:  req.SellingPrice,
		IsActive:      true,
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := h.service.Update(r.Context(), product); err != nil {
		handleServiceError(w, err, "failed to update product")
		return
	}

	updatedProduct, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated product"))
		return
	}
	response.JSONOK(w, updatedProduct)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseProductID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid product id"))
		return
	}

	if err := h.service.SoftDelete(r.Context(), id); err != nil {
		handleServiceError(w, err, "failed to delete product")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func parseProductID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing product id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrProductNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "product not found"))
	case errors.Is(err, ErrProductDuplicateSKU):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "sku already exists"))
	case errors.Is(err, ErrProductDuplicateBarcode):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "barcode already exists"))
	default:
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", validationErr.Error()))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
