package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chats/go-user-api/config"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// RedisClient is a wrapper for redis client
type RedisClient struct {
	client  *redis.Client
	ctx     context.Context
	enabled bool
	ttl     time.Duration
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		// Log the error but continue without Redis
		log.Warn().Err(err).Msg("Failed to connect to Redis, continuing without caching")
		return &RedisClient{
			client:  client,
			ctx:     ctx,
			enabled: false,
			ttl:     time.Duration(cfg.RedisCacheTTL) * time.Second,
		}, nil
	}

	log.Info().Msg("Connected to Redis successfully")
	return &RedisClient{
		client:  client,
		ctx:     ctx,
		enabled: true,
		ttl:     time.Duration(cfg.RedisCacheTTL) * time.Second,
	}, nil
}

// Get retrieves an item from the cache
func (c *RedisClient) Get(key string, dest interface{}) (bool, error) {
	if !c.enabled {
		return false, nil
	}

	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		// Key does not exist
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get from cache: %w", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return true, nil
}

// Set adds an item to the cache with default TTL
func (c *RedisClient) Set(key string, value interface{}) error {
	return c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL adds an item to the cache with a specific TTL
func (c *RedisClient) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal data for caching: %w", err)
	}

	err = c.client.Set(c.ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes an item from the cache
func (c *RedisClient) Delete(key string) error {
	if !c.enabled {
		return nil
	}

	err := c.client.Del(c.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// DeleteByPattern removes items from the cache matching a pattern
func (c *RedisClient) DeleteByPattern(pattern string) error {
	if !c.enabled {
		return nil
	}

	// Find all keys matching the pattern
	keys, err := c.client.Keys(c.ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find keys matching pattern: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all found keys
	err = c.client.Del(c.ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete keys from cache: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (c *RedisClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns whether caching is enabled
func (c *RedisClient) IsEnabled() bool {
	return c.enabled
}
