package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/router"
	"github.com/yorukot/starker/pkg/response"
	"go.uber.org/zap"
)

// startAPI starts the API server
func startAPI(r chi.Router, db *pgxpool.Pool) {
	var err error
	r.Use(middleware.ZapLoggerMiddleware(zap.L()))
	r.Use(chiMiddleware.StripSlashes)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://" + config.Env().FrontendDomain, "https://" + config.Env().FrontendDomain},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Cache-Control", "DNT", "User-Agent", "Referer", "Sec-CH-UA", "Sec-CH-UA-Mobile", "Sec-CH-UA-Platform", "Sec-Fetch-Dest", "Sec-Fetch-Mode", "Sec-Fetch-Site"},
		ExposedHeaders:   []string{"Link", "Cache-Control", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	setupRouter(r, &handler.App{DB: db})

	zap.L().Info("Starting server on http://localhost:" + config.Env().Port)
	zap.L().Info("Environment: " + string(config.Env().AppEnv))

	err = http.ListenAndServe(":"+config.Env().Port, r)
	if err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}

// setupRouter sets up the router
func setupRouter(r chi.Router, app *handler.App) {
	r.Route("/api", func(r chi.Router) {
		router.AuthRouter(r, app)
		router.UserRouter(r, app)
		router.TeamRouter(r, app)
		router.PrivateKeyRouter(r, app)
		router.ServerRouter(r, app)
		router.ProjectRouter(r, app)
		router.ServiceRouter(r, app)
	})

	if config.Env().AppEnv == config.AppEnvDev {
		r.Get("/swagger/*", httpSwagger.WrapHandler)
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Not found handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusNotFound, "Not Found", "NOT_FOUND")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "METHOD_NOT_ALLOWED")
	})

	zap.L().Info("Router setup complete")
}
