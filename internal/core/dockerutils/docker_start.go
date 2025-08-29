package dockerutils

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core/dockersync"
	"github.com/yorukot/starker/internal/repository"
)

// StartDockerCompose starts the docker compose orchestration in a goroutine with streaming output
func (dh *DockerHandler) StartDockerCompose(ctx context.Context) error {
	// Start Docker orchestration in a goroutine for streaming
	go func() {
		// Create a new transaction for the goroutine
		tx, err := repository.StartTransaction(dh.DB, ctx)
		if err != nil {
			zap.L().Error("Failed to begin transaction in StartDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to begin transaction: %w", err)
			return
		}

		defer func() {
			// Rollback transaction if it hasn't been committed
			repository.DeferRollback(tx, ctx)
			dh.StreamChan.DoneChan <- true
		}()

		// Log start of Docker orchestration
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Starting Docker orchestration",
		}

		// Use SyncContainersToDB to sync the container to db first
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Syncing containers to database",
		}

		err = dockersync.SyncContainersToDB(ctx, tx, dh.ConnectionPool, *dh.NamingGenerator, *dh.Project)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to sync containers to database: %v", err),
			}
			dh.StreamChan.FinalError <- fmt.Errorf("failed to sync containers to database: %w", err)
			return
		}

		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogTypeInfo,
			Message: "Successfully synced containers to database",
		}

		// Pull the Docker images
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Starting image pull process",
		}

		// +-------------------------------------------+
		// |Start Docker Pull                          |
		// +-------------------------------------------+
		err = dh.PullDockerImages(ctx, tx)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to pull Docker images: %v", err),
			}
			dh.StreamChan.FinalError <- fmt.Errorf("failed to pull Docker images: %w", err)
			return
		}

		// Create Docker networks
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Creating Docker networks",
		}

		err = dh.StartDockerNetworks(ctx, tx)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to create Docker networks: %v", err),
			}
			dh.StreamChan.FinalError <- fmt.Errorf("failed to create Docker networks: %w", err)
			return
		}

		// Create Docker volumes
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Creating Docker volumes",
		}

		// +-------------------------------------------+
		// |Start Create Volume                        |
		// +-------------------------------------------+
		err = dh.StartDockerVolumes(ctx, tx)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to create Docker volumes: %v", err),
			}
			dh.StreamChan.FinalError <- fmt.Errorf("failed to create Docker volumes: %w", err)
			return
		}

		// Create and start Docker containers
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogStep,
			Message: "Creating and starting Docker containers",
		}

		// +-------------------------------------------+
		// |Start Docker Containers                    |
		// +-------------------------------------------+
		err = dh.StartDockerContainers(ctx, tx)
		if err != nil {
			dh.StreamChan.ErrChan <- LogMessage{
				Type:    LogTypeError,
				Message: fmt.Sprintf("Failed to start Docker containers: %v", err),
			}
			dh.StreamChan.FinalError <- fmt.Errorf("failed to start Docker containers: %w", err)
			return
		}

		// Commit the transaction on successful completion
		if err := tx.Commit(ctx); err != nil {
			zap.L().Error("Failed to commit transaction in StartDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to commit transaction: %w", err)
			return
		}

		// Docker orchestration completed successfully
		dh.StreamChan.LogChan <- LogMessage{
			Type:    LogTypeInfo,
			Message: "Docker orchestration completed successfully",
		}
	}()

	return nil
}
