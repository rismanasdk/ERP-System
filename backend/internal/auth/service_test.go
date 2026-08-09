package auth

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"testing"
	"time"

	"erp-system/backend/internal/users"

	"github.com/DATA-DOG/go-sqlmock"
)

type argMatcher func(driver.Value) bool

func (m argMatcher) Match(value driver.Value) bool {
	return m(value)
}

func TestGenerateRefreshTokenData(t *testing.T) {
	rawToken, tokenHash, familyID, err := GenerateRefreshTokenData()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected non-empty raw token")
	}
	if familyID == "" {
		t.Fatal("expected non-empty family ID")
	}
	if tokenHash == "" {
		t.Fatal("expected non-empty token hash")
	}
	if rawToken == tokenHash {
		t.Fatal("token hash should never equal raw token")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil {
		t.Fatalf("expected raw token to be valid base64 URL encoding, got %v", err)
	}
	if len(decoded) != refreshTokenBytes {
		t.Fatalf("expected raw token to decode to %d bytes, got %d", refreshTokenBytes, len(decoded))
	}

	expectedHash := hashRefreshToken(rawToken)
	if tokenHash != expectedHash {
		t.Fatalf("expected token hash %s, got %s", expectedHash, tokenHash)
	}
}

func TestGenerateRefreshTokenData_Unique(t *testing.T) {
	count := 20
	rawSet := make(map[string]struct{}, count)
	hashSet := make(map[string]struct{}, count)
	familySet := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		rawToken, tokenHash, familyID, err := GenerateRefreshTokenData()
		if err != nil {
			t.Fatalf("expected no error on iteration %d, got %v", i, err)
		}
		if _, ok := rawSet[rawToken]; ok {
			t.Fatalf("expected unique raw token on iteration %d", i)
		}
		if _, ok := hashSet[tokenHash]; ok {
			t.Fatalf("expected unique token hash on iteration %d", i)
		}
		if _, ok := familySet[familyID]; ok {
			t.Fatalf("expected unique family ID on iteration %d", i)
		}
		rawSet[rawToken] = struct{}{}
		hashSet[tokenHash] = struct{}{}
		familySet[familyID] = struct{}{}
	}
}

func TestCreateRefreshToken_StoresHashNotPlaintext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	authService := NewService(userRepo, nil, nil, refreshRepo)

	ctx := context.Background()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `)).WithArgs(
		int64(1),
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && len(str) == 64 && isHex(str)
		}),
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && str != ""
		}),
		argMatcher(func(value driver.Value) bool {
			timeValue, ok := value.(time.Time)
			if !ok {
				return false
			}
			nowLower := now.Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := now.Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	rawToken, err := authService.CreateRefreshToken(ctx, 1)
	if err != nil {
		t.Fatalf("expected no error creating refresh token, got %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected a raw refresh token")
	}
	if _, err := base64.RawURLEncoding.DecodeString(rawToken); err != nil {
		t.Fatalf("expected returned refresh token to be valid base64 URL encoding, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
