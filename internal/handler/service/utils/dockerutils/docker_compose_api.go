package dockerutils

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/handler/service/utils/git"
	"github.com/yorukot/starker/internal/models"
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

// checkServiceHasGitSource checks if service has git source and logs accordingly
// Returns git source config if found, nil if not found
func checkServiceHasGitSource(ctx context.Context, serviceID string, db pgx.Tx, streamResult *StreamingResult) (*models.ServiceSourceGit, error) {
	// Check if service has git source
	gitSource, err := repository.GetServiceSourceGit(ctx, db, serviceID)
	if err != nil {
		// No git source found
		streamResult.LogChan <- "No git source found, using existing compose configuration"
		return nil, nil
	}

	streamResult.LogChan <- fmt.Sprintf("Git source found: %s (branch: %s)", gitSource.RepoURL, gitSource.Branch)
	streamResult.LogChan <- "Git update will be performed before starting service"

	return gitSource, nil
}

// performSyncGitUpdateAndGetConfig performs git update synchronously and returns updated compose config
func performSyncGitUpdateAndGetConfig(ctx context.Context, gitSource *models.ServiceSourceGit, serviceID string, cfg *DockerServiceConfig, db pgx.Tx, streamResult *StreamingResult) (*models.ServiceComposeConfig, error) {
	// Build git workflow config
	gitConfig := git.BuildGitWorkflowConfig(serviceID, gitSource, cfg.ConnectionPool, cfg.ConnectionID, cfg.Host, cfg.PrivateKeyContent)

	// Execute git update workflow
	safeLogWrite(streamResult.LogChan, "Starting git update...")
	gitResult, err := git.ExecuteGitUpdate(ctx, gitConfig)
	if err != nil {
		safeLogWrite(streamResult.LogChan, fmt.Sprintf("Failed to start git update: %v (using existing code)", err))
		// Fall back to existing compose config
		return repository.GetServiceComposeConfig(ctx, db, serviceID)
	}

	// Forward git update streaming output
	go func() {
		for {
			select {
			case log, ok := <-gitResult.LogChan:
				if !ok {
					return
				}
				safeLogWrite(streamResult.LogChan, log)
			case err, ok := <-gitResult.ErrorChan:
				if !ok {
					return
				}
				safeErrorWrite(streamResult.ErrorChan, err)
			case <-gitResult.DoneChan:
				return
			}
		}
	}()

	// Wait for git update to complete
	select {
	case <-gitResult.DoneChan:
		if gitResult.GetFinalError() != nil {
			safeLogWrite(streamResult.LogChan, fmt.Sprintf("Git update failed: %v (using existing code)", gitResult.GetFinalError()))
			// Fall back to existing compose config
			return repository.GetServiceComposeConfig(ctx, db, serviceID)
		}

		// If git update succeeded and we got updated compose content, update the database
		if gitResult.ComposeFile != "" {
			safeLogWrite(streamResult.LogChan, "Git update successful, updating compose configuration in database")

			// Get existing compose config to update it
			existingConfig, err := repository.GetServiceComposeConfig(ctx, db, serviceID)
			if err != nil {
				safeLogWrite(streamResult.LogChan, fmt.Sprintf("Failed to get existing compose config: %v (using git result)", err))
				// Create new config with git result
				newConfig := &models.ServiceComposeConfig{
					ServiceID:   serviceID,
					ComposeFile: gitResult.ComposeFile,
					UpdatedAt:   time.Now(),
				}
				return newConfig, nil
			}

			// Update the compose file content
			existingConfig.ComposeFile = gitResult.ComposeFile
			existingConfig.UpdatedAt = time.Now()

			if err := repository.UpdateServiceComposeConfig(ctx, db, *existingConfig); err != nil {
				safeLogWrite(streamResult.LogChan, fmt.Sprintf("Failed to update compose config in database: %v (using git result)", err))
			} else {
				safeLogWrite(streamResult.LogChan, "Compose configuration updated successfully in database")
			}

			return existingConfig, nil
		}
	case <-ctx.Done():
		safeLogWrite(streamResult.LogChan, "Git update cancelled due to context cancellation (using existing code)")
		// Fall back to existing compose config
		return repository.GetServiceComposeConfig(ctx, db, serviceID)
	}

	safeLogWrite(streamResult.LogChan, "Git update completed, retrieving compose configuration")
	return repository.GetServiceComposeConfig(ctx, db, serviceID)
}

