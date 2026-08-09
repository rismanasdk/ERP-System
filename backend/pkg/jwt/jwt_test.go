package jwt

import (
	"testing"
	"time"
)

func TestConfigure_EmptySecret(t *testing.T) {
	err := Configure("")
	if err == nil {
		t.Fatal("expected error when JWT secret is empty")
	}
}

func TestConfigure_TrimsSecret(t *testing.T) {
	err := Configure("  secret-value  ")
	if err != nil {
		t.Fatalf("expected no error configuring secret, got %v", err)
	}

	if len(jwtSecret) == 0 {
		t.Fatal("jwtSecret should be configured")
	}
}

func TestGenerateAndParseToken_WithConfiguredSecret(t *testing.T) {
	if err := Configure("test-secret"); err != nil {
		t.Fatalf("failed to configure JWT secret: %v", err)
	}

	token, err := GenerateToken(42, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected userID 42, got %d", claims.UserID)
	}
}
