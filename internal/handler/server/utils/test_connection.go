package utils

import (
	"context"
	"fmt"

	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/connection"
)

// TestServerConnection tests a Docker connection using the provided server and private key
func TestServerConnection(ctx context.Context, server models.Server, privateKey models.PrivateKey, dockerPool *connection.ConnectionPool) error {

	if err := connection.TestConnection(server.Host, server.Port, server.User, []byte(privateKey.PrivateKey)); err != nil {
		return fmt.Errorf("failed to test Docker connection: %w", err)
	}

	return nil
}
