package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

const (
	RefreshTokenExpiry      = 30 * 24 * time.Hour
	refreshTokenBytes       = 64
	refreshTokenFamilyBytes = 16
)

var generateRefreshTokenData = defaultGenerateRefreshTokenData

type RefreshToken struct {
	ID        int64      `json:"id,omitempty"`
	UserID    int64      `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	FamilyID  string     `json:"family_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
}

func GenerateRefreshTokenData() (rawToken, tokenHash, familyID string, err error) {
	return generateRefreshTokenData()
}

func defaultGenerateRefreshTokenData() (string, string, string, error) {
	rawBytes := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", "", "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(rawBytes)

	familyBytes := make([]byte, refreshTokenFamilyBytes)
	if _, err := rand.Read(familyBytes); err != nil {
		return "", "", "", err
	}
	familyID := base64.RawURLEncoding.EncodeToString(familyBytes)

	return rawToken, hashRefreshToken(rawToken), familyID, nil
}

func hashRefreshToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}
