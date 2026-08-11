package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/jwt"
	"erp-system/backend/pkg/response"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         any    `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService interface {
	Authenticate(ctx context.Context, email, passwordPlain string) (*users.User, []string, error)
	CreateRefreshToken(ctx context.Context, userID int64) (string, error)
	RefreshAccessToken(ctx context.Context, rawRefreshToken string) (string, string, error)
}

type Handler struct {
	authService AuthService
}

func NewHandler(authService AuthService) *Handler {
	return &Handler{authService: authService}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user, _, err := h.authService.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"))
		return
	}

	refreshToken, err := h.authService.CreateRefreshToken(r.Context(), user.ID)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error"))
		return
	}

	token, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error"))
		return
	}

	response.JSONOK(w, LoginResponse{AccessToken: token, RefreshToken: refreshToken, User: user})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}
	if req.RefreshToken == "" {
		response.JSONError(w, http.StatusBadRequest, response.NewAPIError(http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required"))
		return
	}

	token, refreshToken, err := h.authService.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			response.JSONError(w, http.StatusUnauthorized, response.NewAPIError(http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "invalid refresh token"))
			return
		}
		response.JSONError(w, http.StatusInternalServerError, response.NewAPIError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "internal server error"))
		return
	}

	response.JSONOK(w, RefreshResponse{AccessToken: token, RefreshToken: refreshToken})
}
