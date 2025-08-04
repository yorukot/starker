package database

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

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

func getDatabaseURL() string {
	DB_HOST := os.Getenv("DB_HOST")
	DB_PORT := os.Getenv("DB_PORT")
	DB_USER := os.Getenv("DB_USER")
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	DB_NAME := os.Getenv("DB_NAME")
	DB_SSL_MODE := os.Getenv("DB_SSL_MODE")
	if DB_SSL_MODE == "" {
		DB_SSL_MODE = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME, DB_SSL_MODE,
	)
}

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
