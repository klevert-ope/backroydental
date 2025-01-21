package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

type RedisConfig struct {
	URL          string
	PoolSize     int
	DialTimeout  time.Duration
	MinIdleConns int
	ReadTimeout  time.Duration
	MaxRetries   int
}

// InitializeRedis initializes the Redis client lazily
func InitializeRedis() error {
	config, err := LoadRedisConfig()
	if err != nil {
		return fmt.Errorf("failed to load Redis configuration: %w", err)
	}

	RedisClient, err = NewRedisClient(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	log.Println("Redis connection initialized successfully.")
	return nil
}

// LoadRedisConfig loads configuration from environment variables with default fallbacks
func LoadRedisConfig() (RedisConfig, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return RedisConfig{}, errors.New("REDIS_URL environment variable is not set")
	}

	poolSize := getEnvAsInt("REDIS_POOL_SIZE", 10) // Default: 10
	dialTimeout := getEnvAsDuration("REDIS_DIAL_TIMEOUT", 30*time.Second)
	minIdleConns := getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5) // Default: 5
	readTimeout := getEnvAsDuration("REDIS_READ_TIMEOUT", 10*time.Second)
	maxRetries := getEnvAsInt("REDIS_MAX_RETRIES", 3) // Default: 3

	return RedisConfig{
		URL:          redisURL,
		PoolSize:     poolSize,
		DialTimeout:  dialTimeout,
		MinIdleConns: minIdleConns,
		ReadTimeout:  readTimeout,
		MaxRetries:   maxRetries,
	}, nil
}

func getEnvAsInt(name string, defaultValue int) int {
	if value, exists := os.LookupEnv(name); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s, using default: %d", name, defaultValue)
	}
	return defaultValue
}

func getEnvAsDuration(name string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(name); exists {
		if durationValue, err := time.ParseDuration(value); err == nil {
			return durationValue
		}
		log.Printf("Warning: Invalid duration value for %s, using default: %s", name, defaultValue.String())
	}
	return defaultValue
}

// NewRedisClient creates a Redis client with the provided configuration
func NewRedisClient(config RedisConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	opt.PoolSize = config.PoolSize
	opt.MinIdleConns = config.MinIdleConns
	opt.DialTimeout = config.DialTimeout
	opt.ReadTimeout = config.ReadTimeout
	opt.MaxRetries = config.MaxRetries

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis server: %w", err)
	}

	log.Printf("Redis client initialized with configuration: PoolSize=%d, MinIdleConns=%d, DialTimeout=%s, ReadTimeout=%s, MaxRetries=%d",
		config.PoolSize, config.MinIdleConns, config.DialTimeout.String(), config.ReadTimeout.String(), config.MaxRetries)
	return client, nil
}

// NewLock acquires a distributed lock using Redis
func NewLock(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	if RedisClient == nil {
		return false, errors.New("Redis client is not initialized")
	}

	return RedisClient.SetNX(ctx, key, value, ttl).Result()
}

// ReleaseLock releases a distributed lock using Redis with Lua scripting
func ReleaseLock(ctx context.Context, key string, value string) error {
	if RedisClient == nil {
		return errors.New("Redis client is not initialized")
	}

	const releaseLockScript = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	script := redis.NewScript(releaseLockScript)
	result, err := script.Run(ctx, RedisClient, []string{key}, value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	if result.(int64) == 0 {
		return errors.New("lock release failed: not the lock owner")
	}
	return nil
}

// MonitorRedisPool logs the connection pool statistics for monitoring
func MonitorRedisPool(ctx context.Context) {
	stats := RedisClient.PoolStats()
	log.Printf("Redis pool stats: Total: %d, Idle: %d, Stale: %d", stats.TotalConns, stats.IdleConns, stats.StaleConns)
}
