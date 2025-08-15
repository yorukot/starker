package models

import "time"

// Service represents a service definition with Docker containers
type Service struct {
	ID             string     `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                      // Unique identifier for the service
	TeamID         string     `json:"team_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                 // Associated team ID
	ServerID       string     `json:"server_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`               // Associated server ID
	ProjectID      string     `json:"project_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`              // Associated project ID
	Name           string     `json:"name" example:"web-app"`                                       // Service name
	Description    *string    `json:"description,omitempty" example:"Main web application service"` // Service description
	Type           string     `json:"type" example:"docker"`                                        // Service type (e.g., docker, compose)
	Status         string     `json:"status" example:"running"`                                     // Service status (running, stopped, etc.)
	ContainerID    *string    `json:"container_id,omitempty" example:"abc123..."`                   // Docker container ID
	LastDeployedAt *time.Time `json:"last_deployed_at,omitempty" example:"2023-01-01T12:00:00Z"`    // Timestamp when the service was last deployed
	CreatedAt      time.Time  `json:"created_at" example:"2023-01-01T12:00:00Z"`                    // Timestamp when the service was created
	UpdatedAt      time.Time  `json:"updated_at" example:"2023-01-01T12:00:00Z"`                    // Timestamp when the service was last updated
}

// ServiceEnvironmentVariable represents environment variables for services
type ServiceEnvironmentVariable struct {
	ID        string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`         // Unique identifier for the environment variable
	ServiceID string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"` // Associated service ID
	Key       string    `json:"key" example:"DATABASE_URL"`                      // Environment variable key
	Value     string    `json:"value" example:"postgres://..."`                  // Environment variable value
	IsSecret  bool      `json:"is_secret" example:"true"`                        // Whether this is a secret value
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the variable was created
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`       // Timestamp when the variable was last updated
}

// ServiceComposeConfig represents Docker compose configurations
type ServiceComposeConfig struct {
	ID              string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                           // Unique identifier for the compose config
	ServiceID       string    `json:"service_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                   // Associated service ID
	ComposeFile     string    `json:"compose_file" example:"version: '3.8'..."`                          // Docker compose file content
	ComposeFilePath *string   `json:"compose_file_path,omitempty" example:"/opt/app/docker-compose.yml"` // Path to compose file on server
	CreatedAt       time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`                         // Timestamp when the config was created
	UpdatedAt       time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`                         // Timestamp when the config was last updated
}
