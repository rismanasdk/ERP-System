package password

import (
	"testing"
)

func TestCompare_CorrectPassword(t *testing.T) {
	hash, err := Hash("password123")
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if err := Compare(hash, "password123"); err != nil {
		t.Fatalf("expected no error for correct password, got %v", err)
	}
}

func TestCompare_WrongPassword(t *testing.T) {
	hash, err := Hash("password123")
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if err := Compare(hash, "wrong"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestCompare_EmptyPassword(t *testing.T) {
	hash, err := Hash("password123")
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if err := Compare(hash, ""); err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
}

func TestCompare_MalformedHash(t *testing.T) {
	if err := Compare("not-a-valid-hash", "password123"); err == nil {
		t.Fatal("expected error for malformed hash, got nil")
	}
}
