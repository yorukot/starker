package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockeryaml"
)

func (dh *DockerHandler) StartDockerContainers(ctx context.Context, tx pgx.Tx) error {
	// Resolve service dependencies and get ordered startup sequence
	startupOrder, err := dockeryaml.ResolveDependencyOrder(dh.Project.Services)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to resolve service dependencies: %v", err))
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	// Log the resolved startup order
	dh.StreamChan.LogInfo(fmt.Sprintf("Starting containers in dependency order: %v", startupOrder))

	// Start containers in dependency-resolved order
	for _, serviceName := range startupOrder {
		service := dh.Project.Services[serviceName]

		dh.StreamChan.LogStep(fmt.Sprintf("Starting service: %s", serviceName))

		// Generate the docker container name and create the Docker container
		containerID, err := dh.StartDockerContainer(ctx, serviceName, service)
		if err != nil {
			zap.L().Error("failed to start docker container", zap.Error(err), zap.String("service", serviceName))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to start docker container %s: %v", serviceName, err))
			return fmt.Errorf("failed to start docker container %s (dependency chain broken): %w", serviceName, err)
		}

		// Generate container name for database update
		containerName := dh.NamingGenerator.ContainerName(serviceName)

		// Update container state in database
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateRunning)
		if err != nil {
			zap.L().Error("Failed to update container state in database", zap.String("container", containerName), zap.Error(err))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
			return fmt.Errorf("failed to update container %s state in database: %w", serviceName, err)
		}

		dh.StreamChan.LogInfo(fmt.Sprintf("Container %s created and saved successfully", serviceName))
	}
	return nil
}

// StartDockerContainer creates and starts a Docker container and returns the container ID
func (dh *DockerHandler) StartDockerContainer(ctx context.Context, serviceName string, serviceConfig types.ServiceConfig) (containerID string, err error) {
	// Generate container name using naming generator
	containerName := dh.NamingGenerator.ContainerName(serviceName)

	// Check if a container with this name already exists
	existingContainer, err := dh.checkExistingContainer(ctx, containerName)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to check for existing container %s: %v", containerName, err))
		return "", fmt.Errorf("failed to check for existing container %s: %w", containerName, err)
	}

	// If container exists, handle it appropriately
	if existingContainer != nil {
		dh.StreamChan.LogInfo(fmt.Sprintf("Found existing container %s in state: %s", containerName, existingContainer.State))

		// Check if the existing container is from our service (has our labels)
		serviceIDLabel, hasServiceLabel := existingContainer.Labels["starker.service.id"]
		isOurContainer := hasServiceLabel && serviceIDLabel == dh.NamingGenerator.ServiceID()

		if !isOurContainer {
			dh.StreamChan.LogError(fmt.Sprintf("Container %s exists but doesn't belong to our service (service.id: %s vs expected: %s)",
				containerName, serviceIDLabel, dh.NamingGenerator.ServiceID()))
			return "", fmt.Errorf("container %s exists but belongs to different service", containerName)
		}

		// If it's stopped, remove it so we can create a fresh one
		if err := dh.removeExistingContainer(ctx, existingContainer); err != nil {
			return "", fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Generate project name and labels
	projectName := dh.NamingGenerator.ProjectName()
	labels := dh.NamingGenerator.GetServiceLabels(projectName, serviceName)

	// Log container creation start
	dh.StreamChan.LogInfo(fmt.Sprintf("Creating Docker container: %s", containerName))

	// Convert service configuration to Docker API configurations
	containerConfig, hostConfig, networkConfig, err := dockeryaml.ConvertToDockerConfigs(serviceConfig, labels, dh.NamingGenerator)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to convert service configuration: %v", err))
		return "", fmt.Errorf("failed to convert service configuration: %w", err)
	}

	// Create the Docker container
	resp, err := dh.Client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to create Docker container %s: %v", containerName, err))
		return "", fmt.Errorf("failed to create Docker container %s: %w", containerName, err)
	}

	// Start the container
	err = dh.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to start Docker container %s: %v", containerName, err))
		return "", fmt.Errorf("failed to start Docker container %s: %w", containerName, err)
	}

	// Log successful creation and start
	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully created and started Docker container: %s", resp.ID))

	return resp.ID, nil
}

// StopDockerContainer stops a Docker container with configurable removal option
func (dh *DockerHandler) StopDockerContainer(ctx context.Context, tx pgx.Tx, containerID, containerName string, removeAfterStop bool) error {
	// Log container stop start
	dh.StreamChan.LogStep(fmt.Sprintf("Stopping Docker container: %s", containerName))

	// Stop the Docker container
	timeout := 30
	err := dh.Client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to stop Docker container %s: %v", containerName, err))
		return fmt.Errorf("failed to stop Docker container %s: %w", containerName, err)
	}

	// Log successful stop
	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully stopped Docker container: %s", containerName))

	// Remove the container if requested (default behavior)
	if removeAfterStop {
		dh.StreamChan.LogStep(fmt.Sprintf("Removing Docker container: %s", containerName))

		err = dh.Client.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force: true,
		})
		if err != nil {
			dh.StreamChan.LogError(fmt.Sprintf("Failed to remove Docker container %s: %v", containerName, err))
			return fmt.Errorf("failed to remove Docker container %s: %w", containerName, err)
		}

		dh.StreamChan.LogInfo(fmt.Sprintf("Successfully removed Docker container: %s", containerName))

		// Update container state to indicate it's removed
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateStopped)
		if err != nil {
			dh.StreamChan.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
			return fmt.Errorf("failed to update container state in database: %w", err)
		}
	} else {
		// Update container state in database to stopped (but not removed)
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateStopped)
		if err != nil {
			dh.StreamChan.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
			return fmt.Errorf("failed to update container state in database: %w", err)
		}
	}

	return nil
}

// UpdateContainerState updates the state of a specific container in the database by container name
func (dh *DockerHandler) UpdateContainerState(ctx context.Context, tx pgx.Tx, containerID, containerName string, state models.ContainerState) error {
	// Get the specific service container by name
	serviceContainer, err := repository.GetServiceContainerByName(ctx, tx, dh.NamingGenerator.ServiceID(), containerName)
	if err != nil {
		return fmt.Errorf("failed to get service container by name from database: %w", err)
	}

	// Update the container with the new ID and state
	serviceContainer.ContainerID = &containerID
	serviceContainer.State = state
	serviceContainer.UpdatedAt = time.Now()

	err = repository.UpdateServiceContainer(ctx, tx, *serviceContainer)
	if err != nil {
		return fmt.Errorf("failed to update container state in database: %w", err)
	}

	dh.StreamChan.LogInfo(fmt.Sprintf("Container %s state updated to %s in database", containerName, state))

	return nil
}
