package dockerutils

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yorukot/starker/pkg/connection"
)

// StopDockerCompose stops all services defined in the Docker Compose configuration
// with real-time progress streaming for container shutdown
func (h *DockerHandler) StopDockerCompose(ctx context.Context) error {
	h.StreamChan.LogStep("Stopping Docker Compose services")

	// Get the compose file path
	serviceDataPath := h.NamingGenerator.GenerateServiceDataPath()
	composeFilePath := filepath.Join(serviceDataPath, "compose.yml")

	// Execute Docker Compose stop operation
	if err := h.stopServices(ctx, composeFilePath); err != nil {
		return err // Error already sent through FinalError channel
	}

	h.StreamChan.LogStep("Docker Compose services stopped successfully")
	h.StreamChan.DoneChan <- true
	return nil
}

// stopServices stops all services defined in the compose file
func (h *DockerHandler) stopServices(ctx context.Context, composeFilePath string) error {
	h.StreamChan.LogStep("Stopping Docker services")

	stopCmd := fmt.Sprintf("docker compose -f %s down", composeFilePath)
	
	// Execute the stop command with streaming
	if err := connection.ExecuteCommand(h.Client, stopCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to stop services: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	return nil
}