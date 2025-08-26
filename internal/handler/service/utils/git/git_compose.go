package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/dockeryaml"
)

// DockerComposeExtractResult contains the result of compose file extraction
type DockerComposeExtractResult struct {
	LogChan     chan string
	ErrorChan   chan error
	DoneChan    chan struct{}
	ComposeFile string
	FilePath    string
	finalError  error
}

// NewDockerComposeExtractResult creates a new DockerComposeExtractResult
func NewDockerComposeExtractResult() *DockerComposeExtractResult {
	return &DockerComposeExtractResult{
		LogChan:   make(chan string, 100),
		ErrorChan: make(chan error, 10),
		DoneChan:  make(chan struct{}),
	}
}

// GetFinalError returns the final error from the compose extraction
func (dcer *DockerComposeExtractResult) GetFinalError() error {
	return dcer.finalError
}

// ExtractDockerCompose finds and extracts Docker compose files from the cloned repository
func ExtractDockerCompose(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, repoPath, customComposePath string) (*DockerComposeExtractResult, error) {
	extractResult := NewDockerComposeExtractResult()

	// Execute extraction in a goroutine for streaming
	go func() {
		defer close(extractResult.DoneChan)
		defer close(extractResult.LogChan)
		defer close(extractResult.ErrorChan)

		extractResult.LogChan <- fmt.Sprintf("Searching for Docker Compose files in %s", repoPath)

		// Step 1: Find Docker Compose file
		composePath, err := findDockerComposeFile(ctx, connectionPool, connectionID, host, privateKeyContent, repoPath, customComposePath, extractResult)
		if err != nil {
			extractResult.finalError = err
			extractResult.ErrorChan <- err
			return
		}

		if composePath == "" {
			extractResult.finalError = fmt.Errorf("no Docker Compose file found in repository")
			extractResult.ErrorChan <- extractResult.finalError
			return
		}

		extractResult.FilePath = composePath
		extractResult.LogChan <- fmt.Sprintf("Found Docker Compose file: %s", composePath)

		// Step 2: Read the compose file content
		composeContent, err := readComposeFile(ctx, connectionPool, connectionID, host, privateKeyContent, composePath, extractResult)
		if err != nil {
			extractResult.finalError = err
			extractResult.ErrorChan <- err
			return
		}

		extractResult.ComposeFile = composeContent

		// Step 3: Validate the compose file
		extractResult.LogChan <- "Validating Docker Compose file format"
		if err := validateComposeFile(composeContent); err != nil {
			extractResult.finalError = fmt.Errorf("invalid Docker Compose file: %w", err)
			extractResult.ErrorChan <- extractResult.finalError
			return
		}

		extractResult.LogChan <- "Docker Compose file extracted and validated successfully"
	}()

	return extractResult, nil
}

// findDockerComposeFile searches for Docker Compose files in common locations
func findDockerComposeFile(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, repoPath, customPath string, extractResult *DockerComposeExtractResult) (string, error) {
	// If custom path is provided, check that first
	if customPath != "" {
		customFullPath := fmt.Sprintf("%s/%s", repoPath, strings.TrimPrefix(customPath, "/"))
		extractResult.LogChan <- fmt.Sprintf("Checking custom compose file path: %s", customPath)

		if exists, err := checkFileExists(ctx, connectionPool, connectionID, host, privateKeyContent, customFullPath); err != nil {
			return "", err
		} else if exists {
			return customFullPath, nil
		} else {
			extractResult.LogChan <- fmt.Sprintf("Custom compose file not found: %s", customPath)
		}
	}

	// Common Docker Compose file names to search for
	commonPaths := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"Docker-Compose.yml",
		"docker-compose.prod.yml",
		"docker-compose.production.yml",
	}

	for _, fileName := range commonPaths {
		fullPath := fmt.Sprintf("%s/%s", repoPath, fileName)
		extractResult.LogChan <- fmt.Sprintf("Checking for %s", fileName)

		if exists, err := checkFileExists(ctx, connectionPool, connectionID, host, privateKeyContent, fullPath); err != nil {
			return "", err
		} else if exists {
			return fullPath, nil
		}
	}

	return "", nil
}

// checkFileExists checks if a file exists on the remote server
func checkFileExists(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, filePath string) (bool, error) {
	checkCmd := fmt.Sprintf("test -f %s", filePath)

	sshResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, checkCmd, 10*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	// Wait for command to complete
	select {
	case <-sshResult.DoneChan:
		// Exit code 0 means file exists, non-zero means it doesn't
		return sshResult.GetExitCode() == 0, nil
	case <-ctx.Done():
		return false, fmt.Errorf("file check cancelled: %w", ctx.Err())
	}
}

// readComposeFile reads the content of the Docker Compose file
func readComposeFile(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, filePath string, extractResult *DockerComposeExtractResult) (string, error) {
	readCmd := fmt.Sprintf("cat %s", filePath)
	extractResult.LogChan <- "Reading Docker Compose file content"

	sshResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, readCmd, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to read compose file: %w", err)
	}

	var content strings.Builder

	// Collect all stdout output
	go func() {
		for {
			select {
			case stdout, ok := <-sshResult.StdoutChan:
				if !ok {
					return
				}
				content.WriteString(stdout)
				content.WriteString("\n")
			case stderr, ok := <-sshResult.StderrChan:
				if !ok {
					return
				}
				extractResult.LogChan <- fmt.Sprintf("Read error: %s", stderr)
			case err, ok := <-sshResult.ErrorChan:
				if !ok {
					return
				}
				extractResult.ErrorChan <- err
			}
		}
	}()

	// Wait for read to complete
	select {
	case <-sshResult.DoneChan:
		if sshResult.GetFinalError() != nil {
			return "", fmt.Errorf("failed to read compose file: %w", sshResult.GetFinalError())
		}
	case <-ctx.Done():
		return "", fmt.Errorf("compose file read cancelled: %w", ctx.Err())
	}

	composeContent := strings.TrimSpace(content.String())
	if composeContent == "" {
		return "", fmt.Errorf("docker Compose file is empty")
	}

	return composeContent, nil
}

// validateComposeFile validates the Docker Compose file format
func validateComposeFile(composeContent string) error {
	// Use the existing dockeryaml parser to validate the compose file
	composeProject, err := dockeryaml.ParseComposeContent(composeContent, "git-service")
	if err != nil {
		return fmt.Errorf("compose file validation failed: %w", err)
	}

	// Additional validation using the parsed compose file
	if err := dockeryaml.Validate(composeProject); err != nil {
		return fmt.Errorf("compose file validation failed: %w", err)
	}

	return nil
}
