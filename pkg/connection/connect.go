package connection

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"
)

// sshPipeConn implements net.Conn using SSH session stdin/stdout pipes
type sshPipeConn struct {
	stdin   io.WriteCloser
	stdout  io.Reader
	session *ssh.Session
}

func (c *sshPipeConn) Read(b []byte) (n int, err error) {
	return c.stdout.Read(b)
}

func (c *sshPipeConn) Write(b []byte) (n int, err error) {
	return c.stdin.Write(b)
}

func (c *sshPipeConn) Close() error {
	c.stdin.Close()
	// stdout from SSH session doesn't need explicit closing
	return c.session.Close()
}

func (c *sshPipeConn) LocalAddr() net.Addr {
	return &net.UnixAddr{Name: "ssh-pipe", Net: "unix"}
}

func (c *sshPipeConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{Name: "/var/run/docker.sock", Net: "unix"}
}

func (c *sshPipeConn) SetDeadline(t time.Time) error {
	// SSH sessions don't support deadlines
	return nil
}

func (c *sshPipeConn) SetReadDeadline(t time.Time) error {
	// SSH sessions don't support deadlines
	return nil
}

func (c *sshPipeConn) SetWriteDeadline(t time.Time) error {
	// SSH sessions don't support deadlines
	return nil
}

// ConnectionType represents the type of connection
type ConnectionType int

const (
	DockerConnection ConnectionType = iota
	SSHConnection
)

// ConnectionInfo holds information about a connection
type ConnectionInfo struct {
	dockerClient *client.Client // Docker API client (via SSH)
	sshClient    *ssh.Client    // Raw SSH client
	connType     ConnectionType // Type of primary connection
	lastUsed     time.Time
	createdAt    time.Time
}

// ConnectionPool manages SSH-based connections for both Docker API access and direct SSH commands
type ConnectionPool struct {
	connections map[string]*ConnectionInfo
	mutex       sync.RWMutex
	maxIdle     time.Duration
	maxLifetime time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
}

// NewConnectionPool creates a new connection pool that supports SSH key-based authentication
func NewConnectionPool(maxIdle, maxLifetime time.Duration) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
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

// NewDockerConnectionPool is deprecated, use NewConnectionPool instead
// Kept for backward compatibility
func NewDockerConnectionPool(maxIdle, maxLifetime time.Duration) *ConnectionPool {
	return NewConnectionPool(maxIdle, maxLifetime)
}

// GetDockerConnection gets a Docker client from the pool using connectionID and privateKeyContent
// If the connection is not found, it will create a new SSH key-based connection using the provided private key
func (p *ConnectionPool) GetDockerConnection(connectionID, host string, privateKeyContent []byte, opts ...client.Opt) (*client.Client, error) {
	p.mutex.RLock()
	if connInfo, exists := p.connections[connectionID]; exists {
		// Check if connection needs reconnection due to lifetime expiration
		now := time.Now()
		if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
			// Connection is lifetime-expired, needs reconnection
			p.mutex.RUnlock()
			p.mutex.Lock()
			defer p.mutex.Unlock()

			// Double-check after acquiring write lock
			if connInfo, stillExists := p.connections[connectionID]; stillExists {
				if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
					// Reconnect the connection
					err := p.reconnectConnection(connectionID, connInfo, host, privateKeyContent, opts...)
					if err != nil {
						return nil, fmt.Errorf("failed to reconnect connection: %w", err)
					}
					// Update last used time and return the reconnected client
					connInfo = p.connections[connectionID]
					connInfo.lastUsed = time.Now()
					return connInfo.dockerClient, nil
				}
			}
		}

		// Check if connection is still valid (not lifetime-expired)
		if p.isConnectionValid(connInfo) {
			connInfo.lastUsed = time.Now()
			p.mutex.RUnlock()
			return connInfo.dockerClient, nil
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
			return connInfo.dockerClient, nil
		} else {
			// Clean up invalid connection
			p.removeConnection(connectionID, connInfo)
		}
	}

	// Create new connection with private key
	dockerClient, sshConn, err := p.createDockerClient(host, privateKeyContent, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker connection: %w", err)
	}

	now := time.Now()
	p.connections[connectionID] = &ConnectionInfo{
		dockerClient: dockerClient,
		sshClient:    sshConn,
		connType:     DockerConnection,
		lastUsed:     now,
		createdAt:    now,
	}

	return dockerClient, nil
}

