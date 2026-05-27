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
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	Redis = redis.NewClient(opt)

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")
}
