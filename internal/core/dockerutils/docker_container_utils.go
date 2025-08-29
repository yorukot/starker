package dockerutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// ContainerInfo holds information about an existing container
type ContainerInfo struct {
	ID     string
	Name   string
	State  string
	Status string
	Labels map[string]string
}

// checkExistingContainer checks if a container with the given name already exists
func (dh *DockerHandler) checkExistingContainer(ctx context.Context, containerName string) (*ContainerInfo, error) {
	// Create filter to find container by name
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", containerName)

	// List containers (including stopped ones)
	containers, err := dh.Client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Check if any container matches exactly (Docker API returns partial matches)
	for _, container := range containers {
		for _, name := range container.Names {
			// Container names start with '/' in Docker API
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == containerName {
				return &ContainerInfo{
					ID:     container.ID,
					Name:   cleanName,
					State:  container.State,
					Status: container.Status,
					Labels: container.Labels,
				}, nil
			}
		}
	}

	return nil, nil
}

// removeExistingContainer removes a container if it exists
func (dh *DockerHandler) removeExistingContainer(ctx context.Context, containerInfo *ContainerInfo) error {
	dh.StreamChan.LogStep(fmt.Sprintf("Removing existing container: %s (state: %s)", containerInfo.Name, containerInfo.State))

	// Stop the container if it's running
	if containerInfo.State == "running" {
		dh.StreamChan.LogStep(fmt.Sprintf("Stopping running container: %s", containerInfo.Name))

		timeout := 30
		err := dh.Client.ContainerStop(ctx, containerInfo.ID, container.StopOptions{
			Timeout: &timeout,
		})
		if err != nil {
			dh.StreamChan.LogError(fmt.Sprintf("Failed to stop existing container %s: %v", containerInfo.Name, err))
			return fmt.Errorf("failed to stop existing container %s: %w", containerInfo.Name, err)
		}
	}

	// Remove the container
	err := dh.Client.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{
		Force: true,
	})
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to remove existing container %s: %v", containerInfo.Name, err))
		return fmt.Errorf("failed to remove existing container %s: %w", containerInfo.Name, err)
	}

	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully removed existing container: %s", containerInfo.Name))

	return nil
}
