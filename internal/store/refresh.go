package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrTokenInvalid covers unknown, expired and already-revoked tokens.
	ErrTokenInvalid = errors.New("refresh token invalid")
	// ErrTokenReuse is returned when a revoked token is presented again,
	// which means it may have been stolen. All of the user's tokens are revoked.
	ErrTokenReuse = errors.New("refresh token reuse detected")
)

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (s *Store) CreateRefreshToken(ctx context.Context, userID string, hash []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hash, expiresAt)
	return err
}

// RotateRefreshToken atomically revokes the token identified by oldHash and
// stores newHash for the same user. It returns the user the token belonged to.
func (s *Store) RotateRefreshToken(ctx context.Context, oldHash, newHash []byte, newExpiry time.Time) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var (
		id, userID string
		expiresAt  time.Time
		revokedAt  *time.Time
	)
	err = tx.QueryRow(ctx,
		`SELECT id, user_id, expires_at, revoked_at FROM refresh_tokens
		 WHERE token_hash = $1 FOR UPDATE`, oldHash).Scan(&id, &userID, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrTokenInvalid
	}
	if err != nil {
		return "", err
	}

	if revokedAt != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = now()
			 WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "", ErrTokenReuse
	}
	if time.Now().After(expiresAt) {
		return "", ErrTokenInvalid
	}

	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1`, id); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, newHash, newExpiry); err != nil {
		return "", err
	}
	return userID, tx.Commit(ctx)
}

// RevokeUserTokens revokes every active refresh token of a user.
func (s *Store) RevokeUserTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}
