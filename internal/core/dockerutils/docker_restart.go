package dockerutils

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
)

// RestartDockerCompose restarts the docker compose orchestration by stopping everything and then starting fresh
// This ensures Docker Compose changes are applied by doing a complete stop/remove/start cycle
func (dh *DockerHandler) RestartDockerCompose(ctx context.Context) error {
	// Start Docker restart orchestration in a goroutine for streaming
	go func() {
		// Log start of Docker restart orchestration
		dh.StreamChan.LogChan <- core.LogStep("Starting Docker restart orchestration")

		// Create a channel to monitor stop completion
		stopCompleted := make(chan bool, 1)
		stopError := make(chan error, 1)

		// Monitor the stop operation
		go func() {
			for {
				select {
				case <-dh.StreamChan.DoneChan:
					stopCompleted <- true
					return
				case err := <-dh.StreamChan.FinalError:
					stopError <- err
					return
				}
			}
		}()

		// Phase 1: Stop everything
		dh.StreamChan.LogChan <- core.LogStep("Phase 1: Stopping and removing existing resources")

		err := dh.StopDockerCompose(ctx)
		if err != nil {
			zap.L().Error("Failed to initiate stop phase in RestartDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to initiate stop phase: %w", err)
			return
		}

		// Wait for stop to complete
		select {
		case <-stopCompleted:
			dh.StreamChan.LogChan <- core.LogInfo("Stop phase completed successfully")
		case err := <-stopError:
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed during stop phase: %v", err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed during stop phase: %w", err)
			return
		}

		// Phase 2: Start everything fresh
		dh.StreamChan.LogChan <- core.LogStep("Phase 2: Starting fresh orchestration")

		err = dh.StartDockerCompose(ctx)
		if err != nil {
			zap.L().Error("Failed to initiate start phase in RestartDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to initiate start phase: %w", err)
			return
		}

		// The StartDockerCompose will handle its own completion signaling
		dh.StreamChan.LogChan <- core.LogInfo("Docker restart orchestration initiated successfully")
	}()

	return nil
}
