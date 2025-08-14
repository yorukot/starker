package models

import "time"

// Server represents a server configuration
type Server struct {
	ID           string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                // Unique identifier for the server
	TeamID       string    `json:"team_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`           // Associated team ID
	Name         string    `json:"name" example:"Production Server"`                       // Server name
	Description  *string   `json:"description,omitempty" example:"Main production server"` // Server description
	IP           string    `json:"ip" example:"192.168.1.100"`                             // Server IP address
	Port         string    `json:"port" example:"22"`                                      // SSH port
	User         string    `json:"user" example:"ubuntu"`                                  // SSH username
	PrivateKeyID string    `json:"private_key_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`    // Associated private key ID
	UpdatedAt    time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`              // Timestamp when the server was last updated
	CreatedAt    time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`              // Timestamp when the server was created
}

// PrivateKey represents a private key for SSH authentication
type PrivateKey struct {
	ID          string    `json:"id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                        // Unique identifier for the private key
	TeamID      string    `json:"team_id" example:"01ARZ3NDEKTSV4RRFFQ69G5FAV"`                   // Associated team ID
	Name        string    `json:"name" example:"Production Key"`                                  // Private key name
	Description *string   `json:"description,omitempty" example:"SSH key for production servers"` // Private key description
	PrivateKey  string    `json:"private_key" example:"-----BEGIN PRIVATE KEY-----..."`           // The actual private key content
	Fingerprint string    `json:"fingerprint" example:"SHA256:abc123..."`                         // Key fingerprint
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`                      // Timestamp when the key was created
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`                      // Timestamp when the key was last updated
}
