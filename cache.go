// Package doh provides a simple and flexible Go client library for DNS over HTTPS (DoH).
package doh

import (
	"sync"
	"time"
)

// cache is a basic, thread-safe in-memory cache with TTL support.
type cache struct {
	sync.RWMutex
	items map[string]cacheItem
}

// cacheItem holds the value and expiration time for a cache entry.
type cacheItem struct {
	value      interface{}
	expiration int64
}

// newCache creates and returns a new instance of a cache.
func newCache() *cache {
	return &cache{
		items: make(map[string]cacheItem),
	}
}

// Get retrieves an item from the cache. It returns nil if the item is not found
// or has expired.
func (c *cache) Get(key string) interface{} {
	c.RLock()
	defer c.RUnlock()
	item, found := c.items[key]
	if !found || time.Now().UnixNano() > item.expiration {
		return nil
	}
	return item.value
}

// Set adds or updates an item in the cache with a specified TTL in seconds.
func (c *cache) Set(key string, value interface{}, ttl int64) {
	c.Lock()
	defer c.Unlock()
	expiration := time.Now().UnixNano() + (ttl * int64(time.Second))
	c.items[key] = cacheItem{
		value:      value,
		expiration: expiration,
	}
}

// Close is a no-op for the in-memory cache but is included for
// potential future compatibility with more complex cache implementations.
func (c *cache) Close() {
	// No-op
}
