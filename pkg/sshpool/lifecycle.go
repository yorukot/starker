package sshpool

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// isConnectionValid checks if a connection is valid
func (p *SSHConnectionPool) isConnectionValid(connInfo *ConnectionInfo) bool {
	now := time.Now()

	// Check if exceeded maximum idle time
	if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
		return false
	}

	// Check if exceeded maximum lifetime
	if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
		return false
	}

	// Check if connection is still alive (simple health check)
	return p.isConnectionHealthy(connInfo.client)
}

// isConnectionHealthy checks if a connection is healthy
func (p *SSHConnectionPool) isConnectionHealthy(client *ssh.Client) bool {
	// Try to create a test session
	session, err := client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// markConnectionForRemoval marks a connection for removal (handled in subsequent cleanup)
func (p *SSHConnectionPool) markConnectionForRemoval(host string) {
	go func() {
		time.Sleep(1 * time.Second) // Short delay before cleanup
		p.mutex.Lock()
		defer p.mutex.Unlock()

		if connInfo, exists := p.connections[host]; exists {
			if !p.isConnectionValid(connInfo) {
				p.removeConnection(host, connInfo)
			}
		}
	}()
}

// removeConnection removes a connection (must be called within write lock)
func (p *SSHConnectionPool) removeConnection(host string, connInfo *ConnectionInfo) {
	if connInfo.client != nil {
		connInfo.client.Close()
	}
	delete(p.connections, host)
}

// startCleanup starts the periodic cleanup of expired connections
func (p *SSHConnectionPool) startCleanup() {
	ticker := time.NewTicker(30 * time.Second) // Clean up every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupExpiredConnections()
		case <-p.ctx.Done():
			return
		}
	}
}

// cleanupExpiredConnections cleans up expired connections
func (p *SSHConnectionPool) cleanupExpiredConnections() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for host, connInfo := range p.connections {
		shouldRemove := false

		// Check idle time
		if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
			shouldRemove = true
		}

		// Check lifetime
		if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
			shouldRemove = true
		}

		// Check connection health status
		if !shouldRemove && !p.isConnectionHealthy(connInfo.client) {
			shouldRemove = true
		}

		if shouldRemove {
			toRemove = append(toRemove, host)
		}
	}

	// Remove expired connections
	for _, host := range toRemove {
		if connInfo, exists := p.connections[host]; exists {
			p.removeConnection(host, connInfo)
		}
	}

	if len(toRemove) > 0 {
		fmt.Printf("Cleaned up %d expired connections\n", len(toRemove))
	}
}

// RemoveConnection manually cleans up a connection for a specific host
func (p *SSHConnectionPool) RemoveConnection(host string) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if connInfo, exists := p.connections[host]; exists {
		p.removeConnection(host, connInfo)
		return true
	}
	return false
}

// Close closes the connection pool
func (p *SSHConnectionPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.cancel() // Stop cleanup goroutine

	// Close all connections
	for host, connInfo := range p.connections {
		p.removeConnection(host, connInfo)
	}

	fmt.Println("SSH connection pool closed")
	return nil
}
