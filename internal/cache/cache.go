package cache

import (
	"sync"
	"time"
)

type item struct {
	value      interface{}
	expiration time.Time
}

func (i item) isExpired() bool {
	if i.expiration.IsZero() {
		return false
	}
	return time.Now().After(i.expiration)
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		items: make(map[string]item),
	}

	// Periodically clean up expired keys
	go c.startCleanup(1 * time.Minute)

	return c
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, exists := c.items[key]
	if !exists || it.isExpired() {
		return nil, false
	}

	return it.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}

	c.items[key] = item{
		value:      value,
		expiration: exp,
	}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]item)
}

func (c *MemoryCache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if !v.expiration.IsZero() && now.After(v.expiration) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
