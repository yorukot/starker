package service

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/starker/pkg/sshpool"
)

type ServiceHandler struct {
	DB      *pgxpool.Pool
	SSHPool *sshpool.SSHConnectionPool
	Tx      *pgx.Tx
}
