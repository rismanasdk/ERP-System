package audit

import "time"

type AuditLog struct {
	ID          int64          `json:"id"`
	ActorUserID *int64         `json:"actor_user_id,omitempty"`
	Action      string         `json:"action"`
	Resource    string         `json:"resource"`
	ResourceID  *string        `json:"resource_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
}

type AuditLogFilter struct {
	ActorUserID   *int64     `json:"actor_user_id,omitempty"`
	Action        *string    `json:"action,omitempty"`
	Resource      *string    `json:"resource,omitempty"`
	ResourceID    *string    `json:"resource_id,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}
