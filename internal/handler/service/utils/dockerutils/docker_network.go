package dockerutils

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

// createProjectNetwork creates networks defined in the compose project
func createProjectNetwork(ctx context.Context, dockerClient *client.Client, networkKey string, networkConfig types.NetworkConfig, streamResult *StreamingResult, namingGen *generator.NamingGenerator) error {
	// Check if network already exists
	zap.L().With(zap.Any("network", networkConfig)).Debug("Checking existing networks before creation")
	networks, err := dockerClient.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", networkConfig.Name)),
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}
	zap.L().With(zap.String("network", networkConfig.Name)).Debug("Checking existing networks")
	zap.L().Debug("Existing networks", zap.Any("networks", networks))
	if len(networks) == 0 {
		streamResult.LogChan <- fmt.Sprintf("Creating network: %s", networkConfig.Name)
		_, err = dockerClient.NetworkCreate(ctx, networkConfig.Name, network.CreateOptions{
			Driver: networkConfig.Driver,
			Labels: namingGen.GetNetworkLabels(namingGen.ProjectName(), networkKey),
		})
		if err != nil {
			return fmt.Errorf("failed to create network %s: %w", networkConfig.Name, err)
		}
		streamResult.LogChan <- fmt.Sprintf("Created network: %s", networkConfig.Name)
	} else {
		streamResult.LogChan <- fmt.Sprintf("Network %s already exists", networkConfig.Name)
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
			streamResult.LogChan <- fmt.Sprintf("Failed to inspect network %s: %v", net.Name, err)
			continue
		}

		// If network has no connected containers, remove it
		if len(networkInfo.Containers) == 0 {
			streamResult.LogChan <- fmt.Sprintf("Removing network: %s", net.Name)
			if err := dockerClient.NetworkRemove(ctx, net.ID); err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to remove network %s: %v", net.Name, err)
				continue
			}
			streamResult.LogChan <- fmt.Sprintf("Removed network: %s", net.Name)
		} else {
			streamResult.LogChan <- fmt.Sprintf("Network %s still in use, keeping it", net.Name)
		}
	}

	return nil
}
