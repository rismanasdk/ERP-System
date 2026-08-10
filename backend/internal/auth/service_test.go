package auth

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"erp-system/backend/internal/audit"
	"erp-system/backend/internal/roles"
	"erp-system/backend/internal/users"
	"erp-system/backend/pkg/jwt"
	"erp-system/backend/pkg/password"

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

func TestCreateUser_AuditRecordedWithActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	authService := NewService(userRepo, roleRepo, nil, nil, audit.NewService(auditRepo))

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(42))
	newUser := &users.User{
		Email:        "alice@example.com",
		PasswordHash: "password123",
		Name:         "Alice",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id
    `)).WithArgs(
		"alice@example.com",
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && str != "" && str != "password123"
		}),
		"Alice",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(123)))

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, description
        FROM roles
        WHERE name = $1
    `)).WithArgs("SUPER_ADMIN").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).AddRow(int64(99), "SUPER_ADMIN", "super admin"))

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO user_roles (user_id, role_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `)).WithArgs(int64(123), int64(99)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).WithArgs(
		argMatcher(func(value driver.Value) bool {
			id, ok := value.(int64)
			return ok && id == 42
		}),
		"user.create",
		"user",
		"123",
		metadataArgMatcher(map[string]string{
			"email":        "alice@example.com",
			"name":         "Alice",
			"initial_role": "SUPER_ADMIN",
		}, []string{"password", "password_hash", "access_token", "refresh_token"}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(55)))

	mock.ExpectCommit()

	id, err := authService.CreateUser(ctx, newUser, "SUPER_ADMIN")
	if err != nil {
		t.Fatalf("expected no error creating user, got %v", err)
	}
	if id != 123 {
		t.Fatalf("expected user ID 123, got %d", id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateUser_AuditFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	roleRepo := roles.NewRepository(db)
	auditService := audit.NewService(&failingAuditRepository{})
	authService := NewService(userRepo, roleRepo, nil, nil, auditService)

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(7))
	newUser := &users.User{
		Email:        "bob@example.com",
		PasswordHash: "password123",
		Name:         "Bob",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
        RETURNING id
    `)).WithArgs(
		"bob@example.com",
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && str != "" && str != "password123"
		}),
		"Bob",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(124)))

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, name, description
        FROM roles
        WHERE name = $1
    `)).WithArgs("SUPER_ADMIN").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).AddRow(int64(100), "SUPER_ADMIN", "super admin"))

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO user_roles (user_id, role_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `)).WithArgs(int64(124), int64(100)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectRollback()

	_, err = authService.CreateUser(ctx, newUser, "SUPER_ADMIN")
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if !errors.Is(err, errAuditFailure) {
		t.Fatalf("expected audit failure error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAuthenticate_AuditRecordedWithActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	hashedPassword, err := password.Hash("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userRepo := users.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	authService := NewService(userRepo, nil, nil, nil, audit.NewService(auditRepo))

	ctx := context.Background()
	userID := int64(123)
	email := "alice@example.com"

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE email = $1
    `)).WithArgs(email).WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at", "updated_at"}).AddRow(
		userID,
		email,
		hashedPassword,
		"Alice",
		time.Now(),
		time.Now(),
	))
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT r.name
        FROM roles r
        JOIN user_roles ur ON ur.role_id = r.id
        WHERE ur.user_id = $1
    `)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("admin"))
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT DISTINCT p.name
        FROM permissions p
        JOIN role_permissions rp ON rp.permission_id = p.id
        JOIN user_roles ur ON ur.role_id = rp.role_id
        WHERE ur.user_id = $1
    `)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("users.read"))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).WithArgs(
		argMatcher(func(value driver.Value) bool {
			id, ok := value.(int64)
			return ok && id == userID
		}),
		"auth.login",
		"user",
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && str == fmt.Sprintf("%d", userID)
		}),
		metadataArgMatcher(map[string]string{
			"email": email,
		}, []string{"password", "password_hash", "access_token", "refresh_token", "Authorization", "jwt"}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))

	user, perms, err := authService.Authenticate(ctx, email, "password123")
	if err != nil {
		t.Fatalf("expected no error authenticating, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID != userID {
		t.Fatalf("expected user ID %d, got %d", userID, user.ID)
	}
	if len(perms) != 1 || perms[0] != "users.read" {
		t.Fatalf("expected permissions [users.read], got %v", perms)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAuthenticate_AuditFailureDoesNotFailAuthentication(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	hashedPassword, err := password.Hash("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userRepo := users.NewRepository(db)
	auditService := audit.NewService(&failingAuditRepository{})
	authService := NewService(userRepo, nil, nil, nil, auditService)

	ctx := context.Background()
	userID := int64(123)
	email := "alice@example.com"

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE email = $1
    `)).WithArgs(email).WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at", "updated_at"}).AddRow(
		userID,
		email,
		hashedPassword,
		"Alice",
		time.Now(),
		time.Now(),
	))
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT r.name
        FROM roles r
        JOIN user_roles ur ON ur.role_id = r.id
        WHERE ur.user_id = $1
    `)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("admin"))
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT DISTINCT p.name
        FROM permissions p
        JOIN role_permissions rp ON rp.permission_id = p.id
        JOIN user_roles ur ON ur.role_id = rp.role_id
        WHERE ur.user_id = $1
    `)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("users.read"))

	user, perms, err := authService.Authenticate(ctx, email, "password123")
	if err != nil {
		t.Fatalf("expected authentication to succeed when audit fails, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID != userID {
		t.Fatalf("expected user ID %d, got %d", userID, user.ID)
	}
	if len(perms) != 1 || perms[0] != "users.read" {
		t.Fatalf("expected permissions [users.read], got %v", perms)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshAccessToken_AuditRecordedWithActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	authService := NewService(userRepo, nil, nil, NewRefreshTokenRepository(db), audit.NewService(auditRepo))

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(42))
	oldRefreshToken := "valid-refresh-token-plain"
	hash := hashRefreshToken(oldRefreshToken)
	familyID := "family-1"
	userID := int64(123)
	timeNow := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
		int64(1),
		userID,
		hash,
		familyID,
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
		userID,
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && len(str) == 64 && isHex(str)
		}),
		familyID,
		argMatcher(func(value driver.Value) bool {
			timeValue, ok := value.(time.Time)
			if !ok {
				return false
			}
			nowLower := timeNow.Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := timeNow.Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).WithArgs(
		argMatcher(func(value driver.Value) bool {
			id, ok := value.(int64)
			return ok && id == 42
		}),
		"refresh_token.rotate",
		"refresh_token",
		familyID,
		metadataArgMatcher(map[string]string{
			"user_id": "123",
		}, []string{"password", "password_hash", "access_token", "refresh_token"}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash, name, created_at, updated_at
        FROM users
        WHERE id = $1
    `)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at", "updated_at"}).AddRow(
		userID,
		"test@example.com",
		"hash",
		"Test User",
		timeNow,
		timeNow,
	))

	token, newRefreshToken, err := authService.RefreshAccessToken(ctx, oldRefreshToken)
	if err != nil {
		t.Fatalf("expected no error refreshing access token, got %v", err)
	}
	if token == "" {
		t.Fatal("expected access token")
	}
	if newRefreshToken == "" {
		t.Fatal("expected new refresh token")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshAccessToken_AuditFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	auditService := audit.NewService(&failingAuditRepository{})
	authService := NewService(userRepo, nil, nil, NewRefreshTokenRepository(db), auditService)

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(42))
	oldRefreshToken := "valid-refresh-token-plain"
	hash := hashRefreshToken(oldRefreshToken)
	familyID := "family-1"
	userID := int64(123)
	timeNow := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
		int64(1),
		userID,
		hash,
		familyID,
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
		userID,
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && len(str) == 64 && isHex(str)
		}),
		familyID,
		argMatcher(func(value driver.Value) bool {
			timeValue, ok := value.(time.Time)
			if !ok {
				return false
			}
			nowLower := timeNow.Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := timeNow.Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)))
	mock.ExpectRollback()

	_, _, err = authService.RefreshAccessToken(ctx, oldRefreshToken)
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if !errors.Is(err, errAuditFailure) {
		t.Fatalf("expected audit failure error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

var errAuditFailure = errors.New("audit failure")

type failingAuditRepository struct{}

func (f *failingAuditRepository) Create(ctx context.Context, auditLog audit.AuditLog) (int64, error) {
	return 0, errAuditFailure
}

func (f *failingAuditRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, auditLog audit.AuditLog) (int64, error) {
	return 0, errAuditFailure
}

func metadataArgMatcher(expected map[string]string, forbidden []string) argMatcher {
	return argMatcher(func(value driver.Value) bool {
		var raw []byte
		switch v := value.(type) {
		case []byte:
			raw = v
		case string:
			raw = []byte(v)
		default:
			return false
		}

		var metadata map[string]any
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return false
		}

		for key, expectedValue := range expected {
			actual, ok := metadata[key]
			if !ok {
				return false
			}
			actualStr, ok := fmt.Sprintf("%v", actual), true
			if actualStr != expectedValue {
				return false
			}
		}

		for _, key := range forbidden {
			if _, ok := metadata[key]; ok {
				return false
			}
		}
		return true
	})
}

func TestCreateRefreshToken_StoresHashNotPlaintext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	authService := NewService(userRepo, nil, nil, refreshRepo, nil)

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

func TestCreateRefreshToken_AuditRecordedWithActor(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	auditRepo := audit.NewRepository(db)
	refreshRepo := NewRefreshTokenRepository(db)
	authService := NewService(userRepo, nil, nil, refreshRepo, audit.NewService(auditRepo))

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(42))
	tokenUserID := int64(123)
	var familyID string

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `)).WithArgs(
		tokenUserID,
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && len(str) == 64 && isHex(str)
		}),
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			if !ok || str == "" {
				return false
			}
			familyID = str
			return true
		}),
		argMatcher(func(value driver.Value) bool {
			timeValue, ok := value.(time.Time)
			if !ok {
				return false
			}
			nowLower := time.Now().Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := time.Now().Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO audit_logs (actor_user_id, action, resource, resource_id, metadata)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `)).WithArgs(
		argMatcher(func(value driver.Value) bool {
			id, ok := value.(int64)
			return ok && id == 42
		}),
		"refresh_token.create",
		"refresh_token",
		argMatcher(func(value driver.Value) bool {
			str, ok := value.(string)
			return ok && str == familyID
		}),
		metadataArgMatcher(map[string]string{
			"user_id": "123",
		}, []string{"password", "password_hash", "access_token", "refresh_token"}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))
	mock.ExpectCommit()

	rawToken, err := authService.CreateRefreshToken(ctx, tokenUserID)
	if err != nil {
		t.Fatalf("expected no error creating refresh token, got %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected a raw refresh token")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateRefreshToken_AuditFailureRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	userRepo := users.NewRepository(db)
	auditService := audit.NewService(&failingAuditRepository{})
	refreshRepo := NewRefreshTokenRepository(db)
	authService := NewService(userRepo, nil, nil, refreshRepo, auditService)

	ctx := context.WithValue(context.Background(), userIDContextKey, int64(42))
	tokenUserID := int64(123)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `)).WithArgs(
		tokenUserID,
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
			nowLower := time.Now().Add(RefreshTokenExpiry - 1*time.Minute)
			nowUpper := time.Now().Add(RefreshTokenExpiry + 1*time.Minute)
			return timeValue.After(nowLower) && timeValue.Before(nowUpper)
		}),
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectRollback()

	_, err = authService.CreateRefreshToken(ctx, tokenUserID)
	if err == nil {
		t.Fatal("expected error when audit fails")
	}
	if !errors.Is(err, errAuditFailure) {
		t.Fatalf("expected audit failure error, got %v", err)
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

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db), nil)

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

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db), nil)

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

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db), nil)

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
	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE family_id = $2 AND revoked_at IS NULL
    `)).WithArgs(sqlmock.AnyArg(), "family").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db), nil)

	accessToken, newRefreshToken, err := authService.RefreshAccessToken(context.Background(), oldRefreshToken)
	if err == nil {
		t.Fatal("expected error for revoked refresh token")
	}
	if err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}
	if accessToken != "" {
		t.Fatal("expected no access token for reused refresh token")
	}
	if newRefreshToken != "" {
		t.Fatal("expected no refresh token for reused refresh token")
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

	authService := NewService(users.NewRepository(db), nil, nil, NewRefreshTokenRepository(db), nil)
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
