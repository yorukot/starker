package server

import (
	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/pkg/connection"
)

type ServerHandler struct {
	handler.App
	DockerPool *connection.ConnectionPool
}
