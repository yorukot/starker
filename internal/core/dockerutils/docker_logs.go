package dockerutils

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/yorukot/starker/pkg/connection"
)

// GetContainerLogs retrieves Docker container logs with streaming support
func (h *DockerHandler) GetContainerLogs(ctx context.Context, containerID string, options LogOptions) (io.ReadCloser, error) {
	h.StreamChan.LogLog(fmt.Sprintf("Getting logs for container: %s", containerID))

	// Build docker logs command with options
	logCmd := h.buildLogCommand(containerID, options)
	h.StreamChan.LogLog(fmt.Sprintf("Executing command: %s", logCmd))

	// Create SSH session for log streaming
	session, err := h.Client.NewSession()
	if err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to create SSH session: %v", err))
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Get stdout pipe for log streaming
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		h.StreamChan.LogError(fmt.Sprintf("Failed to get stdout pipe: %v", err))
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the log command
	if err := session.Start(logCmd); err != nil {
		session.Close()
		h.StreamChan.LogError(fmt.Sprintf("Failed to start logs command: %v", err))
		return nil, fmt.Errorf("failed to start logs command: %w", err)
	}

	// Return a custom ReadCloser that handles both stdout and session cleanup
	return &logReader{
		reader:  stdout,
		session: session,
	}, nil
}

// GetContainerLogsStreaming retrieves Docker container logs with streaming via StreamChan
func (h *DockerHandler) GetContainerLogsStreaming(ctx context.Context, containerID string, options LogOptions) error {
	h.StreamChan.LogLog(fmt.Sprintf("Starting log stream for container: %s", containerID))

	// Build docker logs command with options
	logCmd := h.buildLogCommand(containerID, options)

	// Execute the logs command with streaming
	if err := connection.ExecuteCommand(h.Client, logCmd, h.StreamChan); err != nil {
		h.StreamChan.LogError(fmt.Sprintf("Failed to stream container logs: %v", err))
		h.StreamChan.FinalError <- err
		return err
	}

	return nil
}

// buildLogCommand constructs the docker logs command with the specified options
func (h *DockerHandler) buildLogCommand(containerID string, options LogOptions) string {
	cmd := []string{"docker", "logs"}

	// Add follow option for real-time streaming
	if options.Follow {
		cmd = append(cmd, "--follow")
	}

	// Add tail option to limit number of lines
	if options.Tail != "" && options.Tail != "0" {
		cmd = append(cmd, "--tail", options.Tail)
	}

	// Add timestamps if requested
	if options.Timestamps {
		cmd = append(cmd, "--timestamps")
	}

	// Add since option if specified
	if !options.Since.IsZero() {
		cmd = append(cmd, "--since", options.Since.Format(time.RFC3339))
	}

	// Add container ID
	cmd = append(cmd, containerID)

	return strings.Join(cmd, " ")
}

// logReader implements io.ReadCloser to handle both stdout reading and session cleanup
type logReader struct {
	reader  io.Reader
	session *ssh.Session
}

func (lr *logReader) Read(p []byte) (n int, err error) {
	return lr.reader.Read(p)
}

func (lr *logReader) Close() error {
	if lr.session != nil {
		return lr.session.Close()
	}
	return nil
}
