package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
)

func (dh *DockerHandler) StartDockerContainers(ctx context.Context, tx pgx.Tx) error {
	// Resolve service dependencies and get ordered startup sequence
	startupOrder, err := dh.resolveDependencyOrder()
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to resolve service dependencies: %v", err),
		}
		return fmt.Errorf("failed to resolve service dependencies: %w", err)
	}

	// Log the resolved startup order
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogStep,
		Message: fmt.Sprintf("Starting containers in dependency order: %v", startupOrder),
	}

	// Start containers in dependency-resolved order
	for _, serviceName := range startupOrder {
		service := dh.Project.Services[serviceName]

		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: fmt.Sprintf("Starting service: %s", serviceName),
		}

		// Generate the docker container name and create the Docker container
		containerID, err := dh.StartDockerContainer(ctx, serviceName, service)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to start docker container %s: %v", serviceName, err),
			}
			return fmt.Errorf("failed to start docker container %s (dependency chain broken): %w", serviceName, err)
		}

		// Update container state in database
		err = dh.UpdateContainerState(ctx, tx, containerID, models.ContainerStateRunning)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to update container state in database: %v", err),
			}
			return fmt.Errorf("failed to update container %s state in database: %w", serviceName, err)
		}

		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogTypeInfo,
			Message: fmt.Sprintf("Container %s created and saved successfully", serviceName),
		}
	}
	return nil
}

// StartDockerContainer creates and starts a Docker container and returns the container ID
func (dh *DockerHandler) StartDockerContainer(ctx context.Context, serviceName string, serviceConfig types.ServiceConfig) (containerID string, err error) {
	// Generate container name using naming generator
	containerName := dh.NamingGenerator.ContainerName(serviceName)

	// Generate project name and labels
	projectName := dh.NamingGenerator.ProjectName()
	labels := dh.NamingGenerator.GetServiceLabels(projectName, serviceName)

	// Log container creation start
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogStep,
		Message: fmt.Sprintf("Creating Docker container: %s", containerName),
	}

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
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to create Docker container %s: %v", containerName, err),
		}
		return "", fmt.Errorf("failed to create Docker container %s: %w", containerName, err)
	}

	// Start the container
	err = dh.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to start Docker container %s: %v", containerName, err),
		}
		return "", fmt.Errorf("failed to start Docker container %s: %w", containerName, err)
	}

	// Log successful creation and start
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogTypeInfo,
		Message: fmt.Sprintf("Successfully created and started Docker container: %s", resp.ID),
	}

	return resp.ID, nil
}

// StopDockerContainer stops a Docker container and updates its state in the database
func (dh *DockerHandler) StopDockerContainer(ctx context.Context, tx pgx.Tx, containerID, containerName string) error {
	// Log container stop start
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogStep,
		Message: fmt.Sprintf("Stopping Docker container: %s", containerName),
	}

	// Stop the Docker container
	timeout := 30
	err := dh.Client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to stop Docker container %s: %v", containerName, err),
		}
		return fmt.Errorf("failed to stop Docker container %s: %w", containerName, err)
	}

	// Update container state in database
	err = dh.UpdateContainerState(ctx, tx, containerID, models.ContainerStateStopped)
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to update container state in database: %v", err),
		}
		return fmt.Errorf("failed to update container state in database: %w", err)
	}

	// Log successful stop
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogTypeInfo,
		Message: fmt.Sprintf("Successfully stopped Docker container: %s", containerName),
	}

	return nil
}

// RestartDockerContainer restarts a Docker container and updates its state in the database
func (dh *DockerHandler) RestartDockerContainer(ctx context.Context, tx pgx.Tx, containerID, containerName string) error {
	// Log container restart start
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogStep,
		Message: fmt.Sprintf("Restarting Docker container: %s", containerName),
	}

	// Restart the Docker container
	timeout := 30
	err := dh.Client.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to restart Docker container %s: %v", containerName, err),
		}
		return fmt.Errorf("failed to restart Docker container %s: %w", containerName, err)
	}

	// Update container state in database
	err = dh.UpdateContainerState(ctx, tx, containerID, models.ContainerStateRunning)
	if err != nil {
		dh.StreamChan.ErrChan <- LogMessage{
			Type:    LogTypeError,
			Message: fmt.Sprintf("Failed to update container state in database: %v", err),
		}
		return fmt.Errorf("failed to update container state in database: %w", err)
	}

	// Log successful restart
	dh.StreamChan.LogChan <- LogMessage{
		Type:    LogTypeInfo,
		Message: fmt.Sprintf("Successfully restarted Docker container: %s", containerName),
	}

	return nil
}

