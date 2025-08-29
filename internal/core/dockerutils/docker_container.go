package dockerutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockeryaml"
)

func (dh *DockerHandler) StartDockerContainers(ctx context.Context, tx pgx.Tx) error {
	// Resolve service dependencies and get ordered startup sequence
	startupOrder, err := dockeryaml.ResolveDependencyOrder(dh.Project.Services)
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to resolve service dependencies: %v", err))
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	// Log the resolved startup order
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Starting containers in dependency order: %v", startupOrder))

	// Start containers in dependency-resolved order
	for _, serviceName := range startupOrder {
		service := dh.Project.Services[serviceName]

		dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Starting service: %s", serviceName))

		// Generate the docker container name and create the Docker container
		containerID, err := dh.StartDockerContainer(ctx, serviceName, service)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to start docker container %s: %v", serviceName, err))
			return fmt.Errorf("failed to start docker container %s (dependency chain broken): %w", serviceName, err)
		}

		// Generate container name for database update
		containerName := dh.NamingGenerator.ContainerName(serviceName)

		// Update container state in database
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateRunning)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
			return fmt.Errorf("failed to update container %s state in database: %w", serviceName, err)
		}

		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Container %s created and saved successfully", serviceName))
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
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to check for existing container %s: %v", containerName, err))
		return "", fmt.Errorf("failed to check for existing container %s: %w", containerName, err)
	}

	// If container exists, handle it appropriately
	if existingContainer != nil {
		dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Found existing container %s in state: %s", containerName, existingContainer.State))

		// Check if the existing container is from our service (has our labels)
		serviceIDLabel, hasServiceLabel := existingContainer.Labels["starker.service.id"]
		isOurContainer := hasServiceLabel && serviceIDLabel == dh.NamingGenerator.ServiceID()

		if !isOurContainer {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Container %s exists but doesn't belong to our service (service.id: %s vs expected: %s)",
				containerName, serviceIDLabel, dh.NamingGenerator.ServiceID()))
			return "", fmt.Errorf("container %s exists but belongs to different service", containerName)
		}

		// If it's running and belongs to us, we might want to return its ID instead of creating a new one
		if existingContainer.State == "running" {
			dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Container %s is already running, using existing container", containerName))
			return existingContainer.ID, nil
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
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Creating Docker container: %s", containerName))

	// Prepare port bindings
	portBindings := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)

	for _, port := range serviceConfig.Ports {
		containerPort, err := nat.NewPort(port.Protocol, fmt.Sprintf("%d", port.Target))
		if err != nil {
			return "", fmt.Errorf("invalid container port %d/%s: %w", port.Target, port.Protocol, err)
		}

		exposedPorts[containerPort] = struct{}{}

		if port.Published != "" {
			portBindings[containerPort] = []nat.PortBinding{
				{
					HostIP:   port.HostIP,
					HostPort: port.Published,
				},
			}
		}
	}

	// Prepare environment variables
	env := make([]string, 0, len(serviceConfig.Environment))
	for key, value := range serviceConfig.Environment {
		if value != nil {
			env = append(env, fmt.Sprintf("%s=%s", key, *value))
		}
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image:        serviceConfig.Image,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       labels,
		WorkingDir:   serviceConfig.WorkingDir,
	}

	// Add command if specified
	if len(serviceConfig.Command) > 0 {
		containerConfig.Cmd = []string(serviceConfig.Command)
	}

	// Add entrypoint if specified
	if len(serviceConfig.Entrypoint) > 0 {
		containerConfig.Entrypoint = []string(serviceConfig.Entrypoint)
	}

	// Prepare host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(serviceConfig.Restart),
		},
	}

	// Add volume bindings
	if len(serviceConfig.Volumes) > 0 {
		hostConfig.Binds = make([]string, 0, len(serviceConfig.Volumes))
		for _, volume := range serviceConfig.Volumes {
			if volume.Type == types.VolumeTypeBind {
				bind := fmt.Sprintf("%s:%s", volume.Source, volume.Target)
				if volume.ReadOnly {
					bind += ":ro"
				}
				hostConfig.Binds = append(hostConfig.Binds, bind)
			}
		}
	}

	// Prepare network configuration
	networkConfig := &network.NetworkingConfig{}
	if len(serviceConfig.Networks) > 0 {
		networkConfig.EndpointsConfig = make(map[string]*network.EndpointSettings)
		for networkName := range serviceConfig.Networks {
			resolvedNetworkName := dh.NamingGenerator.ResolveNetworkName(networkName, "")
			networkConfig.EndpointsConfig[resolvedNetworkName] = &network.EndpointSettings{}
		}
	}

	// Create the Docker container
	resp, err := dh.Client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		// If we get a name conflict error despite our checks, try one more time after cleanup
		if strings.Contains(err.Error(), "already in use") || strings.Contains(err.Error(), "Conflict") {
			dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Name conflict detected, performing additional cleanup for %s", containerName))

			// Try to find and remove the conflicting container again
			if conflictingContainer, checkErr := dh.checkExistingContainer(ctx, containerName); checkErr == nil && conflictingContainer != nil {
				dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Found conflicting container %s, attempting removal", conflictingContainer.ID))

				if removeErr := dh.removeExistingContainer(ctx, conflictingContainer); removeErr != nil {
					dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove conflicting container %s: %v", containerName, removeErr))
				} else {
					// Retry container creation after cleanup
					resp, err = dh.Client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
				}
			}
		}

		// If creation still fails, return the error
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to create Docker container %s: %v", containerName, err))
			return "", fmt.Errorf("failed to create Docker container %s: %w", containerName, err)
		}
	}

	// Start the container
	err = dh.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to start Docker container %s: %v", containerName, err))
		return "", fmt.Errorf("failed to start Docker container %s: %w", containerName, err)
	}

	// Log successful creation and start
	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully created and started Docker container: %s", resp.ID))

	return resp.ID, nil
}

