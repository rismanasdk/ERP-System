package auth

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRefreshTokenRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	now := time.Now().Truncate(time.Second)
	token := &RefreshToken{
		UserID:    1,
		TokenHash: "hash-value",
		FamilyID:  "family-1",
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `)).
		WithArgs(token.UserID, token.TokenHash, token.FamilyID, token.ExpiresAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	id, err := repo.Create(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 10 {
		t.Fatalf("expected id 10, got %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshTokenRepository_FindByHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `)).
		WithArgs("hash-value").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "token_hash", "family_id", "expires_at", "revoked_at", "created_at"}).AddRow(
			11,
			2,
			"hash-value",
			"family-1",
			now,
			nil,
			now,
		))

	token, err := repo.FindByHash(context.Background(), "hash-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("expected token, got nil")
	}
	if token.ID != 11 || token.UserID != 2 || token.FamilyID != "family-1" {
		t.Fatalf("unexpected token values: %+v", token)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	now := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE token_hash = $2
    `)).
		WithArgs(now, "hash-value").
		WillReturnResult(driver.RowsAffected(1))

	if err := repo.Revoke(context.Background(), "hash-value", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRefreshTokenRepository_RevokeFamily(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	now := time.Now()

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE family_id = $2
    `)).
		WithArgs(now, "family-1").
		WillReturnResult(driver.RowsAffected(2))

	if err := repo.RevokeFamily(context.Background(), "family-1", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
