package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/team"
	"github.com/yorukot/starker/internal/middleware"
)

// TeamRouter sets up the team routes
func TeamRouter(r chi.Router, app *handler.App) {

	teamHandler := team.TeamHandler{
		DB: app.DB,
	}

	r.Route("/teams", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Post("/", teamHandler.CreateTeam)
		r.Get("/", teamHandler.GetTeams)
		r.Get("/{teamID}", teamHandler.GetTeam)
		r.Delete("/{teamID}", teamHandler.DeleteTeam)

		// TODO: Handle other team user management
	})
}
