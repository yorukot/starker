package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/stargo/internal/handler"
)

// AuthRouter sets up the authentication routes
func AuthRouter(r chi.Router, app *handler.App) {

	r.Route("/auth", func(r chi.Router) {

		r.Route("/oauth", func(r chi.Router) {
		})

		r.Post("/register", app.Register)
		r.Post("/login", app.Login)
		// TODO: add refresh token route
		// r.Post("/refresh", oauthHandler.RefreshToken)
	})
}
