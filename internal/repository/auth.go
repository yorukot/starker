package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/stargo/internal/models"
)

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
