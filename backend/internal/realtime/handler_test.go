package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/pkg/jwt"

	"github.com/gorilla/websocket"
)

type allowChecker struct{}

func (a *allowChecker) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return true, nil
}

type fakeIdentityProvider struct {
	identity *auth.Identity
}

func (f *fakeIdentityProvider) GetIdentity(ctx context.Context, userID int64) (*auth.Identity, error) {
	return f.identity, nil
}

type spyHub struct {
	*Hub
	mu           sync.Mutex
	registered   []*Client
	unregistered []*Client
}

func newSpyHub() *spyHub {
	return &spyHub{Hub: NewHub()}
}

func (s *spyHub) Register(client *Client) {
	s.mu.Lock()
	s.registered = append(s.registered, client)
	s.mu.Unlock()
	s.Hub.Register(client)
}

func (s *spyHub) Unregister(client *Client) {
	s.mu.Lock()
	s.unregistered = append(s.unregistered, client)
	s.mu.Unlock()
	s.Hub.Unregister(client)
}

func (s *spyHub) lastRegistered() *Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.registered) == 0 {
		return nil
	}
	return s.registered[len(s.registered)-1]
}

func TestWebSocketAuthenticatedConnection(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}
	token, err := jwt.GenerateToken(42, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	hub := newSpyHub()
	identity := &auth.Identity{UserID: 42, Roles: []string{"sales"}, Permissions: []string{"sales.read"}}
	realtimeHandler := NewHandler(hub, &fakeIdentityProvider{identity: identity})
	authMiddleware := auth.NewMiddleware(&allowChecker{})
	server := httptest.NewServer(authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket)))
	defer server.Close()

	wsURL := url.URL{Scheme: "ws", Host: server.Listener.Addr().String(), Path: "/"}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	if err != nil {
		if resp != nil {
			t.Fatalf("expected websocket upgrade, got status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("expected 1 registered client after connect, got %d", got)
	}

	registered := hub.ClientCount()
	if registered != 1 {
		t.Fatalf("expected one registered client, got %d", registered)
	}

	client := hub.lastRegistered()
	if client == nil {
		t.Fatal("expected a registered client")
	}
	if client.Identity().UserID != 42 {
		t.Fatalf("expected identity user id 42, got %d", client.Identity().UserID)
	}
	if len(client.Identity().Roles) != 1 || client.Identity().Roles[0] != "sales" {
		t.Fatalf("expected identity roles [sales], got %v", client.Identity().Roles)
	}
	if len(client.Identity().Permissions) != 1 || client.Identity().Permissions[0] != "sales.read" {
		t.Fatalf("expected identity permissions [sales.read], got %v", client.Identity().Permissions)
	}

	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Fatalf("failed to write close frame: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close websocket connection: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := hub.ClientCount(); got == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected hub to have 0 clients after disconnect, got %d", got)
	}
}

func TestWebSocketUnauthorizedConnection(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}

	hub := newSpyHub()
	realtimeHandler := NewHandler(hub, nil)
	authMiddleware := auth.NewMiddleware(&allowChecker{})
	server := httptest.NewServer(authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket)))
	defer server.Close()

	wsURL := url.URL{Scheme: "ws", Host: server.Listener.Addr().String(), Path: "/"}
	_, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err == nil {
		t.Fatal("expected websocket dial to fail without Authorization header")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized response, got %v", resp)
	}
}

func TestWebSocketInvalidTokenConnection(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}

	hub := newSpyHub()
	realtimeHandler := NewHandler(hub, nil)
	authMiddleware := auth.NewMiddleware(&allowChecker{})
	server := httptest.NewServer(authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket)))
	defer server.Close()

	wsURL := url.URL{Scheme: "ws", Host: server.Listener.Addr().String(), Path: "/"}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer invalidtoken")
	_, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	if err == nil {
		t.Fatal("expected websocket dial to fail with invalid token")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized response, got %v", resp)
	}
}

func TestWebSocketEventDelivery(t *testing.T) {
	if err := jwt.Configure("test-secret"); err != nil {
		t.Fatal(err)
	}
	token, err := jwt.GenerateToken(7, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	hub := NewHub()
	publisher := NewHubPublisher(hub)
	identity := &auth.Identity{UserID: 7, Roles: []string{"sales"}, Permissions: []string{"sales.read"}}
	realtimeHandler := NewHandler(hub, &fakeIdentityProvider{identity: identity})
	authMiddleware := auth.NewMiddleware(&allowChecker{})
	server := httptest.NewServer(authMiddleware.Authenticate(http.HandlerFunc(realtimeHandler.HandleWebSocket)))
	defer server.Close()

	wsURL := url.URL{Scheme: "ws", Host: server.Listener.Addr().String(), Path: "/"}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	if err != nil {
		if resp != nil {
			t.Fatalf("expected websocket upgrade, got status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close()

	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("expected 1 registered client after connect, got %d", got)
	}

	event := Event{Type: "test.event", Data: map[string]any{"id": 123}}
	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected to receive event message, got %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("expected text message, got %d", messageType)
	}

	var received Event
	if err := json.Unmarshal(message, &received); err != nil {
		t.Fatalf("expected valid json event, got %v", err)
	}
	if received.Type != event.Type {
		t.Fatalf("expected event type %q, got %q", event.Type, received.Type)
	}
	data, ok := received.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data object, got %T", received.Data)
	}
	id, ok := data["id"].(float64)
	if !ok || id != 123 {
		t.Fatalf("expected event data id 123, got %v", data["id"])
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket connection failed: %v", err)
	}
}
