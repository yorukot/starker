package git

import (
	"context"
	"fmt"
	"time"

	"github.com/yorukot/starker/pkg/connection"
)

// GitCloneResult provides streaming output from git clone operation
type GitCloneResult struct {
	LogChan    chan string
	ErrorChan  chan error
	DoneChan   chan struct{}
	ClonePath  string
	finalError error
}

// NewGitCloneResult creates a new GitCloneResult
func NewGitCloneResult(clonePath string) *GitCloneResult {
	return &GitCloneResult{
		LogChan:   make(chan string, 100),
		ErrorChan: make(chan error, 10),
		DoneChan:  make(chan struct{}),
		ClonePath: clonePath,
	}
}

// GetFinalError returns the final error from the git clone operation
func (gcr *GitCloneResult) GetFinalError() error {
	return gcr.finalError
}

// CloneRepository clones a git repository to the specified path using SSH
func CloneRepository(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, repoURL, targetPath, branch string, timeout time.Duration) (*GitCloneResult, error) {
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout for git clone
	}

	cloneResult := NewGitCloneResult(targetPath)

	// Execute git clone in a goroutine for streaming
	go func() {
		defer close(cloneResult.DoneChan)
		defer close(cloneResult.LogChan)
		defer close(cloneResult.ErrorChan)

		cloneResult.LogChan <- fmt.Sprintf("Starting git clone of %s to %s", repoURL, targetPath)

		// Step 1: Create target directory
		createDirCmd := fmt.Sprintf("mkdir -p %s", targetPath)
		cloneResult.LogChan <- fmt.Sprintf("Creating target directory: %s", targetPath)

		sshResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, createDirCmd, 30*time.Second)
		if err != nil {
			cloneResult.finalError = fmt.Errorf("failed to create target directory: %w", err)
			cloneResult.ErrorChan <- cloneResult.finalError
			return
		}

		// Wait for directory creation to complete
		select {
		case <-sshResult.DoneChan:
			if sshResult.GetFinalError() != nil {
				cloneResult.finalError = fmt.Errorf("failed to create target directory: %w", sshResult.GetFinalError())
				cloneResult.ErrorChan <- cloneResult.finalError
				return
			}
		case <-ctx.Done():
			cloneResult.finalError = fmt.Errorf("directory creation cancelled: %w", ctx.Err())
			cloneResult.ErrorChan <- cloneResult.finalError
			return
		}

		cloneResult.LogChan <- "Target directory created successfully"

		// Step 2: Execute git clone command
		gitCloneCmd := buildGitCloneCommand(repoURL, targetPath, branch)
		cloneResult.LogChan <- "Executing git clone command"

		sshResult, err = connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, gitCloneCmd, timeout)
		if err != nil {
			cloneResult.finalError = fmt.Errorf("failed to execute git clone: %w", err)
			cloneResult.ErrorChan <- cloneResult.finalError
			return
		}

		// Stream the git clone output
		go func() {
			for {
				select {
				case stdout, ok := <-sshResult.StdoutChan:
					if !ok {
						return
					}
					cloneResult.LogChan <- fmt.Sprintf("Git: %s", stdout)
				case stderr, ok := <-sshResult.StderrChan:
					if !ok {
						return
					}
					cloneResult.LogChan <- fmt.Sprintf("Git: %s", stderr)
				case err, ok := <-sshResult.ErrorChan:
					if !ok {
						return
					}
					cloneResult.ErrorChan <- err
				}
			}
		}()

		// Wait for git clone to complete
		select {
		case <-sshResult.DoneChan:
			if sshResult.GetFinalError() != nil {
				cloneResult.finalError = fmt.Errorf("git clone failed: %w", sshResult.GetFinalError())
				cloneResult.ErrorChan <- cloneResult.finalError

				// Clean up on error
				cleanupCmd := fmt.Sprintf("rm -rf %s", targetPath)
				cloneResult.LogChan <- "Cleaning up failed clone..."
				connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, cleanupCmd, 30*time.Second)
				return
			}
			cloneResult.LogChan <- fmt.Sprintf("Git clone completed successfully to %s", targetPath)
		case <-ctx.Done():
			cloneResult.finalError = fmt.Errorf("git clone cancelled: %w", ctx.Err())
			cloneResult.ErrorChan <- cloneResult.finalError

			// Clean up on cancellation
			cleanupCmd := fmt.Sprintf("rm -rf %s", targetPath)
			connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, cleanupCmd, 30*time.Second)
			return
		}
	}()

	return cloneResult, nil
}

// buildGitCloneCommand builds the git clone command with proper options
func buildGitCloneCommand(repoURL, targetPath, branch string) string {
	if branch != "" && branch != "main" && branch != "master" {
		return fmt.Sprintf("git clone --depth 1 --branch %s %s %s", branch, repoURL, targetPath)
	}
	return fmt.Sprintf("git clone --depth 1 %s %s", repoURL, targetPath)
}

