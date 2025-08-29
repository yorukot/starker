package dockerutils

import (
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

type DockerHandler struct {
	Client          *client.Client
	Project         *types.Project
	NamingGenerator *generator.NamingGenerator
	DB              *pgxpool.Pool
	ConnectionPool  *connection.ConnectionPool
	StreamChan      core.StreamChan
}
