package sshpool

import (
	"bufio"
	"fmt"
	"sync"

	"golang.org/x/crypto/ssh"
)

// StreamingCommandResult is the result of a streaming command execution
type StreamingCommandResult struct {
	Command    string
	StdoutChan chan string
	StderrChan chan string
	ErrorChan  chan error
	DoneChan   chan bool
	mutex      sync.RWMutex
	stdout     []string
	stderr     []string
	finished   bool
	finalError error
}

// ExcuteCommandStreaming executes a command on the given host and returns the streaming result
func (p *SSHConnectionPool) ExcuteCommandStreaming(host string, config *ssh.ClientConfig, command string) *StreamingCommandResult {
	result := &StreamingCommandResult{
		Command:    command,
		StdoutChan: make(chan string, 100),
		StderrChan: make(chan string, 100),
		ErrorChan:  make(chan error, 1),
		DoneChan:   make(chan bool, 1),
		stdout:     make([]string, 0),
		stderr:     make([]string, 0),
		finished:   false,
	}

	go func() {
		defer close(result.StdoutChan)
		defer close(result.StderrChan)
		defer close(result.ErrorChan)
		defer close(result.DoneChan)
		defer func() {
			result.mutex.Lock()
			result.finished = true
			result.mutex.Unlock()
		}()

		client, err := p.GetConnection(host, config)
		if err != nil {
			result.ErrorChan <- err
			result.DoneChan <- true
			return
		}

		session, err := client.NewSession()
		if err != nil {
			p.markConnectionForRemoval(host)
			result.ErrorChan <- fmt.Errorf("failed to create session for command '%s': %w", command, err)
			result.DoneChan <- true
			return
		}
		defer session.Close()

		// Get pipes for stdout and stderr
		stdoutPipe, err := session.StdoutPipe()
		if err != nil {
			result.ErrorChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
			result.DoneChan <- true
			return
		}

		stderrPipe, err := session.StderrPipe()
		if err != nil {
			result.ErrorChan <- fmt.Errorf("failed to create stderr pipe: %w", err)
			result.DoneChan <- true
			return
		}

		// Start the command
		if err := session.Start(command); err != nil {
			result.ErrorChan <- fmt.Errorf("failed to start command '%s': %w", command, err)
			result.DoneChan <- true
			return
		}

		// Wait group to wait for both stdout and stderr readers
		var wg sync.WaitGroup
		wg.Add(2)

		// Read stdout
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				line := scanner.Text()
				result.mutex.Lock()
				result.stdout = append(result.stdout, line)
				result.mutex.Unlock()
				result.StdoutChan <- line
			}
		}()

		// Read stderr
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				line := scanner.Text()
				result.mutex.Lock()
				result.stderr = append(result.stderr, line)
				result.mutex.Unlock()
				result.StderrChan <- line
			}
		}()

		// Wait for command to finish
		err = session.Wait()
		if err != nil {
			result.mutex.Lock()
			result.finalError = err
			result.mutex.Unlock()
			result.ErrorChan <- err
		}

		// Wait for all output to be read
		wg.Wait()
		result.DoneChan <- true
	}()

	return result
}

// GetAccumulatedStdout returns the accumulated stdout output
func (scr *StreamingCommandResult) GetAccumulatedStdout() []string {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()
	result := make([]string, len(scr.stdout))
	copy(result, scr.stdout)
	return result
}

// GetAccumulatedStderr returns the accumulated stderr output
func (scr *StreamingCommandResult) GetAccumulatedStderr() []string {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()
	result := make([]string, len(scr.stderr))
	copy(result, scr.stderr)
	return result
}

// IsFinished returns whether the command execution has completed
func (scr *StreamingCommandResult) IsFinished() bool {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()
	return scr.finished
}

// GetFinalError returns the final error if any occurred during execution
func (scr *StreamingCommandResult) GetFinalError() error {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()
	return scr.finalError
}
