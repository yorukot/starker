package router

import (
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/service"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/pkg/sshpool"
)

func ServiceRouter(r chi.Router, app *handler.App) {
	sshPool := sshpool.NewSSHConnectionPool(10*time.Minute, 1*time.Hour)

	serviceHandler := service.ServiceHandler{
		DB:      app.DB,
		SSHPool: sshPool,
	}

	r.Route("/teams/{teamID}/projects/{projectID}/services", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Post("/", serviceHandler.CreateService)
		r.Patch("/{serviceID}/", serviceHandler.UpdateService)
		r.Put("/{serviceID/state", serviceHandler.UpdateServiceState)
		// r.Get("/", serviceHandler.GetServices)
	})
}
