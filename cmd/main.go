package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/database"
	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/middleware"
	"github.com/yorukot/stargo/internal/router"
	"github.com/yorukot/stargo/pkg/logger"

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
	r := chi.NewRouter()

	logger.InitLogger()

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

	zap.L().Info("Starting server on http://localhost:" + os.Getenv("PORT"))
	zap.L().Info("Environment: " + os.Getenv("APP_ENV"))

	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}

// setupRouter sets up the router
func setupRouter(r chi.Router, app *handler.App) {
	r.Route("/api", func(r chi.Router) {
		router.AuthRouter(r, app)
	})

	if os.Getenv("APP_ENV") == "dev" {
		r.Get("/swagger/*", httpSwagger.WrapHandler)
	}

	zap.L().Info("Router setup complete")
}
