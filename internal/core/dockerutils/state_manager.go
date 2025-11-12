package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/dockeryaml"
)

// VerifyAndUpdateStates checks container status and updates both service and container states in database
func (h *DockerHandler) VerifyAndUpdateStates(ctx context.Context, composeFilePath string) error {
	h.StreamChan.LogLog("Verifying service status")

	// Check service status using simple command execution
	statusCmd := fmt.Sprintf("docker compose -f %s ps --format json -a", composeFilePath)

	stdout, stderr, err := connection.ExecuteSimpleCommand(h.Client, statusCmd)
	if err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to verify services: %v", err))
		if stderr != "" {
			h.StreamChan.LogError(fmt.Sprintf("Docker error: %s", stderr))
		}
		h.StreamChan.FinalError <- err
		return err
	}

	// Parse container statuses from JSON output
	containers, err := dockeryaml.ParseContainerStatuses(stdout)
	if err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to parse container status JSON: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Determine overall service status based on container states
	serviceState := h.determineServiceStateFromContainers(containers)

	// Start a new transaction for the database updates
	tx, err := repository.StartTransaction(h.DB, ctx)
	if err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to start transaction: %v", err))
		h.StreamChan.FinalError <- err
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer repository.DeferRollback(tx, ctx)

	// Update service status
	if err := h.updateServiceStatusInTx(ctx, tx, serviceState); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to update service status: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Sync container states
	if err := h.syncContainerStates(ctx, tx, containers); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to sync container states: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Commit the transaction
	repository.CommitTransaction(tx, ctx)

	h.StreamChan.LogLog(fmt.Sprintf("Service verification completed - status: %s", serviceState))
	return nil
}

// syncContainerStates creates or updates container records based on current Docker status
// and marks containers that no longer exist in Docker as removed
func (h *DockerHandler) syncContainerStates(ctx context.Context, tx pgx.Tx, containers []dockeryaml.DockerContainerStatus) error {
	serviceID := h.NamingGenerator.ServiceID()

	// Get all existing containers from database for comparison
	existingContainers, err := repository.GetServiceContainers(ctx, tx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get existing containers: %w", err)
	}

	// Create maps for efficient lookups
	existingByName := make(map[string]*models.ServiceContainer)
	for i := range existingContainers {
		existingByName[existingContainers[i].ContainerName] = &existingContainers[i]
	}

	dockerContainerNames := make(map[string]bool)
	for _, container := range containers {
		dockerContainerNames[container.Name] = true
	}

	now := time.Now()

	// Process containers that exist in Docker
	for _, container := range containers {
		containerState := h.mapDockerStateToContainerState(container.State)

		if existingContainer, exists := existingByName[container.Name]; exists {
			// Update existing container
			existingContainer.ContainerID = &container.ID
			existingContainer.State = containerState
			existingContainer.UpdatedAt = now

			if err := repository.UpdateServiceContainer(ctx, tx, *existingContainer); err != nil {
				return fmt.Errorf("failed to update container: %w", err)
			}
		} else {
			// Create new container record
			newContainer := models.ServiceContainer{
				ID:            ksuid.New().String(),
				ServiceID:     serviceID,
				ContainerID:   &container.ID,
				ContainerName: container.Name,
				State:         containerState,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			if err := repository.CreateServiceContainer(ctx, tx, newContainer); err != nil {
				return fmt.Errorf("failed to create container: %w", err)
			}
			zap.L().Debug("Created new container", zap.String("container", container.Name), zap.String("state", string(containerState)))
		}
	}

	// Handle containers that exist in database but not in Docker (mark as removed)
	for _, existingContainer := range existingContainers {
		if !dockerContainerNames[existingContainer.ContainerName] {
			// Container exists in DB but not in Docker - mark as removed
			if existingContainer.State != models.ContainerStateRemoved {
				existingContainer.State = models.ContainerStateRemoved
				existingContainer.UpdatedAt = now
				existingContainer.ContainerID = nil // Clear container ID since it no longer exists

				if err := repository.UpdateServiceContainer(ctx, tx, existingContainer); err != nil {
					return fmt.Errorf("failed to mark container as removed: %w", err)
				}
				zap.L().Debug("Marked container as removed", zap.String("container", existingContainer.ContainerName))
			}
		}
	}

	return nil
}

// mapDockerStateToContainerState converts Docker container state to models.ContainerState
func (h *DockerHandler) mapDockerStateToContainerState(dockerState string) models.ContainerState {
	switch dockerState {
	case "running":
		return models.ContainerStateRunning
	case "exited":
		return models.ContainerStateExited
	case "stopped":
		return models.ContainerStateStopped
	default:
		// For unknown states, default to stopped for safety
		return models.ContainerStateStopped
	}
}

// determineServiceStateFromContainers analyzes container states and determines the overall service state
func (h *DockerHandler) determineServiceStateFromContainers(containers []dockeryaml.DockerContainerStatus) models.ServiceState {
	if len(containers) == 0 {
		// No containers found, service is stopped
		return models.ServiceStateStopped
	}

	runningCount := 0
	totalCount := len(containers)

	for _, container := range containers {
		if container.State == "running" {
			runningCount++
		}
	}

	// If all containers are running, service is running
	if runningCount == totalCount {
		return models.ServiceStateRunning
	}

	// If no containers are running, service is stopped
	if runningCount == 0 {
		return models.ServiceStateStopped
	}

	// Mixed state - some running, some not - consider it stopped for safety
	return models.ServiceStateStopped
}

// updateServiceStatusInTx updates the service status in the database within a transaction
func (h *DockerHandler) updateServiceStatusInTx(ctx context.Context, tx pgx.Tx, state models.ServiceState) error {
	// Get service ID from naming generator
	serviceID := h.NamingGenerator.ServiceID()

	// Get current service data to preserve other fields
	service, err := repository.GetServiceByID(ctx, tx, serviceID, h.NamingGenerator.TeamID(), h.NamingGenerator.ProjectID())
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}
	if service == nil {
		return fmt.Errorf("service not found")
	}

	// Update only the state
	service.State = state

	// Save the updated service
	if err := repository.UpdateService(ctx, tx, *service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}