// GetConnection is deprecated, use GetDockerConnection instead
// GetConnection gets an SSH connection from the pool using connectionID and privateKeyContent
// If the connection is not found, it will create a new SSH key-based connection using the provided private key
func (p *ConnectionPool) GetConnection(connectionID, host string, privateKeyContent []byte, opts ...client.Opt) (*client.Client, error) {
	return p.GetDockerConnection(connectionID, host, privateKeyContent, opts...)
}

// GetSSHConnection gets a raw SSH client from the pool using connectionID and privateKeyContent
// If the connection is not found, it will create a new SSH connection using the provided private key
func (p *ConnectionPool) GetSSHConnection(connectionID, host string, privateKeyContent []byte) (*ssh.Client, error) {
	p.mutex.RLock()
	if connInfo, exists := p.connections[connectionID]; exists {
		// Check if connection needs reconnection due to lifetime expiration
		now := time.Now()
		if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
			// Connection is lifetime-expired, needs reconnection
			p.mutex.RUnlock()
			p.mutex.Lock()
			defer p.mutex.Unlock()

			// Double-check after acquiring write lock
			if connInfo, stillExists := p.connections[connectionID]; stillExists {
				if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
					// Reconnect the SSH connection
					err := p.reconnectSSHConnection(connectionID, connInfo, host, privateKeyContent)
					if err != nil {
						return nil, fmt.Errorf("failed to reconnect SSH connection: %w", err)
					}
					// Update last used time and return the reconnected SSH client
					connInfo = p.connections[connectionID]
					connInfo.lastUsed = time.Now()
					return connInfo.sshClient, nil
				}
			}
		}

		// Check if connection is still valid (not lifetime-expired)
		if p.isConnectionValid(connInfo) {
			connInfo.lastUsed = time.Now()
			p.mutex.RUnlock()
			return connInfo.sshClient, nil
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
			return connInfo.sshClient, nil
		} else {
			// Clean up invalid connection
			p.removeConnection(connectionID, connInfo)
		}
	}

	// Create new SSH connection
	sshConn, err := p.createRawSSHConnection(host, privateKeyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	now := time.Now()
	p.connections[connectionID] = &ConnectionInfo{
		dockerClient: nil, // No Docker client for raw SSH connections
		sshClient:    sshConn,
		connType:     SSHConnection,
		lastUsed:     now,
		createdAt:    now,
	}

	return sshConn, nil
}

// validateSSHKeyAuth validates that the SSH connection uses key-based authentication
func (p *ConnectionPool) validateSSHKeyAuth(host string) error {
	// Check for password authentication indicators (which we want to reject)
	if strings.Contains(host, "password") || strings.Contains(host, "passwd") {
		return fmt.Errorf("password-based SSH authentication is not allowed, only SSH key authentication is supported")
	}

	// Parse the SSH URL to check for key-based authentication requirements
	parsedURL, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("invalid SSH URL format: %w", err)
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

// isConnectionValid checks if a connection is still valid and not expired
func (p *ConnectionPool) isConnectionValid(connInfo *ConnectionInfo) bool {
	now := time.Now()

	// Check if connection has been idle too long - this should still invalidate
	if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
		return false
	}

	// Don't invalidate based on lifetime here - let cleanup handle reconnection
	// if connection is still being used

	// Check connection based on type
	if connInfo.connType == DockerConnection && connInfo.dockerClient != nil {
		// Ping the Docker daemon to check if connection is alive
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := connInfo.dockerClient.Ping(ctx)
		return err == nil
	} else if connInfo.connType == SSHConnection && connInfo.sshClient != nil {
		// For SSH connections, we assume they're valid if not nil
		// More sophisticated checking could be added here
		return true
	}

	return false
}

// removeConnection removes a connection from the pool and closes it
func (p *ConnectionPool) removeConnection(connectionID string, connInfo *ConnectionInfo) {
	delete(p.connections, connectionID)
	if connInfo.dockerClient != nil {
		connInfo.dockerClient.Close()
	}
	if connInfo.sshClient != nil {
		connInfo.sshClient.Close()
	}
}