// StopDockerContainer stops a Docker container with configurable removal option
func (dh *DockerHandler) StopDockerContainer(ctx context.Context, tx pgx.Tx, containerID, containerName string, removeAfterStop bool) error {
	// Log container stop start
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Stopping Docker container: %s", containerName))

	// Stop the Docker container
	timeout := 30
	err := dh.Client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to stop Docker container %s: %v", containerName, err))
		return fmt.Errorf("failed to stop Docker container %s: %w", containerName, err)
	}

	// Log successful stop
	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully stopped Docker container: %s", containerName))

	// Remove the container if requested (default behavior)
	if removeAfterStop {
		dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Removing Docker container: %s", containerName))

		err = dh.Client.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force: true,
		})
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove Docker container %s: %v", containerName, err))
			return fmt.Errorf("failed to remove Docker container %s: %w", containerName, err)
		}

		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully removed Docker container: %s", containerName))

		// Update container state to indicate it's removed
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateStopped)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
			return fmt.Errorf("failed to update container state in database: %w", err)
		}
	} else {
		// Update container state in database to stopped (but not removed)
		err = dh.UpdateContainerState(ctx, tx, containerID, containerName, models.ContainerStateStopped)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to update container state in database: %v", err))
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

	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Container %s state updated to %s in database", containerName, state))

	return nil
}

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
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Removing existing container: %s (state: %s)", containerInfo.Name, containerInfo.State))

	// Stop the container if it's running
	if containerInfo.State == "running" {
		dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Stopping running container: %s", containerInfo.Name))

		timeout := 30
		err := dh.Client.ContainerStop(ctx, containerInfo.ID, container.StopOptions{
			Timeout: &timeout,
		})
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to stop existing container %s: %v", containerInfo.Name, err))
			return fmt.Errorf("failed to stop existing container %s: %w", containerInfo.Name, err)
		}
	}

	// Remove the container
	err := dh.Client.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{
		Force: true,
	})
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove existing container %s: %v", containerInfo.Name, err))
		return fmt.Errorf("failed to remove existing container %s: %w", containerInfo.Name, err)
	}

	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully removed existing container: %s", containerInfo.Name))

	return nil
}
