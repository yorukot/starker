package dockerutils

import (
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

type DockerHandler struct {
	Client          *ssh.Client
	Project         *types.Project
	NamingGenerator *generator.NamingGenerator
	DB              *pgxpool.Pool
	ConnectionPool  *connection.ConnectionPool
	StreamChan      core.StreamChan
}

// LogOptions represents options for Docker container logs
type LogOptions struct {
	Follow     bool      `json:"follow"`     // Follow log output (stream continuously)
	Tail       string    `json:"tail"`       // Number of lines to show from end of logs
	Timestamps bool      `json:"timestamps"` // Include timestamps in log output
	Since      time.Time `json:"since"`      // Show logs since timestamp
}
