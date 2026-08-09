package realtime

import (
	"context"
	"sync"
	"time"

	"erp-system/backend/internal/auth"
	"erp-system/backend/pkg/logger"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	maxMessageSize = 8192
)

type websocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	SetReadLimit(limit int64)
	SetWriteDeadline(t time.Time) error
	Close() error
}

type Client struct {
	hub       HubClientRegistrar
	conn      websocketConn
	send      chan []byte
	identity  *auth.Identity
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

func NewClient(hub HubClientRegistrar, conn websocketConn, identity *auth.Identity) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 16),
		identity: identity,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Client) Identity() *auth.Identity {
	return c.identity
}

func (c *Client) Start() {
	c.hub.Register(c)
	go c.writePump()
	go c.readPump()
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()
		c.hub.Unregister(c)
		_ = c.conn.Close()
	})
}

func (c *Client) readPump() {
	defer c.Close()
	c.conn.SetReadLimit(maxMessageSize)
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			logger.Info("WebSocket client disconnected")
			return
		}
	}
}

func (c *Client) writePump() {
	defer c.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("WebSocket write error: %v", err)
				return
			}
		}
	}
}
