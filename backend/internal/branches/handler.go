package branches

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type BranchService interface {
	List(ctx context.Context, filter BranchFilter) ([]Branch, error)
	GetByID(ctx context.Context, id int64) (*Branch, error)
	Create(ctx context.Context, branch *Branch) (int64, error)
	Update(ctx context.Context, branch *Branch) error
}

type Handler struct {
	service BranchService
}

func NewHandler(service BranchService) *Handler {
	return &Handler{service: service}
}

type createBranchRequest struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	IsActive *bool  `json:"is_active,omitempty"`
}

type updateBranchRequest = createBranchRequest

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var active *bool
	if raw := r.URL.Query().Get("active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid active query parameter"))
			return
		}
		active = &parsed
	}

	branches, err := h.service.List(r.Context(), BranchFilter{Active: active})
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list branches"))
		return
	}
	response.JSONOK(w, branches)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseBranchID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch id"))
		return
	}

	branch, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch branch")
		return
	}
	response.JSONOK(w, branch)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	branch := &Branch{
		Name:     req.Name,
		Code:     req.Code,
		IsActive: true,
	}
	if req.IsActive != nil {
		branch.IsActive = *req.IsActive
	}

	id, err := h.service.Create(r.Context(), branch)
	if err != nil {
		handleServiceError(w, err, "failed to create branch")
		return
	}

	createdBranch, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created branch"))
		return
	}
	response.JSONOK(w, createdBranch)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseBranchID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid branch id"))
		return
	}

	var req updateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	branch := &Branch{
		ID:       id,
		Name:     req.Name,
		Code:     req.Code,
		IsActive: true,
	}
	if req.IsActive != nil {
		branch.IsActive = *req.IsActive
	}

	if err := h.service.Update(r.Context(), branch); err != nil {
		handleServiceError(w, err, "failed to update branch")
		return
	}

	updatedBranch, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated branch"))
		return
	}
	response.JSONOK(w, updatedBranch)
}

func parseBranchID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing branch id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrBranchNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "branch not found"))
	case errors.Is(err, ErrBranchAccessDenied):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", "branch access denied"))
	case errors.Is(err, ErrBranchNameRequired), errors.Is(err, ErrBranchCodeRequired):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrBranchCodeDuplicate):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "branch code already exists"))
	default:
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
