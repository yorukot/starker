package project

import "github.com/jackc/pgx/v5/pgxpool"

type ProjectHandler struct {
	DB *pgxpool.Pool
}

