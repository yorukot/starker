package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/network"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
)

func (dh *DockerHandler) StartDockerNetworks(ctx context.Context, tx pgx.Tx) error {

	for _, network := range dh.Project.Networks {
		// Generate the docker network name and create the Docker network
		networkID, err := dh.StartDockerNetwork(ctx, network)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to start docker network %s: %v", network.Name, err))
			return err
		}

		// Create the network record in database
		serviceNetwork := models.ServiceNetwork{
			ID:          ksuid.New().String(),
			ServiceID:   dh.NamingGenerator.ServiceID(),
			NetworkID:   &networkID,
			NetworkName: dh.NamingGenerator.NetworkName(network.Name),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = repository.CreateServiceNetwork(ctx, tx, serviceNetwork)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to save network to database: %v", err))
			return fmt.Errorf("failed to save network %s to database: %w", network.Name, err)
		}

		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Network %s created and saved successfully", network.Name))
	}
	return nil
}

// checkNetworkExists checks if a Docker network exists and returns its ID
func (dh *DockerHandler) checkNetworkExists(ctx context.Context, networkName string) (string, bool) {
	networkResource, err := dh.Client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		return "", false
	}
	return networkResource.ID, true
}

// StartDockerNetwork creates a Docker network and returns the network ID
func (dh *DockerHandler) StartDockerNetwork(ctx context.Context, networkConfig types.NetworkConfig) (networkID string, err error) {
	// Generate network name using naming generator
	networkName := dh.NamingGenerator.NetworkName(networkConfig.Name)

	zap.L().Debug(fmt.Sprintf("Creating Docker network: %s", networkName))

	// Check if network already exists
	if existingID, exists := dh.checkNetworkExists(ctx, networkName); exists {
		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Docker network %s already exists, using existing network", networkName))
		return existingID, nil
	}

	// Generate project name and labels
	projectName := dh.NamingGenerator.ProjectName()
	labels := dh.NamingGenerator.GetNetworkLabels(projectName, networkConfig.Name)

	// Log network creation start
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Creating Docker network: %s", networkName))

	// Convert IPAM configuration if present
	var ipam *network.IPAM
	if networkConfig.Ipam.Driver != "" || len(networkConfig.Ipam.Config) > 0 {
		ipamConfig := make([]network.IPAMConfig, len(networkConfig.Ipam.Config))
		for i, pool := range networkConfig.Ipam.Config {
			ipamConfig[i] = network.IPAMConfig{
				Subnet:     pool.Subnet,
				IPRange:    pool.IPRange,
				Gateway:    pool.Gateway,
				AuxAddress: pool.AuxiliaryAddresses,
			}
		}
		ipam = &network.IPAM{
			Driver: networkConfig.Ipam.Driver,
			Config: ipamConfig,
		}
	}

	// Prepare network creation options
	createOptions := network.CreateOptions{
		Labels:     labels,
		Driver:     networkConfig.Driver,
		Options:    networkConfig.DriverOpts,
		Internal:   networkConfig.Internal,
		Attachable: networkConfig.Attachable,
		EnableIPv4: networkConfig.EnableIPv4,
		EnableIPv6: networkConfig.EnableIPv6,
		IPAM:       ipam,
	}

	// Create the Docker network
	dockerNetwork, err := dh.Client.NetworkCreate(ctx, networkName, createOptions)
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to create Docker network %s: %v", networkName, err))
		return "", fmt.Errorf("failed to create Docker network %s: %w", networkName, err)
	}

	// Log successful creation
	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully created Docker network: %s", dockerNetwork.ID))

	return dockerNetwork.ID, nil
}

func (dh *DockerHandler) RemoveDockerNetworks(ctx context.Context, tx pgx.Tx) error {

	// Get all service networks from database
	serviceNetworks, err := repository.GetServiceNetworks(ctx, tx, dh.NamingGenerator.ServiceID())
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to get service networks from database: %v", err))
		return fmt.Errorf("failed to get service networks from database: %w", err)
	}

	// Remove each Docker network
	for _, serviceNetwork := range serviceNetworks {
		if serviceNetwork.NetworkID == nil {
			continue
		}
		err := dh.RemoveDockerNetwork(ctx, *serviceNetwork.NetworkID, serviceNetwork.NetworkName)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove Docker network %s: %v", serviceNetwork.NetworkName, err))
			return err
		}
	}

	// Delete all network records from database
	err = repository.DeleteServiceNetworks(ctx, tx, dh.NamingGenerator.ServiceID())
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to delete service networks from database: %v", err))
		return fmt.Errorf("failed to delete service networks from database: %w", err)
	}

	dh.StreamChan.LogChan <- core.LogInfo("All Docker networks removed successfully")

	return nil
}

// RemoveDockerNetwork removes a Docker network by ID and returns any error
func (dh *DockerHandler) RemoveDockerNetwork(ctx context.Context, networkID, networkName string) error {
	// Log network removal start
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Removing Docker network: %s", networkName))

	// Remove the Docker network
	err := dh.Client.NetworkRemove(ctx, networkID)
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to remove Docker network %s: %v", networkName, err))
		return fmt.Errorf("failed to remove Docker network %s: %w", networkName, err)
	}

	// Log successful removal
	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully removed Docker network: %s", networkName))

	return nil
}
