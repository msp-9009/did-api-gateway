package cache

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"
	"time"
)

// MultiLayerCache provides L1 (in-memory) + L2 (Redis) caching
type MultiLayerCache struct {
	l1     *RistrettoCache
	l2     *RedisCache
	mu     sync.RWMutex
	onHit  func() // Metrics callback
	onMiss func() // Metrics callback
}

// NewMultiLayerCache creates a new multi-layer cache
func NewMultiLayerCache(l1 *RistrettoCache, l2 *RedisCache, onHit, onMiss func()) *MultiLayerCache {
	return &MultiLayerCache{
		l1:     l1,
		l2:     l2,
		onHit:  onHit,
		onMiss: onMiss,
	}
}

// Get retrieves a value, checking L1 then L2
func (m *MultiLayerCache) Get(ctx context.Context, key string) (interface{}, error) {
	// Try L1 first (in-memory, fastest)
	if val, ok := m.l1.Get(key); ok {
		if m.onHit != nil {
			m.onHit()
		}
		return val, nil
	}

	// Try L2 (Redis, distributed)
	val, err := m.l2.Get(ctx, key)
	if err == nil {
		// Populate L1 for next time
		m.l1.Set(key, val, 1, time.Hour)
		if m.onHit != nil {
			m.onHit()
		}
		return val, nil
	}

	if m.onMiss != nil {
		m.onMiss()
	}
	return nil, ErrCacheMiss
}

// Set stores a value in both L1 and L2
func (m *MultiLayerCache) Set(ctx context.Context, key string, value interface{}, cost int64, ttl time.Duration) error {
	// Set in L1 (in-memory)
	m.l1.Set(key, value, cost, ttl)

	// Set in L2 (Redis)
	return m.l2.Set(ctx, key, value, ttl)
}

// Delete removes a key from both caches
func (m *MultiLayerCache) Delete(ctx context.Context, key string) error {
	m.l1.Delete(key)
	return m.l2.Delete(ctx, key)
}

// GetOrLoad retrieves from cache or loads using the provided function
func (m *MultiLayerCache) GetOrLoad(
	ctx context.Context,
	key string,
	loader func(ctx context.Context) (interface{}, error),
	cost int64,
	ttl time.Duration,
) (interface{}, error) {
	// Try to get from cache
	val, err := m.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	// Cache miss - load the value
	val, err = loader(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := m.Set(ctx, key, val, cost, ttl); err != nil {
		// Log error but return the value anyway
		fmt.Printf("cache set error: %v\n", err)
	}

	return val, nil
}

// DIDCache is a specialized cache for DID public keys
type DIDCache struct {
	cache *MultiLayerCache
}

// NewDIDCache creates a cache optimized for DID resolution
func NewDIDCache(l1 *RistrettoCache, l2 *RedisCache, onHit, onMiss func()) *DIDCache {
	return &DIDCache{
		cache: NewMultiLayerCache(l1, l2, onHit, onMiss),
	}
}

// GetPublicKey retrieves a cached public key for a DID
func (d *DIDCache) GetPublicKey(ctx context.Context, did string) (ed25519.PublicKey, error) {
	val, err := d.cache.Get(ctx, "did:"+did)
	if err != nil {
		return nil, err
	}

	// Handle different value types
	switch v := val.(type) {
	case ed25519.PublicKey:
		return v, nil
	case []byte:
		if len(v) == ed25519.PublicKeySize {
			return ed25519.PublicKey(v), nil
		}
		return nil, fmt.Errorf("invalid public key size: %d", len(v))
	case string:
		// Assume hex or base64 encoded
		return nil, fmt.Errorf("string public key not yet supported")
	default:
		return nil, fmt.Errorf("unexpected public key type: %T", v)
	}
}

// SetPublicKey stores a public key for a DID
func (d *DIDCache) SetPublicKey(ctx context.Context, did string, pubKey ed25519.PublicKey, ttl time.Duration) error {
	return d.cache.Set(ctx, "did:"+did, pubKey, int64(len(pubKey)), ttl)
}

// Invalidate removes a DID from cache
func (d *DIDCache) Invalidate(ctx context.Context, did string) error {
	return d.cache.Delete(ctx, "did:"+did)
}
