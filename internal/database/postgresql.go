package database

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/config"
)

// InitDatabase initialize the database connection pool and return the pool and also migrate the database
func InitDatabase() (*pgxpool.Pool, error) {
	ctx := context.Background()

	// Configure connection pool to handle concurrent operations better
	config, err := pgxpool.ParseConfig(getDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Increase pool size to handle more concurrent connections
	config.MaxConns = 25
	config.MinConns = 5

	// Reduce prepared statement cache to prevent "conn busy" errors
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	zap.L().Info("Database initialized")

	Migrator()

	return pool, nil
}

// getDatabaseURL return a pgsql connection uri by the environment variables
func getDatabaseURL() string {
	dbHost := config.Env().DBHost
	dbPort := config.Env().DBPort
	dbUser := config.Env().DBUser
	dbPassword := config.Env().DBPassword
	dbName := config.Env().DBName
	dbSSLMode := config.Env().DBSSLMode
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
}

// Migrator the database
func Migrator() {
	zap.L().Info("Migrating database")

	wd, _ := os.Getwd()

	databaseURL := getDatabaseURL()
	migrationsPath := "file://" + wd + "/migrations"

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		zap.L().Fatal("failed to create migrator", zap.Error(err))
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		zap.L().Fatal("failed to migrate database", zap.Error(err))
	}

	zap.L().Info("Database migrated")
}
