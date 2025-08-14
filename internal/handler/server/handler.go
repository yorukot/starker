package server

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/sshpool"
)

type ServerHandler struct {
	DB      *pgxpool.Pool
	SSHPool *sshpool.SSHConnectionPool
}
