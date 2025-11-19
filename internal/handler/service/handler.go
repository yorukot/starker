package service

import (

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/pkg/connection"
)

type ServiceHandler struct {
	handler.App
	ConnectionPool *connection.ConnectionPool
	DockerPool     *connection.ConnectionPool
}
