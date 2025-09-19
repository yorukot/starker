package dockeryaml

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DockerContainerStatus represents the JSON structure returned by docker compose ps --format json
type DockerContainerStatus struct {
	State   string `json:"State"`
	Service string `json:"Service"`
	Name    string `json:"Name"`
	ID      string `json:"ID"`
}

// ParseContainerStatuses parses Docker Compose container status output and returns container statuses
func ParseContainerStatuses(stdout string) ([]DockerContainerStatus, error) {
	var containers []DockerContainerStatus

	if stdout == "" {
		return containers, nil
	}

	// Docker compose returns one JSON object per line when there are containers
	lines := strings.SplitSeq(strings.TrimSpace(stdout), "\n")
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var container DockerContainerStatus
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			return nil, fmt.Errorf("failed to parse container status JSON: %w", err)
		}
		containers = append(containers, container)
	}

	return containers, nil
}
