package service

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/connection"
)

type ServiceHandler struct {
	DB             *pgxpool.Pool
	ConnectionPool *connection.ConnectionPool
	DockerPool     *connection.ConnectionPool
}
