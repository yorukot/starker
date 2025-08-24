package dockerpool

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

// ConnectionInfo is the information of a connection
type ConnectionInfo struct {
	client    *client.Client
	sshConn   *ssh.Client
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

// GetConnection gets an SSH connection from the pool using connectionID and privateKeyContent
// If the connection is not found, it will create a new SSH key-based connection using the provided private key
func (p *DockerConnectionPool) GetConnection(connectionID, host string, privateKeyContent []byte, opts ...client.Opt) (*client.Client, error) {
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

	// Create new connection with private key
	dockerClient, sshConn, err := p.createDockerClient(host, privateKeyContent, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker connection: %w", err)
	}

	now := time.Now()
	p.connections[connectionID] = &ConnectionInfo{
		client:    dockerClient,
		sshConn:   sshConn,
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
	if connInfo.sshConn != nil {
		connInfo.sshConn.Close()
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

// TestConnection creates a new Docker connection using the provided private key, tests it, and immediately closes it
// This function bypasses the connection pool and accepts private key content directly
func (p *DockerConnectionPool) TestConnection(host string, privateKeyContent []byte, opts ...client.Opt) error {
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
func (p *DockerConnectionPool) createDockerClient(host string, privateKeyContent []byte, opts ...client.Opt) (*client.Client, *ssh.Client, error) {
	// Only allow SSH connections
	if len(host) < 6 || host[:6] != "ssh://" {
		return nil, nil, fmt.Errorf("only SSH connections are allowed, got: %s", host)
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

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: parsedURL.User.Username(),
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
