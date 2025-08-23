package user

import "github.com/jackc/pgx/v5/pgxpool"

type UserHandler struct {
	DB *pgxpool.Pool
}
