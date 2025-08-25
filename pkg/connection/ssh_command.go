package connection

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHCommandResult provides streaming output from SSH command execution
type SSHCommandResult struct {
	StdoutChan chan string
	StderrChan chan string
	ErrorChan  chan error
	DoneChan   chan struct{}
	finalError error
	exitCode   int
}

// NewSSHCommandResult creates a new SSHCommandResult
func NewSSHCommandResult() *SSHCommandResult {
	return &SSHCommandResult{
		StdoutChan: make(chan string, 100),
		StderrChan: make(chan string, 100),
		ErrorChan:  make(chan error, 10),
		DoneChan:   make(chan struct{}),
		exitCode:   -1,
	}
}

// GetFinalError returns the final error from the command execution
func (sr *SSHCommandResult) GetFinalError() error {
	return sr.finalError
}

// GetExitCode returns the exit code of the executed command
func (sr *SSHCommandResult) GetExitCode() int {
	return sr.exitCode
}

// ExecuteSSHCommand executes a command via SSH with real-time streaming output using existing connection infrastructure
func (p *ConnectionPool) ExecuteSSHCommand(ctx context.Context, connectionID, host string, privateKeyContent []byte, command string, timeout time.Duration) (*SSHCommandResult, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Try to reuse existing SSH connection from Docker connection first
	sshClient, err := p.getOrCreateSSHConnection(connectionID, host, privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection: %w", err)
	}

	// Create streaming result
	streamResult := NewSSHCommandResult()

	// Execute command in a goroutine for streaming
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.StdoutChan)
		defer close(streamResult.StderrChan)
		defer close(streamResult.ErrorChan)
		// Note: Don't close sshClient here since it's managed by the pool

		// Create SSH session
		session, err := sshClient.NewSession()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create SSH session: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer session.Close()

		// Get pipes for stdout and stderr
		stdout, err := session.StdoutPipe()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create stdout pipe: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		stderr, err := session.StderrPipe()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create stderr pipe: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Start the command
		if err := session.Start(command); err != nil {
			streamResult.finalError = fmt.Errorf("failed to start command: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Stream stdout and stderr concurrently
		done := make(chan struct{}, 2)

		// Stream stdout
		go func() {
			defer func() { done <- struct{}{} }()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				select {
				case streamResult.StdoutChan <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				streamResult.ErrorChan <- fmt.Errorf("stdout scanning error: %w", err)
			}
		}()

		// Stream stderr
		go func() {
			defer func() { done <- struct{}{} }()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				select {
				case streamResult.StderrChan <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				streamResult.ErrorChan <- fmt.Errorf("stderr scanning error: %w", err)
			}
		}()

		// Wait for the command to finish with timeout
		cmdDone := make(chan error, 1)
		go func() {
			cmdDone <- session.Wait()
		}()

		// Wait for both stdout/stderr streaming to complete and command to finish
		streamingDone := 0
		for streamingDone < 2 {
			select {
			case <-done:
				streamingDone++
			case err := <-cmdDone:
				if err != nil {
					if exitErr, ok := err.(*ssh.ExitError); ok {
						streamResult.exitCode = exitErr.ExitStatus()
						streamResult.finalError = fmt.Errorf("command failed with exit code %d", exitErr.ExitStatus())
					} else {
						streamResult.finalError = fmt.Errorf("command execution failed: %w", err)
					}
				} else {
					streamResult.exitCode = 0
				}
			case <-ctx.Done():
				streamResult.finalError = fmt.Errorf("command execution cancelled: %w", ctx.Err())
				return
			case <-time.After(timeout):
				streamResult.finalError = fmt.Errorf("command execution timed out after %v", timeout)
				session.Signal(ssh.SIGKILL)
				return
			}
		}

		// Wait for command completion if not already done
		if streamResult.exitCode == -1 {
			select {
			case err := <-cmdDone:
				if err != nil {
					if exitErr, ok := err.(*ssh.ExitError); ok {
						streamResult.exitCode = exitErr.ExitStatus()
						streamResult.finalError = fmt.Errorf("command failed with exit code %d", exitErr.ExitStatus())
					} else {
						streamResult.finalError = fmt.Errorf("command execution failed: %w", err)
					}
				} else {
					streamResult.exitCode = 0
				}
			case <-ctx.Done():
				streamResult.finalError = fmt.Errorf("command execution cancelled: %w", ctx.Err())
				return
			case <-time.After(5 * time.Second):
				streamResult.finalError = fmt.Errorf("command completion timed out")
				return
			}
		}
	}()

	return streamResult, nil
}

