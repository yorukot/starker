package dockerutils

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/jackc/pgx/v5"
)

// startComposeServicesWithDependencies starts all services with proper Docker Compose dependency orchestration
func startComposeServicesWithDependencies(ctx context.Context, cfg *DockerServiceConfig, project *types.Project, streamResult *StreamingResult, db pgx.Tx) error {
	streamResult.LogChan <- fmt.Sprintf("Starting Docker Compose project %s with %d services", project.Name, len(project.Services))

	// Step 1: Create networks first
	for networkKey, network := range project.Networks {
		if err := createProjectNetwork(ctx, cfg.Client, networkKey, network, streamResult, cfg.Generator); err != nil {
			return fmt.Errorf("failed to create network %s: %w", network.Name, err)
		}
	}

	// Step 2: Create volumes
	for _, volume := range project.Volumes {
		if err := createProjectVolume(ctx, cfg.Client, volume, streamResult, cfg.Generator); err != nil {
			return fmt.Errorf("failed to create volume %s: %w", volume.Name, err)
		}
	}

	// Step 3: Pull images for all services
	for _, service := range project.Services {
		if err := pullServiceImage(ctx, cfg.Client, service, streamResult); err != nil {
			return fmt.Errorf("failed to pull image for service %s: %w", service.Name, err)
		}
	}

	// Step 4: Build images for services that have build configurations

	for _, service := range project.Services {
		if err := buildServiceImages(ctx, cfg.Client, service, cfg.ServiceID, streamResult, cfg.Generator, cfg.ConnectionPool, cfg.ConnectionID, cfg.Host, cfg.PrivateKeyContent); err != nil {
			return fmt.Errorf("failed to build images: %w", err)
		}
	}

	// Step 5: Start services in proper dependency order using Docker Compose v2 API
	serviceStartFunc := func(ctx context.Context, serviceName string) error {
		// Find the service configuration
		var serviceConfig *types.ServiceConfig
		for i := range project.Services {
			if project.Services[i].Name == serviceName {
				// Create a copy to avoid address-of-map-index issue
				svc := project.Services[i]
				serviceConfig = &svc
				break
			}
		}

		if serviceConfig == nil {
			return fmt.Errorf("service %s not found in project", serviceName)
		}

		return startSingleServiceFromProject(ctx, cfg.Client, serviceConfig, project, streamResult, cfg.Generator, db, cfg.ServiceID)
	}

	// Use Docker Compose v2's InDependencyOrder to start services properly
	if err := compose.InDependencyOrder(ctx, project, serviceStartFunc); err != nil {
		return fmt.Errorf("failed to start services in dependency order: %w", err)
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully started Docker Compose project %s", project.Name)
	return nil
}
