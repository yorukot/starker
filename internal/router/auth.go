package router

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/handler/auth"
	"github.com/yorukot/stargo/internal/middleware"
)

// AuthRouter sets up the authentication routes
func AuthRouter(r chi.Router, app *handler.App) {

	authHandler := auth.AuthHandler{
		DB: app.DB,
	}

	oauthConfig, err := config.GetOAuthConfig()
	if err != nil {
		zap.L().Panic("failed to get oauth config", zap.Error(err))
		return
	}

	oauthHandler := auth.OAuthHandler{
		DB:          app.DB,
		OAuthConfig: oauthConfig,
	}

	r.Route("/auth", func(r chi.Router) {

		r.Route("/oauth", func(r chi.Router) {
			// We use AuthOptionalMiddleware because we want to allow users to access the OAuth session without being authenticated (first time login/register)
			r.With(middleware.AuthOptionalMiddleware).Get("/{provider}", oauthHandler.OAuthEntry)
			r.Get("/{provider}/callback", oauthHandler.OAuthCallback)
		})

		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.RefreshToken)
	})
}
