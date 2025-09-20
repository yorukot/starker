package dockersync

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

// SyncContainersToDB synchronizes containers defined in compose project with database records
func SyncContainersToDB(ctx context.Context, dbTx pgx.Tx, connPool *connection.ConnectionPool, namingGenerator generator.NamingGenerator, composeProject types.Project) error {
	// Get the service ID from the naming generator
	serviceID := namingGenerator.GetLabels()["starker.service.id"]

	zap.L().Debug("Starting container sync to database", zap.String("serviceID", serviceID), zap.Int("composeServices", len(composeProject.Services)))

	// Get all existing containers from the database for this service
	existingContainers, err := repository.GetServiceContainers(ctx, dbTx, serviceID)
	if err != nil {
		zap.L().Error("Failed to get existing containers from database", zap.Error(err), zap.String("serviceID", serviceID))
		return fmt.Errorf("failed to get existing containers: %w", err)
	}

	// Create a map of existing container names for efficient lookup
	existingContainerMap := make(map[string]*models.ServiceContainer)
	for i := range existingContainers {
		existingContainerMap[existingContainers[i].ContainerName] = &existingContainers[i]
	}

	// Create a map to track which containers are present in the compose file
	composeContainerNames := make(map[string]bool)

	// Process each service in the compose project
	for _, service := range composeProject.Services {
		containerName := namingGenerator.ContainerName(service.Name)
		composeContainerNames[containerName] = true

		// Check if this container already exists in the database
		if _, exists := existingContainerMap[containerName]; !exists {
			// Container doesn't exist in database, create it
			now := time.Now()
			newContainer := models.ServiceContainer{
				ID:            ksuid.New().String(),
				ServiceID:     serviceID,
				ContainerID:   nil, // Will be populated when container is actually created
				ContainerName: containerName,
				State:         models.ContainerStateStopped, // Default state - will be updated by state verification
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			if err := repository.CreateServiceContainer(ctx, dbTx, newContainer); err != nil {
				zap.L().Error("Failed to create container in database", zap.Error(err), zap.String("containerName", containerName))
				return fmt.Errorf("failed to create container %s: %w", containerName, err)
			}
			zap.L().Debug("Created new container record", zap.String("containerName", containerName), zap.String("containerID", newContainer.ID))
		}
	}

	// Check for containers in database that are not in the compose file anymore
	for _, existingContainer := range existingContainers {
		if !composeContainerNames[existingContainer.ContainerName] {
			// Container exists in database but not in compose file, mark as removed
			if existingContainer.State != models.ContainerStateRemoved {
				existingContainer.State = models.ContainerStateRemoved
				existingContainer.UpdatedAt = time.Now()

				if err := repository.UpdateServiceContainer(ctx, dbTx, existingContainer); err != nil {
					zap.L().Error("Failed to mark container as removed in database", zap.Error(err), zap.String("containerName", existingContainer.ContainerName))
					return fmt.Errorf("failed to update container %s as removed: %w", existingContainer.ContainerName, err)
				}
				zap.L().Debug("Marked container as removed", zap.String("containerName", existingContainer.ContainerName))
			}
		}
	}

	zap.L().Debug("Container sync to database completed successfully", zap.String("serviceID", serviceID))
	return nil
}
