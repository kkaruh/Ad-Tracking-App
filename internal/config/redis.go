package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis(trigger chan bool) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URI"),
		DB:   0,
	})
	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		fmt.Println("⛒ Failed to connect to Redis:", err)
		os.Exit(1)
	}
	log.Println("Connected to Redis")
	trigger <- true
}
