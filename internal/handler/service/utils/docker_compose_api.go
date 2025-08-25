package utils

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/dockerpool"
	"github.com/yorukot/starker/pkg/dockeryaml"
	"github.com/yorukot/starker/pkg/generator"
)

// DockerServiceConfig holds Docker API connection configuration for service operations
type DockerServiceConfig struct {
	Client      *client.Client
	ServiceID   string
	ProjectName string
	Generator   *generator.NamingGenerator
}

// StreamingResult provides streaming output from Docker operations
type StreamingResult struct {
	LogChan    chan string
	ErrorChan  chan error
	DoneChan   chan struct{}
	finalError error
}

// NewStreamingResult creates a new StreamingResult
func NewStreamingResult() *StreamingResult {
	return &StreamingResult{
		LogChan:   make(chan string, 100),
		ErrorChan: make(chan error, 10),
		DoneChan:  make(chan struct{}),
	}
}

// GetFinalError returns the final error from the operation
func (sr *StreamingResult) GetFinalError() error {
	return sr.finalError
}

// prepareDockerConfig prepares Docker API configuration for service operations
func prepareDockerConfig(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dockerPool *dockerpool.DockerConnectionPool) (*DockerServiceConfig, error) {
	// Get the service by ID
	service, err := repository.GetServiceByID(ctx, db, serviceID, teamID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Get the server details
	server, err := repository.GetServerByID(ctx, db, service.ServerID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	// Get the private key for authentication
	privateKey, err := repository.GetPrivateKeyByID(ctx, db, server.PrivateKeyID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	// Create naming generator for consistent naming
	namingGenerator := generator.NewNamingGenerator(serviceID, teamID, server.ID)

	// Create host string for Docker connection
	host := fmt.Sprintf("ssh://%s@%s:%s", server.User, server.IP, server.Port)

	// Get Docker client from pool using generated connection ID
	connectionID := namingGenerator.ConnectionID()
	dockerClient, err := dockerPool.GetConnection(connectionID, host, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker connection: %w", err)
	}

	return &DockerServiceConfig{
		Client:      dockerClient,
		ServiceID:   serviceID,
		ProjectName: namingGenerator.ProjectName(),
		Generator:   namingGenerator,
	}, nil
}

// validateComposeFile validates the compose file using dockeryaml parser
func validateComposeFile(composeContent string) error {
	composeFile, err := dockeryaml.ParseComposeContent(composeContent, "validation")
	if err != nil {
		return fmt.Errorf("failed to parse compose content: %w", err)
	}

	if err := composeFile.Validate(); err != nil {
		return fmt.Errorf("compose file validation failed: %w", err)
	}

	return nil
}

// StartService starts a service using Docker API
func StartService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *dockerpool.DockerConnectionPool) (*StreamingResult, error) {
	// Prepare Docker configuration
	cfg, err := prepareDockerConfig(ctx, serviceID, teamID, projectID, db, dockerPool)
	if err != nil {
		return nil, err
	}

	// Get the compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(ctx, db, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compose config: %w", err)
	}

	// Validate compose file
	if err := validateComposeFile(composeConfig.ComposeFile); err != nil {
		return nil, fmt.Errorf("compose validation failed: %w", err)
	}

	// Parse compose file to get service definitions
	composeFile, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get the Docker Compose v2 Project directly from the parser for dependency orchestration
	project := composeFile.GetProject()
	project.Name = cfg.ProjectName // Set the project name for this deployment

	// Create streaming result
	streamResult := NewStreamingResult()

	// Start containers in a goroutine for streaming with proper dependency order
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.LogChan)
		defer close(streamResult.ErrorChan)

		// Create a new transaction for the goroutine
		tx, err := repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, project, cfg, tx, streamResult)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Synchronize Docker Compose resources to database
		err = DockerComposeToDatabase(ctx, project, cfg, tx, streamResult)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to sync compose resources to database: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		err = startComposeServicesWithDependencies(ctx, cfg, project, streamResult)
		if err != nil {
			streamResult.finalError = err
			streamResult.ErrorChan <- err
			return
		}

	}()

	return streamResult, nil
}

// StopService stops a service using Docker API with proper dependency ordering
func StopService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *dockerpool.DockerConnectionPool) (*StreamingResult, error) {
	// Prepare Docker configuration
	cfg, err := prepareDockerConfig(ctx, serviceID, teamID, projectID, db, dockerPool)
	if err != nil {
		return nil, err
	}

	// Get the compose configuration for dependency information
	composeConfig, err := repository.GetServiceComposeConfig(ctx, db, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compose config: %w", err)
	}

	// Parse compose file to get service definitions
	composeFile, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get the Docker Compose v2 Project directly from the parser for dependency orchestration
	project := composeFile.GetProject()
	project.Name = cfg.ProjectName // Set the project name for this deployment

	// Create streaming result
	streamResult := NewStreamingResult()

	// Stop containers in a goroutine for streaming with reverse dependency order
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.LogChan)
		defer close(streamResult.ErrorChan)

		// Create a new transaction for the goroutine
		tx, err := repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, project, cfg, tx, streamResult)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		err = stopComposeServicesWithDependencies(ctx, cfg, project, streamResult)
		if err != nil {
			streamResult.finalError = err
			streamResult.ErrorChan <- err
			return
		}

	}()

	return streamResult, nil
}

// RestartService restarts a service using Docker API with proper dependency ordering
func RestartService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *dockerpool.DockerConnectionPool) (*StreamingResult, error) {
	// Prepare Docker configuration
	cfg, err := prepareDockerConfig(ctx, serviceID, teamID, projectID, db, dockerPool)
	if err != nil {
		return nil, err
	}

	// Get the compose configuration
	composeConfig, err := repository.GetServiceComposeConfig(ctx, db, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compose config: %w", err)
	}

	// Validate compose file
	if err := validateComposeFile(composeConfig.ComposeFile); err != nil {
		return nil, fmt.Errorf("compose validation failed: %w", err)
	}

	// Parse compose file to get service definitions
	composeFile, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get the Docker Compose v2 Project directly from the parser for dependency orchestration
	project := composeFile.GetProject()
	project.Name = cfg.ProjectName // Set the project name for this deployment

	// Create streaming result
	streamResult := NewStreamingResult()

	// Restart containers in a goroutine for streaming with proper dependency ordering
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.LogChan)
		defer close(streamResult.ErrorChan)

		// Create a new transaction for the goroutine
		tx, err := repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, project, cfg, tx, streamResult)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		// First stop services in reverse dependency order
		err = stopComposeServicesWithDependencies(ctx, cfg, project, streamResult)
		if err != nil {
			streamResult.finalError = err
			streamResult.ErrorChan <- err
			return
		}

		// Create a new transaction for the goroutine
		tx, err = repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Synchronize Docker Compose resources to database
		err = DockerComposeToDatabase(ctx, project, cfg, tx, streamResult)
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to sync compose resources to database: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		// Then start them again in proper dependency order
		err = startComposeServicesWithDependencies(ctx, cfg, project, streamResult)
		if err != nil {
			streamResult.finalError = err
			streamResult.ErrorChan <- err
			return
		}

	}()

	return streamResult, nil
}
