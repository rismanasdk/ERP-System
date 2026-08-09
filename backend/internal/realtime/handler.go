package realtime

import (
	"net/http"

	"erp-system/backend/internal/auth"
	"erp-system/backend/pkg/logger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type HubClientRegistrar interface {
	Register(*Client)
	Unregister(*Client)
}

type Handler struct {
	hub              HubClientRegistrar
	identityProvider auth.IdentityProvider
}

func NewHandler(hub HubClientRegistrar, identityProvider auth.IdentityProvider) *Handler {
	return &Handler{hub: hub, identityProvider: identityProvider}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	identity := &auth.Identity{UserID: userID}
	if h.identityProvider != nil {
		id, err := h.identityProvider.GetIdentity(r.Context(), userID)
		if err != nil {
			logger.Error("WebSocket identity fetch failed: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if id != nil {
			identity = id
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: %v", err)
		return
	}

	logger.Info("WebSocket connection accepted")

	client := NewClient(h.hub, conn, identity)
	client.Start()
}
