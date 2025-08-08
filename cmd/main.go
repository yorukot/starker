package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/config"
	"github.com/yorukot/stargo/internal/database"
	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/middleware"
	"github.com/yorukot/stargo/internal/router"
	"github.com/yorukot/stargo/pkg/logger"
	"github.com/yorukot/stargo/pkg/response"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/yorukot/stargo/docs"
)

// @title Stargo Go API Template
// @version 1.0
// @description Stargo Go API Template
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /api
// @schemes http https

// Run starts the server
func main() {
	logger.InitLogger()

	_, err := config.InitConfig()
	if err != nil {
		zap.L().Fatal("Error initializing config", zap.Error(err))
		return
	}

	r := chi.NewRouter()

	db, err := database.InitDatabase()
	if err != nil {
		zap.L().Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	r.Use(middleware.ZapLoggerMiddleware(zap.L()))
	r.Use(chiMiddleware.StripSlashes)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello stargo!"))
	})

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
