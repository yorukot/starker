package connection

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/yorukot/starker/internal/core"
)

// ExecuteCommand executes a single SSH command with real-time output streaming
func ExecuteCommand(sshClient *ssh.Client, command string, streamChan core.StreamChan) error {

	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		streamChan.LogError(fmt.Sprintf("Failed to create SSH session: %v", err))
		streamChan.FinalError <- err
		return err
	}
	defer session.Close()

	streamChan.LogLog(fmt.Sprintf("Executing command: %s", command))

	// Get stdout and stderr pipes
	stdout, err := session.StdoutPipe()
	if err != nil {
		streamChan.LogError(fmt.Sprintf("Failed to get stdout pipe: %v", err))
		streamChan.FinalError <- err
		return err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		streamChan.LogError(fmt.Sprintf("Failed to get stderr pipe: %v", err))
		streamChan.FinalError <- err
		return err
	}

	// Start the command
	if err := session.Start(command); err != nil {
		streamChan.LogError(fmt.Sprintf("Failed to start command: %v", err))
		streamChan.FinalError <- err
		return err
	}

	// Use WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stdout in real-time
	go func() {
		defer wg.Done()
		streamOutput(stdout, streamChan, false)
	}()

	// Stream stderr in real-time
	go func() {
		defer wg.Done()
		streamOutput(stderr, streamChan, true)
	}()

	// Wait for command to complete
	err = session.Wait()

	// Wait for all output to be processed
	wg.Wait()

	if err != nil {
		streamChan.LogError(fmt.Sprintf("Command execution failed: %v", err))
		streamChan.FinalError <- err
	} else {
		streamChan.LogLog("Command executed successfully")
	}

	streamChan.DoneChan <- true
	return err
}

// ExecuteMultipleCommands executes multiple SSH commands sequentially with real-time output streaming
func ExecuteMultipleCommands(sshClient *ssh.Client, commands []string, streamChan core.StreamChan) error {
	streamChan.LogLog(fmt.Sprintf("Executing %d commands sequentially", len(commands)))

	for i, command := range commands {
		streamChan.LogLog(fmt.Sprintf("Command %d/%d: %s", i+1, len(commands), command))

		// Execute each command using the same connection
		if err := ExecuteCommand(sshClient, command, streamChan); err != nil {
			streamChan.LogError(fmt.Sprintf("Failed at command %d/%d: %v", i+1, len(commands), err))
			return err
		}
	}

	streamChan.LogLog("All commands executed successfully")
	return nil
}

// ExecuteSimpleCommand executes a single SSH command and returns the output synchronously
func ExecuteSimpleCommand(sshClient *ssh.Client, command string) (stdout, stderr string, err error) {
	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Create buffers to capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run the command
	err = session.Run(command)

	// Return captured output
	return stdoutBuf.String(), stderrBuf.String(), err
}

// streamOutput reads from an io.Reader and streams the output line by line
func streamOutput(reader io.Reader, streamChan core.StreamChan, isError bool) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if isError {
				// For stderr, check if it's actually a Docker progress message
				if isDockerProgressMessage(line) {
					streamChan.LogLog(line)
				} else {
					streamChan.LogError(line)
				}
			} else {
				streamChan.LogLog(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		streamChan.LogError(fmt.Sprintf("Error reading output: %v", err))
	}
}

// isDockerProgressMessage determines if a stderr line is actually Docker progress info
func isDockerProgressMessage(line string) bool {
	line = strings.ToLower(line)

	// Docker progress and status messages that appear on stderr but aren't errors
	progressKeywords := []string{
		"pulling",
		"pulled",
		"downloading",
		"downloaded",
		"extracting",
		"extracted",
		"creating",
		"created",
		"starting",
		"started",
		"stopping",
		"stopped",
		"removing",
		"removed",
		"restarting",
		"restarted",
		"waiting",
		"running",
		"exited",
	}

	for _, keyword := range progressKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	// Check for container/service name patterns followed by colons (e.g., "app: Pulling")
	if strings.Contains(line, ":") && (strings.Contains(line, "pull") || strings.Contains(line, "start") || strings.Contains(line, "creat")) {
		return true
	}

	// Check for progress indicators (percentages, progress bars)
	if strings.Contains(line, "%") || strings.Contains(line, "[") && strings.Contains(line, "]") {
		return true
	}

	// Check for image/layer hash patterns (common in Docker progress)
	if strings.Contains(line, "sha256:") {
		return true
	}

	return false
}
