package dockerutils

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
)

// LogOptions configures container log retrieval options
type LogOptions struct {
	Follow     bool      // Stream logs continuously
	Tail       string    // Number of lines from the end of the logs to show (e.g., "100", "all")
	Timestamps bool      // Include timestamps in log output
	Since      time.Time // Show logs since timestamp
}

// GetContainerLogs retrieves logs from a Docker container by container ID
func (dh *DockerHandler) GetContainerLogs(ctx context.Context, containerID string, options LogOptions) (io.ReadCloser, error) {
	dh.StreamChan.LogStep(fmt.Sprintf("Retrieving logs for container: %s", containerID))

	// Configure log options for Docker API
	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     options.Follow,
		Timestamps: options.Timestamps,
		Tail:       options.Tail,
	}

	// Set since timestamp if provided
	if !options.Since.IsZero() {
		logOptions.Since = options.Since.Format(time.RFC3339Nano)
	}

	// Get container logs from Docker API
	logs, err := dh.Client.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to get container logs for %s: %v", containerID, err))
		return nil, fmt.Errorf("failed to get container logs for %s: %w", containerID, err)
	}

	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully retrieved logs for container: %s", containerID))

	return logs, nil
}

// GetContainerLogsByName retrieves logs from a Docker container by container name
func (dh *DockerHandler) GetContainerLogsByName(ctx context.Context, containerName string, options LogOptions) (io.ReadCloser, error) {
	dh.StreamChan.LogStep(fmt.Sprintf("Looking up container by name: %s", containerName))

	// Check if container exists and get its info
	containerInfo, err := dh.checkExistingContainer(ctx, containerName)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to check for container %s: %v", containerName, err))
		return nil, fmt.Errorf("failed to check for container %s: %w", containerName, err)
	}

	if containerInfo == nil {
		dh.StreamChan.LogError(fmt.Sprintf("Container not found: %s", containerName))
		return nil, fmt.Errorf("container not found: %s", containerName)
	}

	dh.StreamChan.LogStep(fmt.Sprintf("Found container %s with ID: %s", containerName, containerInfo.ID))

	// Get logs using the container ID
	return dh.GetContainerLogs(ctx, containerInfo.ID, options)
}
