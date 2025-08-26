package git

import (
	"context"
	"fmt"
	"time"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/connection"
)

// GitWorkflowResult provides streaming output from the complete git workflow
type GitWorkflowResult struct {
	LogChan     chan string
	ErrorChan   chan error
	DoneChan    chan struct{}
	ComposeFile string
	ClonePath   string
	finalError  error
}

// NewGitWorkflowResult creates a new GitWorkflowResult
func NewGitWorkflowResult() *GitWorkflowResult {
	return &GitWorkflowResult{
		LogChan:   make(chan string, 200), // Larger buffer for combined operations
		ErrorChan: make(chan error, 20),
		DoneChan:  make(chan struct{}),
	}
}

// GetFinalError returns the final error from the workflow
func (gwr *GitWorkflowResult) GetFinalError() error {
	return gwr.finalError
}

// GitWorkflowConfig contains configuration for the git workflow
type GitWorkflowConfig struct {
	ServiceID             string
	RepoURL               string
	Branch                string
	DockerComposeFilePath *string
	ConnectionPool        *connection.ConnectionPool
	ConnectionID          string
	Host                  string
	PrivateKeyContent     []byte
	Timeout               time.Duration
}

// ExecuteGitWorkflow executes the complete git workflow as described in workflow.md
// This implements the workflow:
// 1. Clone git repository to /data/starker/services/{serviceID}
// 2. Extract Docker Compose file from the repository
// 3. Validate the compose file
// 4. Return the compose file content
// 5. Handle errors and cleanup on failure
func ExecuteGitWorkflow(ctx context.Context, config *GitWorkflowConfig) (*GitWorkflowResult, error) {
	if config == nil {
		return nil, fmt.Errorf("git workflow config cannot be nil")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute // Default total workflow timeout
	}

	// Create workflow result
	workflowResult := NewGitWorkflowResult()

	// Set the clone path following the workflow specification
	clonePath := fmt.Sprintf("/data/starker/services/%s", config.ServiceID)
	workflowResult.ClonePath = clonePath

	// Execute workflow in a goroutine for streaming
	go func() {
		defer close(workflowResult.DoneChan)
		defer close(workflowResult.LogChan)
		defer close(workflowResult.ErrorChan)

		workflowResult.LogChan <- "=== Starting Git Workflow ==="
		workflowResult.LogChan <- fmt.Sprintf("Service ID: %s", config.ServiceID)
		workflowResult.LogChan <- fmt.Sprintf("Repository: %s", config.RepoURL)
		workflowResult.LogChan <- fmt.Sprintf("Branch: %s", config.Branch)
		workflowResult.LogChan <- fmt.Sprintf("Target path: %s", clonePath)

		// Step 1: Clone the repository
		workflowResult.LogChan <- "=== Step 1: Cloning Repository ==="
		cloneResult, err := CloneRepository(ctx, config.ConnectionPool, config.ConnectionID, config.Host, config.PrivateKeyContent, config.RepoURL, clonePath, config.Branch, config.Timeout/2)
		if err != nil {
			workflowResult.finalError = fmt.Errorf("failed to start git clone: %w", err)
			workflowResult.ErrorChan <- workflowResult.finalError
			return
		}

		// Forward clone streaming output in a separate goroutine
		cloneDone := make(chan struct{})
		go func() {
			defer close(cloneDone)
			forwardCloneStreaming(cloneResult, workflowResult)
		}()

		// Wait for clone to complete
		select {
		case <-cloneResult.DoneChan:
			if cloneResult.GetFinalError() != nil {
				workflowResult.finalError = fmt.Errorf("git clone failed: %w", cloneResult.GetFinalError())
				workflowResult.ErrorChan <- workflowResult.finalError
				return
			}
		case <-ctx.Done():
			workflowResult.finalError = fmt.Errorf("git workflow cancelled during clone: %w", ctx.Err())
			workflowResult.ErrorChan <- workflowResult.finalError
			return
		}

		// Wait for clone streaming to finish with timeout
		select {
		case <-cloneDone:
		case <-time.After(5 * time.Second):
			// Streaming timeout, continue anyway
		}

		workflowResult.LogChan <- "=== Step 2: Extracting Docker Compose File ==="

		// Step 2: Extract Docker Compose file
		customComposePath := ""
		if config.DockerComposeFilePath != nil {
			customComposePath = *config.DockerComposeFilePath
		}

		extractResult, err := ExtractDockerCompose(ctx, config.ConnectionPool, config.ConnectionID, config.Host, config.PrivateKeyContent, clonePath, customComposePath)
		if err != nil {
			workflowResult.finalError = fmt.Errorf("failed to start compose extraction: %w", err)
			workflowResult.ErrorChan <- workflowResult.finalError
			// Cleanup on failure
			CleanupRepository(ctx, config.ConnectionPool, config.ConnectionID, config.Host, config.PrivateKeyContent, clonePath)
			return
		}

		// Forward extraction streaming output in a separate goroutine
		extractDone := make(chan struct{})
		go func() {
			defer close(extractDone)
			forwardExtractionStreaming(extractResult, workflowResult)
		}()

		// Wait for extraction to complete
		select {
		case <-extractResult.DoneChan:
			if extractResult.GetFinalError() != nil {
				workflowResult.finalError = fmt.Errorf("compose extraction failed: %w", extractResult.GetFinalError())
				workflowResult.ErrorChan <- workflowResult.finalError
				// Cleanup on failure
				CleanupRepository(ctx, config.ConnectionPool, config.ConnectionID, config.Host, config.PrivateKeyContent, clonePath)
				return
			}
		case <-ctx.Done():
			workflowResult.finalError = fmt.Errorf("git workflow cancelled during extraction: %w", ctx.Err())
			workflowResult.ErrorChan <- workflowResult.finalError
			// Cleanup on cancellation
			CleanupRepository(ctx, config.ConnectionPool, config.ConnectionID, config.Host, config.PrivateKeyContent, clonePath)
			return
		}

		// Wait for extraction streaming to finish with timeout
		select {
		case <-extractDone:
		case <-time.After(5 * time.Second):
			// Streaming timeout, continue anyway
		}

		// Set the extracted compose file content
		workflowResult.ComposeFile = extractResult.ComposeFile

		workflowResult.LogChan <- "=== Git Workflow Completed Successfully ==="
		workflowResult.LogChan <- fmt.Sprintf("Repository cloned to: %s", clonePath)
		workflowResult.LogChan <- fmt.Sprintf("Docker Compose file found: %s", extractResult.FilePath)
		workflowResult.LogChan <- fmt.Sprintf("Compose file size: %d bytes", len(extractResult.ComposeFile))
	}()

	return workflowResult, nil
}

