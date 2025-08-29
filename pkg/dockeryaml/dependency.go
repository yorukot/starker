package dockeryaml

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// ResolveDependencyOrder resolves service dependencies using topological sorting
// Returns services in dependency order (dependencies first, dependents last)
func ResolveDependencyOrder(services types.Services) ([]string, error) {
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
func validateDependencies(_ map[string][]string, services types.Services) error {
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
