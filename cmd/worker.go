package main

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/yorukot/starker/internal/config"
	"github.com/yorukot/starker/internal/worker/handler"
	"github.com/yorukot/starker/internal/worker/tasks"
)

func startWorker(db *pgxpool.Pool, redisClient *redis.Client) {
	cfg := config.Env()

	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: cfg.RedisPassword,
		},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	h := handler.NewHandler(db, redisClient)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeStartService, h.HandleStartServiceTask)

	if err := srv.Run(mux); err != nil {
		panic(err)
	}
}