// RemoveConnection removes a connection from the pool by connectionID
func (p *ConnectionPool) RemoveConnection(connectionID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if connInfo, exists := p.connections[connectionID]; exists {
		p.removeConnection(connectionID, connInfo)
		return nil
	}
	return fmt.Errorf("connection with ID %s not found", connectionID)
}

// startCleanup starts the cleanup goroutine
func (p *ConnectionPool) startCleanup() {
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

// cleanup handles expired connections - reconnects active ones, removes idle ones
func (p *ConnectionPool) cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	for connectionID, connInfo := range p.connections {
		// Check idle timeout - remove idle connections
		if p.maxIdle > 0 && now.Sub(connInfo.lastUsed) > p.maxIdle {
			p.removeConnection(connectionID, connInfo)
			continue
		}

		// Check maximum lifetime for reconnection
		if p.maxLifetime > 0 && now.Sub(connInfo.createdAt) > p.maxLifetime {
			// If connection was used recently (within idle timeout), reconnect it
			if p.maxIdle == 0 || now.Sub(connInfo.lastUsed) <= p.maxIdle {
				// Attempt to reconnect - we need to store the connection details
				// Since we don't have them here, just reset the creation time
				// The actual reconnection will happen on next use if ping fails
				connInfo.createdAt = now
			} else {
				// Connection is old and not recently used, remove it
				p.removeConnection(connectionID, connInfo)
			}
		}
	}
}

// TestConnection creates a new Docker connection using the provided private key, tests it, and immediately closes it
// This function bypasses the connection pool and accepts private key content directly
func (p *ConnectionPool) TestConnection(host string, privateKeyContent []byte, opts ...client.Opt) error {
	// Create a new Docker client for testing with the provided key
	dockerClient, sshConn, err := p.createDockerClient(host, privateKeyContent, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Docker connection: %w", err)
	}
	defer dockerClient.Close()
	if sshConn != nil {
		defer sshConn.Close()
	}

	// Test the connection with a ping
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker connection test failed: %w", err)
	}

	return nil
}

// createDockerClient creates a new Docker client with SSH key-based authentication using provided key content
func (p *ConnectionPool) createDockerClient(host string, privateKeyContent []byte, opts ...client.Opt) (*client.Client, *ssh.Client, error) {
	// Normalize host format - if it doesn't have ssh:// scheme, add it
	if !strings.HasPrefix(host, "ssh://") {
		// Assume user@host:port or host:port format and prepend ssh://
		host = "ssh://" + host
	}

	// Validate SSH key authentication
	if err := p.validateSSHKeyAuth(host); err != nil {
		return nil, nil, fmt.Errorf("SSH key validation failed: %w", err)
	}

	// Parse the SSH URL
	parsedURL, err := url.Parse(host)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid SSH URL format: %w", err)
	}

	// Parse the private key
	block, _ := pem.Decode(privateKeyContent)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to parse private key PEM block")
	}

	var privateKey any
	var parseErr error

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		privateKey, parseErr = x509.ParseECPrivateKey(block.Bytes)
	case "OPENSSH PRIVATE KEY":
		privateKey, parseErr = ssh.ParseRawPrivateKey(privateKeyContent)
	default:
		return nil, nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	if parseErr != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", parseErr)
	}

	// Create SSH signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SSH signer: %w", err)
	}

	// Extract username from URL, default to "root" if not provided
	username := "root"
	if parsedURL.User != nil {
		if user := parsedURL.User.Username(); user != "" {
			username = user
		}
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key validation
		Timeout:         10 * time.Second,
	}

	// Establish persistent SSH connection
	sshConn, err := ssh.Dial("tcp", parsedURL.Host, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect via SSH: %w", err)
	}

	// Create custom dialer that uses the established SSH connection
	dialer := func(ctx context.Context, network, addr string) (net.Conn, error) {

		// Use SSH to execute a command that connects to the Docker socket
		// This is more compatible than direct unix socket forwarding
		session, err := sshConn.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create SSH session: %w", err)
		}

		// Use socat to bridge the connection to the Docker socket
		// This creates a bidirectional pipe to the Unix socket
		cmd := "socat STDIO UNIX-CONNECT:/var/run/docker.sock"

		stdin, err := session.StdinPipe()
		if err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		if err := session.Start(cmd); err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to start socat command: %w", err)
		}

		// Create a bidirectional connection using stdin/stdout pipes
		return &sshPipeConn{
			stdin:   stdin,
			stdout:  stdout,
			session: session,
		}, nil
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: dialer,
		},
	}

	clientOpts := []client.Opt{
		client.WithHTTPClient(httpClient),
		client.WithAPIVersionNegotiation(),
	}

	// Append any additional options provided
	clientOpts = append(clientOpts, opts...)

	dockerClient, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		sshConn.Close()
		return nil, nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return dockerClient, sshConn, nil
}

