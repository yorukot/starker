package dockersync

import (
	"context"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

// This function going to sync all the container to database
func SyncContainersToDB(ctx context.Context, dbTx pgx.Tx, connPool *connection.ConnectionPool, namingGenerator generator.NamingGenerator, composeProject types.Project) error {
	// Get the service ID from the naming generator
	serviceID := namingGenerator.GetLabels()["starker.service.id"]

	// Get all existing containers from the database for this service
	existingContainers, err := repository.GetServiceContainers(ctx, dbTx, serviceID)
	if err != nil {
		return err
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
				State:         models.ContainerStateStopped, // Default state
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			if err := repository.CreateServiceContainer(ctx, dbTx, newContainer); err != nil {
				return err
			}
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
					return err
				}
			}
		}
	}

	return nil
}
