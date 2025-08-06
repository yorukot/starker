package database

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// InitDatabase initialize the database connection pool and return the pool and also migrate the database
func InitDatabase() (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, getDatabaseURL())
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
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSL_MODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode,
	)
}

// Migrator migrate the database
func Migrator() {
	zap.L().Info("Migrating database")

	wd, _ := os.Getwd()

	databaseURL := getDatabaseURL()
	migrationsPath := "file://" + wd + "/migrations"

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		zap.L().Fatal("failed to create migrator", zap.Error(err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		zap.L().Fatal("failed to migrate database", zap.Error(err))
	}

	zap.L().Info("Database migrated")
}
