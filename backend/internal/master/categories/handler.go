package categories

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type request struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	ExpectedVersion int64   `json:"expected_version,omitempty"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var active *bool
	if raw := r.URL.Query().Get("active"); raw != "" {
		value := raw == "true"
		active = &value
	}
	search := r.URL.Query().Get("search")
	var query *string
	if search != "" {
		query = &search
	}
	items, err := h.service.List(r.Context(), Filter{Search: query, Active: active})
	if err != nil {
		categoryError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list categories")
		return
	}
	response.JSONOK(w, items)
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := categoryID(r)
	if err != nil {
		categoryError(w, 400, "INVALID_REQUEST", "invalid category id")
		return
	}
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	response.JSONOK(w, item)
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req request
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		categoryError(w, 400, "INVALID_REQUEST", "invalid request body")
		return
	}
	item := &Category{Name: req.Name, Description: req.Description, IsActive: req.IsActive == nil || *req.IsActive}
	id, err := h.service.Create(r.Context(), item)
	if err != nil {
		handleError(w, err)
		return
	}
	created, err := h.service.Get(r.Context(), id)
	if err != nil {
		categoryError(w, 500, "INTERNAL_SERVER_ERROR", "failed to load category")
		return
	}
	response.JSONOK(w, created)
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := categoryID(r)
	if err != nil {
		categoryError(w, 400, "INVALID_REQUEST", "invalid category id")
		return
	}
	var req request
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		categoryError(w, 400, "INVALID_REQUEST", "invalid request body")
		return
	}
	item := &Category{ID: id, Name: req.Name, Description: req.Description, Version: req.ExpectedVersion, IsActive: req.IsActive == nil || *req.IsActive}
	if err := h.service.Update(r.Context(), item); err != nil {
		handleError(w, err)
		return
	}
	updated, err := h.service.Get(r.Context(), id)
	if err != nil {
		categoryError(w, 500, "INTERNAL_SERVER_ERROR", "failed to load category")
		return
	}
	response.JSONOK(w, updated)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := categoryID(r)
	if err != nil {
		categoryError(w, 400, "INVALID_REQUEST", "invalid category id")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}
func categoryID(r *http.Request) (int64, error) { return strconv.ParseInt(mux.Vars(r)["id"], 10, 64) }
func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		categoryError(w, 404, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrDuplicate), errors.Is(err, ErrNameRequired):
		categoryError(w, 400, "INVALID_REQUEST", err.Error())
	case errors.Is(err, ErrVersionRequired):
		categoryError(w, 400, "INVALID_REQUEST", err.Error())
	case errors.Is(err, ErrInUse):
		categoryError(w, 409, "CATEGORY_IN_USE", err.Error())
	case errors.Is(err, ErrVersionConflict):
		categoryError(w, 409, "CONFLICT", "category was modified by another request; reload and try again")
	default:
		categoryError(w, 500, "INTERNAL_SERVER_ERROR", "failed to save category")
	}
}
func categoryError(w http.ResponseWriter, status int, code, message string) {
	response.JSONError(w, status, response.NewAPIError(status, code, message))
}
