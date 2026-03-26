package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCacheFromEnv() (*RedisCache, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options, configured, err := redisOptionsFromEnv()
	if err != nil {
		return nil, err
	}
	if !configured {
		return nil, nil
	}

	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func (redisCache *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := redisCache.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return value, true, nil
}

func (redisCache *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return redisCache.client.Set(ctx, key, value, ttl).Err()
}

func redisOptionsFromEnv() (*redis.Options, bool, error) {
	url := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if url != "" {
		options, err := redis.ParseURL(url)
		if err != nil {
			return nil, false, fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		return options, true, nil
	}

	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil, false, nil
	}

	options := &redis.Options{
		Addr:     addr,
		Password: strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		DB:       envInt("REDIS_DB", 0),
	}

	if envBool("REDIS_TLS", false) {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	return options, true, nil
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
