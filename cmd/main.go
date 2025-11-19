package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/joho/godotenv/autoload"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	_ "github.com/yorukot/starker/docs"
	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/database"
	"github.com/yorukot/starker/pkg/logger"
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

	app := &cli.App{
		Name:  "starker",
		Usage: "Starker server application",
		Commands: []*cli.Command{
			{
				Name:  "api",
				Usage: "Run API server only",
				Action: func(c *cli.Context) error {
					return run("api")
				},
			},
			{
				Name:  "worker",
				Usage: "Run worker only",
				Action: func(c *cli.Context) error {
					return run("worker")
				},
			},
			{
				Name:  "both",
				Usage: "Run both API server and worker",
				Action: func(c *cli.Context) error {
					return run("both")
				},
			},
		},
		Action: func(c *cli.Context) error {
			// Default action: run both
			return run("both")
		},
	}

	if err := app.Run(os.Args); err != nil {
		zap.L().Fatal("Application error", zap.Error(err))
	}
}

func run(mode string) error {
	_, err := config.InitConfig()
	if err != nil {
		zap.L().Fatal("Error initializing config", zap.Error(err))
		return err
	}

	db, err := database.InitDatabase()
	if err != nil {
		zap.L().Fatal("Failed to initialize database", zap.Error(err))
		return err
	}
	defer db.Close()

	// Start services based on mode
	if mode == "api" || mode == "both" {
		r := chi.NewRouter()
		go startAPI(r, db)
		zap.L().Info("API server started")
	}

	if mode == "worker" || mode == "both" {
		go startWorker()
		zap.L().Info("Worker started")
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutting down...")
	return nil
}
