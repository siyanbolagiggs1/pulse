package database

import (
	"context"
	"log"

	"github.com/pulse/api/internal/config"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func ConnectRedis() {
	opt, err := redis.ParseURL(config.App.RedisURL)
	if err != nil {
		log.Printf("WARNING: Failed to parse Redis URL (%v) — rate limiting and token blacklisting disabled", err)
		return
	}

	Redis = redis.NewClient(opt)

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Printf("WARNING: Redis unavailable (%v) — rate limiting and token blacklisting disabled", err)
		Redis = nil
		return
	}

	log.Println("Connected to Redis")
}
