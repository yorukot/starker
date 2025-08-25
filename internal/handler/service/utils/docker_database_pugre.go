package utils

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/generator"
)

// DockerDatabaseToPurge purges Docker resources and removes database records
// This function ensures that Docker resources (containers, networks) are properly
// cleaned up and corresponding database records are removed for service cleanup
func DockerDatabaseToPurge(ctx context.Context, project *types.Project, cfg *DockerServiceConfig, db pgx.Tx, streamResult *StreamingResult) error {
	// Step 1: Validate inputs
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("docker service config cannot be nil")
	}
	if cfg.Generator == nil {
		return fmt.Errorf("naming generator cannot be nil")
	}
	if db == nil {
		return fmt.Errorf("database transaction cannot be nil")
	}
	if streamResult == nil {
		return fmt.Errorf("streaming result cannot be nil")
	}

	streamResult.LogChan <- fmt.Sprintf("Starting resource purge for project %s", project.Name)

	// Step 2: Purge containers (stop and remove)
	streamResult.LogChan <- fmt.Sprintf("Purging %d service containers", len(project.Services))
	if err := purgeContainersFromDocker(ctx, cfg.Client, project.Services, cfg.ServiceID, db, cfg.Generator, streamResult); err != nil {
		return fmt.Errorf("failed to purge containers: %w", err)
	}

	// Step 3: Purge networks (with safety checks)
	streamResult.LogChan <- fmt.Sprintf("Purging %d networks", len(project.Networks))
	if err := purgeNetworksFromDocker(ctx, cfg.Client, project.Networks, cfg.ServiceID, db, cfg.Generator, streamResult); err != nil {
		return fmt.Errorf("failed to purge networks: %w", err)
	}

	streamResult.LogChan <- fmt.Sprintf("Resource purge completed successfully for project %s", project.Name)
	return nil
}

func purgeContainersFromDocker(ctx context.Context, dockerClient *client.Client, services types.Services, serviceID string, db pgx.Tx, generator *generator.NamingGenerator, streamResult *StreamingResult) error {
	// Get all containers from database first
	containers, err := repository.GetServiceContainers(ctx, db, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service containers from database: %w", err)
	}

	if len(containers) == 0 {
		streamResult.LogChan <- "No containers found in database to purge"
		return nil
	}

	// Stop and remove each container
	timeout := int(30) // 30 seconds timeout
	for _, dbContainer := range containers {
		containerName := dbContainer.ContainerName
		streamResult.LogChan <- fmt.Sprintf("Purging container: %s", containerName)

		// Find the actual Docker container by name
		dockerContainers, err := dockerClient.ContainerList(ctx, container.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", containerName)),
			All:     true,
		})
		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Failed to list Docker containers: %v", err)
			continue
		}

		// Process each matching Docker container
		for _, dockerContainer := range dockerContainers {
			// Stop container if running
			if dockerContainer.State == "running" {
				streamResult.LogChan <- fmt.Sprintf("Stopping container: %s", containerName)
				if err := dockerClient.ContainerStop(ctx, dockerContainer.ID, container.StopOptions{Timeout: &timeout}); err != nil {
					streamResult.LogChan <- fmt.Sprintf("Failed to stop container %s: %v", containerName, err)
					continue
				}
				streamResult.LogChan <- fmt.Sprintf("Stopped container: %s", containerName)
			}

			// Remove container
			streamResult.LogChan <- fmt.Sprintf("Removing container: %s", containerName)
			if err := dockerClient.ContainerRemove(ctx, dockerContainer.ID, container.RemoveOptions{
				RemoveVolumes: false, // Don't remove named volumes
				Force:         true,  // Force remove even if running
			}); err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to remove container %s: %v", containerName, err)
				continue
			}
			streamResult.LogChan <- fmt.Sprintf("Removed container: %s", containerName)
		}
	}

	// Clean up database records after successful Docker operations
	if err := repository.DeleteServiceContainers(ctx, db, serviceID); err != nil {
		return fmt.Errorf("failed to delete service containers from database: %w", err)
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully purged %d containers and cleaned database records", len(containers))
	return nil
}

func purgeNetworksFromDocker(ctx context.Context, dockerClient *client.Client, networks map[string]types.NetworkConfig, serviceID string, db pgx.Tx, generator *generator.NamingGenerator, streamResult *StreamingResult) error {
	// Get all networks from database first
	dbNetworks, err := repository.GetServiceNetworks(ctx, db, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service networks from database: %w", err)
	}

	if len(dbNetworks) == 0 {
		streamResult.LogChan <- "No networks found in database to purge"
		return nil
	}

	// Process each network from database
	for _, dbNetwork := range dbNetworks {
		networkName := dbNetwork.NetworkName
		streamResult.LogChan <- fmt.Sprintf("Checking network for purge: %s", networkName)

		// Find the actual Docker network by name
		dockerNetworks, err := dockerClient.NetworkList(ctx, network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", networkName)),
		})
		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Failed to list Docker networks: %v", err)
			continue
		}

		// Process each matching Docker network
		for _, dockerNetwork := range dockerNetworks {
			// Inspect network to check if it's in use
			networkInfo, err := dockerClient.NetworkInspect(ctx, dockerNetwork.ID, network.InspectOptions{})
			if err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to inspect network %s: %v", networkName, err)
				continue
			}

			// Check if network has connected containers
			if len(networkInfo.Containers) > 0 {
				streamResult.LogChan <- fmt.Sprintf("Network %s still in use by %d containers, skipping removal", networkName, len(networkInfo.Containers))
				continue
			}

			// Network is not in use, safe to remove
			streamResult.LogChan <- fmt.Sprintf("Removing network: %s", networkName)
			if err := dockerClient.NetworkRemove(ctx, dockerNetwork.ID); err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to remove network %s: %v", networkName, err)
				continue
			}
			streamResult.LogChan <- fmt.Sprintf("Removed network: %s", networkName)
		}
	}

	// Clean up database records after successful Docker operations
	if err := repository.DeleteServiceNetworks(ctx, db, serviceID); err != nil {
		return fmt.Errorf("failed to delete service networks from database: %w", err)
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully processed %d networks and cleaned database records", len(dbNetworks))
	return nil
}
