package auth

import (
	"context"
	"database/sql"
	"time"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *RefreshToken) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, token.UserID, token.TokenHash, token.FamilyID, token.ExpiresAt).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *RefreshTokenRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, token *RefreshToken) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
        INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, token.UserID, token.TokenHash, token.FamilyID, token.ExpiresAt).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	token := &RefreshToken{}
	err := r.db.QueryRowContext(ctx, `
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.FamilyID,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return token, nil
}

func (r *RefreshTokenRepository) FindByHashWithTx(ctx context.Context, tx *sql.Tx, tokenHash string) (*RefreshToken, error) {
	token := &RefreshToken{}
	err := tx.QueryRowContext(ctx, `
        SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
        FROM refresh_tokens
        WHERE token_hash = $1
    `, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.FamilyID,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE token_hash = $2
    `, revokedAt, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeWithTx(ctx context.Context, tx *sql.Tx, tokenHash string, revokedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE token_hash = $2
    `, revokedAt, tokenHash)
	return err
}

func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE family_id = $2
    `, revokedAt, familyID)
	return err
}

func (r *RefreshTokenRepository) RevokeFamilyWithTx(ctx context.Context, tx *sql.Tx, familyID string, revokedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
        UPDATE refresh_tokens
        SET revoked_at = $1
        WHERE family_id = $2 AND revoked_at IS NULL
    `, revokedAt, familyID)
	return err
}
