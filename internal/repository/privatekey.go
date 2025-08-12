package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
)

// GetPrivateKeysByTeamID gets all private keys for a team
func GetPrivateKeysByTeamID(ctx context.Context, db pgx.Tx, teamID string) ([]models.PrivateKey, error) {
	query := `
		SELECT id, team_id, name, description, private_key, fingerprint, created_at, updated_at
		FROM private_keys
		WHERE team_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privateKeys []models.PrivateKey
	for rows.Next() {
		var privateKey models.PrivateKey
		err := rows.Scan(
			&privateKey.ID,
			&privateKey.TeamID,
			&privateKey.Name,
			&privateKey.Description,
			&privateKey.PrivateKey,
			&privateKey.Fingerprint,
			&privateKey.CreatedAt,
			&privateKey.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		privateKeys = append(privateKeys, privateKey)
	}

	return privateKeys, rows.Err()
}

// GetPrivateKeyByID gets a private key by ID and team ID
func GetPrivateKeyByID(ctx context.Context, db pgx.Tx, privateKeyID, teamID string) (*models.PrivateKey, error) {
	query := `
		SELECT id, team_id, name, description, private_key, fingerprint, created_at, updated_at
		FROM private_keys
		WHERE id = $1 AND team_id = $2
	`
	var privateKey models.PrivateKey
	err := db.QueryRow(ctx, query, privateKeyID, teamID).Scan(
		&privateKey.ID,
		&privateKey.TeamID,
		&privateKey.Name,
		&privateKey.Description,
		&privateKey.PrivateKey,
		&privateKey.Fingerprint,
		&privateKey.CreatedAt,
		&privateKey.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("private key not found")
		}
		return nil, err
	}

	return &privateKey, nil
}

// CreatePrivateKey creates a new private key
func CreatePrivateKey(ctx context.Context, db pgx.Tx, privateKey models.PrivateKey) error {
	query := `
		INSERT INTO private_keys (id, team_id, name, description, private_key, fingerprint, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := db.Exec(ctx, query,
		privateKey.ID,
		privateKey.TeamID,
		privateKey.Name,
		privateKey.Description,
		privateKey.PrivateKey,
		privateKey.Fingerprint,
		privateKey.CreatedAt,
		privateKey.UpdatedAt,
	)
	return err
}

// DeletePrivateKeyByID deletes a private key by ID and team ID
func DeletePrivateKeyByID(ctx context.Context, db pgx.Tx, privateKeyID, teamID string) error {
	query := `
		DELETE FROM private_keys
		WHERE id = $1 AND team_id = $2
	`
	_, err := db.Exec(ctx, query, privateKeyID, teamID)
	return err
}
