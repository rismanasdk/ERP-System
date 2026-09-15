package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type UserService interface {
	List(ctx context.Context, filter UserFilter) ([]User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user *User, roleNames []string, branchIDs []int64) (int64, error)
	Update(ctx context.Context, user *User, roleNames []string, branchIDs []int64) error
	Delete(ctx context.Context, id int64) error
}

type Handler struct {
	service UserService
}

func NewHandler(service UserService) *Handler {
	return &Handler{service: service}
}

type createUserRequest struct {
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles,omitempty"`
	BranchIDs []int64  `json:"branch_ids,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
}

type updateUserRequest struct {
	createUserRequest
	ExpectedVersion int64 `json:"expected_version"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	var filter UserFilter
	if search != "" {
		filter.Search = &search
	}

	usersList, err := h.service.List(r.Context(), filter)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list users"))
		return
	}
	response.JSONOK(w, usersList)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid user id"))
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err, "failed to fetch user")
		return
	}
	response.JSONOK(w, user)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user := &User{
		Email:        req.Email,
		PasswordHash: req.Password,
		Name:         req.Name,
		IsActive:     req.IsActive == nil || *req.IsActive,
	}
	id, err := h.service.Create(r.Context(), user, req.Roles, req.BranchIDs)
	if err != nil {
		handleServiceError(w, err, "failed to create user")
		return
	}

	createdUser, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created user"))
		return
	}
	response.JSONOK(w, createdUser)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid user id"))
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user := &User{
		ID:           id,
		Email:        req.Email,
		Version:      req.ExpectedVersion,
		PasswordHash: req.Password,
		Name:         req.Name,
		IsActive:     req.IsActive == nil || *req.IsActive,
	}
	if err := h.service.Update(r.Context(), user, req.Roles, req.BranchIDs); err != nil {
		handleServiceError(w, err, "failed to update user")
		return
	}

	updatedUser, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated user"))
		return
	}
	response.JSONOK(w, updatedUser)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(r)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid user id"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err, "failed to delete user")
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func parseUserID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	rawID, ok := vars["id"]
	if !ok {
		return 0, errors.New("missing user id")
	}
	return strconv.ParseInt(rawID, 10, 64)
}

func handleServiceError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "user not found"))
	case errors.Is(err, ErrUserDuplicateEmail):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "email already exists"))
	case errors.Is(err, ErrUserEmailRequired), errors.Is(err, ErrUserNameRequired), errors.Is(err, ErrUserPasswordNeeded), errors.Is(err, ErrUserRoleNotFound), errors.Is(err, ErrUserBranchNotFound):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	case errors.Is(err, ErrUserAccessDenied), errors.Is(err, ErrUserPrivilege):
		response.JSONError(w, http.StatusForbidden, response.NewAPIError(http.StatusForbidden, "FORBIDDEN", err.Error()))
	case errors.Is(err, ErrUserSelfDelete), errors.Is(err, ErrProtectedUser):
		response.JSONError(w, http.StatusConflict, response.NewAPIError(http.StatusConflict, "PROTECTED_USER", err.Error()))
	case errors.Is(err, ErrUserVersionConflict):
		response.JSONError(w, http.StatusConflict, response.NewAPIError(http.StatusConflict, "CONFLICT", err.Error()))
	case errors.Is(err, ErrUserVersionRequired):
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", err.Error()))
	default:
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message))
	}
}
