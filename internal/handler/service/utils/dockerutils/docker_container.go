package dockerutils

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/generator"
)

// startSingleServiceFromProject starts a single service from the project configuration
func startSingleServiceFromProject(ctx context.Context, dockerClient *client.Client, service *types.ServiceConfig, project *types.Project, streamResult *StreamingResult, namingGen *generator.NamingGenerator, db pgx.Tx, serviceID string) error {
	streamResult.LogChan <- fmt.Sprintf("Starting service: %s", service.Name)

	containerName := namingGen.ContainerName(service.Name)

	// Check if container already exists
	existingContainers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("name", containerName),
			filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", project.Name)),
		),
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	if len(existingContainers) > 0 {
		// Container exists, just start it
		containerID = existingContainers[0].ID
		streamResult.LogChan <- fmt.Sprintf("Starting existing container: %s", containerName)

		if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start existing container %s: %w", service.Name, err)
		}

		// Update container state in database to running with the Docker container ID
		if err := UpdateContainerStateByName(ctx, db, serviceID, containerName, models.ContainerStateRunning, &containerID); err != nil {
			streamResult.LogChan <- fmt.Sprintf("Warning: failed to update container state for %s: %v", containerName, err)
		}
	} else {
		// Create new container with proper compose configuration
		containerID, err = createComposeContainerFromProject(ctx, dockerClient, service, project, containerName, streamResult, namingGen, db, serviceID)
		if err != nil {
			return fmt.Errorf("failed to create container %s: %w", service.Name, err)
		}
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully started service: %s", service.Name)

	return nil
}

// createComposeContainerFromProject creates a container from project service configuration
func createComposeContainerFromProject(ctx context.Context, dockerClient *client.Client, service *types.ServiceConfig, project *types.Project, containerName string, streamResult *StreamingResult, namingGen *generator.NamingGenerator, db pgx.Tx, serviceID string) (string, error) {
	// Create port bindings
	exposedPorts := make(nat.PortSet)
	portBindings := make(nat.PortMap)

	for _, portConfig := range service.Ports {
		natPort, err := nat.NewPort(portConfig.Protocol, strconv.Itoa(int(portConfig.Target)))
		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Invalid port format %d/%s: %v", portConfig.Target, portConfig.Protocol, err)
			continue
		}

		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: portConfig.Published,
			},
		}
	}

	// Create volume mounts
	var mounts []mount.Mount
	for _, volumeConfig := range service.Volumes {
		source := volumeConfig.Source
		target := volumeConfig.Target

		mountType := mount.TypeBind
		if volumeConfig.Type == types.VolumeTypeVolume {
			mountType = mount.TypeVolume
			source = namingGen.VolumeName(source)
		}

		mounts = append(mounts, mount.Mount{
			Type:   mountType,
			Source: source,
			Target: target,
		})
	}

	// Convert environment variables
	env := make([]string, 0)
	for key, valuePtr := range service.Environment {
		value := ""
		if valuePtr != nil {
			value = *valuePtr
		}
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Determine the image name - if service has build config but no image, generate one
	imageName := service.Image
	if imageName == "" && service.Build != nil {
		// Generate image name for built services (same as what would be built)
		imageName = fmt.Sprintf("%s-%s:latest", project.Name, service.Name)
		streamResult.LogChan <- fmt.Sprintf("Service %s has no image specified, using generated name: %s", service.Name, imageName)
	}

	if imageName == "" {
		return "", fmt.Errorf("service %s has no image specified and no build configuration", service.Name)
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:        imageName,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       namingGen.GetServiceLabels(project.Name, service.Name),
		Cmd:          []string(service.Command),
		Entrypoint:   []string(service.Entrypoint),
		WorkingDir:   service.WorkingDir,
		User:         service.User,
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
	}

	// Create network configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings),
	}

	// Add service to networks using consistent network name resolution
	for networkName := range service.Networks {
		// Resolve network name consistently with how networks were created
		var configuredName string
		if netConfig, exists := project.Networks[networkName]; exists {
			configuredName = netConfig.Name
		}
		fullNetworkName := namingGen.ResolveNetworkName(networkName, configuredName)
		zap.L().Debug("Resolved network name", zap.String("service", service.Name), zap.String("network", networkName), zap.String("fullNetworkName", fullNetworkName))
		networkConfig.EndpointsConfig[fullNetworkName] = &network.EndpointSettings{
			Aliases: []string{service.Name},
		}
	}

	streamResult.LogChan <- fmt.Sprintf("Creating container: %s", containerName)
	zap.L().Debug("Creating container", zap.Any("networkconfig", networkConfig))
	// Create container with detailed error logging
	resp, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		streamResult.LogChan <- fmt.Sprintf("Failed to create container %s: %v", containerName, err)
		streamResult.LogChan <- fmt.Sprintf("Container config - Image: %s, Cmd: %v, Entrypoint: %v",
			containerConfig.Image, containerConfig.Cmd, containerConfig.Entrypoint)

		// Check if this is an image not found error
		if strings.Contains(err.Error(), "No such image") || strings.Contains(err.Error(), "pull access denied") {
			streamResult.LogChan <- fmt.Sprintf("Image %s not found. For services with build config, the image should be built first or a fallback image should be specified.", containerConfig.Image)
		}

		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Update container state in database to running with the Docker container ID
	if err := UpdateContainerStateByName(ctx, db, serviceID, containerName, models.ContainerStateRunning, &resp.ID); err != nil {
		streamResult.LogChan <- fmt.Sprintf("Warning: failed to update container state for %s: %v", containerName, err)
	}

	streamResult.LogChan <- fmt.Sprintf("Created and started container: %s", containerName)
	return resp.ID, nil
}

