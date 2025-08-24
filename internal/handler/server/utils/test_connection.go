package utils

import (
	"context"
	"fmt"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/dockerpool"
)

// TestServerConnection tests a Docker connection using the provided server and private key
func TestServerConnection(ctx context.Context, server models.Server, privateKey models.PrivateKey, dockerPool *dockerpool.DockerConnectionPool) error {
	// Build SSH connection string for Docker
	host := fmt.Sprintf("ssh://%s@%s:%s", server.User, server.IP, server.Port)

	// Test the connection using the docker pool's test function with private key content
	if err := dockerPool.TestConnection(host, []byte(privateKey.PrivateKey)); err != nil {
		return fmt.Errorf("failed to test Docker connection: %w", err)
	}

	return nil
}
