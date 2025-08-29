package dockerutils

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockeryaml"
)

// StopDockerCompose stops the docker compose orchestration in a goroutine with streaming output
func (dh *DockerHandler) StopDockerCompose(ctx context.Context) error {
	// Start Docker stop orchestration in a goroutine for streaming
	go func() {
		// Create a new transaction for the goroutine
		tx, err := repository.StartTransaction(dh.DB, ctx)
		if err != nil {
			zap.L().Error("Failed to begin transaction in StopDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to begin transaction: %w", err)
			return
		}

		defer func() {
			// Rollback transaction if it hasn't been committed
			repository.DeferRollback(tx, ctx)
			dh.StreamChan.DoneChan <- true
		}()

		// Log start of Docker stop orchestration
		dh.StreamChan.LogChan <- core.LogStep("Starting Docker stop orchestration")

		// Get all containers for this service from database
		dh.StreamChan.LogChan <- core.LogStep("Retrieving service containers from database")

		serviceContainers, err := repository.GetServiceContainers(ctx, tx, dh.NamingGenerator.ServiceID())
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to get service containers from database: %v", err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to get service containers from database: %w", err)
			return
		}

		if len(serviceContainers) == 0 {
			dh.StreamChan.LogChan <- core.LogInfo("No containers found to stop")
		} else {
			// Stop containers in reverse dependency order (dependents first, dependencies last)
			dh.StreamChan.LogChan <- core.LogStep("Stopping Docker containers")

			// +-------------------------------------------+
			// |Stop Docker Containers                    |
			// +-------------------------------------------+
			err = dh.StopDockerContainers(ctx, tx, serviceContainers)
			if err != nil {
				dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to stop Docker containers: %v", err))
				dh.StreamChan.FinalError <- fmt.Errorf("failed to stop Docker containers: %w", err)
				return
			}
		}

		// Remove Docker networks
		dh.StreamChan.LogChan <- core.LogStep("Removing Docker networks")

		// +-------------------------------------------+
		// |Remove Docker Networks                     |
		// +-------------------------------------------+
		err = dh.RemoveDockerNetworks(ctx, tx)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove Docker networks: %v", err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to remove Docker networks: %w", err)
			return
		}

		// Commit the transaction on successful completion
		if err := tx.Commit(ctx); err != nil {
			zap.L().Error("Failed to commit transaction in StopDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to commit transaction: %w", err)
			return
		}

		// Docker stop orchestration completed successfully
		dh.StreamChan.LogChan <- core.LogInfo("Docker stop orchestration completed successfully")
	}()

	return nil
}

// StopDockerContainers stops and removes all Docker containers for the service in reverse dependency order
func (dh *DockerHandler) StopDockerContainers(ctx context.Context, tx pgx.Tx, serviceContainers []models.ServiceContainer) error {
	if len(serviceContainers) == 0 {
		dh.StreamChan.LogChan <- core.LogInfo("No containers found to stop")
		return nil
	}

	// Get dependency order for proper stopping sequence (reverse order)
	startupOrder, err := dockeryaml.ResolveDependencyOrder(dh.Project.Services)
	if err != nil {
		// If we can't resolve dependencies, just stop containers as found
		dh.StreamChan.LogChan <- core.LogStep("Unable to resolve dependency order, stopping containers in database order")

		for _, serviceContainer := range serviceContainers {
			if serviceContainer.ContainerID != nil && serviceContainer.State != models.ContainerStateStopped && serviceContainer.State != models.ContainerStateRemoved {
				err := dh.StopDockerContainer(ctx, tx, *serviceContainer.ContainerID, serviceContainer.ContainerName, true)
				if err != nil {
					dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to stop and remove container %s: %v", serviceContainer.ContainerName, err))
					// Continue with other containers even if one fails
					continue
				}
			}
		}
		return nil
	}

	// Stop containers in reverse dependency order (dependents first, dependencies last)
	reverseOrder := make([]string, len(startupOrder))
	for i, serviceName := range startupOrder {
		reverseOrder[len(startupOrder)-1-i] = serviceName
	}

	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Stopping containers in reverse dependency order: %v", reverseOrder))

	// Stop containers in reverse order
	for _, serviceName := range reverseOrder {
		// Find the container for this service
		var targetContainer *models.ServiceContainer
		for _, serviceContainer := range serviceContainers {
			// Match by service name in container name (assuming naming convention)
			if serviceContainer.ContainerName == dh.NamingGenerator.ContainerName(serviceName) {
				targetContainer = &serviceContainer
				break
			}
		}

		if targetContainer == nil {
			dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("No container found for service %s, skipping", serviceName))
			continue
		}

		if targetContainer.ContainerID == nil {
			dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Container %s has no Docker ID, skipping", targetContainer.ContainerName))
			continue
		}

		if targetContainer.State == models.ContainerStateStopped || targetContainer.State == models.ContainerStateRemoved {
			dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Container %s is already stopped, skipping", targetContainer.ContainerName))
			continue
		}

		dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Stopping and removing service: %s", serviceName))

		err := dh.StopDockerContainer(ctx, tx, *targetContainer.ContainerID, targetContainer.ContainerName, true)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to stop and remove container %s: %v", targetContainer.ContainerName, err))
			// Continue with other containers even if one fails
			continue
		}

		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Container %s stopped and removed successfully", serviceName))
	}

	return nil
}
