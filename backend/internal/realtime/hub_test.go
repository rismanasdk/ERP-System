package realtime

import (
	"sync"
	"testing"
	"time"
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

func (f *fakeConn) SetReadLimit(limit int64)           {}
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }
func (f *fakeConn) Close() error {
	close(f.readCh)
	return nil
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	conn := newFakeConn()
	client := NewClient(hub, conn, 1)

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

	client1 := NewClient(hub, conn1, 1)
	client2 := NewClient(hub, conn2, 2)

	client1.Start()
	client2.Start()
	time.Sleep(20 * time.Millisecond)
	defer client1.Close()
	defer client2.Close()

	hub.Broadcast([]byte("hello"))
	time.Sleep(10 * time.Millisecond)

	if len(conn1.messages) != 1 || string(conn1.messages[0]) != "hello" {
		t.Fatalf("expected client1 to receive hello, got %v", conn1.messages)
	}
	if len(conn2.messages) != 1 || string(conn2.messages[0]) != "hello" {
		t.Fatalf("expected client2 to receive hello, got %v", conn2.messages)
	}
}

func TestHubConcurrentRegisterBroadcastUnregister(t *testing.T) {
	hub := NewHub()
	wg := sync.WaitGroup{}
	clients := make([]*Client, 50)

	for i := 0; i < 50; i++ {
		conn := newFakeConn()
		clients[i] = NewClient(hub, conn, int64(i+1))
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

func TestClientCloseCleanup(t *testing.T) {
	hub := NewHub()
	conn := newFakeConn()
	client := NewClient(hub, conn, 1)
	client.Start()

	client.Close()
	time.Sleep(10 * time.Millisecond)

	if got := hub.ClientCount(); got != 0 {
		t.Fatalf("expected 0 clients after close, got %d", got)
	}
}
