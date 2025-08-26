package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types/filters"
)

const (
	StarkerPrefix = "starker"
	MaxNameLength = 50
)

type NamingGenerator struct {
	serviceID string
	teamID    string
	serverID  string
}

func NewNamingGenerator(serviceID, teamID, serverID string) *NamingGenerator {
	return &NamingGenerator{
		serviceID: serviceID,
		teamID:    teamID,
		serverID:  serverID,
	}
}

func (ng *NamingGenerator) ProjectName() string {
	sanitizedServiceID := sanitizeProjectName(ng.serviceID)
	return fmt.Sprintf("%s-%s", StarkerPrefix, sanitizedServiceID)
}

func (ng *NamingGenerator) ContainerName(serviceName string) string {
	return fmt.Sprintf("%s-%s", serviceName, ng.serviceID)
}

func (ng *NamingGenerator) NetworkName(networkName string) string {
	return fmt.Sprintf("%s-%s", networkName, ng.serviceID)
}

// ResolveNetworkName resolves the correct network name based on Docker Compose network configuration
// This ensures consistency between network creation and container network attachment
func (ng *NamingGenerator) ResolveNetworkName(networkName, configuredName string) string {
	// If the network has an explicit name in the compose config, use it
	if configuredName != "" {
		return configuredName
	}
	// Otherwise, generate the name using our naming convention
	return ng.NetworkName(networkName)
}

func (ng *NamingGenerator) VolumeName(volumeName string) string {
	return fmt.Sprintf("%s-%s", volumeName, ng.serviceID)
}

func (ng *NamingGenerator) ConnectionID() string {
	return fmt.Sprintf("%s-%s", ng.teamID, ng.serverID)
}

func (ng *NamingGenerator) GetLabels() map[string]string {
	return map[string]string{
		"starker.service.id": ng.serviceID,
		"starker.team.id":    ng.teamID,
		"starker.server.id":  ng.serverID,
	}
}

func (ng *NamingGenerator) GetComposeLabels(projectName string) map[string]string {
	labels := ng.GetLabels()
	labels["com.docker.compose.project"] = projectName
	return labels
}

func (ng *NamingGenerator) GetServiceLabels(projectName, serviceName string) map[string]string {
	labels := ng.GetComposeLabels(projectName)
	labels["com.docker.compose.service"] = serviceName
	return labels
}

func (ng *NamingGenerator) GetNetworkLabels(projectName, networkName string) map[string]string {
	labels := ng.GetComposeLabels(projectName)
	labels["com.docker.compose.network"] = networkName
	return labels
}

func (ng *NamingGenerator) GetVolumeLabels(projectName, volumeName string) map[string]string {
	labels := ng.GetComposeLabels(projectName)
	labels["com.docker.compose.volume"] = volumeName
	return labels
}

func (ng *NamingGenerator) GenerateServiceDataPath() string {
	return fmt.Sprintf("/data/starker/services/%s", ng.serviceID)
}

func sanitizeProjectName(name string) string {
	name = strings.ToLower(name)

	reg := regexp.MustCompile(`[^a-z0-9_-]`)
	name = reg.ReplaceAllString(name, "-")

	if len(name) > 0 && !regexp.MustCompile(`^[a-z0-9]`).MatchString(name) {
		name = "s" + name
	}

	if len(name) > MaxNameLength {
		name = name[:MaxNameLength]
	}

	return name
}

type FilterBuilder struct {
	generator *NamingGenerator
}

func NewFilterBuilder(generator *NamingGenerator) *FilterBuilder {
	return &FilterBuilder{generator: generator}
}

func (fb *FilterBuilder) ServiceFilters(projectName, serviceName string) filters.Args {
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))
	args.Add("label", fmt.Sprintf("com.docker.compose.service=%s", serviceName))
	args.Add("label", fmt.Sprintf("starker.service.id=%s", fb.generator.serviceID))
	return args
}

func (fb *FilterBuilder) ProjectFilters(projectName string) filters.Args {
	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))
	args.Add("label", fmt.Sprintf("starker.service.id=%s", fb.generator.serviceID))
	return args
}

func (fb *FilterBuilder) NetworkFilters(projectName string) filters.Args {
	return fb.ProjectFilters(projectName)
}

func (fb *FilterBuilder) VolumeFilters(projectName string) filters.Args {
	return fb.ProjectFilters(projectName)
}
