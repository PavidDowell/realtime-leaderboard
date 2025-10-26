package db

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

// NewRedis connects to Redis and pings it with retry logic (like Postgres).
func NewRedis(ctx context.Context) (*Redis, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	var rdb *redis.Client
	var err error

	for i := 0; i < 10; i++ {
		rdb = redis.NewClient(&redis.Options{
			Addr:         addr,
			ReadTimeout:  500 * time.Millisecond,
			WriteTimeout: 500 * time.Millisecond,
			DialTimeout:  500 * time.Millisecond,
		})

		pingErr := rdb.Ping(ctx).Err()
		if pingErr == nil {
			return &Redis{Client: rdb}, nil
		}

		err = pingErr
		time.Sleep(500 * time.Millisecond)
	}

	return nil, err
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
