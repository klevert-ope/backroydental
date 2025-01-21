package cache

import (
	"RoyDental/database"
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	client *redis.Client
}

// NewCache creates a new Cache instance, ensuring that RedisClient is not nil.
func NewCache() (*Cache, error) {
	if database.RedisClient == nil {
		return nil, errors.New("Redis client is not initialized")
	}
	return &Cache{client: database.RedisClient}, nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if c.client == nil {
		return errors.New("Redis client is not initialized")
	}
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) DeleteAll(ctx context.Context, pattern string) error {
	if c.client == nil {
		return errors.New("Redis client is not initialized")
	}
	// Use SCAN for better efficiency on large datasets
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if c.client == nil {
		return errors.New("Redis client is not initialized")
	}
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c.client == nil {
		return "", errors.New("Redis client is not initialized")
	}
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // key does not exist
	}
	return val, err
}

func (c *Cache) DeleteBatch(ctx context.Context, keys ...string) error {
	if c.client == nil {
		return errors.New("Redis client is not initialized")
	}
	return c.client.Del(ctx, keys...).Err()
}
