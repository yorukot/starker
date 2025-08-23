package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/user"
	"github.com/yorukot/starker/internal/middleware"
)

// TeamRouter sets up the team routes
func UserRouter(r chi.Router, app *handler.App) {

	userHandler := user.UserHandler{
		DB: app.DB,
	}

	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Get("/me", userHandler.GetMe)
	})
}