// safeLogWrite safely writes to log channel, recovering from panic if channel is closed
func safeLogWrite(logChan chan string, message string) {
	defer func() {
		recover() // Recover from panic if channel is closed
	}()
	select {
	case logChan <- message:
		// Message sent successfully
	default:
		// Channel is closed or full, ignore silently
	}
}

// safeErrorWrite safely writes to error channel, recovering from panic if channel is closed
func safeErrorWrite(errorChan chan error, err error) {
	defer func() {
		recover() // Recover from panic if channel is closed
	}()
	select {
	case errorChan <- err:
		// Error sent successfully
	default:
		// Channel is closed or full, ignore silently
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

	// Create streaming result immediately and return it for immediate SSE streaming
	streamResult := NewStreamingResult()

	// Start containers in a goroutine for streaming with proper dependency order
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.LogChan)
		defer close(streamResult.ErrorChan)

		// Create a new transaction for git and compose operations
		tx, err := repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Check for git source
		gitSource, err := checkServiceHasGitSource(ctx, serviceID, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to check git source: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Get compose config - either from git update or existing database
		var composeConfig *models.ServiceComposeConfig
		if gitSource != nil {
			// Perform synchronous git update and get updated compose config
			composeConfig, err = performSyncGitUpdateAndGetConfig(ctx, gitSource, serviceID, cfg, tx, streamResult)
			if err != nil {
				streamResult.FinalError = fmt.Errorf("failed to update from git and get compose config: %w", err)
				streamResult.ErrorChan <- streamResult.FinalError
				return
			}
		} else {
			// No git source, get existing compose config
			composeConfig, err = repository.GetServiceComposeConfig(ctx, tx, serviceID)
			if err != nil {
				streamResult.FinalError = fmt.Errorf("failed to get compose config: %w", err)
				streamResult.ErrorChan <- streamResult.FinalError
				return
			}
		}

		// Parse compose file to get service definitions
		composeProject, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to parse compose file: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Validate the compose file
		if err := dockeryaml.Validate(composeProject); err != nil {
			streamResult.FinalError = fmt.Errorf("compose file validation failed: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		streamResult.LogChan <- "Compose file validated successfully, proceeding with Docker operations"

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

	// Create streaming result immediately and return it for immediate SSE streaming
	streamResult := NewStreamingResult()

	// Restart containers in a goroutine for streaming with proper dependency ordering
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.LogChan)
		defer close(streamResult.ErrorChan)

		// Create a new transaction for git and compose operations
		tx, err := repository.StartTransaction(dbPool, ctx)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to start transaction: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}
		defer repository.DeferRollback(tx, ctx)

		// Check for git source
		gitSource, err := checkServiceHasGitSource(ctx, serviceID, tx, streamResult)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to check git source: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Get compose config - either from git update or existing database
		var composeConfig *models.ServiceComposeConfig
		if gitSource != nil {
			// Perform synchronous git update and get updated compose config
			composeConfig, err = performSyncGitUpdateAndGetConfig(ctx, gitSource, serviceID, cfg, tx, streamResult)
			if err != nil {
				streamResult.FinalError = fmt.Errorf("failed to update from git and get compose config: %w", err)
				streamResult.ErrorChan <- streamResult.FinalError
				return
			}
		} else {
			// No git source, get existing compose config
			composeConfig, err = repository.GetServiceComposeConfig(ctx, tx, serviceID)
			if err != nil {
				streamResult.FinalError = fmt.Errorf("failed to get compose config: %w", err)
				streamResult.ErrorChan <- streamResult.FinalError
				return
			}
		}

		// Parse compose file to get service definitions
		composeProject, err := dockeryaml.ParseComposeContent(composeConfig.ComposeFile, cfg.ProjectName)
		if err != nil {
			streamResult.FinalError = fmt.Errorf("failed to parse compose file: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		// Validate the compose file
		if err := dockeryaml.Validate(composeProject); err != nil {
			streamResult.FinalError = fmt.Errorf("compose file validation failed: %w", err)
			streamResult.ErrorChan <- streamResult.FinalError
			return
		}

		streamResult.LogChan <- "Compose file validated successfully, proceeding with Docker restart operations"

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

		// Create a new transaction for starting services
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
