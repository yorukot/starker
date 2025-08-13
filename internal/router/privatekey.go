package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/handler/privatekey"
	"github.com/yorukot/starker/internal/middleware"
)

// PrivateKeyRouter sets up the private key routes
func PrivateKeyRouter(r chi.Router, app *handler.App) {

	privateKeyHandler := privatekey.PrivateKeyHandler{
		DB: app.DB,
	}

	r.Route("/teams/{teamID}/private-keys", func(r chi.Router) {
		r.Use(middleware.AuthRequiredMiddleware)

		r.Post("/", privateKeyHandler.CreatePrivateKey)
		r.Get("/", privateKeyHandler.GetPrivateKeys)
		r.Get("/{privateKeyID}", privateKeyHandler.GetPrivateKey)
		r.Delete("/{privateKeyID}", privateKeyHandler.DeletePrivateKey)

		// TODO: Handle other team user management
	})
}
