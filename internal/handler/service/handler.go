package service

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/dockerpool"
)

type ServiceHandler struct {
	DB         *pgxpool.Pool
	DockerPool *dockerpool.DockerConnectionPool
	Tx         *pgx.Tx
}
