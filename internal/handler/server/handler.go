package server

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/connection"
)

type ServerHandler struct {
	DB         *pgxpool.Pool
	DockerPool *connection.ConnectionPool
}
