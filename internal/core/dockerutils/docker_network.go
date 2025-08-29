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

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
)

func (dh *DockerHandler) StartDockerNetworks(ctx context.Context, tx pgx.Tx) error {

	for _, network := range dh.Project.Networks {
		// Generate the docker network name and create the Docker network
		networkID, err := dh.StartDockerNetwork(ctx, network)
		if err != nil {
			zap.L().Error("failed to start docker network", zap.Error(err), zap.String("network", network.Name))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to start docker network %s: %v", network.Name, err))
			return err
		}

		// Create the network record to database
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
			zap.L().Error("failed to save network to database", zap.Error(err), zap.String("network", network.Name))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to save network to database: %v", err))
			return fmt.Errorf("failed to save network %s to database: %w", network.Name, err)
		}

		dh.StreamChan.LogInfo(fmt.Sprintf("Network %s created and saved successfully", network.Name))
	}
	return nil
}

// StartDockerNetwork creates a Docker network and returns the network ID
func (dh *DockerHandler) StartDockerNetwork(ctx context.Context, networkConfig types.NetworkConfig) (networkID string, err error) {
	// Generate network name using naming generator
	networkName := dh.NamingGenerator.NetworkName(networkConfig.Name)

	// Check if network already exists
	networkResource, err := dh.Client.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err == nil {
		dh.StreamChan.LogInfo(fmt.Sprintf("Docker network %s already exists, using existing network", networkName))
		return networkResource.ID, nil
	}

	// Generate project name and labels
	projectName := dh.NamingGenerator.ProjectName()
	labels := dh.NamingGenerator.GetNetworkLabels(projectName, networkConfig.Name)

	// Log network creation start
	dh.StreamChan.LogStep(fmt.Sprintf("Creating Docker network: %s", networkName))

	// Prepare network creation options
	createOptions := network.CreateOptions{
		Labels:     labels,
		Driver:     networkConfig.Driver,
		Options:    networkConfig.DriverOpts,
		Internal:   networkConfig.Internal,
		Attachable: networkConfig.Attachable,
		EnableIPv4: networkConfig.EnableIPv4,
		EnableIPv6: networkConfig.EnableIPv6,
	}

	// Create the Docker network
	dockerNetwork, err := dh.Client.NetworkCreate(ctx, networkName, createOptions)
	if err != nil {
		zap.L().Error("failed to create Docker network", zap.Error(err), zap.String("network", networkName))
		dh.StreamChan.LogError(fmt.Sprintf("Failed to create Docker network %s: %v", networkName, err))
		return "", fmt.Errorf("failed to create Docker network %s: %w", networkName, err)
	}

	// Log successful creation
	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully created Docker network: %s", dockerNetwork.ID))

	return dockerNetwork.ID, nil
}

// RemoveDockerNetworks removes all Docker networks associated with the project
func (dh *DockerHandler) RemoveDockerNetworks(ctx context.Context, tx pgx.Tx) error {

	// Get all service networks from database
	serviceNetworks, err := repository.GetServiceNetworks(ctx, tx, dh.NamingGenerator.ServiceID())
	if err != nil {
		zap.L().Error("failed to get service networks from database", zap.Error(err))
		dh.StreamChan.LogError(fmt.Sprintf("Failed to get service networks from database: %v", err))
		return fmt.Errorf("failed to get service networks from database: %w", err)
	}

	// Remove each Docker network
	for _, serviceNetwork := range serviceNetworks {
		if serviceNetwork.NetworkID == nil {
			continue
		}
		useByOther, err := dh.RemoveDockerNetwork(ctx, *serviceNetwork.NetworkID, serviceNetwork.NetworkName)
		if err != nil {
			zap.L().Error("failed to remove docker network", zap.Error(err), zap.String("network", serviceNetwork.NetworkName))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to remove Docker network %s: %v", serviceNetwork.NetworkName, err))
			return err
		}
		if useByOther {
			// If the network is still in use, we can't remove it
			dh.StreamChan.LogInfo(fmt.Sprintf("Docker network %s is still in use by other containers, skipping removal", serviceNetwork.NetworkName))
			continue
		}
	}

	// Delete all network records from database
	err = repository.DeleteServiceNetworks(ctx, tx, dh.NamingGenerator.ServiceID())
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to delete service networks from database: %v", err))
		return fmt.Errorf("failed to delete service networks from database: %w", err)
	}

	dh.StreamChan.LogInfo("All Docker networks removed successfully")

	return nil
}

// RemoveDockerNetwork removes a Docker network by ID and returns any error
func (dh *DockerHandler) RemoveDockerNetwork(ctx context.Context, networkID, networkName string) (useByOther bool, err error) {
	// Log network removal start
	dh.StreamChan.LogStep(fmt.Sprintf("Removing Docker network: %s", networkName))

	// Check if the network is using by other containers
	networkInspect, err := dh.Client.NetworkInspect(ctx, networkID, network.InspectOptions{})
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to inspect Docker network %s: %v", networkName, err))
		return false, fmt.Errorf("failed to inspect Docker network %s: %w", networkName, err)
	}
	if len(networkInspect.Containers) > 0 {
		dh.StreamChan.LogError(fmt.Sprintf("Docker network %s is still in use by other containers", networkName))
		return true, fmt.Errorf("docker network %s is still in use by other containers", networkName)
	}

	// Remove the Docker network
	err = dh.Client.NetworkRemove(ctx, networkID)
	if err != nil {
		dh.StreamChan.LogError(fmt.Sprintf("Failed to remove Docker network %s: %v", networkName, err))
		return false, fmt.Errorf("failed to remove Docker network %s: %w", networkName, err)
	}

	// Log successful removal
	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully removed Docker network: %s", networkName))

	return false, nil
}
