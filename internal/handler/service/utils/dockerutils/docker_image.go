package dockerutils

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
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

// pullServiceImages pulls Docker images for all services in the project with real-time progress streaming
// Modified to handle both image pulls and builds
func pullServiceImage(ctx context.Context, dockerClient *client.Client, serviceConfig types.ServiceConfig, streamResult *StreamingResult) error {
	// Handle services with pre-built images
	if serviceConfig.Image != "" {
		streamResult.LogChan <- fmt.Sprintf("Starting pull for image: %s", serviceConfig.Image)

		pullResponse, err := dockerClient.ImagePull(ctx, serviceConfig.Image, dockerimage.PullOptions{})
		if err != nil {
			streamResult.ErrorChan <- fmt.Errorf("failed to initiate pull for image %s: %v", serviceConfig.Image, err)
			return nil
		}

		// Stream Docker pull progress in real-time
		err = streamDockerPullProgress(pullResponse, serviceConfig.Image, streamResult)
		pullResponse.Close()

		if err != nil {
			streamResult.ErrorChan <- fmt.Errorf("error during pull of %s: %v", serviceConfig.Image, err)
			return nil
		}

		streamResult.LogChan <- fmt.Sprintf("Successfully pulled image: %s", serviceConfig.Image)
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

// buildServiceImages builds Docker images for services that have build configurations
func buildServiceImages(ctx context.Context, dockerClient *client.Client, serviceConfig types.ServiceConfig, serviceID string, streamResult *StreamingResult, namingGen *generator.NamingGenerator, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte) error {
	if serviceConfig.Build == nil {
		return nil
	}

	streamResult.LogChan <- fmt.Sprintf("Starting build for service: %s", serviceConfig.Name)

	// Determine the target image name
	imageName := serviceConfig.Image

	if imageName == "" {
		// Generate image name for services without explicit image
		imageName = generateImageTag(serviceConfig.Name, namingGen.ProjectName())
		streamResult.LogChan <- fmt.Sprintf("Service %s has no image specified, will build as: %s", serviceConfig.Name, imageName)
	}

	// Check if image already exists locally
	images, err := dockerClient.ImageList(ctx, dockerimage.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil {
		streamResult.ErrorChan <- fmt.Errorf("warning: failed to check for existing image %s: %v", imageName, err)
	} else if len(images) > 0 {
		streamResult.LogChan <- fmt.Sprintf("Image %s already exists locally, skipping build", imageName)
		return nil
	}

	// Build the image since it doesn't exist
	streamResult.LogChan <- fmt.Sprintf("Building image %s for service %s", imageName, serviceConfig.Name)

	buildContextPath := resolveBuildContext(serviceID, serviceConfig.Build.Context)
	streamResult.LogChan <- fmt.Sprintf("Using build context: %s (on remote server)", buildContextPath)

	// Use relative path to Dockerfile within the build context tar
	DockerfilePath := serviceConfig.Build.Dockerfile
	if DockerfilePath == "" {
		DockerfilePath = "Dockerfile" // Default to "Dockerfile" if not specified
	}
	streamResult.LogChan <- fmt.Sprintf("Using Dockerfile path: %s (relative to build context)", DockerfilePath)

	// Create build context tar from the remote directory via SSH
	buildContextReader, err := createRemoteBuildContextTar(ctx, connectionPool, connectionID, host, privateKeyContent, buildContextPath, streamResult)
	if err != nil {
		streamResult.ErrorChan <- fmt.Errorf("failed to create remote build context tar for service %s: %v", serviceConfig.Name, err)
		return err
	}

	// Set up build options
	buildOptions := build.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: DockerfilePath,
		BuildArgs:  convertBuildArgs(serviceConfig.Build.Args),
		Target:     serviceConfig.Build.Target,
	}

	// Start the build
	buildResponse, err := dockerClient.ImageBuild(ctx, buildContextReader, buildOptions)
	if err != nil {
		streamResult.LogChan <- fmt.Sprintf("Failed to start build for %s: %v", serviceConfig.Name, err)
		return fmt.Errorf("failed to start build for service %s: %w", serviceConfig.Name, err)
	}

	// Stream build progress
	err = streamDockerBuildProgress(buildResponse.Body, serviceConfig.Name, streamResult)
	buildResponse.Body.Close()

	if err != nil {
		streamResult.LogChan <- fmt.Sprintf("Build failed for %s: %v", serviceConfig.Name, err)
		return fmt.Errorf("build failed for service %s: %w", serviceConfig.Name, err)
	}

	streamResult.LogChan <- fmt.Sprintf("Successfully built image %s for service %s", imageName, serviceConfig.Name)

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
// Note: This runs on remote server via SSH, so no local filesystem validation needed
func resolveBuildContext(serviceID, buildContext string) string {
	// Base path where services are stored on the remote server
	basePath := fmt.Sprintf("/data/starker/services/%s", serviceID)

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
func convertBuildArgs(args types.MappingWithEquals) map[string]*string {
	buildArgs := make(map[string]*string)
	for key, value := range args {
		buildArgs[key] = value
	}
	return buildArgs
}

// createRemoteBuildContextTar creates a tar archive from a remote directory via SSH
// This function executes tar command on the remote server and streams the binary data
func createRemoteBuildContextTar(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, buildContextPath string, streamResult *StreamingResult) (io.ReadCloser, error) {
	// Validate that build context path exists on remote server first
	checkCmd := fmt.Sprintf("test -d %s", buildContextPath)
	checkResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, checkCmd, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to execute remote path check: %w", err)
	}

	// Wait for check command to complete
	select {
	case <-checkResult.DoneChan:
		if checkResult.GetFinalError() != nil {
			return nil, fmt.Errorf("build context path %s does not exist on remote server: %w", buildContextPath, checkResult.GetFinalError())
		}
		if checkResult.GetExitCode() != 0 {
			return nil, fmt.Errorf("build context path %s does not exist on remote server (exit code: %d)", buildContextPath, checkResult.GetExitCode())
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("build context path check cancelled: %w", ctx.Err())
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("build context path check timed out")
	}

	streamResult.LogChan <- fmt.Sprintf("Build context path %s verified on remote server", buildContextPath)

	// For tar binary data, we need to use the raw SSH client directly rather than ExecuteSSHCommand
	// which is designed for text-based commands
	sshClient, err := connectionPool.GetSSHConnection(connectionID, host, privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection: %w", err)
	}

	// Create SSH session for tar command
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Get stdout pipe for binary tar data
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Get stderr pipe for error messages
	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Create tar command to archive the build context directory
	tarCmd := fmt.Sprintf("tar -cf - -C %s .", buildContextPath)
	streamResult.LogChan <- fmt.Sprintf("Creating build context tar with command: %s", tarCmd)

	// Start the tar command
	if err := session.Start(tarCmd); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to start tar command: %w", err)
	}

	// Create a pipe to return to Docker client
	reader, writer := io.Pipe()

	// Handle the tar streaming in a goroutine
	go func() {
		defer writer.Close()
		defer session.Close()

		// Handle stderr in a separate goroutine
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				streamResult.LogChan <- fmt.Sprintf("Tar stderr: %s", scanner.Text())
			}
		}()

		// Stream tar data directly from stdout to writer
		_, err := io.Copy(writer, stdout)
		if err != nil {
			writer.CloseWithError(fmt.Errorf("failed to stream tar data: %w", err))
			return
		}

		// Wait for the command to complete
		if err := session.Wait(); err != nil {
			writer.CloseWithError(fmt.Errorf("tar command failed: %w", err))
			return
		}

		streamResult.LogChan <- "Build context tar created successfully on remote server"
	}()

	return reader, nil
}
