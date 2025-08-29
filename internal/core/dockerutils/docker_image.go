package dockerutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/core"
)

// PullDockerImages pulls all required Docker images from the compose project
func (dh *DockerHandler) PullDockerImages(ctx context.Context, tx pgx.Tx) error {
	for _, service := range dh.Project.Services {
		if service.Image == "" {
			continue
		}

		// Pull the Docker image
		err := dh.PullDockerImage(ctx, service.Image)
		if err != nil {
			zap.L().Error("failed to pull docker image", zap.Error(err), zap.String("image", service.Image))
			dh.StreamChan.LogError(fmt.Sprintf("Failed to pull docker image %s: %v", service.Image, err))
			return err
		}

		dh.StreamChan.LogInfo(fmt.Sprintf("Image %s pulled successfully", service.Image))
	}

	return nil
}

// PullDockerImage pulls a specific Docker image with streaming progress
func (dh *DockerHandler) PullDockerImage(ctx context.Context, imageName string) (err error) {
	// Log image pull start
	dh.StreamChan.LogInfo(fmt.Sprintf("Pulling Docker image: %s", imageName))

	// Pull the Docker image
	reader, err := dh.Client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		zap.L().Error("failed to start pulling Docker image", zap.Error(err), zap.String("image", imageName))
		dh.StreamChan.LogError(fmt.Sprintf("Failed to start pulling Docker image %s: %v", imageName, err))
		return fmt.Errorf("failed to start pulling Docker image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Stream the pull progress in real-time
	err = dh.streamDockerProgress(reader, imageName)
	if err != nil {
		zap.L().Error("failed during image pull streaming", zap.Error(err), zap.String("image", imageName))
		dh.StreamChan.LogError(fmt.Sprintf("Failed during image pull streaming: %v", err))
		return fmt.Errorf("failed during image pull streaming: %w", err)
	}

	dh.StreamChan.LogInfo(fmt.Sprintf("Successfully pulled Docker image: %s", imageName))
	return nil
}

// streamDockerProgress streams the Docker image pull progress in real-time
func (dh *DockerHandler) streamDockerProgress(reader io.ReadCloser, taskName string) error {
	decoder := json.NewDecoder(reader)

	for {
		var progress core.ProgressMessage
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			zap.L().Error("failed to decode pull progress", zap.Error(err))
			return fmt.Errorf("failed to decode pull progress: %w", err)
		}

		dh.StreamChan.LogProgress(progress)
	}

	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully pulled Docker image: %s", taskName))

	return nil
}
