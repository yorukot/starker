package utils

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/compose"
)

// stopComposeServicesWithDependencies stops all services with proper Docker Compose reverse dependency ordering
func stopComposeServicesWithDependencies(ctx context.Context, cfg *DockerServiceConfig, project *types.Project, streamResult *StreamingResult) error {
	streamResult.LogChan <- fmt.Sprintf("Stopping Docker Compose project: %s", project.Name)

	// Step 1: Stop services in reverse dependency order using Docker Compose v2 API
	serviceStopFunc := func(ctx context.Context, serviceName string) error {
		return stopSingleServiceFromProject(ctx, cfg.Client, serviceName, project.Name, streamResult, cfg.Generator)
	}

	// Use Docker Compose v2's InReverseDependencyOrder to stop services properly
	if err := compose.InReverseDependencyOrder(ctx, project, serviceStopFunc); err != nil {
		return fmt.Errorf("failed to stop services in reverse dependency order: %w", err)
	}

	// Step 2: Remove networks (but keep default network if other containers are using it)
	if err := cleanupProjectNetworks(ctx, cfg.Client, project, streamResult, cfg.Generator); err != nil {
		streamResult.LogChan <- fmt.Sprintf("Warning: Failed to cleanup networks: %v", err)
		// Continue execution - network cleanup is not critical
	}

	// Note: We don't remove volumes by default in stop operation (like docker-compose down)
	// Volumes are typically preserved unless explicitly requested with --volumes flag

	streamResult.LogChan <- fmt.Sprintf("Successfully stopped Docker Compose project: %s", project.Name)
	return nil
}