// reconnectConnection recreates a connection with the same parameters, preserving lastUsed time
func (p *ConnectionPool) reconnectConnection(connectionID string, oldConnInfo *ConnectionInfo, host string, privateKeyContent []byte, opts ...client.Opt) error {
	// Preserve the last used time
	lastUsed := oldConnInfo.lastUsed

	// Close the old connection
	if oldConnInfo.dockerClient != nil {
		oldConnInfo.dockerClient.Close()
	}
	if oldConnInfo.sshClient != nil {
		oldConnInfo.sshClient.Close()
	}

	// Create new connection
	dockerClient, sshConn, err := p.createDockerClient(host, privateKeyContent, opts...)
	if err != nil {
		return fmt.Errorf("failed to create new Docker connection: %w", err)
	}

	// Update connection info with new client but preserve lastUsed
	p.connections[connectionID] = &ConnectionInfo{
		dockerClient: dockerClient,
		sshClient:    sshConn,
		connType:     DockerConnection,
		lastUsed:     lastUsed,   // Preserve the last used time
		createdAt:    time.Now(), // Reset creation time
	}

	return nil
}

// createRawSSHConnection creates a raw SSH client connection using the provided private key
func (p *ConnectionPool) createRawSSHConnection(host string, privateKeyContent []byte) (*ssh.Client, error) {
	// Normalize host format - if it doesn't have ssh:// scheme, add it
	if !strings.HasPrefix(host, "ssh://") {
		// Assume user@host:port or host:port format and prepend ssh://
		host = "ssh://" + host
	}

	// Validate SSH key authentication
	if err := p.validateSSHKeyAuth(host); err != nil {
		return nil, fmt.Errorf("SSH key validation failed: %w", err)
	}

	// Parse the SSH URL
	parsedURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH URL format: %w", err)
	}

	// Parse the private key
	block, _ := pem.Decode(privateKeyContent)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key PEM block")
	}

	var privateKey any
	var parseErr error

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		privateKey, parseErr = x509.ParseECPrivateKey(block.Bytes)
	case "OPENSSH PRIVATE KEY":
		privateKey, parseErr = ssh.ParseRawPrivateKey(privateKeyContent)
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", parseErr)
	}

	// Create SSH signer
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH signer: %w", err)
	}

	// Extract username from URL, default to "root" if not provided
	username := "root"
	if parsedURL.User != nil {
		if user := parsedURL.User.Username(); user != "" {
			username = user
		}
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key validation
		Timeout:         10 * time.Second,
	}

	// Establish SSH connection
	sshConn, err := ssh.Dial("tcp", parsedURL.Host, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect via SSH: %w", err)
	}

	return sshConn, nil
}

// reconnectSSHConnection recreates an SSH connection with the same parameters, preserving lastUsed time
func (p *ConnectionPool) reconnectSSHConnection(connectionID string, oldConnInfo *ConnectionInfo, host string, privateKeyContent []byte) error {
	// Preserve the last used time
	lastUsed := oldConnInfo.lastUsed

	// Close the old connection
	if oldConnInfo.sshClient != nil {
		oldConnInfo.sshClient.Close()
	}

	// Create new SSH connection
	sshConn, err := p.createRawSSHConnection(host, privateKeyContent)
	if err != nil {
		return fmt.Errorf("failed to create new SSH connection: %w", err)
	}

	// Update connection info with new SSH client but preserve lastUsed
	p.connections[connectionID] = &ConnectionInfo{
		dockerClient: oldConnInfo.dockerClient, // Keep existing Docker client if any
		sshClient:    sshConn,
		connType:     SSHConnection, // This is primarily an SSH connection
		lastUsed:     lastUsed,      // Preserve the last used time
		createdAt:    time.Now(),    // Reset creation time
	}

	return nil
}

// Close closes the connection pool and all connections
func (p *ConnectionPool) Close() {
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
