package roles

import (
	"net/http"
	"strconv"

	"erp-system/backend/pkg/response"

	"github.com/gorilla/mux"
)

type Handler struct {
	repo *Repository
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
	out := map[string]interface{}{"id": role.ID, "name": role.Name, "description": role.Description, "permissions": perms}
	response.JSONOK(w, out)
}