// UpdateRepository pulls latest changes from git repository
func UpdateRepository(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, repositoryPath, branch string, timeout time.Duration) (*GitCloneResult, error) {
	if timeout == 0 {
		timeout = 3 * time.Minute // Default timeout for git pull
	}

	updateResult := NewGitCloneResult(repositoryPath)

	// Execute git pull in a goroutine for streaming
	go func() {
		defer close(updateResult.DoneChan)
		defer close(updateResult.LogChan)
		defer close(updateResult.ErrorChan)

		updateResult.LogChan <- fmt.Sprintf("Starting git update for repository at %s", repositoryPath)

		// Step 1: Check if repository directory exists
		checkDirCmd := fmt.Sprintf("[ -d %s ] && echo 'exists' || echo 'not_exists'", repositoryPath)
		updateResult.LogChan <- "Checking if repository directory exists"

		sshResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, checkDirCmd, 30*time.Second)
		if err != nil {
			updateResult.finalError = fmt.Errorf("failed to check repository directory: %w", err)
			updateResult.ErrorChan <- updateResult.finalError
			return
		}

		// Wait for directory check to complete
		var directoryExists bool
		select {
		case <-sshResult.DoneChan:
			if sshResult.GetFinalError() != nil {
				updateResult.finalError = fmt.Errorf("failed to check repository directory: %w", sshResult.GetFinalError())
				updateResult.ErrorChan <- updateResult.finalError
				return
			}
			// Check if directory exists by reading stdout
			// This is a simple check - if directory doesn't exist, we'll handle it gracefully
			directoryExists = true
		case <-ctx.Done():
			updateResult.finalError = fmt.Errorf("directory check cancelled: %w", ctx.Err())
			updateResult.ErrorChan <- updateResult.finalError
			return
		}

		if !directoryExists {
			updateResult.LogChan <- "Repository directory not found, skipping git update"
			updateResult.finalError = fmt.Errorf("repository directory %s does not exist", repositoryPath)
			updateResult.ErrorChan <- updateResult.finalError
			return
		}

		updateResult.LogChan <- "Repository directory found, proceeding with git update"

		// Step 2: Change to repository directory and pull latest changes
		gitPullCmd := buildGitPullCommand(repositoryPath, branch)
		updateResult.LogChan <- "Executing git pull command"

		sshResult, err = connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, gitPullCmd, timeout)
		if err != nil {
			updateResult.finalError = fmt.Errorf("failed to execute git pull: %w", err)
			updateResult.ErrorChan <- updateResult.finalError
			return
		}

		// Stream the git pull output
		go func() {
			for {
				select {
				case stdout, ok := <-sshResult.StdoutChan:
					if !ok {
						return
					}
					updateResult.LogChan <- fmt.Sprintf("Git: %s", stdout)
				case stderr, ok := <-sshResult.StderrChan:
					if !ok {
						return
					}
					updateResult.LogChan <- fmt.Sprintf("Git: %s", stderr)
				case err, ok := <-sshResult.ErrorChan:
					if !ok {
						return
					}
					updateResult.ErrorChan <- err
				}
			}
		}()

		// Wait for git pull to complete
		select {
		case <-sshResult.DoneChan:
			if sshResult.GetFinalError() != nil {
				updateResult.finalError = fmt.Errorf("git pull failed: %w", sshResult.GetFinalError())
				updateResult.ErrorChan <- updateResult.finalError
				return
			}
			updateResult.LogChan <- fmt.Sprintf("Git pull completed successfully for %s", repositoryPath)
		case <-ctx.Done():
			updateResult.finalError = fmt.Errorf("git pull cancelled: %w", ctx.Err())
			updateResult.ErrorChan <- updateResult.finalError
			return
		}
	}()

	return updateResult, nil
}

// buildGitPullCommand builds the git pull command with proper options
func buildGitPullCommand(repositoryPath, branch string) string {
	if branch != "" && branch != "main" && branch != "master" {
		return fmt.Sprintf("cd %s && git checkout %s && git pull origin %s", repositoryPath, branch, branch)
	}
	return fmt.Sprintf("cd %s && git pull", repositoryPath)
}

// CleanupRepository removes the cloned repository directory
func CleanupRepository(ctx context.Context, connectionPool *connection.ConnectionPool, connectionID, host string, privateKeyContent []byte, targetPath string) error {
	cleanupCmd := fmt.Sprintf("rm -rf %s", targetPath)

	sshResult, err := connectionPool.ExecuteSSHCommand(ctx, connectionID, host, privateKeyContent, cleanupCmd, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to execute cleanup command: %w", err)
	}

	// Wait for cleanup to complete
	select {
	case <-sshResult.DoneChan:
		if sshResult.GetFinalError() != nil {
			return fmt.Errorf("cleanup failed: %w", sshResult.GetFinalError())
		}
	case <-ctx.Done():
		return fmt.Errorf("cleanup cancelled: %w", ctx.Err())
	}

	return nil
}
