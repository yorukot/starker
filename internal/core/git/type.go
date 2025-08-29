package git

import (
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/internal/core"
	"github.com/yorukot/starker/internal/models"
	"github.com/yorukot/starker/pkg/connection"
	"github.com/yorukot/starker/pkg/generator"
)

type GitHandler struct {
	Client          *client.Client
	GitModel        *models.ServiceSourceGit
	NamingGenerator *generator.NamingGenerator
	DB              *pgxpool.Pool
	ConnectionPool  *connection.ConnectionPool
	StreamChan      core.StreamChan
}
