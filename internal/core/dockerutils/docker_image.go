package dockerutils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/internal/repository"
)

// DockerPullProgress represents the progress information from Docker ImagePull API
type DockerPullProgress struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}

// PullDockerImages pulls all required Docker images from the compose project
func (dh *DockerHandler) PullDockerImages(ctx context.Context, tx pgx.Tx) error {
	// Collect unique images from all services
	imageMap := make(map[string]bool)
	for _, service := range dh.Project.Services {
		if service.Image != "" {
			imageMap[service.Image] = true
		}
	}

	// Pull each unique image and save to database
	for imageName := range imageMap {
		// Pull the Docker image
		imageID, err := dh.PullDockerImage(ctx, imageName)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to pull docker image %s: %v", imageName, err))
			return err
		}

		// Create the image record in database
		serviceImage := models.ServiceImage{
			ID:        ksuid.New().String(),
			ServiceID: dh.NamingGenerator.ServiceID(),
			ImageID:   &imageID,
			ImageName: imageName,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = repository.CreateServiceImage(ctx, tx, serviceImage)
		if err != nil {
			dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to save image to database: %v", err))
			return fmt.Errorf("failed to save image %s to database: %w", imageName, err)
		}

		dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Image %s pulled and saved successfully", imageName))
	}
	return nil
}

// PullDockerImage pulls a specific Docker image with streaming progress
func (dh *DockerHandler) PullDockerImage(ctx context.Context, imageName string) (imageID string, err error) {
	// Log image pull start
	dh.StreamChan.LogChan <- core.LogStep(fmt.Sprintf("Pulling Docker image: %s", imageName))

	// Prepare image pull options
	pullOptions := image.PullOptions{
		// Add authentication here if needed in the future
		// RegistryAuth: "base64-encoded-auth",
	}

	// Pull the Docker image
	reader, err := dh.Client.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed to start pulling Docker image %s: %v", imageName, err))
		return "", fmt.Errorf("failed to start pulling Docker image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Stream the pull progress in real-time
	err = dh.streamImagePullProgress(reader, imageName)
	if err != nil {
		dh.StreamChan.ErrChan <- core.LogError(fmt.Sprintf("Failed during image pull streaming: %v", err))
		return "", fmt.Errorf("failed during image pull streaming: %w", err)
	}

	// Get the image ID after successful pull
	images, err := dh.Client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list images after pull: %w", err)
	}

	// Find the pulled image
	for _, img := range images {
		for _, repoTag := range img.RepoTags {
			if repoTag == imageName {
				// Use the first 12 characters of the ID for consistency
				fullID := strings.TrimPrefix(img.ID, "sha256:")
				if len(fullID) > 12 {
					return fullID[:12], nil
				}
				return fullID, nil
			}
		}
	}

	// If we can't find by RepoTags, just return a generic success message
	// This can happen with digests or other image references
	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully pulled Docker image: %s", imageName))
	return "pulled", nil
}

// streamImagePullProgress streams the Docker image pull progress in real-time
func (dh *DockerHandler) streamImagePullProgress(reader io.ReadCloser, imageName string) error {
	decoder := json.NewDecoder(reader)

	for {
		var progress DockerPullProgress
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull progress: %w", err)
		}

		// Create a meaningful progress message
		message := progress.Status
		if progress.ID != "" {
			message = fmt.Sprintf("%s: %s", progress.ID, progress.Status)
		}
		if progress.Progress != "" {
			message = fmt.Sprintf("%s %s", message, progress.Progress)
		}

		// Skip empty or redundant messages
		if strings.TrimSpace(message) == "" {
			continue
		}

		// Determine log message based on status and send to appropriate channel
		if strings.Contains(strings.ToLower(progress.Status), "error") {
			dh.StreamChan.ErrChan <- core.LogError(message)
		} else if strings.Contains(progress.Status, "Downloading") ||
			strings.Contains(progress.Status, "Extracting") ||
			strings.Contains(progress.Status, "Pulling") {
			dh.StreamChan.LogChan <- core.LogStep(message)
		} else {
			dh.StreamChan.LogChan <- core.LogInfo(message)
		}
	}

	dh.StreamChan.LogChan <- core.LogInfo(fmt.Sprintf("Successfully pulled Docker image: %s", imageName))

	return nil
}
