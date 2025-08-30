package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	_ "github.com/yorukot/starker/docs"
	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/database"
	"github.com/yorukot/starker/internal/handler"
	"github.com/yorukot/starker/internal/middleware"
	"github.com/yorukot/starker/internal/router"
	"github.com/yorukot/starker/pkg/logger"
	"github.com/yorukot/starker/pkg/response"
)

// @title starker Go API Template
// @version 1.0
//
// @description starker Go API Template
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /api
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter JWT Bearer token in the format: Bearer {token}

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
