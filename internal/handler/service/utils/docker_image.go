package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// DockerPullProgress represents Docker pull progress event
type DockerPullProgress struct {
	Status         string `json:"status"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	ID           string `json:"id"`
	ErrorMessage string `json:"error"`
}

// pullProjectImages pulls Docker images for all services in the project with real-time progress streaming
func pullProjectImages(ctx context.Context, dockerClient *client.Client, project *types.Project, streamResult *StreamingResult) error {
	for _, service := range project.Services {
		if service.Image == "" {
			continue // Skip services without image (build-only)
		}

		streamResult.StdoutChan <- fmt.Sprintf("Starting pull for image: %s", service.Image)

		pullResponse, err := dockerClient.ImagePull(ctx, service.Image, image.PullOptions{})
		if err != nil {
			streamResult.StderrChan <- fmt.Sprintf("Failed to initiate pull for image %s: %v", service.Image, err)
			continue // Continue with existing local image if pull fails
		}

		// Stream Docker pull progress in real-time
		err = streamDockerPullProgress(pullResponse, service.Image, streamResult)
		pullResponse.Close()

		if err != nil {
			streamResult.StderrChan <- fmt.Sprintf("Warning: Error during pull of %s: %v", service.Image, err)
			continue // Continue with existing local image if pull encounters issues
		}

		streamResult.StdoutChan <- fmt.Sprintf("Successfully pulled image: %s", service.Image)
	}

	return nil
}

// streamDockerPullProgress streams Docker pull progress events to SSE
func streamDockerPullProgress(pullResponse io.ReadCloser, imageName string, streamResult *StreamingResult) error {
	scanner := bufio.NewScanner(pullResponse)
	layerProgress := make(map[string]DockerPullProgress)

	for scanner.Scan() {
		var progress DockerPullProgress
		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			// Skip malformed JSON lines but continue processing
			continue
		}

		// Handle error events
		if progress.ErrorMessage != "" {
			streamResult.StderrChan <- fmt.Sprintf("Pull error for %s: %s", imageName, progress.ErrorMessage)
			return fmt.Errorf("pull failed: %s", progress.ErrorMessage)
		}

		// Stream meaningful progress updates
		if progress.ID != "" {
			// Track layer-specific progress
			layerProgress[progress.ID] = progress

			switch progress.Status {
			case "Pulling fs layer":
				streamResult.StdoutChan <- fmt.Sprintf("[%s] %s: %s", imageName, progress.ID, progress.Status)
			case "Downloading":
				if progress.Progress != "" {
					streamResult.StdoutChan <- fmt.Sprintf("[%s] %s: Downloading %s", imageName, progress.ID, progress.Progress)
				}
			case "Download complete":
				streamResult.StdoutChan <- fmt.Sprintf("[%s] %s: Download complete", imageName, progress.ID)
			case "Extracting":
				if progress.Progress != "" {
					streamResult.StdoutChan <- fmt.Sprintf("[%s] %s: Extracting %s", imageName, progress.ID, progress.Progress)
				}
			case "Pull complete":
				streamResult.StdoutChan <- fmt.Sprintf("[%s] %s: Pull complete", imageName, progress.ID)
			}
		} else {
			// Handle image-level status updates
			switch progress.Status {
			case "Status: Image is up to date":
				streamResult.StdoutChan <- fmt.Sprintf("[%s] Image is up to date", imageName)
			case "Status: Downloaded newer image":
				streamResult.StdoutChan <- fmt.Sprintf("[%s] Downloaded newer image", imageName)
			default:
				if progress.Status != "" {
					streamResult.StdoutChan <- fmt.Sprintf("[%s] %s", imageName, progress.Status)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading pull response: %w", err)
	}

	return nil
}
