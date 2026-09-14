package roles

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type Handler struct {
	repo *Repository
}

type roleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roles, err := h.repo.List(ctx)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to list roles"))
		return
	}

	// attach permissions for each role
	out := []map[string]interface{}{}
	for _, role := range roles {
		perms, err := h.repo.GetPermissionNamesByRoleID(ctx, role.ID)
		if err != nil {
			response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load role permissions"))
			return
		}
		out = append(out, map[string]interface{}{"id": role.ID, "name": role.Name, "description": role.Description, "permissions": perms})
	}
	response.JSONOK(w, out)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_ID", "invalid role id"))
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_ID", "invalid role id"))
		return
	}
	role, err := h.repo.GetByID(ctx, id)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load role"))
		return
	}
	if role == nil {
		response.JSONError(w, http.StatusNotFound, response.NewAPIError(http.StatusNotFound, "NOT_FOUND", "role not found"))
		return
	}
	perms, err := h.repo.GetPermissionNamesByRoleID(ctx, role.ID)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load role permissions"))
		return
	}
	out := map[string]interface{}{"id": role.ID, "name": role.Name, "description": role.Description, "user_count": role.UserCount, "permissions": perms}
	response.JSONOK(w, out)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		roleError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		roleError(w, http.StatusBadRequest, "INVALID_REQUEST", "role name is required")
		return
	}
	id, err := h.repo.Create(r.Context(), &Role{Name: req.Name, Description: req.Description}, req.Permissions)
	if err != nil {
		handleRoleError(w, err)
		return
	}
	role, err := h.repo.GetByID(r.Context(), id)
	if err != nil || role == nil {
		roleError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created role")
		return
	}
	perms, err := h.repo.GetPermissionNamesByRoleID(r.Context(), id)
	if err != nil {
		roleError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created role permissions")
		return
	}
	response.JSONOK(w, map[string]interface{}{"id": role.ID, "name": role.Name, "description": role.Description, "user_count": role.UserCount, "permissions": perms})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := roleID(r)
	if err != nil {
		roleError(w, http.StatusBadRequest, "INVALID_ID", "invalid role id")
		return
	}
	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		roleError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		roleError(w, http.StatusBadRequest, "INVALID_REQUEST", "role name is required")
		return
	}
	if err := h.repo.Update(r.Context(), &Role{ID: id, Name: req.Name, Description: req.Description}, req.Permissions); err != nil {
		handleRoleError(w, err)
		return
	}
	role, err := h.repo.GetByID(r.Context(), id)
	if err != nil || role == nil {
		roleError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated role")
		return
	}
	perms, err := h.repo.GetPermissionNamesByRoleID(r.Context(), id)
	if err != nil {
		roleError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load updated role permissions")
		return
	}
	response.JSONOK(w, map[string]interface{}{"id": role.ID, "name": role.Name, "description": role.Description, "user_count": role.UserCount, "permissions": perms})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := roleID(r)
	if err != nil {
		roleError(w, http.StatusBadRequest, "INVALID_ID", "invalid role id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleRoleError(w, err)
		return
	}
	response.JSONOK(w, map[string]int64{"id": id})
}

func roleID(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
}

func handleRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRoleNotFound):
		roleError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
	case errors.Is(err, ErrProtectedRole):
		roleError(w, http.StatusForbidden, "PROTECTED_ROLE", "the SUPER_ADMIN role cannot be changed")
	case errors.Is(err, ErrRoleHasUsers):
		roleError(w, http.StatusConflict, "ROLE_IN_USE", "role is assigned to users")
	case errors.Is(err, ErrPermissionNotFound):
		roleError(w, http.StatusBadRequest, "INVALID_PERMISSION", err.Error())
	default:
		roleError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to save role")
	}
}

func roleError(w http.ResponseWriter, status int, code, message string) {
	response.JSONError(w, status, response.NewAPIError(status, code, message))
}
