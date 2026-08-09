package realtime

import (
	"context"
	"encoding/json"
)

type HubPublisher struct {
	hub *Hub
}

func NewHubPublisher(hub *Hub) *HubPublisher {
	return &HubPublisher{hub: hub}
}

func (p *HubPublisher) Publish(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	p.hub.Broadcast(payload)
	return nil
}
