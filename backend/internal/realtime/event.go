package realtime

import "context"

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
