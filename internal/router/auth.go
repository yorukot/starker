package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/handler/auth"
)

// AuthRouter sets up the authentication routes
func AuthRouter(r chi.Router, app *handler.App) {

	authHandler := auth.AuthHandler{
		DB: app.DB,
	}

	r.Route("/auth", func(r chi.Router) {

		r.Route("/oauth", func(r chi.Router) {
		})

		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.RefreshToken)
	})
}