// ExecuteInteractiveSSHCommand executes a command that may require interactive input
func (p *ConnectionPool) ExecuteInteractiveSSHCommand(ctx context.Context, connectionID, host string, privateKeyContent []byte, command string, input string, timeout time.Duration) (*SSHCommandResult, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Try to reuse existing SSH connection from Docker connection first
	sshClient, err := p.getOrCreateSSHConnection(connectionID, host, privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection: %w", err)
	}

	// Create streaming result
	streamResult := NewSSHCommandResult()

	// Execute command in a goroutine for streaming
	go func() {
		defer close(streamResult.DoneChan)
		defer close(streamResult.StdoutChan)
		defer close(streamResult.StderrChan)
		defer close(streamResult.ErrorChan)
		// Note: Don't close sshClient here since it's managed by the pool

		// Create SSH session
		session, err := sshClient.NewSession()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create SSH session: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}
		defer session.Close()

		// Get pipes for stdin, stdout and stderr
		stdin, err := session.StdinPipe()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create stdin pipe: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create stdout pipe: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		stderr, err := session.StderrPipe()
		if err != nil {
			streamResult.finalError = fmt.Errorf("failed to create stderr pipe: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Start the command
		if err := session.Start(command); err != nil {
			streamResult.finalError = fmt.Errorf("failed to start command: %w", err)
			streamResult.ErrorChan <- streamResult.finalError
			return
		}

		// Send input if provided
		if input != "" {
			go func() {
				defer stdin.Close()
				if _, err := io.WriteString(stdin, input); err != nil {
					streamResult.ErrorChan <- fmt.Errorf("failed to write input: %w", err)
				}
			}()
		} else {
			stdin.Close()
		}

		// Stream stdout and stderr concurrently
		done := make(chan struct{}, 2)

		// Stream stdout
		go func() {
			defer func() { done <- struct{}{} }()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				select {
				case streamResult.StdoutChan <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				streamResult.ErrorChan <- fmt.Errorf("stdout scanning error: %w", err)
			}
		}()

		// Stream stderr
		go func() {
			defer func() { done <- struct{}{} }()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				select {
				case streamResult.StderrChan <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				streamResult.ErrorChan <- fmt.Errorf("stderr scanning error: %w", err)
			}
		}()

		// Wait for the command to finish with timeout
		cmdDone := make(chan error, 1)
		go func() {
			cmdDone <- session.Wait()
		}()

		// Wait for both stdout/stderr streaming to complete and command to finish
		streamingDone := 0
		for streamingDone < 2 {
			select {
			case <-done:
				streamingDone++
			case err := <-cmdDone:
				if err != nil {
					if exitErr, ok := err.(*ssh.ExitError); ok {
						streamResult.exitCode = exitErr.ExitStatus()
						streamResult.finalError = fmt.Errorf("command failed with exit code %d", exitErr.ExitStatus())
					} else {
						streamResult.finalError = fmt.Errorf("command execution failed: %w", err)
					}
				} else {
					streamResult.exitCode = 0
				}
			case <-ctx.Done():
				streamResult.finalError = fmt.Errorf("command execution cancelled: %w", ctx.Err())
				return
			case <-time.After(timeout):
				streamResult.finalError = fmt.Errorf("command execution timed out after %v", timeout)
				session.Signal(ssh.SIGKILL)
				return
			}
		}

		// Wait for command completion if not already done
		if streamResult.exitCode == -1 {
			select {
			case err := <-cmdDone:
				if err != nil {
					if exitErr, ok := err.(*ssh.ExitError); ok {
						streamResult.exitCode = exitErr.ExitStatus()
						streamResult.finalError = fmt.Errorf("command failed with exit code %d", exitErr.ExitStatus())
					} else {
						streamResult.finalError = fmt.Errorf("command execution failed: %w", err)
					}
				} else {
					streamResult.exitCode = 0
				}
			case <-ctx.Done():
				streamResult.finalError = fmt.Errorf("command execution cancelled: %w", ctx.Err())
				return
			case <-time.After(5 * time.Second):
				streamResult.finalError = fmt.Errorf("command completion timed out")
				return
			}
		}
	}()

	return streamResult, nil
}

// getOrCreateSSHConnection tries to reuse an existing SSH connection from a Docker connection
// or creates a new SSH connection if needed. This optimizes resource usage.
func (p *ConnectionPool) getOrCreateSSHConnection(connectionID, host string, privateKeyContent []byte) (*ssh.Client, error) {
	p.mutex.RLock()

	// Check if we already have a connection with this ID
	if connInfo, exists := p.connections[connectionID]; exists {
		// If we have a Docker connection, we can reuse its SSH client
		if connInfo.connType == DockerConnection && connInfo.sshClient != nil {
			// Check if connection is still valid
			if p.isConnectionValid(connInfo) {
				connInfo.lastUsed = time.Now()
				p.mutex.RUnlock()
				return connInfo.sshClient, nil
			}
		}
		// If we have an SSH connection, use it directly
		if connInfo.connType == SSHConnection && connInfo.sshClient != nil {
			if p.isConnectionValid(connInfo) {
				connInfo.lastUsed = time.Now()
				p.mutex.RUnlock()
				return connInfo.sshClient, nil
			}
		}
	}
	p.mutex.RUnlock()

	// No reusable connection found, create a new SSH connection
	return p.GetSSHConnection(connectionID, host, privateKeyContent)
}
