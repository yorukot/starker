package dockerutils

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types"
	dockerimage "github.com/docker/docker/api/types/image"
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

// DockerBuildProgress represents Docker build progress event
type DockerBuildProgress struct {
	Stream      string `json:"stream"`
	Status      string `json:"status"`
	Progress    string `json:"progress"`
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
	ID string `json:"id"`
}

// pullProjectImages pulls Docker images for all services in the project with real-time progress streaming
// Modified to handle both image pulls and builds
func pullProjectImages(ctx context.Context, dockerClient *client.Client, project *composetypes.Project, streamResult *StreamingResult) error {
	for _, service := range project.Services {
		// Handle services with pre-built images
		if service.Image != "" {
			streamResult.LogChan <- fmt.Sprintf("Starting pull for image: %s", service.Image)

			pullResponse, err := dockerClient.ImagePull(ctx, service.Image, dockerimage.PullOptions{})
			if err != nil {
				streamResult.LogChan <- fmt.Sprintf("Failed to initiate pull for image %s: %v", service.Image, err)
				continue // Continue with existing local image if pull fails
			}

			// Stream Docker pull progress in real-time
			err = streamDockerPullProgress(pullResponse, service.Image, streamResult)
			pullResponse.Close()

			if err != nil {
				streamResult.LogChan <- fmt.Sprintf("Warning: Error during pull of %s: %v", service.Image, err)
				continue // Continue with existing local image if pull encounters issues
			}

			streamResult.LogChan <- fmt.Sprintf("Successfully pulled image: %s", service.Image)
		}
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
			streamResult.LogChan <- fmt.Sprintf("Pull error for %s: %s", imageName, progress.ErrorMessage)
			return fmt.Errorf("pull failed: %s", progress.ErrorMessage)
		}

		// Stream meaningful progress updates
		if progress.ID != "" {
			// Track layer-specific progress
			layerProgress[progress.ID] = progress

			switch progress.Status {
			case "Pulling fs layer":
				streamResult.LogChan <- fmt.Sprintf("[%s] %s: %s", imageName, progress.ID, progress.Status)
			case "Downloading":
				if progress.Progress != "" {
					streamResult.LogChan <- fmt.Sprintf("[%s] %s: Downloading %s", imageName, progress.ID, progress.Progress)
				}
			case "Download complete":
				streamResult.LogChan <- fmt.Sprintf("[%s] %s: Download complete", imageName, progress.ID)
			case "Extracting":
				if progress.Progress != "" {
					streamResult.LogChan <- fmt.Sprintf("[%s] %s: Extracting %s", imageName, progress.ID, progress.Progress)
				}
			case "Pull complete":
				streamResult.LogChan <- fmt.Sprintf("[%s] %s: Pull complete", imageName, progress.ID)
			}
		} else {
			// Handle image-level status updates
			switch progress.Status {
			case "Status: Image is up to date":
				streamResult.LogChan <- fmt.Sprintf("[%s] Image is up to date", imageName)
			case "Status: Downloaded newer image":
				streamResult.LogChan <- fmt.Sprintf("[%s] Downloaded newer image", imageName)
			default:
				if progress.Status != "" {
					streamResult.LogChan <- fmt.Sprintf("[%s] %s", imageName, progress.Status)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading pull response: %w", err)
	}

	return nil
}

// buildProjectImages builds Docker images for services that have build configurations
func buildProjectImages(ctx context.Context, dockerClient *client.Client, project *composetypes.Project, serviceID string, streamResult *StreamingResult) error {
	for _, service := range project.Services {
		if service.Build == nil {
			continue // Skip services without build configuration
		}

		streamResult.LogChan <- fmt.Sprintf("Starting build for service: %s", service.Name)

		// Resolve build context path - services are stored at /data/starker/{serviceID}/
		buildContextPath := resolveBuildContext(serviceID, service.Build.Context)

		// Create tar archive from build context
		buildContextTar, err := createBuildContextTar(buildContextPath)
		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Failed to create build context for service %s: %v", service.Name, err)
			continue
		}
		defer buildContextTar.Close()

		// Create build options
		buildOptions := types.ImageBuildOptions{
			Dockerfile: service.Build.Dockerfile,
			Tags:       []string{generateImageTag(service.Name, project.Name)},
			BuildArgs:  convertBuildArgs(service.Build.Args),
			Remove:     true,
		}

		streamResult.LogChan <- fmt.Sprintf("[%s] Building with context: %s", service.Name, buildContextPath)

		buildResponse, err := dockerClient.ImageBuild(ctx, buildContextTar, buildOptions)
		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Failed to initiate build for service %s: %v", service.Name, err)
			continue // Continue with other services if build fails
		}

		// Stream Docker build progress in real-time
		err = streamDockerBuildProgress(buildResponse.Body, service.Name, streamResult)
		buildResponse.Body.Close()

		if err != nil {
			streamResult.LogChan <- fmt.Sprintf("Warning: Error during build of %s: %v", service.Name, err)
			continue // Continue with other services if build encounters issues
		}

		streamResult.LogChan <- fmt.Sprintf("Successfully built image for service: %s", service.Name)
	}

	return nil
}

