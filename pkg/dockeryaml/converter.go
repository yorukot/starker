package dockeryaml

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"

	"github.com/yorukot/starker/pkg/generator"
)

// ConvertToPorts converts Docker Compose port configurations to Docker API port types
func ConvertToPorts(ports []types.ServicePortConfig) (nat.PortMap, nat.PortSet, error) {
	portBindings := make(nat.PortMap)
	exposedPorts := make(nat.PortSet)

	for _, port := range ports {
		containerPort, err := nat.NewPort(port.Protocol, fmt.Sprintf("%d", port.Target))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid container port %d/%s: %w", port.Target, port.Protocol, err)
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

	return portBindings, exposedPorts, nil
}

// ConvertToEnvironment converts Docker Compose environment variables to Docker API format
func ConvertToEnvironment(environment types.MappingWithEquals) []string {
	env := make([]string, 0, len(environment))
	for key, value := range environment {
		if value != nil {
			env = append(env, fmt.Sprintf("%s=%s", key, *value))
		}
	}
	return env
}

// ConvertToContainerConfig creates a Docker container configuration from service config
func ConvertToContainerConfig(serviceConfig types.ServiceConfig, exposedPorts nat.PortSet, env []string, labels map[string]string) *container.Config {
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

	return containerConfig
}

// ConvertToHostConfig creates a Docker host configuration from service config
func ConvertToHostConfig(serviceConfig types.ServiceConfig, portBindings nat.PortMap) *container.HostConfig {
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

	return hostConfig
}

// ConvertToNetworkConfig creates a Docker network configuration from service config
func ConvertToNetworkConfig(serviceConfig types.ServiceConfig, namingGenerator *generator.NamingGenerator) *network.NetworkingConfig {
	networkConfig := &network.NetworkingConfig{}
	if len(serviceConfig.Networks) > 0 {
		networkConfig.EndpointsConfig = make(map[string]*network.EndpointSettings)
		for networkName := range serviceConfig.Networks {
			resolvedNetworkName := namingGenerator.ResolveNetworkName(networkName, "")
			networkConfig.EndpointsConfig[resolvedNetworkName] = &network.EndpointSettings{}
		}
	}
	return networkConfig
}

// ConvertToDockerConfigs converts a Docker Compose service configuration to Docker API configurations
func ConvertToDockerConfigs(serviceConfig types.ServiceConfig, labels map[string]string, namingGenerator *generator.NamingGenerator) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	// Convert port configurations
	portBindings, exposedPorts, err := ConvertToPorts(serviceConfig.Ports)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to convert ports: %w", err)
	}

	// Convert environment variables
	env := ConvertToEnvironment(serviceConfig.Environment)

	// Create container configuration
	containerConfig := ConvertToContainerConfig(serviceConfig, exposedPorts, env, labels)

	// Create host configuration
	hostConfig := ConvertToHostConfig(serviceConfig, portBindings)

	// Create network configuration
	networkConfig := ConvertToNetworkConfig(serviceConfig, namingGenerator)

	return containerConfig, hostConfig, networkConfig, nil
}