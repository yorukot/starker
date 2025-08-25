package models

import "time"

type ServiceState string

const (
	ServiceStateRunning    ServiceState = "running"
	ServiceStateStopped    ServiceState = "stopped"
	ServiceStateStarting   ServiceState = "starting"
	ServiceStateStopping   ServiceState = "stopping"
	ServiceStateRestarting ServiceState = "restarting"
)

// Service represents a service definition with Docker containers
type Service struct {
	ID             string       `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                      // Unique identifier for the service
	TeamID         string       `json:"team_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                 // Associated team ID
	ServerID       string       `json:"server_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`               // Associated server ID
	ProjectID      string       `json:"project_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`              // Associated project ID
	Name           string       `json:"name" example:"web-app"`                                       // Service name
	Description    *string      `json:"description,omitempty" example:"Main web application service"` // Service description
	Type           string       `json:"type" example:"docker"`                                        // Service type (e.g., docker, compose)
	State          ServiceState `json:"state" example:"running"`                                      // Service state (running, stopped, etc.)
	ContainerID    *string      `json:"container_id,omitempty" example:"abc123..."`                   // Docker container ID
	LastDeployedAt *time.Time   `json:"last_deployed_at,omitempty" example:"2023-01-01T12:00:00Z"`    // Timestamp when the service was last deployed
	CreatedAt      time.Time    `json:"created_at" example:"2023-01-01T12:00:00Z"`                    // Timestamp when the service was created
	UpdatedAt      time.Time    `json:"updated_at" example:"2023-01-01T12:00:00Z"`                    // Timestamp when the service was last updated
}

// ServiceComposeConfig represents Docker compose configurations
type ServiceComposeConfig struct {
	ID          string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the compose config
	ServiceID   string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	ComposeFile string    `json:"compose_file" example:"version: '3.8'..."`        // Docker compose file content
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the config was created
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the config was last updated
}

// ServiceSourceGit represents Git source configuration for services
type ServiceSourceGit struct {
	ID                    string     `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                         // Unique identifier for the Git source
	ServiceID             string     `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                 // Associated service ID
	RepoURL               string     `json:"repo_url" example:"https://github.com/user/repo.git"`             // Git repository URL
	Branch                string     `json:"branch" example:"main"`                                           // Git branch
	AutoDeploy            bool       `json:"auto_deploy" example:"true"`                                      // Whether to auto-deploy on changes
	DockerComposeFilePath *string    `json:"docker_compose_file_path,omitempty" example:"docker-compose.yml"` // Path to Docker Compose file in repo
	WebhookSecret         string     `json:"webhook_secret" example:"secret123"`                              // Webhook secret for Git events
	UpdatedAt             *time.Time `json:"updated_at,omitempty" example:"2023-01-01T12:00:00Z"`             // Timestamp when the source was last updated
	CreatedAt             *time.Time `json:"created_at,omitempty" example:"2023-01-01T12:00:00Z"`             // Timestamp when the source was created
}

// ServiceContainer represents Docker containers associated with a service
type ServiceContainer struct {
	ID            string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the container record
	ServiceID     string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	ContainerID   string    `json:"container_id" example:"abc123def456"`             // Docker container ID
	ContainerName string    `json:"container_name" example:"web-app-container"`      // Docker container name
	UpdatedAt     time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the container was last updated
	CreatedAt     time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the container was created
}

// ServiceImage represents Docker images associated with a service
type ServiceImage struct {
	ID        string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the image record
	ServiceID string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	ImageID   *string   `json:"image_id,omitempty" example:"sha256:abc123..."`   // Docker image ID
	ImageName string    `json:"image_name" example:"nginx:latest"`               // Docker image name
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the image was last updated
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the image was created
}

// ServiceNetwork represents Docker networks associated with a service
type ServiceNetwork struct {
	ID          string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the network record
	ServiceID   string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	NetworkID   *string   `json:"network_id,omitempty" example:"abc123def456"`     // Docker network ID
	NetworkName string    `json:"network_name" example:"my-app-network"`           // Docker network name
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the network was last updated
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the network was created
}

// ServiceVolume represents Docker volumes associated with a service
type ServiceVolume struct {
	ID         string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the volume record
	ServiceID  string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	VolumeID   *string   `json:"volume_id,omitempty" example:"abc123def456"`      // Docker volume ID
	VolumeName string    `json:"volume_name" example:"my-app-data"`               // Docker volume name
	UpdatedAt  time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the volume was last updated
	CreatedAt  time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the volume was created
}