// streamDockerBuildProgress streams Docker build progress events to SSE
func streamDockerBuildProgress(buildResponse io.ReadCloser, serviceName string, streamResult *StreamingResult) error {
	scanner := bufio.NewScanner(buildResponse)

	for scanner.Scan() {
		var progress DockerBuildProgress
		if err := json.Unmarshal(scanner.Bytes(), &progress); err != nil {
			// Skip malformed JSON lines but continue processing
			continue
		}

		// Handle error events
		if progress.Error != "" {
			streamResult.LogChan <- fmt.Sprintf("Build error for %s: %s", serviceName, progress.Error)
			return fmt.Errorf("build failed: %s", progress.Error)
		}

		if progress.ErrorDetail.Message != "" {
			streamResult.LogChan <- fmt.Sprintf("Build error for %s: %s", serviceName, progress.ErrorDetail.Message)
			return fmt.Errorf("build failed: %s", progress.ErrorDetail.Message)
		}

		// Stream meaningful progress updates
		if progress.Stream != "" {
			// Clean up the stream output (remove newlines)
			cleanStream := strings.TrimSpace(progress.Stream)
			if cleanStream != "" {
				streamResult.LogChan <- fmt.Sprintf("[%s] %s", serviceName, cleanStream)
			}
		} else if progress.Status != "" {
			if progress.Progress != "" {
				streamResult.LogChan <- fmt.Sprintf("[%s] %s %s", serviceName, progress.Status, progress.Progress)
			} else {
				streamResult.LogChan <- fmt.Sprintf("[%s] %s", serviceName, progress.Status)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading build response: %w", err)
	}

	return nil
}

// resolveBuildContext resolves the build context path for a service
func resolveBuildContext(serviceID, buildContext string) string {
	// Base path where services are stored
	basePath := fmt.Sprintf("/data/starker/%s", serviceID)

	// If build context is empty or ".", use the base path
	if buildContext == "" || buildContext == "." {
		return basePath
	}

	// If build context is relative, join with base path
	if !filepath.IsAbs(buildContext) {
		return filepath.Join(basePath, buildContext)
	}

	// If absolute, use as is (though this is unusual)
	return buildContext
}

// generateImageTag generates a consistent image tag for built images
func generateImageTag(serviceName, projectName string) string {
	return fmt.Sprintf("%s-%s:latest", projectName, serviceName)
}

// convertBuildArgs converts Docker Compose build args to Docker API format
func convertBuildArgs(args composetypes.MappingWithEquals) map[string]*string {
	buildArgs := make(map[string]*string)
	for key, value := range args {
		buildArgs[key] = value
	}
	return buildArgs
}

// createBuildContextTar creates a tar archive from a directory for Docker build context
func createBuildContextTar(buildContextPath string) (io.ReadCloser, error) {
	// Create a pipe for tar data
	reader, writer := io.Pipe()

	// Start a goroutine to write tar data
	go func() {
		defer writer.Close()

		tarWriter := tar.NewWriter(writer)
		defer tarWriter.Close()

		// Walk the build context directory
		err := filepath.Walk(buildContextPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get relative path from build context
			relPath, err := filepath.Rel(buildContextPath, path)
			if err != nil {
				return err
			}

			// Skip the root directory itself
			if relPath == "." {
				return nil
			}

			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			// Use forward slashes in tar paths
			header.Name = filepath.ToSlash(relPath)

			// Write header
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			// If it's a regular file, write content
			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(tarWriter, file)
				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			writer.CloseWithError(err)
		}
	}()

	return reader, nil
}
