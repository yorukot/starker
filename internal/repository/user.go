package repository

import (
	"context"
	"errors"

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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &account, nil
}

// GetAccountByEmail checks if the user already exists
func GetAccountByEmail(ctx context.Context, db pgx.Tx, email string) (*models.Account, error) {
	query := `
		SELECT * FROM accounts WHERE email = $1 AND provider = $2
	`
	var account models.Account
	err := pgxscan.Get(ctx, db, &account, query, email, models.ProviderEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &account, nil
}

// GetUserByEmail checks if the user already exists
func GetUserByEmail(ctx context.Context, db pgx.Tx, email string) (*models.User, error) {
	// This need to get the account first and then join the user
	query := `
		SELECT u.* FROM users u
		JOIN accounts a ON u.id = a.user_id
		WHERE a.email = $1 AND a.provider = $2
	`
	var user models.User
	err := pgxscan.Get(ctx, db, &user, query, email, models.ProviderEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, db pgx.Tx, user models.User) error {
	query := `
		INSERT INTO users (id, password_hash, avatar, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query,
		user.ID,
		user.PasswordHash,
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

// CreateUserAndAccount creates a user and account
func CreateUserAndAccount(ctx context.Context, db pgx.Tx, user models.User, account models.Account) error {
	if err := CreateUser(ctx, db, user); err != nil {
		return err
	}
	if err := CreateAccount(ctx, db, account); err != nil {
		return err
	}
	return nil
}
