package router

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/server"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/pkg/connection"
)

// ServerRouter sets up the server routes
func ServerRouter(r chi.Router, app *handler.App) {

	dockerPool := connection.NewConnectionPool(20*time.Minute, 1*time.Hour)

	serverHandler := server.ServerHandler{
		DB:         app.DB,
		DockerPool: dockerPool,
	}

	r.Route("/teams/{teamID}/servers", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Post("/", serverHandler.CreateServer)
		r.Get("/", serverHandler.GetServers)
		r.Get("/{serverID}", serverHandler.GetServer)
		r.Patch("/{serverID}", serverHandler.UpdateServer)
		r.Delete("/{serverID}", serverHandler.DeleteServer)
	})
}
