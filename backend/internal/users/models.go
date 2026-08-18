package users

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name,omitempty"`
	RoleNames    []string  `json:"roles,omitempty"`
	BranchIDs    []int64   `json:"branch_ids,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

type UserFilter struct {
	Search *string
}
