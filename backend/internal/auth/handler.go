package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
	AccessToken string `json:"access_token"`
}

type Handler struct {
	authService *Service
}

func NewHandler(authService *Service) *Handler {
	return &Handler{authService: authService}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, err)
		return
	}

	user, _, err := h.authService.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		response.JSONError(w, http.StatusUnauthorized, err)
		return
	}

	refreshToken, err := h.authService.CreateRefreshToken(r.Context(), user.ID)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, err)
		return
	}

	token, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, err)
		return
	}

	response.JSONOK(w, LoginResponse{AccessToken: token, RefreshToken: refreshToken, User: user})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, err)
		return
	}
	if req.RefreshToken == "" {
		response.JSONError(w, http.StatusBadRequest, errors.New("refresh_token is required"))
		return
	}

	token, err := h.authService.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			response.JSONError(w, http.StatusUnauthorized, err)
			return
		}
		response.JSONError(w, http.StatusInternalServerError, err)
		return
	}

	response.JSONOK(w, RefreshResponse{AccessToken: token})
}
