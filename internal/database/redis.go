package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yorukot/starker/internal/config"
)

// InitRedis initializes the Redis client and returns it
func InitRedis() (*redis.Client, error) {
	ctx := context.Background()
	cfg := config.Env()

	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.RedisPassword,
		DB:       0, // use default DB
	})

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	zap.L().Info("Redis initialized", zap.String("addr", redisAddr))

	return client, nil
}
