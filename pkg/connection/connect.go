package connection

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectionInfo holds connection details and metadata
type ConnectionInfo struct {
	id          string
	client      *ssh.Client
	host        string
	createdAt   time.Time
	lastUsed    time.Time
	maxLifetime time.Duration
	idleTimeout time.Duration
}

// ConnectionPool manages SSH and Docker connections with connection pooling
type ConnectionPool struct {
	connections map[string]*ConnectionInfo
	mutex       sync.RWMutex

	// Configuration
	defaultTimeout     time.Duration
	defaultIdleTimeout time.Duration
	defaultMaxLifetime time.Duration
	maxConnections     int

	// Cleanup management
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupOnce   sync.Once
}

// NewConnectionPool creates a new ConnectionPool with default settings
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections:        make(map[string]*ConnectionInfo),
		defaultTimeout:     30 * time.Second,
		defaultIdleTimeout: 5 * time.Minute,
		defaultMaxLifetime: 1 * time.Hour,
		maxConnections:     100,
		stopCleanup:        make(chan struct{}),
	}

	// Start cleanup goroutine
	pool.startCleanupRoutine()

	return pool
}

// createSSHConfig creates SSH client configuration with private key authentication
func createSSHConfig(user string, privateKeyContent []byte) (*ssh.ClientConfig, error) {
	// Parse the private key
	signer, err := ssh.ParsePrivateKey(privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	return config, nil
}

// GetSSHConnection gets or creates an SSH connection with connection pooling
func (p *ConnectionPool) GetSSHConnection(connectionID, host string, port int, user string, privateKeyContent []byte) (*ssh.Client, error) {
	p.mutex.RLock()

	// Check if we already have a valid connection
	if connInfo, exists := p.connections[connectionID]; exists {
		connInfo.lastUsed = time.Now()
		p.mutex.RUnlock()
		return connInfo.client, nil
	}
	p.mutex.RUnlock()

	// Need to create a new connection
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check after acquiring write lock
	if connInfo, exists := p.connections[connectionID]; exists {
		connInfo.lastUsed = time.Now()
		return connInfo.client, nil
	}

	// Create SSH config
	config, err := createSSHConfig(user, privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Create SSH connection
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	// Store the connection
	connInfo := &ConnectionInfo{
		id:          connectionID,
		client:      sshClient,
		host:        host,
		createdAt:   time.Now(),
		lastUsed:    time.Now(),
		maxLifetime: p.defaultMaxLifetime,
		idleTimeout: p.defaultIdleTimeout,
	}

	p.connections[connectionID] = connInfo

	return sshClient, nil
}

// isConnectionValid checks if a connection is still valid and not expired
func (p *ConnectionPool) isConnectionValid(connInfo *ConnectionInfo) bool {
	now := time.Now()

	// Check if connection has exceeded maximum lifetime
	if now.Sub(connInfo.createdAt) > connInfo.maxLifetime {
		return false
	}

	// Check if connection has been idle too long
	if now.Sub(connInfo.lastUsed) > connInfo.idleTimeout {
		return false
	}

	// Test if SSH connection is still alive by creating a simple session
	if connInfo.client != nil {
		session, err := connInfo.client.NewSession()
		if err != nil {
			return false
		}
		session.Close()
	}

	return true
}

// startCleanupRoutine starts a goroutine that periodically cleans up stale connections
func (p *ConnectionPool) startCleanupRoutine() {
	p.cleanupOnce.Do(func() {
		// Start cleanup routine every 2 minutes
		p.cleanupTicker = time.NewTicker(2 * time.Minute)

		go func() {
			defer p.cleanupTicker.Stop()

			for {
				select {
				case <-p.cleanupTicker.C:
					p.cleanupStaleConnections()
				case <-p.stopCleanup:
					return
				}
			}
		}()
	})
}

// cleanupStaleConnections removes invalid or expired connections from the pool
func (p *ConnectionPool) cleanupStaleConnections() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for connectionID, connInfo := range p.connections {
		if !p.isConnectionValid(connInfo) {
			// Close the SSH connection
			if connInfo.client != nil {
				connInfo.client.Close()
			}

			// Remove from pool
			delete(p.connections, connectionID)
		}
	}
}

// Shutdown gracefully shuts down the connection pool
func (p *ConnectionPool) Shutdown() {
	// Signal cleanup routine to stop
	close(p.stopCleanup)

	// Stop the cleanup ticker if it exists
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	// Close all connections
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for connectionID, connInfo := range p.connections {
		if connInfo.client != nil {
			connInfo.client.Close()
		}
		delete(p.connections, connectionID)
	}
}

// TestConnection tests an SSH connection without using the connection pool
func TestConnection(host string, port int, user string, privateKeyContent []byte) error {
	if port == 0 {
		port = 22
	}

	// Create SSH config using existing function
	config, err := createSSHConfig(user, privateKeyContent)
	if err != nil {
		return fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Create SSH connection
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer client.Close()

	// Test the connection by creating a session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	return nil
}
