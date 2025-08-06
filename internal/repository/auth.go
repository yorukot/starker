package repository

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/stargo/internal/models"
)

// GetRefreshTokenByToken gets a refresh token by token
func GetRefreshTokenByToken(ctx context.Context, db pgx.Tx, token string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, user_agent, ip::text, used_at, created_at FROM refresh_tokens WHERE token = $1
	`
	var refreshToken models.RefreshToken
	err := pgxscan.Get(ctx, db, &refreshToken, query, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

// CreateRefreshToken creates a new refresh token
func CreateRefreshToken(ctx context.Context, db pgx.Tx, refreshToken models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, user_agent, ip, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(ctx,
		query,
		refreshToken.ID,
		refreshToken.UserID,
		refreshToken.Token,
		refreshToken.UserAgent,
		refreshToken.IP,
		refreshToken.UsedAt,
		refreshToken.CreatedAt,
	)
	return err
}

// UpdateRefreshTokenUsedAt updates the used_at of a refresh token
func UpdateRefreshTokenUsedAt(ctx context.Context, db pgx.Tx, refreshToken models.RefreshToken) error {
	query := `
		UPDATE refresh_tokens SET used_at = $1 WHERE id = $2
	`
	_, err := db.Exec(ctx, query, refreshToken.UsedAt, refreshToken.ID)
	return err
}
