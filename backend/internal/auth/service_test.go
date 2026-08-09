package auth

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/jwt"

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

func TestRefreshAccessToken_ValidRotation(t *testing.T) {
	oldRefreshToken := "valid-refresh-token-plain"
	hash := hashRefreshToken(oldRefreshToken)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	timeNow := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
		int64(1),
		int64(123),
		hash,
		"family",
		timeNow.Add(24*time.Hour),
		nil,
		timeNow,
	))
	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE token_hash = $2
    `)).WithArgs(sqlmock.AnyArg(), hash).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `)).WithArgs(
		int64(123),
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && len(str) == 64 && isHex(str)
		}),
		"family",
		argMatcher(func(value driver.Value) bool {
			timeValue, ok := value.(time.Time)
			if !ok {
				return false
			}
			nowLower := timeNow.Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := timeNow.Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE id = $1
    `)).WithArgs(int64(123)).WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at", "updated_at"}).AddRow(
		int64(123),
		"test@example.com",
		"hash",
		"Test User",
		timeNow,
		timeNow,
	))

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db))

	token, newRefreshToken, err := authService.RefreshAccessToken(context.Background(), oldRefreshToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected access token")
	}
	if newRefreshToken == "" {
		t.Fatal("expected new refresh token")
	}
	if _, err := jwt.ParseToken(token); err != nil {
		t.Fatalf("expected valid jwt token, got %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(newRefreshToken); err != nil {
		t.Fatalf("expected valid base64 refresh token, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshAccessToken_InvalidTokenNotFound(t *testing.T) {
	oldRefreshToken := "missing-refresh-token"
	hash := hashRefreshToken(oldRefreshToken)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db))

	_, _, err = authService.RefreshAccessToken(context.Background(), oldRefreshToken)
	if err == nil {
		t.Fatal("expected error for missing refresh token")
	}
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshAccessToken_ExpiredToken(t *testing.T) {
	oldRefreshToken := "expired-refresh-token"
	hash := hashRefreshToken(oldRefreshToken)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
		int64(1),
		int64(123),
		hash,
		"family",
		time.Now().Add(-1*time.Hour),
		nil,
		time.Now().Add(-24*time.Hour),
	))
	mock.ExpectRollback()

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db))

	_, _, err = authService.RefreshAccessToken(context.Background(), oldRefreshToken)
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshAccessToken_RevokedToken(t *testing.T) {
	oldRefreshToken := "revoked-refresh-token"
	hash := hashRefreshToken(oldRefreshToken)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
		int64(1),
		int64(123),
		hash,
		"family",
		time.Now().Add(24*time.Hour),
		time.Now(),
		time.Now().Add(-24*time.Hour),
	))
	mock.ExpectRollback()

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db))

	_, _, err = authService.RefreshAccessToken(context.Background(), oldRefreshToken)
	if err == nil {
		t.Fatal("expected error for revoked refresh token")
	}
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshHandler_MalformedRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db))
	handler := NewHandler(authService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	handler.Refresh(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.StatusCode)
	}
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
