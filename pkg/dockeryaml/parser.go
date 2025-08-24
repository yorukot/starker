package dockeryaml

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// ComposeFile represents a parsed Docker Compose file
type ComposeFile struct {
	Project *types.Project
}

// ServiceInfo contains essential service information with Docker API compatibility
type ServiceInfo struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Ports       []string          `json:"ports"`
	Environment map[string]string `json:"environment"`
	Volumes     []string          `json:"volumes"`
	Networks    []string          `json:"networks"`
	// Raw Docker API types for direct access
	PortConfigs   []types.ServicePortConfig              `json:"port_configs"`
	VolumeConfigs []types.ServiceVolumeConfig            `json:"volume_configs"`
	NetworkMap    map[string]*types.ServiceNetworkConfig `json:"network_map"`
}

// ParseComposeContent parses Docker Compose YAML content and returns a ComposeFile
func ParseComposeContent(yamlContent string, projectName string) (*ComposeFile, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("compose content cannot be empty")
	}

	configFiles := []types.ConfigFile{
		{
			Filename: "docker-compose.yml",
			Content:  []byte(yamlContent),
		},
	}

	project, err := loader.LoadWithContext(context.Background(), types.ConfigDetails{
		ConfigFiles: configFiles,
		WorkingDir:  ".",
	}, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &ComposeFile{
		Project: project,
	}, nil
}

// ParseComposeFile parses a Docker Compose file from filesystem path
func ParseComposeFile(filePath string) (*ComposeFile, error) {
	options, err := cli.NewProjectOptions(
		[]string{filePath},
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithName("file-project"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	project, err := options.LoadProject(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &ComposeFile{
		Project: project,
	}, nil
}

// GetServices returns simplified service information with Docker API compatibility
func (cf *ComposeFile) GetServices() []ServiceInfo {
	var services []ServiceInfo

	for _, service := range cf.Project.Services {
		serviceInfo := ServiceInfo{
			Name:        service.Name,
			Image:       service.Image,
			Ports:       extractPorts(service.Ports),
			Environment: extractEnvironment(service.Environment),
			Volumes:     extractVolumes(service.Volumes),
			Networks:    extractNetworks(service.Networks),
			// Include raw Docker API types for direct access
			PortConfigs:   service.Ports,
			VolumeConfigs: service.Volumes,
			NetworkMap:    service.Networks,
		}
		services = append(services, serviceInfo)
	}

	return services
}

// GetProject returns the underlying Docker Compose project for direct API access
func (cf *ComposeFile) GetProject() *types.Project {
	return cf.Project
}

// GetServiceNames returns a list of service names
func (cf *ComposeFile) GetServiceNames() []string {
	var names []string
	for _, service := range cf.Project.Services {
		names = append(names, service.Name)
	}
	return names
}

// Validate checks if the compose file is valid
func (cf *ComposeFile) Validate() error {
	if cf.Project == nil {
		return fmt.Errorf("project is nil")
	}

	if len(cf.Project.Services) == 0 {
		return fmt.Errorf("compose file must contain at least one service")
	}

	for _, service := range cf.Project.Services {
		if service.Name == "" {
			return fmt.Errorf("service name cannot be empty")
		}
		if service.Image == "" && service.Build == nil {
			return fmt.Errorf("service '%s' must specify either image or build", service.Name)
		}
	}

	return nil
}

// extractPorts converts port configurations to string slice
func extractPorts(ports []types.ServicePortConfig) []string {
	var result []string
	for _, port := range ports {
		if port.Published != "" && port.Target != 0 {
			result = append(result, fmt.Sprintf("%s:%d", port.Published, port.Target))
		} else if port.Target != 0 {
			result = append(result, fmt.Sprintf("%d", port.Target))
		}
	}
	return result
}

// extractEnvironment converts environment variables to map
func extractEnvironment(env types.MappingWithEquals) map[string]string {
	result := make(map[string]string)
	for key, value := range env {
		if value != nil {
			result[key] = *value
		} else {
			result[key] = ""
		}
	}
	return result
}

// extractVolumes converts volume configurations to string slice
func extractVolumes(volumes []types.ServiceVolumeConfig) []string {
	var result []string
	for _, volume := range volumes {
		switch volume.Type {
		case types.VolumeTypeBind, types.VolumeTypeVolume:
			result = append(result, fmt.Sprintf("%s:%s", volume.Source, volume.Target))
		default:
			result = append(result, volume.Target)
		}
	}
	return result
}

// extractNetworks converts network configurations to string slice
func extractNetworks(networks map[string]*types.ServiceNetworkConfig) []string {
	var result []string
	for networkName := range networks {
		result = append(result, networkName)
	}
	return result
}
