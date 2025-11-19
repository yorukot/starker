package handler

import (
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// App is the application context
type App struct {
	DB          *pgxpool.Pool
	Redis       *redis.Client
	AsynqClient *asynq.Client
}
