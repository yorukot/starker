package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/generator"
)

// DockerComposeToDatabase synchronizes the Docker Compose project state with the database
// This function ensures that all Docker resources defined in the compose project are properly
// tracked in the database for management and monitoring purposes
func DockerComposeToDatabase(ctx context.Context, project *types.Project, cfg *DockerServiceConfig, db pgx.Tx, streamResult *StreamingResult) error {
	// Step 1: Validate inputs
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("docker service config cannot be nil")
	}
	if cfg.Generator == nil {
		return fmt.Errorf("naming generator cannot be nil")
	}
	if db == nil {
		return fmt.Errorf("database transaction cannot be nil")
	}
	if streamResult == nil {
		return fmt.Errorf("streaming result cannot be nil")
	}

	streamResult.LogChan <- fmt.Sprintf("Starting database synchronization for project %s", project.Name)

	// Step 2: Synchronize networks
	streamResult.LogChan <- fmt.Sprintf("Synchronizing %d networks", len(project.Networks))
	if err := syncNetworksToDatabase(ctx, project.Networks, cfg.ServiceID, db, cfg.Generator, streamResult); err != nil {
		return fmt.Errorf("failed to sync networks: %w", err)
	}

	// Step 3: Synchronize volumes
	streamResult.LogChan <- fmt.Sprintf("Synchronizing %d volumes", len(project.Volumes))
	if err := syncVolumesToDatabase(ctx, project.Volumes, cfg.ServiceID, db, cfg.Generator, streamResult); err != nil {
		return fmt.Errorf("failed to sync volumes: %w", err)
	}

	// Step 4: Synchronize containers (prepare container names from services)
	streamResult.LogChan <- fmt.Sprintf("Synchronizing %d service containers", len(project.Services))
	if err := syncContainersToDatabase(ctx, project.Services, cfg.ServiceID, db, cfg.Generator, streamResult); err != nil {
		return fmt.Errorf("failed to sync containers: %w", err)
	}

	streamResult.LogChan <- fmt.Sprintf("Database synchronization completed successfully for project %s", project.Name)
	return nil
}

func syncContainersToDatabase(ctx context.Context, services types.Services, serviceID string, db pgx.Tx, generator *generator.NamingGenerator, streamResult *StreamingResult) error {
	// Create new container records for each service in the compose file
	for _, service := range services {
		// Generate consistent container name using the naming generator
		containerName := generator.ContainerName(service.Name)

		// Create the container model
		container := models.ServiceContainer{
			ID:            ksuid.New().String(), // Generate unique ID
			ServiceID:     serviceID,
			ContainerID:   "", // Will be populated when actual Docker container is created
			ContainerName: containerName,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Insert the container record
		if err := repository.CreateServiceContainer(ctx, db, container); err != nil {
			return fmt.Errorf("failed to create container record for %s: %w", service.Name, err)
		}

		streamResult.LogChan <- fmt.Sprintf("Container '%s' -> '%s' synchronized", service.Name, containerName)
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully synchronized %d containers", len(services))
	return nil
}

func syncNetworksToDatabase(ctx context.Context, networks map[string]types.NetworkConfig, serviceID string, db pgx.Tx, generator *generator.NamingGenerator, streamResult *StreamingResult) error {
	// Create new network records for each network in the compose file
	for networkKey, networkConfig := range networks {
		// Generate consistent network name using the naming generator
		networkName := generator.NetworkName(networkKey)

		// Create the network model
		network := models.ServiceNetwork{
			ID:          ksuid.New().String(), // Generate unique ID
			ServiceID:   serviceID,
			NetworkID:   nil, // Will be populated when actual Docker network is created
			NetworkName: networkName,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Insert the network record
		if err := repository.CreateServiceNetwork(ctx, db, network); err != nil {
			return fmt.Errorf("failed to create network record for %s: %w", networkKey, err)
		}

		streamResult.LogChan <- fmt.Sprintf("Network '%s' -> '%s' synchronized", networkKey, networkName)

		// Avoid unused variable warning
		_ = networkConfig
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully synchronized %d networks", len(networks))
	return nil
}

func syncVolumesToDatabase(ctx context.Context, volumes map[string]types.VolumeConfig, serviceID string, db pgx.Tx, generator *generator.NamingGenerator, streamResult *StreamingResult) error {
	// Create new volume records for each volume in the compose file
	for volumeKey, volumeConfig := range volumes {
		// Generate consistent volume name using the naming generator
		volumeName := generator.VolumeName(volumeKey)

		// Create the volume model
		volume := models.ServiceVolume{
			ID:         ksuid.New().String(), // Generate unique ID
			ServiceID:  serviceID,
			VolumeID:   nil, // Will be populated when actual Docker volume is created
			VolumeName: volumeName,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Insert the volume record
		if err := repository.CreateServiceVolume(ctx, db, volume); err != nil {
			return fmt.Errorf("failed to create volume record for %s: %w", volumeKey, err)
		}

		streamResult.LogChan <- fmt.Sprintf("Volume '%s' -> '%s' synchronized", volumeKey, volumeName)

		// Avoid unused variable warning
		_ = volumeConfig
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully synchronized %d volumes", len(volumes))
	return nil
}
