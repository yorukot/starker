package cmd

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/yorukot/stargo/internal/database"
	"github.com/yorukot/stargo/internal/handler"
	"github.com/yorukot/stargo/internal/middleware"
	"github.com/yorukot/stargo/internal/router"
	"github.com/yorukot/stargo/internal/logger"

	_ "github.com/joho/godotenv/autoload"
)

// @title Stargo API
// @version 1.0
// @description OAuth API for Stargo application
// @host localhost:8080
// @BasePath /api

// Run starts the server
func Run() {
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

	zap.L().Info("Starting server on port " + os.Getenv("PORT"))

	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}

// setupRouter sets up the router
func setupRouter(r chi.Router, app *handler.App) {
	r.Route("/api", func(r chi.Router) {
		router.AuthRouter(r, app)
	})

	zap.L().Info("Router setup complete")
}
