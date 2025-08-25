package router

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/service"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/pkg/connection"
)

func ServiceRouter(r chi.Router, app *handler.App) {
	dockerPool := connection.NewConnectionPool(20*time.Minute, 1*time.Hour)

	serviceHandler := service.ServiceHandler{
		DB:         app.DB,
		DockerPool: dockerPool,
	}

	r.Route("/teams/{teamID}/projects/{projectID}/services", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Get("/", serviceHandler.GetServices)
		r.Get("/{serviceID}", serviceHandler.GetService)

		r.Post("/compose", serviceHandler.CreateServiceCompose)
		r.Post("/git", serviceHandler.CreateServiceCompose)

		r.Patch("/{serviceID}/", serviceHandler.UpdateService)
		r.Patch("/{serviceID}/state", serviceHandler.UpdateServiceState)

		r.Get("/{serviceID}/compose", serviceHandler.GetServiceCompose)
		r.Patch("/{serviceID}/compose", serviceHandler.UpdateServiceCompose)
	})
}
