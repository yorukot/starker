package connection

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/yorukot/starker/internal/core"
	"golang.org/x/crypto/ssh"
)

// CommandExecutor handles SSH command execution with real-time streaming
type CommandExecutor struct {}

// NewCommandExecutor creates a new CommandExecutor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// ExecuteCommand executes a single SSH command with real-time output streaming
func (e *CommandExecutor) ExecuteCommand(sshClient *ssh.Client, command string, streamChan core.StreamChan) error {

	// Create SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		streamChan.LogError(fmt.Sprintf("Failed to create SSH session: %v", err))
		streamChan.FinalError <- err
		return err
	}
	defer session.Close()

	streamChan.LogStep(fmt.Sprintf("Executing command: %s", command))

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
		e.streamOutput(stdout, streamChan, false)
	}()

	// Stream stderr in real-time
	go func() {
		defer wg.Done()
		e.streamOutput(stderr, streamChan, true)
	}()

	// Wait for command to complete
	err = session.Wait()

	// Wait for all output to be processed
	wg.Wait()

	if err != nil {
		streamChan.LogError(fmt.Sprintf("Command execution failed: %v", err))
		streamChan.FinalError <- err
	} else {
		streamChan.LogStep("Command executed successfully")
	}

	streamChan.DoneChan <- true
	return err
}

// ExecuteMultipleCommands executes multiple SSH commands sequentially with real-time output streaming
func (e *CommandExecutor) ExecuteMultipleCommands(sshClient *ssh.Client, commands []string, streamChan core.StreamChan) error {
	streamChan.LogStep(fmt.Sprintf("Executing %d commands sequentially", len(commands)))

	for i, command := range commands {
		streamChan.LogStep(fmt.Sprintf("Command %d/%d: %s", i+1, len(commands), command))

		// Execute each command using the same connection
		if err := e.ExecuteCommand(sshClient, command, streamChan); err != nil {
			streamChan.LogError(fmt.Sprintf("Failed at command %d/%d: %v", i+1, len(commands), err))
			return err
		}
	}

	streamChan.LogStep("All commands executed successfully")
	return nil
}

// ExecuteSimpleCommand executes a single SSH command and returns the output synchronously
func (e *CommandExecutor) ExecuteSimpleCommand(sshClient *ssh.Client, command string) (stdout, stderr string, err error) {
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
func (e *CommandExecutor) streamOutput(reader io.Reader, streamChan core.StreamChan, isError bool) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if isError {
				streamChan.LogError(line)
			} else {
				streamChan.LogInfo(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		streamChan.LogError(fmt.Sprintf("Error reading output: %v", err))
	}
}
