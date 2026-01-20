package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

// RedisCache provides a distributed L2 cache using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache client
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Get retrieves a value from Redis
func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}

	// Try to unmarshal as generic interface{}
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// If unmarshaling fails, return raw string
		return val, nil
	}
	return result, nil
}

// GetBytes retrieves raw bytes from Redis
func (r *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

// Set stores a value in Redis with TTL
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// SetBytes stores raw bytes in Redis with TTL
func (r *RedisCache) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from Redis
func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Pipeline returns a Redis pipeline for batch operations
func (r *RedisCache) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// MGet gets multiple keys at once (pipelining)
func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return r.client.MGet(ctx, keys...).Result()
}

// MSet sets multiple keys at once
func (r *RedisCache) MSet(ctx context.Context, values map[string]interface{}, ttl time.Duration) error {
	pipe := r.client.Pipeline()
	
	for key, val := range values {
		data, err := json.Marshal(val)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, ttl)
	}
	
	_, err := pipe.Exec(ctx)
	return err
}
