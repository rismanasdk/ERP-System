package realtime

import (
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
)

type fakeConn struct {
	messages [][]byte
	mu       sync.Mutex
	readCh   chan struct{}
}

func newFakeConn() *fakeConn {
	return &fakeConn{readCh: make(chan struct{})}
}

func (f *fakeConn) ReadMessage() (messageType int, p []byte, err error) {
	<-f.readCh
	return 0, nil, nil
}

func (f *fakeConn) WriteMessage(messageType int, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, append([]byte(nil), data...))
	return nil
}

func (f *fakeConn) MessageCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.messages)
}

func (f *fakeConn) Messages() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	copies := make([][]byte, len(f.messages))
	for i, msg := range f.messages {
		copies[i] = append([]byte(nil), msg...)
	}
	return copies
}

func waitForMessages(t *testing.T, conn *fakeConn, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn.MessageCount() >= expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d messages within %v, got %d", expected, timeout, conn.MessageCount())
}

func (f *fakeConn) SetReadLimit(limit int64)           {}
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }
func (f *fakeConn) Close() error {
	close(f.readCh)
	return nil
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	conn := newFakeConn()
	client := NewClient(hub, conn, &auth.Identity{UserID: 1})

	hub.Register(client)
	if got := hub.ClientCount(); got != 1 {
		t.Fatalf("expected 1 client, got %d", got)
	}

	hub.Unregister(client)
	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected 0 clients, got %d", got)
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	conn1 := newFakeConn()
	conn2 := newFakeConn()

	client1 := NewClient(hub, conn1, &auth.Identity{UserID: 1})
	client2 := NewClient(hub, conn2, &auth.Identity{UserID: 2})
	client1.Start()
	client2.Start()
	time.Sleep(20 * time.Millisecond)
	defer client1.Close()
	defer client2.Close()

	hub.Broadcast([]byte("hello"))
	waitForMessages(t, conn1, 1, 200*time.Millisecond)
	waitForMessages(t, conn2, 1, 200*time.Millisecond)

	conn1Messages := conn1.Messages()
	conn2Messages := conn2.Messages()
	if string(conn1Messages[0]) != "hello" {
		t.Fatalf("expected client1 to receive hello, got %v", conn1Messages)
	}
	if string(conn2Messages[0]) != "hello" {
		t.Fatalf("expected client2 to receive hello, got %v", conn2Messages)
	}
}

func TestHubConcurrentRegisterBroadcastUnregister(t *testing.T) {
	hub := NewHub()
	wg := sync.WaitGroup{}
	clients := make([]*Client, 50)

	for i := 0; i < 50; i++ {
		conn := newFakeConn()
		clients[i] = NewClient(hub, conn, &auth.Identity{UserID: int64(i + 1)})
	}

	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			hub.Register(c)
			wg.Done()
		}(client)
	}

	wg.Wait()
	if got := hub.ClientCount(); got != 50 {
		t.Fatalf("expected 50 clients, got %d", got)
	}

	hub.Broadcast([]byte("broadcast"))
	if got := hub.ClientCount(); got != 50 {
		t.Fatalf("expected 50 clients after broadcast, got %d", got)
	}

	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			hub.Unregister(c)
			wg.Done()
		}(client)
	}

	wg.Wait()
	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected 0 clients after unregister, got %d", got)
	}
}

func TestHubBroadcastDoesNotSendToClosedClient(t *testing.T) {
	hub := NewHub()
	conn1 := newFakeConn()
	conn2 := newFakeConn()

	client1 := NewClient(hub, conn1, &auth.Identity{UserID: 1})
	client2 := NewClient(hub, conn2, &auth.Identity{UserID: 2})
	client1.Start()
	client2.Start()
	time.Sleep(20 * time.Millisecond)

	client1.Close()
	time.Sleep(20 * time.Millisecond)

	hub.Broadcast([]byte("world"))
	time.Sleep(10 * time.Millisecond)

	if conn1.MessageCount() != 0 {
		t.Fatalf("expected closed client to receive 0 messages, got %d", conn1.MessageCount())
	}
	conn2Messages := conn2.Messages()
	if conn2.MessageCount() != 1 || string(conn2Messages[0]) != "world" {
		t.Fatalf("expected active client2 to receive world, got %v", conn2Messages)
	}
}

func TestHubShutdownRemovesClients(t *testing.T) {
	hub := NewHub()
	conn1 := newFakeConn()
	conn2 := newFakeConn()

	client1 := NewClient(hub, conn1, &auth.Identity{UserID: 1})
	client2 := NewClient(hub, conn2, &auth.Identity{UserID: 2})
	client1.Start()
	client2.Start()
	time.Sleep(20 * time.Millisecond)

	hub.Shutdown()
	time.Sleep(20 * time.Millisecond)

	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected 0 clients after shutdown, got %d", got)
	}
}

func TestClientCloseCleanup(t *testing.T) {
	hub := NewHub()
	conn := newFakeConn()
	client := NewClient(hub, conn, &auth.Identity{UserID: 1})

	client.Close()
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected 0 clients after close, got %d", got)
	}
}
