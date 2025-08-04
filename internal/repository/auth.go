package repository

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/stargo/internal/models"
)

// Check if the account already exists
func GetAccountByProviderAndProviderUserID(ctx context.Context, db pgx.Tx, provider models.Provider, providerUserID string) (*models.Account, error) {
	query := `
		SELECT * FROM accounts WHERE provider = $1 AND provider_user_id = $2
	`
	var account models.Account
	err := pgxscan.Get(ctx, db, &account, query, provider, providerUserID)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, db pgx.Tx, user models.User) error {
	query := `
		INSERT INTO users (id, password, avatar, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query,
		user.ID,
		user.Password,
		user.Avatar,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// CreateAccount creates a new account
func CreateAccount(ctx context.Context, db pgx.Tx, account models.Account) error {
	query := `
		INSERT INTO accounts (id, provider, provider_user_id, user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(ctx, query,
		account.ID,
		account.Provider,
		account.ProviderUserID,
		account.UserID,
		account.Email,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
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
