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
		dh.StreamChan.LogStep("Starting Docker restart orchestration")

		// Phase 1: Stop everything
		dh.StreamChan.LogStep("Stopping and removing existing resources")

		err := dh.StopDockerCompose(ctx)
		if err != nil {
			zap.L().Error("Failed to initiate stop phase in RestartDockerCompose", zap.Error(err))
			dh.StreamChan.FinalError <- fmt.Errorf("failed to initiate stop phase: %w", err)
			return
		}

		// Phase 2: Start everything fresh
		dh.StreamChan.LogStep("Starting fresh orchestration")

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
