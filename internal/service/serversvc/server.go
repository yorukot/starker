package serversvc

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/ssh"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/sshpool"
)

type CreateServerRequest struct {
	Name         string  `json:"name" validate:"required,min=3,max=255"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=500"`
	IP           string  `json:"ip" validate:"required,ip"`
	Port         string  `json:"port" validate:"required,min=1,max=5"`
	User         string  `json:"user" validate:"required,min=1,max=255"`
	PrivateKeyID string  `json:"private_key_id" validate:"required"`
}

type UpdateServerRequest struct {
	Name         *string `json:"name,omitempty" validate:"omitempty,min=3,max=255"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=500"`
	IP           *string `json:"ip,omitempty" validate:"omitempty,ip"`
	Port         *string `json:"port,omitempty" validate:"omitempty,min=1,max=5"`
	User         *string `json:"user,omitempty" validate:"omitempty,min=1,max=255"`
	PrivateKeyID *string `json:"private_key_id,omitempty" validate:"omitempty"`
}

// ServerValidate validates the create server request
func ServerValidate(createServerRequest CreateServerRequest) error {
	return validator.New().Struct(createServerRequest)
}

// ServerUpdateValidate validates the update server request
func ServerUpdateValidate(updateServerRequest UpdateServerRequest) error {
	return validator.New().Struct(updateServerRequest)
}

// GenerateServer generates a server model for the create request
func GenerateServer(createServerRequest CreateServerRequest, teamID string) models.Server {
	now := time.Now()

	return models.Server{
		ID:           ksuid.New().String(),
		TeamID:       teamID,
		Name:         createServerRequest.Name,
		Description:  createServerRequest.Description,
		IP:           createServerRequest.IP,
		Port:         createServerRequest.Port,
		User:         createServerRequest.User,
		PrivateKeyID: createServerRequest.PrivateKeyID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UpdateServerFromRequest updates a server model with new values from update request
func UpdateServerFromRequest(existingServer models.Server, updateServerRequest UpdateServerRequest) models.Server {
	if updateServerRequest.Name != nil {
		existingServer.Name = *updateServerRequest.Name
	}
	if updateServerRequest.Description != nil {
		existingServer.Description = updateServerRequest.Description
	}
	if updateServerRequest.IP != nil {
		existingServer.IP = *updateServerRequest.IP
	}
	if updateServerRequest.Port != nil {
		existingServer.Port = *updateServerRequest.Port
	}
	if updateServerRequest.User != nil {
		existingServer.User = *updateServerRequest.User
	}
	if updateServerRequest.PrivateKeyID != nil {
		existingServer.PrivateKeyID = *updateServerRequest.PrivateKeyID
	}
	existingServer.UpdatedAt = time.Now()

	return existingServer
}

// TestServerConnection tests the SSH connection to a server using the provided private key
func TestServerConnection(ctx context.Context, server models.Server, privateKey models.PrivateKey, pool *sshpool.SSHConnectionPool) error {
	// Parse the private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKey.PrivateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: server.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing purposes
		Timeout:         10 * time.Second,
	}

	// Test connection by creating a host string
	host := fmt.Sprintf("%s:%s", server.IP, server.Port)

	// Try to establish connection
	client, err := pool.GetConnection(host, config)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Test by creating a simple session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Run a simple test command
	if err := session.Run("echo 'connection test'"); err != nil {
		return fmt.Errorf("connection test command failed: %w", err)
	}

	return nil
}
