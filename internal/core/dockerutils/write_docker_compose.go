package dockerutils

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteDockerCompose inject some label and necessary info into docker-compose.yml
// file and write it to the server file system at the /data/starker/{serviceID}/docker-compose.yml
func (h *DockerHandler) WriteDockerCompose() error {
	// Inject Starker labels and metadata into the compose file
	modifiedComposeContent, err := h.injectDockerCompose()
	if err != nil {
		return fmt.Errorf("failed to inject compose metadata: %w", err)
	}

	// Write the modified compose file to the remote server
	if err := h.writeToFile(modifiedComposeContent); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

func (h *DockerHandler) injectDockerCompose() (string, error) {
	if h.Project == nil {
		return "", fmt.Errorf("project is nil")
	}

	if h.NamingGenerator == nil {
		return "", fmt.Errorf("naming generator is nil")
	}

	// Get the project name from naming generator
	projectName := h.NamingGenerator.ProjectName()

	// Create a copy of the project for modification
	modifiedProject := *h.Project
	modifiedProject.Name = projectName

	// Apply Starker labels to all services
	for serviceName, service := range modifiedProject.Services {
		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}

		// Add Starker-specific labels
		starkerLabels := h.NamingGenerator.GetServiceLabels(projectName, serviceName)
		for key, value := range starkerLabels {
			service.Labels[key] = value
		}

		// Update container name with Starker naming convention
		service.ContainerName = h.NamingGenerator.ContainerName(serviceName)

		modifiedProject.Services[serviceName] = service
	}

	// Apply Starker labels to all networks
	for networkName, network := range modifiedProject.Networks {
		if network.Labels == nil {
			network.Labels = make(map[string]string)
		}

		// Add Starker-specific labels
		networkLabels := h.NamingGenerator.GetNetworkLabels(projectName, networkName)
		for key, value := range networkLabels {
			network.Labels[key] = value
		}

		// Update network name if not explicitly set
		if network.Name == "" {
			network.Name = h.NamingGenerator.NetworkName(networkName)
		}

		modifiedProject.Networks[networkName] = network
	}

	// Apply Starker labels to all volumes
	for volumeName, volume := range modifiedProject.Volumes {
		if volume.Labels == nil {
			volume.Labels = make(map[string]string)
		}

		// Add Starker-specific labels
		volumeLabels := h.NamingGenerator.GetVolumeLabels(projectName, volumeName)
		for key, value := range volumeLabels {
			volume.Labels[key] = value
		}

		// Update volume name if not explicitly set
		if volume.Name == "" {
			volume.Name = h.NamingGenerator.VolumeName(volumeName)
		}

		modifiedProject.Volumes[volumeName] = volume
	}

	// Convert the modified project back to YAML
	yamlBytes, err := yaml.Marshal(&modifiedProject)
	if err != nil {
		return "", fmt.Errorf("failed to marshal compose to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (h *DockerHandler) writeToFile(composeContent string) error {
	if h.Client == nil {
		return fmt.Errorf("SSH client is nil")
	}

	if h.NamingGenerator == nil {
		return fmt.Errorf("naming generator is nil")
	}

	// Get the service data path
	serviceDataPath := h.NamingGenerator.GenerateServiceDataPath()
	composeFilePath := filepath.Join(serviceDataPath, "compose.yml")

	// Create SSH session
	session, err := h.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Create the service directory if it doesn't exist
	mkdirCmd := fmt.Sprintf("mkdir -p %s", serviceDataPath)
	if err := session.Run(mkdirCmd); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Create a new session for writing the file
	session2, err := h.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create second SSH session: %w", err)
	}
	defer session2.Close()

	// Write the compose file content
	writeCmd := fmt.Sprintf("cat > %s", composeFilePath)
	session2.Stdin = strings.NewReader(composeContent)
	if err := session2.Run(writeCmd); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Create a third session to set permissions
	session3, err := h.Client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create third SSH session: %w", err)
	}
	defer session3.Close()

	// Set appropriate permissions (readable and writable by owner)
	chmodCmd := fmt.Sprintf("chmod 644 %s", composeFilePath)
	if err := session3.Run(chmodCmd); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
