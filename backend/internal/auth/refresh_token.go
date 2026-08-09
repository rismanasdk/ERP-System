package auth

import "time"

type RefreshToken struct {
    ID        int64      `json:"id,omitempty"`
    UserID    int64      `json:"user_id"`
    TokenHash string     `json:"token_hash"`
    FamilyID  string     `json:"family_id"`
    ExpiresAt time.Time  `json:"expires_at"`
    RevokedAt *time.Time `json:"revoked_at,omitempty"`
    CreatedAt time.Time  `json:"created_at,omitempty"`
}
