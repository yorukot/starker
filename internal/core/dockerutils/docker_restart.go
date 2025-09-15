package dockerutils

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yorukot/starker/pkg/connection"
)

// RestartDockerCompose restarts all services defined in the Docker Compose configuration
// with real-time progress streaming for container restart
func (h *DockerHandler) RestartDockerCompose(ctx context.Context) error {
	h.StreamChan.LogStep("Restarting Docker Compose services")

	// Get the compose file path
	serviceDataPath := h.NamingGenerator.GenerateServiceDataPath()
	composeFilePath := filepath.Join(serviceDataPath, "compose.yml")

	// First, ensure the compose file is written to the server
	if err := h.WriteDockerCompose(); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to write compose file: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	// Execute Docker Compose restart operations in sequence
	if err := h.restartServices(ctx, composeFilePath); err != nil {
		return err // Error already sent through FinalError channel
	}

	h.StreamChan.LogStep("Docker Compose services restarted successfully")
	h.StreamChan.DoneChan <- true
	return nil
}

// restartServices restarts all services defined in the compose file
func (h *DockerHandler) restartServices(ctx context.Context, composeFilePath string) error {
	h.StreamChan.LogStep("Restarting Docker services")

	// Use docker compose restart command for graceful restart
	restartCmd := fmt.Sprintf("docker compose -f %s restart", composeFilePath)
	
	// Execute the restart command with streaming
	if err := connection.ExecuteCommand(h.Client, restartCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to restart services: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	return nil
}