// stopSingleServiceFromProject stops a single service from the project
func stopSingleServiceFromProject(ctx context.Context, dockerClient *client.Client, serviceName, projectName string, streamResult *StreamingResult, namingGen *generator.NamingGenerator, db pgx.Tx, serviceID string) error {
	streamResult.LogChan <- fmt.Sprintf("Stopping service: %s", serviceName)

	// Find containers for this specific service using generator filters
	fb := generator.NewFilterBuilder(namingGen)
	filterArgs := fb.ServiceFilters(projectName, serviceName)

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
		All:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for service %s: %w", serviceName, err)
	}

	if len(containers) == 0 {
		streamResult.LogChan <- fmt.Sprintf("No containers found for service %s", serviceName)
		return nil
	}

	// Stop and remove each container for this service
	timeout := int(30) // 30 seconds timeout
	for _, cont := range containers {
		// Stop container if running
		if cont.State == "running" {
			streamResult.LogChan <- fmt.Sprintf("Stopping container for service: %s", serviceName)
			if err := dockerClient.ContainerStop(ctx, cont.ID, container.StopOptions{Timeout: &timeout}); err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to stop container for service %s: %v", serviceName, err)
				continue
			}
			streamResult.LogChan <- fmt.Sprintf("Stopped container for service: %s", serviceName)
		}

		// Remove container
		streamResult.LogChan <- fmt.Sprintf("Removing container for service: %s", serviceName)
		if err := dockerClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{
			RemoveVolumes: false, // Don't remove named volumes
			Force:         true,  // Force remove even if running
		}); err != nil {
			streamResult.LogChan <- fmt.Sprintf("Failed to remove container for service %s: %v", serviceName, err)
			continue
		}
		streamResult.LogChan <- fmt.Sprintf("Removed container for service: %s", serviceName)

		// Update container state in database to stopped after successful stop and removal
		containerName := namingGen.ContainerName(serviceName)
		if err := UpdateContainerStateByName(ctx, db, serviceID, containerName, models.ContainerStateStopped, nil); err != nil {
			streamResult.LogChan <- fmt.Sprintf("Warning: failed to update container state for %s: %v", containerName, err)
		}
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully stopped service: %s", serviceName)
	return nil
}
