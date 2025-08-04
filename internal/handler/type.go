package handler

import "github.com/jackc/pgx/v5/pgxpool"

// App is the application context
type App struct {
	DB *pgxpool.Pool
}
