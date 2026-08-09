package realtime

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventCreation(t *testing.T) {
	event := Event{Type: "test.event", Data: map[string]any{"value": 42}}

	if event.Type != "test.event" {
		t.Fatalf("expected event type test.event, got %s", event.Type)
	}

	data, ok := event.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected event data map, got %T", event.Data)
	}
	if data["value"] != 42 {
		t.Fatalf("expected event data value 42, got %v", data["value"])
	}
}

func TestHubPublisherPublishesToHub(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	conn := newFakeConn()
	client := NewClient(hub, conn, 1)
	client.Start()
	time.Sleep(20 * time.Millisecond)
	defer client.Close()

	err := publisher.Publish(context.Background(), Event{Type: "test.event", Data: map[string]any{"value": "ok"}})
	if err != nil {
		t.Fatalf("expected no error publishing event, got %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if len(conn.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conn.messages))
	}
}

func TestPublisherNoClientsDoesNotError(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	if err := publisher.Publish(context.Background(), Event{Type: "test.event", Data: nil}); err != nil {
		t.Fatalf("expected no error when no clients are connected, got %v", err)
	}
}

func TestMultipleClientsReceiveEvent(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	clients := make([]*Client, 5)
	conns := make([]*fakeConn, 5)
	for i := 0; i < 5; i++ {
		conns[i] = newFakeConn()
		clients[i] = NewClient(hub, conns[i], int64(i+1))
		clients[i].Start()
		defer clients[i].Close()
	}
	time.Sleep(20 * time.Millisecond)

	if err := publisher.Publish(context.Background(), Event{Type: "test.event", Data: map[string]any{"value": true}}); err != nil {
		t.Fatalf("expected no error publishing event, got %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	for i, conn := range conns {
		if len(conn.messages) != 1 {
			t.Fatalf("expected client %d to receive event, got %d messages", i+1, len(conn.messages))
		}
	}
}

func TestConcurrentPublish(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	conn := newFakeConn()
	client := NewClient(hub, conn, 1)
	client.Start()
	time.Sleep(20 * time.Millisecond)
	defer client.Close()

	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = publisher.Publish(context.Background(), Event{Type: "test.event", Data: map[string]any{"index": i}})
		}(i)
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	if len(conn.messages) != 20 {
		t.Fatalf("expected 20 messages, got %d", len(conn.messages))
	}
}

func TestPublishDoesNotChangeBusinessData(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)
	businessData := map[string]any{"count": 0}

	_ = publisher.Publish(context.Background(), Event{Type: "test.event", Data: businessData})

	if businessData["count"] != 0 {
		t.Fatalf("expected business data unchanged, got %v", businessData["count"])
	}
}
