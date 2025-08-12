package team

import "github.com/jackc/pgx/v5/pgxpool"

type TeamHandler struct {
	DB *pgxpool.Pool
}

