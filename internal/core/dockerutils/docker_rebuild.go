package dockerutils

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yorukot/starker/internal/core/dockersync"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/connection"
)

// RebuildDockerCompose performs a complete rebuild of Docker Compose services
// This operation: stops all containers -> syncs database -> starts with updated config
func (h *DockerHandler) RebuildDockerCompose(ctx context.Context) error {
	h.StreamChan.LogLog("Starting Docker Compose rebuild operation")

	// Get the compose file path
	serviceDataPath := h.NamingGenerator.GenerateServiceDataPath()
	composeFilePath := filepath.Join(serviceDataPath, "compose.yml")

	// Step 1: Write the updated compose file to the server
	h.StreamChan.LogLog("Writing updated compose configuration")
	if err := h.WriteDockerCompose(); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to write compose file: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Step 2: Stop all containers
	h.StreamChan.LogLog("Stopping existing containers")
	if err := h.stopContainersForRebuild(composeFilePath); err != nil {
		return err // Error already sent through FinalError channel
	}

	// Step 3: Sync containers to database
	h.StreamChan.LogLog("Synchronizing container states to database")
	if err := h.syncContainersToDatabase(); err != nil {
		return err // Error already sent through FinalError channel
	}

	// Step 4: Start containers with updated configuration
	h.StreamChan.LogLog("Starting containers with updated configuration")
	if err := h.startContainersForRebuild(composeFilePath); err != nil {
		return err // Error already sent through FinalError channel
	}

	// Step 5: Verify and update final states
	h.StreamChan.LogLog("Verifying final container states")
	if err := h.VerifyAndUpdateStates(ctx, composeFilePath); err != nil {
		return err // Error already sent through FinalError channel
	}

	h.StreamChan.LogLog("Docker Compose rebuild completed successfully")
	h.StreamChan.DoneChan <- true
	return nil
}

// stopContainersForRebuild stops all containers for rebuild operation
func (h *DockerHandler) stopContainersForRebuild(composeFilePath string) error {
	h.StreamChan.LogLog("Bringing down Docker Compose stack")

	// Use docker compose down to stop and remove containers, networks
	downCmd := fmt.Sprintf("docker compose -f %s down", composeFilePath)

	// Execute the down command with streaming
	if err := connection.ExecuteCommand(h.Client, downCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to stop containers: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	h.StreamChan.LogLog("All containers stopped successfully")
	return nil
}

// syncContainersToDatabase synchronizes container state with database
func (h *DockerHandler) syncContainersToDatabase() error {
	// Start a database transaction for the sync operation
	tx, err := repository.StartTransaction(h.DB, context.Background())
	if err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to start database transaction: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}
	defer repository.DeferRollback(tx, context.Background())

	// Perform the container sync using the existing sync functionality
	if err := dockersync.SyncContainersToDB(context.Background(), tx, h.ConnectionPool, *h.NamingGenerator, *h.Project); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to sync containers to database: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Commit the transaction
	repository.CommitTransaction(tx, context.Background())

	h.StreamChan.LogLog("Container states synchronized with database")
	return nil
}

// startContainersForRebuild starts containers with updated configuration
func (h *DockerHandler) startContainersForRebuild(composeFilePath string) error {
	h.StreamChan.LogLog("Starting containers with updated configuration")

	// Use docker compose up to start containers with new configuration
	upCmd := fmt.Sprintf("docker compose -f %s up -d", composeFilePath)

	// Execute the up command with streaming
	if err := connection.ExecuteCommand(h.Client, upCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to start containers: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	h.StreamChan.LogLog("All containers started successfully")
	return nil
}
