package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler holds dependencies for task handlers.
type Handler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(db *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}
