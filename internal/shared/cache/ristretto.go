package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

// RistrettoCache provides an in-memory L1 cache using Ristretto
type RistrettoCache struct {
	cache *ristretto.Cache
}

// NewRistrettoCache creates a new L1 cache
// maxCost: maximum total cost of items (in bytes, typically)
// numCounters: number of keys to track frequency (10x maxCost recommended)
func NewRistrettoCache(maxCost int64, numCounters int64) (*RistrettoCache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,      // 10x maxCost recommended
		MaxCost:     maxCost,           // Total cache size
		BufferItems: 64,                // Number of keys per Get buffer
		Metrics:     true,              // Enable metrics
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoCache{cache: cache}, nil
}

// Get retrieves a value from the cache
func (r *RistrettoCache) Get(key string) (interface{}, bool) {
	return r.cache.Get(key)
}

// Set stores a value in the cache with TTL
// cost should represent the size/weight of the item
func (r *RistrettoCache) Set(key string, value interface{}, cost int64, ttl time.Duration) bool {
	return r.cache.SetWithTTL(key, value, cost, ttl)
}

// Delete removes a key from the cache
func (r *RistrettoCache) Delete(key string) {
	r.cache.Del(key)
}

// Clear removes all items from the cache
func (r *RistrettoCache) Clear() {
	r.cache.Clear()
}

// Metrics returns cache metrics
func (r *RistrettoCache) Metrics() *ristretto.Metrics {
	return r.cache.Metrics
}

// Close closes the cache
func (r *RistrettoCache) Close() {
	r.cache.Close()
}
