package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
)

// GetServices gets all services for a team and project
func GetServices(ctx context.Context, db pgx.Tx, teamID, projectID string) ([]models.Service, error) {
	query := `
		SELECT id, team_id, server_id, project_id, name, description, type, state,
		       container_id, last_deployed_at, created_at, updated_at
		FROM services
		WHERE team_id = $1 AND project_id = $2
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, teamID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var service models.Service
		err := rows.Scan(
			&service.ID,
			&service.TeamID,
			&service.ServerID,
			&service.ProjectID,
			&service.Name,
			&service.Description,
			&service.Type,
			&service.State,
			&service.ContainerID,
			&service.LastDeployedAt,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, rows.Err()
}

// GetServiceByID gets a service by ID, team ID, and project ID
func GetServiceByID(ctx context.Context, db pgx.Tx, serviceID, teamID, projectID string) (*models.Service, error) {
	query := `
		SELECT id, team_id, server_id, project_id, name, description, type, state,
		       container_id, last_deployed_at, created_at, updated_at
		FROM services
		WHERE id = $1 AND team_id = $2 AND project_id = $3
	`
	var service models.Service
	err := db.QueryRow(ctx, query, serviceID, teamID, projectID).Scan(
		&service.ID,
		&service.TeamID,
		&service.ServerID,
		&service.ProjectID,
		&service.Name,
		&service.Description,
		&service.Type,
		&service.State,
		&service.ContainerID,
		&service.LastDeployedAt,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("service not found")
		}
		return nil, err
	}

	return &service, nil
}

// CreateService creates a new service
func CreateService(ctx context.Context, db pgx.Tx, service models.Service) error {
	query := `
		INSERT INTO services (id, team_id, server_id, project_id, name, description, type, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.Exec(ctx, query,
		service.ID,
		service.TeamID,
		service.ServerID,
		service.ProjectID,
		service.Name,
		service.Description,
		service.Type,
		service.State,
		service.CreatedAt,
		service.UpdatedAt,
	)
	return err
}

// UpdateService updates an existing service
func UpdateService(ctx context.Context, db pgx.Tx, service models.Service) error {
	query := `
		UPDATE services
		SET name = $2, description = $3, type = $4, state = $5,
		    container_id = $6, last_deployed_at = $7, updated_at = $8
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query,
		service.ID,
		service.Name,
		service.Description,
		service.Type,
		service.State,
		service.ContainerID,
		service.LastDeployedAt,
		service.UpdatedAt,
	)
	return err
}

// DeleteService deletes a service
func DeleteService(ctx context.Context, db pgx.Tx, serviceID, teamID, projectID string) error {
	query := `DELETE FROM services WHERE id = $1 AND team_id = $2 AND project_id = $3`
	result, err := db.Exec(ctx, query, serviceID, teamID, projectID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("service not found")
	}

	return nil
}

// GetServiceComposeConfig gets the compose config for a service
func GetServiceComposeConfig(ctx context.Context, db pgx.Tx, serviceID string) (*models.ServiceComposeConfig, error) {
	query := `
		SELECT id, service_id, compose_file, compose_file_path, created_at, updated_at
		FROM service_compose_configs
		WHERE service_id = $1
	`
	var config models.ServiceComposeConfig
	err := db.QueryRow(ctx, query, serviceID).Scan(
		&config.ID,
		&config.ServiceID,
		&config.ComposeFile,
		&config.ComposeFilePath,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("compose config not found")
		}
		return nil, err
	}

	return &config, nil
}

// CreateServiceComposeConfig creates a new compose config
func CreateServiceComposeConfig(ctx context.Context, db pgx.Tx, config models.ServiceComposeConfig) error {
	query := `
		INSERT INTO service_compose_configs (id, service_id, compose_file, compose_file_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query,
		config.ID,
		config.ServiceID,
		config.ComposeFile,
		config.ComposeFilePath,
		config.CreatedAt,
		config.UpdatedAt,
	)
	return err
}

// UpdateServiceComposeConfig updates an existing compose config
func UpdateServiceComposeConfig(ctx context.Context, db pgx.Tx, config models.ServiceComposeConfig) error {
	query := `
		UPDATE service_compose_configs
		SET compose_file = $2, compose_file_path = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query,
		config.ID,
		config.ComposeFile,
		config.ComposeFilePath,
		config.UpdatedAt,
	)
	return err
}

// DeleteServiceComposeConfig deletes a compose config
func DeleteServiceComposeConfig(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_compose_configs WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}
