package dockerutils

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yorukot/starker/pkg/connection"
)

// StartDockerCompose starts all services defined in the Docker Compose configuration
// with real-time progress streaming for image pulls and container startup
func (h *DockerHandler) StartDockerCompose(ctx context.Context) error {
	h.StreamChan.LogLog("Starting Docker Compose services")

	// First, ensure the compose file is written to the server
	if err := h.WriteDockerCompose(); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to write compose file: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Get the compose file path
	serviceDataPath := h.NamingGenerator.GenerateServiceDataPath()
	composeFilePath := filepath.Join(serviceDataPath, "compose.yml")

	// Execute Docker Compose operations in sequence
	if err := h.pullImages(composeFilePath); err != nil {
		return err
	}

	if err := h.startServices(composeFilePath); err != nil {
		return err
	}

	if err := h.VerifyAndUpdateStates(ctx, composeFilePath); err != nil {
		return err
	}

	h.StreamChan.LogLog("Docker Compose services started successfully")
	h.StreamChan.DoneChan <- true
	return nil
}

// pullImages pulls all required Docker images with progress streaming
func (h *DockerHandler) pullImages(composeFilePath string) error {
	h.StreamChan.LogLog("Pulling Docker images")

	pullCmd := fmt.Sprintf("docker compose -f %s pull", composeFilePath)

	// Execute the pull command with streaming
	if err := connection.ExecuteCommand(h.Client, pullCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to pull images: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	return nil
}

// startServices starts all services defined in the compose file
func (h *DockerHandler) startServices(composeFilePath string) error {
	h.StreamChan.LogLog("Starting Docker services")

	startCmd := fmt.Sprintf("docker compose -f %s up -d", composeFilePath)

	// Execute the start command with streaming
	if err := connection.ExecuteCommand(h.Client, startCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to start services: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	return nil
}
