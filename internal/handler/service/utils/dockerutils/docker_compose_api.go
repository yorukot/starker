package dockerutils

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/repository"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/dockeryaml"
	"github.com/yorukot/starker/pkg/generator"
)

// DockerServiceConfig holds Docker API connection configuration for service operations
type DockerServiceConfig struct {
	Client            *client.Client
	ServiceID         string
	ProjectName       string
	Generator         *generator.NamingGenerator
	ConnectionPool    *connection.ConnectionPool
	ConnectionID      string
	Host              string
	PrivateKeyContent []byte
}

// StreamingResult provides streaming output from Docker operations
type StreamingResult struct {
	LogChan    chan string
	ErrorChan  chan error
	DoneChan   chan struct{}
	FinalError error
}

// NewStreamingResult creates a new StreamingResult
func NewStreamingResult() *StreamingResult {
	return &StreamingResult{
		LogChan:   make(chan string, 100),
		ErrorChan: make(chan error, 10),
		DoneChan:  make(chan struct{}),
	}
}

// prepareDockerConfig prepares Docker API configuration for service operations
func prepareDockerConfig(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dockerPool *connection.ConnectionPool) (*DockerServiceConfig, error) {
	service, err := repository.GetServiceByID(ctx, db, serviceID, teamID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	server, err := repository.GetServerByID(ctx, db, service.ServerID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	privateKey, err := repository.GetPrivateKeyByID(ctx, db, server.PrivateKeyID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	namingGenerator := generator.NewNamingGenerator(serviceID, teamID, server.ID)

	// Create host string for Docker connection
	host := fmt.Sprintf("ssh://%s@%s:%s", server.User, server.IP, server.Port)

	// Get Docker client from pool using generated connection ID
	connectionID := namingGenerator.ConnectionID()
	dockerClient, err := dockerPool.GetDockerConnection(connectionID, host, []byte(privateKey.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker connection: %w", err)
	}

	return &DockerServiceConfig{
		Client:            dockerClient,
		ServiceID:         serviceID,
		ProjectName:       namingGenerator.ProjectName(),
		Generator:         namingGenerator,
		ConnectionPool:    dockerPool,
		ConnectionID:      connectionID,
		Host:              host,
		PrivateKeyContent: []byte(privateKey.PrivateKey),
	}, nil
}

// StartService starts a service using Docker API
func StartService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *connection.ConnectionPool) (*StreamingResult, error) {
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

	// Parse compose file to get service definitions
	composeProject, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate the compose file
	if err := dockeryaml.Validate(composeProject); err != nil {
		return nil, fmt.Errorf("compose file validation failed: %w", err)
	}

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
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, composeProject, cfg, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Synchronize Docker Compose resources to database
		err = DockerComposeToDatabase(ctx, composeProject, cfg, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to sync compose resources to database: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		err = startComposeServicesWithDependencies(ctx, cfg, composeProject, streamResult)
		if err != nil {
			streamResult.FinalError = err
			streamResult.ErrorChan <- err
			return
		}
	}()

	return streamResult, nil
}

// StopService stops a service using Docker API with proper dependency ordering
func StopService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *connection.ConnectionPool) (*StreamingResult, error) {
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

	// Parse compose file to get service definitions
	composeProject, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate the compose file
	if err := dockeryaml.Validate(composeProject); err != nil {
		return nil, fmt.Errorf("compose file validation failed: %w", err)
	}

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
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, composeProject, cfg, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		err = stopComposeServicesWithDependencies(ctx, cfg, composeProject, streamResult)
		if err != nil {
			streamResult.FinalError = err
			streamResult.ErrorChan <- err
			return
		}

	}()

	return streamResult, nil
}

// RestartService restarts a service using Docker API with proper dependency ordering
func RestartService(ctx context.Context, serviceID, teamID, projectID string, db pgx.Tx, dbPool *pgxpool.Pool, dockerPool *connection.ConnectionPool) (*StreamingResult, error) {
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

	// Parse compose file to get service definitions
	composeProject, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate the compose file
	if err := dockeryaml.Validate(composeProject); err != nil {
		return nil, fmt.Errorf("compose file validation failed: %w", err)
	}

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
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Purge existing Docker resources and clean database records first
		err = DockerDatabaseToPurge(ctx, composeProject, cfg, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to purge existing resources: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		// First stop services in reverse dependency order
		err = stopComposeServicesWithDependencies(ctx, cfg, composeProject, streamResult)
		if err != nil {
			streamResult.FinalError = err
			streamResult.ErrorChan <- err
			return
		}

		// Create a new transaction for the goroutine
		tx, err = repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Synchronize Docker Compose resources to database
		err = DockerComposeToDatabase(ctx, composeProject, cfg, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to sync compose resources to database: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Commit the transaction if successful
		repository.CommitTransaction(tx, ctx)

		// Then start them again in proper dependency order
		err = startComposeServicesWithDependencies(ctx, cfg, composeProject, streamResult)
		if err != nil {
			streamResult.FinalError = err
			streamResult.ErrorChan <- err
			return
		}

	}()

	return streamResult, nil
}
