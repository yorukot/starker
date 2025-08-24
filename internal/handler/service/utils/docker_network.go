package utils

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"github.com/yorukot/starker/pkg/generator"
)

// createProjectNetworks creates networks defined in the compose project
func createProjectNetworks(ctx context.Context, dockerClient *client.Client, project *types.Project, streamResult *StreamingResult, namingGen *generator.NamingGenerator) error {
	for networkName, networkConfig := range project.Networks {
		fullNetworkName := namingGen.ResolveNetworkName(networkName, networkConfig.Name)

		// Check if network already exists
		networks, err := dockerClient.NetworkList(ctx, network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", fullNetworkName)),
		})
		if err != nil {
			return fmt.Errorf("failed to list networks: %w", err)
		}
		zap.L().Debug("Existing networks", zap.Any("networks", networks))
		if len(networks) == 0 {
			streamResult.StdoutChan <- fmt.Sprintf("Creating network: %s", fullNetworkName)
			_, err = dockerClient.NetworkCreate(ctx, fullNetworkName, network.CreateOptions{
				Driver: networkConfig.Driver,
				Labels: namingGen.GetNetworkLabels(project.Name, networkName),
			})
			if err != nil {
				return fmt.Errorf("failed to create network %s: %w", fullNetworkName, err)
			}
			streamResult.StdoutChan <- fmt.Sprintf("Created network: %s", fullNetworkName)
		} else {
			streamResult.StdoutChan <- fmt.Sprintf("Network %s already exists", fullNetworkName)
		}
	}

	return nil
}

// cleanupProjectNetworks removes networks created for the project if they're not in use
func cleanupProjectNetworks(ctx context.Context, dockerClient *client.Client, project *types.Project, streamResult *StreamingResult, namingGen *generator.NamingGenerator) error {
	// List networks for this project using generator filters
	fb := generator.NewFilterBuilder(namingGen)
	filterArgs := fb.ProjectFilters(project.Name)

	networks, err := dockerClient.NetworkList(ctx, network.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		// Check if network is in use by other containers
		networkInfo, err := dockerClient.NetworkInspect(ctx, net.ID, network.InspectOptions{})
		if err != nil {
			streamResult.StderrChan <- fmt.Sprintf("Failed to inspect network %s: %v", net.Name, err)
			continue
		}

		// If network has no connected containers, remove it
		if len(networkInfo.Containers) == 0 {
			streamResult.StdoutChan <- fmt.Sprintf("Removing network: %s", net.Name)
			if err := dockerClient.NetworkRemove(ctx, net.ID); err != nil {
				streamResult.StderrChan <- fmt.Sprintf("Failed to remove network %s: %v", net.Name, err)
				continue
			}
			streamResult.StdoutChan <- fmt.Sprintf("Removed network: %s", net.Name)
		} else {
			streamResult.StdoutChan <- fmt.Sprintf("Network %s still in use, keeping it", net.Name)
		}
	}

	return nil
}
