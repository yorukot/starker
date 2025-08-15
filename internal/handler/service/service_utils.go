package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"

	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/sshpool"
)

// StartService starts a service on a remote server using Docker Compose
func StartService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, sshPool *sshpool.SSHConnectionPool) error {
	// Get the service by ID
	service, err := repository.GetServiceByID(ctx, db, serviceID, teamID, projectID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Get the server details
	server, err := repository.GetServerByID(ctx, db, service.ServerID, teamID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Get the private key for SSH authentication
	privateKey, err := repository.GetPrivateKeyByID(ctx, db, server.PrivateKeyID, teamID)
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Get the compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(ctx, db, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get compose config: %w", err)
	}

	// Parse the private key for SSH authentication
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// Create host string for SSH connection
	host := fmt.Sprintf("%s:%s", server.IP, server.Port)

	// Define service directory path on server
	servicePath := fmt.Sprintf("/data/starker/services/%s", serviceID)
	composeFilePath := fmt.Sprintf("%s/docker-compose.yml", servicePath)

	// Check if the service folder exists on the server
	checkDirCmd := fmt.Sprintf("[ -d %s ]", servicePath)
	result := sshPool.ExecuteCommands(host, config, checkDirCmd)

	// If directory doesn't exist, create it
	if result.Error != nil {
		createDirCmd := fmt.Sprintf("mkdir -p %s", servicePath)
		result = sshPool.ExecuteCommands(host, config, createDirCmd)
		if result.Error != nil {
			return fmt.Errorf("failed to create service directory: %w", result.Error)
		}
	}

	// Create/overwrite the docker-compose file with the latest content
	createComposeCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", composeFilePath, composeConfig.ComposeFile)
	result = sshPool.ExecuteCommands(host, config, createComposeCmd)
	if result.Error != nil {
		return fmt.Errorf("failed to create compose file: %w", result.Error)
	}

	// Change to service directory and start the service using docker-compose
	startServiceCmd := fmt.Sprintf("cd %s && docker-compose up -d", servicePath)
	result = sshPool.ExecuteCommands(host, config, startServiceCmd)
	if result.Error != nil {
		return fmt.Errorf("failed to start service: %w", result.Error)
	}

	return nil
}

// StopService stops a service on a remote server using Docker Compose
func StopService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, sshPool *sshpool.SSHConnectionPool) error {
	// Get the service by ID
	service, err := repository.GetServiceByID(ctx, db, serviceID, teamID, projectID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Get the server details
	server, err := repository.GetServerByID(ctx, db, service.ServerID, teamID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Get the private key for SSH authentication
	privateKey, err := repository.GetPrivateKeyByID(ctx, db, server.PrivateKeyID, teamID)
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Parse the private key for SSH authentication
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// Create host string for SSH connection
	host := fmt.Sprintf("%s:%s", server.IP, server.Port)

	// Define service directory path on server
	servicePath := fmt.Sprintf("/data/starker/services/%s", serviceID)

	// Stop the service using docker-compose
	stopServiceCmd := fmt.Sprintf("cd %s && docker-compose down", servicePath)
	result := sshPool.ExecuteCommands(host, config, stopServiceCmd)
	if result.Error != nil {
		return fmt.Errorf("failed to stop service: %w", result.Error)
	}

	return nil
}
