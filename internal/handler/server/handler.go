package server

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/dockerpool"
)

type ServerHandler struct {
	DB         *pgxpool.Pool
	DockerPool *dockerpool.DockerConnectionPool
}
