package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"erp-system/backend/internal/auth"
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

func TestEventSerialization(t *testing.T) {
	event := Event{Type: "test.event", Data: map[string]any{"id": 123}}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("expected marshal to succeed, got %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("expected unmarshal to succeed, got %v", err)
	}

	if parsed.Type != event.Type {
		t.Fatalf("expected type %q, got %q", event.Type, parsed.Type)
	}

	parsedData, ok := parsed.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected parsed data map, got %T", parsed.Data)
	}

	id, ok := parsedData["id"].(float64)
	if !ok || id != 123 {
		t.Fatalf("expected parsed data id 123, got %v", parsedData["id"])
	}
}

func TestHubPublisherPublishesToHub(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	conn := newFakeConn()
	client := NewClient(hub, conn, &auth.Identity{UserID: 1})
	client.Start()
	time.Sleep(20 * time.Millisecond)
	defer client.Close()

	err := publisher.Publish(context.Background(), Event{Type: "test.event", Data: map[string]any{"value": "ok"}})
	if err != nil {
		t.Fatalf("expected no error publishing event, got %v", err)
	}

	waitForMessages(t, conn, 1, 200*time.Millisecond)
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
		clients[i] = NewClient(hub, conns[i], &auth.Identity{UserID: int64(i + 1)})
		clients[i].Start()
		defer clients[i].Close()
	}
	time.Sleep(20 * time.Millisecond)

	if err := publisher.Publish(context.Background(), Event{Type: "test.event", Data: map[string]any{"value": true}}); err != nil {
		t.Fatalf("expected no error publishing event, got %v", err)
	}

	for _, conn := range conns {
		waitForMessages(t, conn, 1, 200*time.Millisecond)
	}
}

func TestConcurrentPublish(t *testing.T) {
	hub := NewHub()
	publisher := NewHubPublisher(hub)

	conn := newFakeConn()
	client := NewClient(hub, conn, &auth.Identity{UserID: 1})
	client.send = make(chan []byte, 64)
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

	waitForMessages(t, conn, 20, 500*time.Millisecond)
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
