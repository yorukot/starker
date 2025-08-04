package router

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/handler/auth"
)

// AuthRouter sets up the authentication routes
func AuthRouter(r chi.Router, app *handler.App) {

	oauthConfig, err := config.OauthConfig()
	if err != nil {
		zap.L().Panic("failed to create oauth config", zap.Error(err))
		return
	}

	oauthHandler := &auth.OAuthHandler{
		DB:          app.DB,
		OAuthConfig: oauthConfig,
	}

	r.Route("/auth", func(r chi.Router) {

		r.Route("/oauth", func(r chi.Router) {
			r.Get("/{provider}", oauthHandler.OauthEntry)
			r.Get("/{provider}/callback", oauthHandler.OauthCallback)
		})

		// TODO: add refresh token route
		// r.Post("/refresh", oauthHandler.RefreshToken)
	})
}
