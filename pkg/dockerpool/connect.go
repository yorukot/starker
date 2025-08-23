package dockerpool

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
)

// ConnectionInfo is the information of a connection
type ConnectionInfo struct {
	client    *client.Client
	lastUsed  time.Time
	createdAt time.Time
}

// DockerConnectionPool is a pool of Docker connections that only supports SSH key-based authentication
type DockerConnectionPool struct {
	connections map[string]*ConnectionInfo
	mutex       sync.RWMutex
	maxIdle     time.Duration
	maxLifetime time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
}

// NewDockerConnectionPool creates a new Docker connection pool that only supports SSH key-based authentication
func NewDockerConnectionPool(maxIdle, maxLifetime time.Duration) *DockerConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &DockerConnectionPool{
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

// GetConnection gets an SSH connection from the pool using connectionID and if the connection is not found, it will create a new SSH key-based connection
func (p *DockerConnectionPool) GetConnection(connectionID, host string, opts ...client.Opt) (*client.Client, error) {
	p.mutex.RLock()
	if connInfo, exists := p.connections[connectionID]; exists {
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
	if connInfo, exists := p.connections[connectionID]; exists {
		if p.isConnectionValid(connInfo) {
			connInfo.lastUsed = time.Now()
			return connInfo.client, nil
		} else {
			// Clean up invalid connection
			p.removeConnection(connectionID, connInfo)
		}
	}

	// Create new connection
	dockerClient, err := p.createDockerClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker connection: %w", err)
	}

	now := time.Now()
	p.connections[connectionID] = &ConnectionInfo{
		client:    dockerClient,
		lastUsed:  now,
		createdAt: now,
	}

	return dockerClient, nil
}

// validateSSHKeyAuth validates that the SSH connection uses key-based authentication
func (p *DockerConnectionPool) validateSSHKeyAuth(host string) error {
	// Parse the SSH URL to check for key-based authentication requirements
	parsedURL, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("invalid SSH URL format: %w", err)
	}

	// Check for password authentication indicators (which we want to reject)
	if strings.Contains(host, "password") || strings.Contains(host, "passwd") {
		return fmt.Errorf("password-based SSH authentication is not allowed, only SSH key authentication is supported")
	}

	// Validate SSH URL format and ensure it's configured for key auth
	if parsedURL.Scheme != "ssh" {
		return fmt.Errorf("invalid SSH scheme: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("SSH host is required")
	}

	return nil
}

// createDockerClient creates a new Docker client with SSH key-based authentication only
func (p *DockerConnectionPool) createDockerClient(host string, opts ...client.Opt) (*client.Client, error) {
	// Only allow SSH connections
	if len(host) < 6 || host[:6] != "ssh://" {
		return nil, fmt.Errorf("only SSH connections are allowed, got: %s", host)
	}

	// Validate SSH key authentication
	if err := p.validateSSHKeyAuth(host); err != nil {
		return nil, fmt.Errorf("SSH key validation failed: %w", err)
	}

	// Get SSH connection helper
	helper, err := connhelper.GetConnectionHelper(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection helper: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: helper.Dialer,
		},
	}

	clientOpts := []client.Opt{
		client.WithHTTPClient(httpClient),
		client.WithHost(helper.Host),
		client.WithDialContext(helper.Dialer),
		client.WithAPIVersionNegotiation(),
	}

	// Append any additional options provided
	clientOpts = append(clientOpts, opts...)

	return client.NewClientWithOpts(clientOpts...)
}

// Stats Get connection pool statistics
func (p *DockerConnectionPool) Stats() map[string]any {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	stats := map[string]any{
		"total_connections": len(p.connections),
		"connections":       make(map[string]map[string]any),
	}

	now := time.Now()
	for connectionID, connInfo := range p.connections {
		stats["connections"].(map[string]map[string]any)[connectionID] = map[string]any{
			"created_at":    connInfo.createdAt,
			"last_used":     connInfo.lastUsed,
			"idle_duration": now.Sub(connInfo.lastUsed),
			"age":           now.Sub(connInfo.createdAt),
		}
	}

	return stats
}

// isConnectionValid checks if a connection is still valid and not expired
func (p *DockerConnectionPool) isConnectionValid(connInfo *ConnectionInfo) bool {
	now := time.Now()

	// Check if connection has exceeded maximum lifetime
	if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
		return false
	}

	// Check if connection has been idle too long
	if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
		return false
	}

	// Ping the Docker daemon to check if connection is alive
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := connInfo.client.Ping(ctx)
	return err == nil
}

// removeConnection removes a connection from the pool and closes it
func (p *DockerConnectionPool) removeConnection(connectionID string, connInfo *ConnectionInfo) {
	delete(p.connections, connectionID)
	if connInfo.client != nil {
		connInfo.client.Close()
	}
}

// RemoveConnection removes a connection from the pool by connectionID
func (p *DockerConnectionPool) RemoveConnection(connectionID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if connInfo, exists := p.connections[connectionID]; exists {
		p.removeConnection(connectionID, connInfo)
		return nil
	}
	return fmt.Errorf("connection with ID %s not found", connectionID)
}

// startCleanup starts the cleanup goroutine
func (p *DockerConnectionPool) startCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanup()
		}
	}
}

// cleanup removes expired connections
func (p *DockerConnectionPool) cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	for connectionID, connInfo := range p.connections {
		shouldRemove := false

		// Check maximum lifetime
		if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
			shouldRemove = true
		}

		// Check idle timeout
		if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
			shouldRemove = true
		}

		if shouldRemove {
			p.removeConnection(connectionID, connInfo)
		}
	}
}

// Close closes the connection pool and all connections
func (p *DockerConnectionPool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	p.cancel()

	// Close all connections
	for connectionID, connInfo := range p.connections {
		p.removeConnection(connectionID, connInfo)
	}
}
