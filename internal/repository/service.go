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
		SELECT id, service_id, compose_file, created_at, updated_at
		FROM service_compose_configs
		WHERE service_id = $1
	`
	var config models.ServiceComposeConfig
	err := db.QueryRow(ctx, query, serviceID).Scan(
		&config.ID,
		&config.ServiceID,
		&config.ComposeFile,
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
		INSERT INTO service_compose_configs (id, service_id, compose_file, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query,
		config.ID,
		config.ServiceID,
		config.ComposeFile,
		config.CreatedAt,
		config.UpdatedAt,
	)
	return err
}

// UpdateServiceComposeConfig updates an existing compose config
func UpdateServiceComposeConfig(ctx context.Context, db pgx.Tx, config models.ServiceComposeConfig) error {
	query := `
		UPDATE service_compose_configs
		SET compose_file = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query,
		config.ID,
		config.ComposeFile,
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

// +----------------------------------------------+
// | Service Container Functions                  |
// +----------------------------------------------+

// GetServiceContainers gets all containers for a service
func GetServiceContainers(ctx context.Context, db pgx.Tx, serviceID string) ([]models.ServiceContainer, error) {
	query := `
		SELECT id, service_id, container_id, container_name, state, created_at, updated_at
		FROM service_containers
		WHERE service_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []models.ServiceContainer
	for rows.Next() {
		var container models.ServiceContainer
		err := rows.Scan(
			&container.ID,
			&container.ServiceID,
			&container.ContainerID,
			&container.ContainerName,
			&container.State,
			&container.CreatedAt,
			&container.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		containers = append(containers, container)
	}

	return containers, rows.Err()
}

// CreateServiceContainer creates a new service container
func CreateServiceContainer(ctx context.Context, db pgx.Tx, container models.ServiceContainer) error {
	query := `
		INSERT INTO service_containers (id, service_id, container_id, container_name, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(ctx, query,
		container.ID,
		container.ServiceID,
		container.ContainerID,
		container.ContainerName,
		container.State,
		container.CreatedAt,
		container.UpdatedAt,
	)
	return err
}

// DeleteServiceContainers deletes all containers for a service
func DeleteServiceContainers(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_containers WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}

// UpdateServiceContainer updates an existing service container
func UpdateServiceContainer(ctx context.Context, db pgx.Tx, container models.ServiceContainer) error {
	query := `
		UPDATE service_containers 
		SET container_id = $1, container_name = $2, state = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := db.Exec(ctx, query,
		container.ContainerID,
		container.ContainerName,
		container.State,
		container.UpdatedAt,
		container.ID,
	)
	return err
}

// +----------------------------------------------+
// | Service Image Functions                      |
// +----------------------------------------------+

// GetServiceImages gets all images for a service
func GetServiceImages(ctx context.Context, db pgx.Tx, serviceID string) ([]models.ServiceImage, error) {
	query := `
		SELECT id, service_id, image_id, image_name, created_at, updated_at
		FROM service_images
		WHERE service_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []models.ServiceImage
	for rows.Next() {
		var image models.ServiceImage
		err := rows.Scan(
			&image.ID,
			&image.ServiceID,
			&image.ImageID,
			&image.ImageName,
			&image.CreatedAt,
			&image.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	return images, rows.Err()
}

// CreateServiceImage creates a new service image
func CreateServiceImage(ctx context.Context, db pgx.Tx, image models.ServiceImage) error {
	query := `
		INSERT INTO service_images (id, service_id, image_id, image_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query,
		image.ID,
		image.ServiceID,
		image.ImageID,
		image.ImageName,
		image.CreatedAt,
		image.UpdatedAt,
	)
	return err
}

// DeleteServiceImages deletes all images for a service
func DeleteServiceImages(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_images WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}

// +----------------------------------------------+
// | Service Network Functions                    |
// +----------------------------------------------+

// GetServiceNetworks gets all networks for a service
func GetServiceNetworks(ctx context.Context, db pgx.Tx, serviceID string) ([]models.ServiceNetwork, error) {
	query := `
		SELECT id, service_id, network_id, network_name, created_at, updated_at
		FROM service_networks
		WHERE service_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []models.ServiceNetwork
	for rows.Next() {
		var network models.ServiceNetwork
		err := rows.Scan(
			&network.ID,
			&network.ServiceID,
			&network.NetworkID,
			&network.NetworkName,
			&network.CreatedAt,
			&network.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}

	return networks, rows.Err()
}

// CreateServiceNetwork creates a new service network
func CreateServiceNetwork(ctx context.Context, db pgx.Tx, network models.ServiceNetwork) error {
	query := `
		INSERT INTO service_networks (id, service_id, network_id, network_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query,
		network.ID,
		network.ServiceID,
		network.NetworkID,
		network.NetworkName,
		network.CreatedAt,
		network.UpdatedAt,
	)
	return err
}

// DeleteServiceNetworks deletes all networks for a service
func DeleteServiceNetworks(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_networks WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}

// UpdateServiceNetwork updates an existing service network
func UpdateServiceNetwork(ctx context.Context, db pgx.Tx, network models.ServiceNetwork) error {
	query := `
		UPDATE service_networks 
		SET network_id = $1, network_name = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := db.Exec(ctx, query,
		network.NetworkID,
		network.NetworkName,
		network.UpdatedAt,
		network.ID,
	)
	return err
}

// +----------------------------------------------+
// | Service Volume Functions                     |
// +----------------------------------------------+

// GetServiceVolumes gets all volumes for a service
func GetServiceVolumes(ctx context.Context, db pgx.Tx, serviceID string) ([]models.ServiceVolume, error) {
	query := `
		SELECT id, service_id, volume_id, volume_name, created_at, updated_at
		FROM service_volumes
		WHERE service_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var volumes []models.ServiceVolume
	for rows.Next() {
		var volume models.ServiceVolume
		err := rows.Scan(
			&volume.ID,
			&volume.ServiceID,
			&volume.VolumeID,
			&volume.VolumeName,
			&volume.CreatedAt,
			&volume.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, volume)
	}

	return volumes, rows.Err()
}

// CreateServiceVolume creates a new service volume
func CreateServiceVolume(ctx context.Context, db pgx.Tx, volume models.ServiceVolume) error {
	query := `
		INSERT INTO service_volumes (id, service_id, volume_id, volume_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query,
		volume.ID,
		volume.ServiceID,
		volume.VolumeID,
		volume.VolumeName,
		volume.CreatedAt,
		volume.UpdatedAt,
	)
	return err
}

// DeleteServiceVolumes deletes all volumes for a service
func DeleteServiceVolumes(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_volumes WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}

// +----------------------------------------------+
// | Service Source Git Functions                 |
// +----------------------------------------------+

// GetServiceSourceGit gets the git source config for a service
func GetServiceSourceGit(ctx context.Context, db pgx.Tx, serviceID string) (*models.ServiceSourceGit, error) {
	query := `
		SELECT id, service_id, repo_url, branch, auto_deploy, docker_compose_file_path, webhook_secret, created_at, updated_at
		FROM service_source_gits
		WHERE service_id = $1
	`
	var gitSource models.ServiceSourceGit
	err := db.QueryRow(ctx, query, serviceID).Scan(
		&gitSource.ID,
		&gitSource.ServiceID,
		&gitSource.RepoURL,
		&gitSource.Branch,
		&gitSource.AutoDeploy,
		&gitSource.DockerComposeFilePath,
		&gitSource.WebhookSecret,
		&gitSource.CreatedAt,
		&gitSource.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("git source not found")
		}
		return nil, err
	}

	return &gitSource, nil
}

// CreateServiceSourceGit creates a new git source config
func CreateServiceSourceGit(ctx context.Context, db pgx.Tx, gitSource models.ServiceSourceGit) error {
	query := `
		INSERT INTO service_source_gits (id, service_id, repo_url, branch, auto_deploy, docker_compose_file_path, webhook_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := db.Exec(ctx, query,
		gitSource.ID,
		gitSource.ServiceID,
		gitSource.RepoURL,
		gitSource.Branch,
		gitSource.AutoDeploy,
		gitSource.DockerComposeFilePath,
		gitSource.WebhookSecret,
		gitSource.CreatedAt,
		gitSource.UpdatedAt,
	)
	return err
}

// UpdateServiceSourceGit updates an existing git source config
func UpdateServiceSourceGit(ctx context.Context, db pgx.Tx, gitSource models.ServiceSourceGit) error {
	query := `
		UPDATE service_source_gits
		SET repo_url = $2, branch = $3, auto_deploy = $4, docker_compose_file_path = $5, 
		    webhook_secret = $6, updated_at = $7
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query,
		gitSource.ID,
		gitSource.RepoURL,
		gitSource.Branch,
		gitSource.AutoDeploy,
		gitSource.DockerComposeFilePath,
		gitSource.WebhookSecret,
		gitSource.UpdatedAt,
	)
	return err
}

// DeleteServiceSourceGit deletes a git source config
func DeleteServiceSourceGit(ctx context.Context, db pgx.Tx, serviceID string) error {
	query := `DELETE FROM service_source_gits WHERE service_id = $1`
	_, err := db.Exec(ctx, query, serviceID)
	return err
}