// forwardCloneStreaming forwards streaming output from clone operation
func forwardCloneStreaming(cloneResult *GitCloneResult, workflowResult *GitWorkflowResult) {
	defer func() {
		if r := recover(); r != nil {
			// Recover from panic if trying to send on closed channel
		}
	}()

	for {
		select {
		case log, ok := <-cloneResult.LogChan:
			if !ok {
				return
			}
			// Safe channel write with recovery
			func() {
				defer func() { recover() }()
				workflowResult.LogChan <- fmt.Sprintf("[CLONE] %s", log)
			}()
		case err, ok := <-cloneResult.ErrorChan:
			if !ok {
				return
			}
			// Safe channel write with recovery
			func() {
				defer func() { recover() }()
				workflowResult.ErrorChan <- fmt.Errorf("clone error: %w", err)
			}()
		}
	}
}

// forwardExtractionStreaming forwards streaming output from extraction operation
func forwardExtractionStreaming(extractResult *DockerComposeExtractResult, workflowResult *GitWorkflowResult) {
	defer func() {
		if r := recover(); r != nil {
			// Recover from panic if trying to send on closed channel
		}
	}()

	for {
		select {
		case log, ok := <-extractResult.LogChan:
			if !ok {
				return
			}
			// Safe channel write with recovery
			func() {
				defer func() { recover() }()
				workflowResult.LogChan <- fmt.Sprintf("[EXTRACT] %s", log)
			}()
		case err, ok := <-extractResult.ErrorChan:
			if !ok {
				return
			}
			// Safe channel write with recovery
			func() {
				defer func() { recover() }()
				workflowResult.ErrorChan <- fmt.Errorf("extraction error: %w", err)
			}()
		}
	}
}

// BuildGitWorkflowConfig builds a GitWorkflowConfig from service models and connection info
func BuildGitWorkflowConfig(serviceID string, gitSource *models.ServiceSourceGit, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte) *GitWorkflowConfig {
	return &GitWorkflowConfig{
		ServiceID:             serviceID,
		RepoURL:               gitSource.RepoURL,
		Branch:                gitSource.Branch,
		DockerComposeFilePath: gitSource.DockerComposeFilePath,
		ConnectionPool:        connectionPool,
		ConnectionID:          connectionID,
		Host:                  host,
		PrivateKeyContent:     privateKeyContent,
		Timeout:               10 * time.Minute,
	}
}
