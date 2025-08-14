package sshpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectionInfo is the information of a connection
type ConnectionInfo struct {
	client    *ssh.Client
	lastUsed  time.Time
	createdAt time.Time
}

// CommandResult is the result of a command execution
type CommandResult struct {
	Command string
	Stdout  string
	Stderr  string
	Error   error
}

// SSHConnectionPool is a pool of SSH connections
type SSHConnectionPool struct {
	connections map[string]*ConnectionInfo
	mutex       sync.RWMutex
	maxIdle     time.Duration
	maxLifetime time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
}

// NewSSHConnectionPool creates a new SSH connection pool
func NewSSHConnectionPool(maxIdle, maxLifetime time.Duration) *SSHConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &SSHConnectionPool{
		connections: make(map[string]*ConnectionInfo),
		maxIdle:     maxIdle,
		maxLifetime: maxLifetime,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start cleanup goroutine
	go pool.startCleanup()

	return pool
}

// GetConnection gets a connection from the pool and if the connection is not found, it will create a new connection
func (p *SSHConnectionPool) GetConnection(host string, config *ssh.ClientConfig) (*ssh.Client, error) {
	p.mutex.RLock()
	if connInfo, exists := p.connections[host]; exists {
		// Check if connection is still valid and not expired
		if p.isConnectionValid(connInfo) {
			connInfo.lastUsed = time.Now()
			p.mutex.RUnlock()
			return connInfo.client, nil
		}
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if pool is closed
	if p.closed {
		return nil, fmt.Errorf("connection pool is closed")
	}

	// Double-check pattern
	if connInfo, exists := p.connections[host]; exists {
		if p.isConnectionValid(connInfo) {
			connInfo.lastUsed = time.Now()
			return connInfo.client, nil
		} else {
			// Clean up invalid connection
			p.removeConnection(host, connInfo)
		}
	}

	// Create new connection
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	now := time.Now()
	p.connections[host] = &ConnectionInfo{
		client:    client,
		lastUsed:  now,
		createdAt: now,
	}

	return client, nil
}

// NewSession creates a new session on the given host and returns the session
func (p *SSHConnectionPool) NewSession(host string, config *ssh.ClientConfig) (*ssh.Session, error) {
	client, err := p.GetConnection(host, config)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		// If session creation fails, connection might be broken, mark for removal
		p.markConnectionForRemoval(host)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ExecuteCommands executes a command on the given host and returns the result
func (p *SSHConnectionPool) ExecuteCommands(host string, config *ssh.ClientConfig, command string) CommandResult {
	client, err := p.GetConnection(host, config)
	if err != nil {
		return CommandResult{
			Command: command,
			Error:   err,
		}
	}

	session, err := client.NewSession()
	if err != nil {
		p.markConnectionForRemoval(host)
		return CommandResult{
			Command: command,
			Error:   fmt.Errorf("failed to create session for command '%s': %w", command, err),
		}
	}
	defer session.Close()

	var stdout, stderr []byte

	stdout, err = session.Output(command)
	if err != nil {
		// Try to get stderr if command failed
		if exitError, ok := err.(*ssh.ExitError); ok {
			stderr = []byte(exitError.Error())
		}
		p.markConnectionForRemoval(host)
		return CommandResult{
			Command: command,
			Stdout:  string(stdout),
			Stderr:  string(stderr),
			Error:   fmt.Errorf("command '%s' failed: %w", command, err),
		}
	}

	return CommandResult{
		Command: command,
		Stdout:  string(stdout),
		Stderr:  string(stderr),
		Error:   nil,
	}
}

// Stats Get connection pool statistics
func (p *SSHConnectionPool) Stats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(p.connections),
		"connections":       make(map[string]map[string]interface{}),
	}

	now := time.Now()
	for host, connInfo := range p.connections {
		stats["connections"].(map[string]map[string]interface{})[host] = map[string]interface{}{
			"created_at":    connInfo.createdAt,
			"last_used":     connInfo.lastUsed,
			"idle_duration": now.Sub(connInfo.lastUsed),
			"age":           now.Sub(connInfo.createdAt),
		}
	}

	return stats
}