// UpdateContainerState updates the state of a container in the database by service name
func (dh *DockerHandler) UpdateContainerState(ctx context.Context, tx pgx.Tx, containerID string, state models.ContainerState) error {
	// Get all service containers for this service
	serviceContainers, err := repository.GetServiceContainers(ctx, tx, dh.NamingGenerator.ServiceID())
	if err != nil {
		return fmt.Errorf("failed to get service containers from database: %w", err)
	}

	// Update all containers for this service (usually just one)
	updated := false
	for _, serviceContainer := range serviceContainers {
		// Update the container with the new ID and state
		serviceContainer.ContainerID = &containerID
		serviceContainer.State = state
		serviceContainer.UpdatedAt = time.Now()

		err = repository.UpdateServiceContainer(ctx, tx, serviceContainer)
		if err != nil {
			return fmt.Errorf("failed to update container state in database: %w", err)
		}

		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogTypeInfo,
			Message: fmt.Sprintf("Container state updated to %s in database", state),
		}

		updated = true
	}

	if !updated {
		return fmt.Errorf("no containers found for service %s in database", dh.NamingGenerator.ServiceID())
	}

	return nil
}

// resolveDependencyOrder resolves service dependencies using topological sorting
// Returns services in dependency order (dependencies first, dependents last)
func (dh *DockerHandler) resolveDependencyOrder() ([]string, error) {
	services := dh.Project.Services

	// Handle simple case: no dependencies
	if !hasDependencies(services) {
		serviceNames := make([]string, 0, len(services))
		for serviceName := range services {
			serviceNames = append(serviceNames, serviceName)
		}
		return serviceNames, nil
	}

	// Build dependency graph and perform topological sort
	graph := buildDependencyGraph(services)

	// Validate all dependencies exist
	if err := validateDependencies(graph, services); err != nil {
		return nil, err
	}

	// Detect circular dependencies
	if err := detectCircularDependencies(graph); err != nil {
		return nil, err
	}

	// Perform topological sort using Kahn's algorithm
	sortedServices, err := topologicalSort(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to sort dependencies: %w", err)
	}

	return sortedServices, nil
}

// hasDependencies checks if any service has dependencies
func hasDependencies(services types.Services) bool {
	for _, service := range services {
		if len(service.DependsOn) > 0 {
			return true
		}
	}
	return false
}

// buildDependencyGraph creates a dependency graph from services
func buildDependencyGraph(services types.Services) map[string][]string {
	graph := make(map[string][]string)

	// Initialize all services in the graph
	for serviceName := range services {
		if graph[serviceName] == nil {
			graph[serviceName] = []string{}
		}
	}

	// Add dependencies
	for serviceName, service := range services {
		for dependency := range service.DependsOn {
			// dependency -> serviceName (dependency points to dependent)
			graph[dependency] = append(graph[dependency], serviceName)
		}
	}

	return graph
}

// validateDependencies ensures all dependencies reference existing services
func validateDependencies(graph map[string][]string, services types.Services) error {
	for serviceName, service := range services {
		for dependency := range service.DependsOn {
			if _, exists := services[dependency]; !exists {
				return fmt.Errorf("service '%s' depends on non-existent service '%s'", serviceName, dependency)
			}
		}
	}
	return nil
}

// detectCircularDependencies uses DFS to detect cycles in the dependency graph
func detectCircularDependencies(graph map[string][]string) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var dfs func(string, []string) error
	dfs = func(node string, path []string) error {
		visited[node] = true
		recursionStack[node] = true

		for _, dependent := range graph[node] {
			if !visited[dependent] {
				if err := dfs(dependent, append(path, node)); err != nil {
					return err
				}
			} else if recursionStack[dependent] {
				// Found a cycle
				cycle := append(path, node, dependent)
				return fmt.Errorf("circular dependency detected: %v", cycle)
			}
		}

		recursionStack[node] = false
		return nil
	}

	for node := range graph {
		if !visited[node] {
			if err := dfs(node, []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// topologicalSort performs Kahn's algorithm to sort services by dependencies
func topologicalSort(graph map[string][]string) ([]string, error) {
	// Calculate in-degrees (number of dependencies for each service)
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for node := range graph {
		inDegree[node] = 0
	}

	// Calculate actual in-degrees
	for _, dependents := range graph {
		for _, dependent := range dependents {
			inDegree[dependent]++
		}
	}

	// Queue for nodes with no dependencies (in-degree 0)
	queue := []string{}
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	result := []string{}

	// Process nodes with no dependencies
	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree of dependent nodes
		for _, dependent := range graph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If result doesn't contain all nodes, there's a cycle
	if len(result) != len(graph) {
		return nil, fmt.Errorf("dependency cycle detected - unable to resolve startup order")
	}

	return result, nil
}
