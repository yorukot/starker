package service

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/connection"
)

type ServiceHandler struct {
	DB         *pgxpool.Pool
	DockerPool *connection.ConnectionPool
	Tx         *pgx.Tx
}
