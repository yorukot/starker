package repository

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
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

// CreateOAuthToken creates a new OAuth token
func CreateOAuthToken(ctx context.Context, db pgx.Tx, oauthToken models.OAuthToken) error {
	query := `
		INSERT INTO oauth_tokens (
			account_id,
			access_token,
			refresh_token,
			expiry,
			token_type,
			provider,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (account_id)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expiry = EXCLUDED.expiry,
			token_type = EXCLUDED.token_type,
			updated_at = EXCLUDED.updated_at
	`

	_, err := db.Exec(ctx,
		query,
		oauthToken.AccountID,
		oauthToken.AccessToken,
		oauthToken.RefreshToken,
		oauthToken.Expiry,
		oauthToken.TokenType,
		oauthToken.Provider,
		oauthToken.CreatedAt,
		oauthToken.UpdatedAt,
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
