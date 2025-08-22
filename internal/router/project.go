package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/project"
	"github.com/yorukot/starker/internal/middleware"
)

// PrivateKeyRouter sets up the private key routes
func ProjectRouter(r chi.Router, app *handler.App) {

	projectHandler := project.ProjectHandler{
		DB: app.DB,
	}

	r.Route("/teams/{teamID}/projects", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Post("/", projectHandler.CreateProject)
		r.Get("/", projectHandler.GetProjects)
		r.Get("/{projectID}", projectHandler.GetProject)
		r.Patch("/{projectID}", projectHandler.UpdateProject)
		r.Delete("/{projectID}", projectHandler.DeleteProject)
	})
}
