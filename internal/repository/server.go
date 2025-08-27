package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
)

// GetServersByTeamID gets all servers for a team
func GetServersByTeamID(ctx context.Context, db pgx.Tx, teamID string) ([]models.Server, error) {
	query := `
		SELECT id, team_id, name, description, ip, port, "user", private_key_id, created_at, updated_at
		FROM servers
		WHERE team_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var server models.Server
		err := rows.Scan(
			&server.ID,
			&server.TeamID,
			&server.Name,
			&server.Description,
			&server.IP,
			&server.Port,
			&server.User,
			&server.PrivateKeyID,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, rows.Err()
}

// GetServerByID gets a server by ID and team ID
func GetServerByID(ctx context.Context, db pgx.Tx, serverID, teamID string) (*models.Server, error) {
	query := `
		SELECT id, team_id, name, description, ip, port, "user", private_key_id, created_at, updated_at
		FROM servers
		WHERE id = $1 AND team_id = $2
	`
	var server models.Server
	err := db.QueryRow(ctx, query, serverID, teamID).Scan(
		&server.ID,
		&server.TeamID,
		&server.Name,
		&server.Description,
		&server.IP,
		&server.Port,
		&server.User,
		&server.PrivateKeyID,
		&server.CreatedAt,
		&server.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No server found
		}
		return nil, err
	}

	return &server, nil
}

// CreateServer creates a new server
func CreateServer(ctx context.Context, db pgx.Tx, server models.Server) error {
	query := `
		INSERT INTO servers (id, team_id, name, description, ip, port, "user", private_key_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.Exec(ctx, query,
		server.ID,
		server.TeamID,
		server.Name,
		server.Description,
		server.IP,
		server.Port,
		server.User,
		server.PrivateKeyID,
		server.CreatedAt,
		server.UpdatedAt,
	)
	return err
}

// UpdateServerByID updates a server by ID and team ID
func UpdateServerByID(ctx context.Context, db pgx.Tx, serverID, teamID string, server models.Server) error {
	query := `
		UPDATE servers
		SET name = $3, description = $4, ip = $5, port = $6, "user" = $7, private_key_id = $8, updated_at = $9
		WHERE id = $1 AND team_id = $2
	`
	_, err := db.Exec(ctx, query,
		serverID,
		teamID,
		server.Name,
		server.Description,
		server.IP,
		server.Port,
		server.User,
		server.PrivateKeyID,
		server.UpdatedAt,
	)
	return err
}

// UpdateServer updates an existing server
func UpdateServer(ctx context.Context, db pgx.Tx, teamID, serverID string, server models.Server) (*models.Server, error) {
	query := `
		UPDATE servers
		SET name = $1, description = $2, ip = $3, port = $4, "user" = $5, private_key_id = $6, updated_at = $7
		WHERE id = $8 AND team_id = $9
	`
	_, err := db.Exec(ctx, query,
		server.Name,
		server.Description,
		server.IP,
		server.Port,
		server.User,
		server.PrivateKeyID,
		server.UpdatedAt,
		serverID,
		teamID,
	)
	if err != nil {
		return nil, err
	}

	return &server, nil
}

// DeleteServerByID deletes a server by ID and team ID
func DeleteServerByID(ctx context.Context, db pgx.Tx, serverID, teamID string) error {
	query := `
		DELETE FROM servers
		WHERE id = $1 AND team_id = $2
	`
	_, err := db.Exec(ctx, query, serverID, teamID)
	return err
}
