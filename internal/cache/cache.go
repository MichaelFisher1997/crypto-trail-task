package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Value stores the cached balance and when it was fetched.
type Value struct {
	Lamports  uint64
	FetchedAt time.Time
}

type item struct {
	val       Value
	expiresAt time.Time
}

// Cache provides a TTL cache with singleflight coalescing per key.
type Cache struct {
	mu     sync.RWMutex
	items  map[string]item
	ttl    time.Duration
	group  singleflight.Group
}

func New(ttl time.Duration) *Cache {
	return &Cache{items: make(map[string]item), ttl: ttl}
}

// GetOrFetch returns a cached value if valid; otherwise it coalesces concurrent
// fetches for the same key using singleflight and stores the result.
// Returns the value, source ("cache" or "rpc"), and error if fetching failed.
func (c *Cache) GetOrFetch(ctx context.Context, key string, fetch func(context.Context) (Value, error)) (Value, string, error) {
	// fast path: cache hit
	c.mu.RLock()
	it, ok := c.items[key]
	if ok && time.Now().Before(it.expiresAt) {
		v := it.val
		c.mu.RUnlock()
		return v, "cache", nil
	}
	c.mu.RUnlock()

	// singleflight to coalesce concurrent misses
	res, err, _ := c.group.Do(key, func() (interface{}, error) {
		v, err := fetch(ctx)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.items[key] = item{val: v, expiresAt: time.Now().Add(c.ttl)}
		c.mu.Unlock()
		return v, nil
	})
	if err != nil {
		return Value{}, "", err
	}
	return res.(Value), "rpc", nil
}

// Len returns the number of items in the cache (for tests).
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
