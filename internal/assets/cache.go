package assets

import (
	"reflect"
	"sync"
	"time"
)

type cacheKey struct {
	id     string
	typ    reflect.Type
	format string
}

type cacheEntry struct {
	value      any
	version    uint64
	loadedAt   time.Time
	accessedAt time.Time
	size       int64
}

// cache stores decoded values. Versioning and eviction policy can be added
// without changing the public loader contract.
type cache struct {
	mu      sync.RWMutex
	entries map[cacheKey]cacheEntry
}

func newCache() *cache {
	return &cache{entries: make(map[cacheKey]cacheEntry)}
}

func (c *cache) get(key cacheKey) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return entry.value, ok
}

func (c *cache) put(key cacheKey, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value}
}

func (c *cache) invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if key.id == id {
			delete(c.entries, key)
		}
	}
}

func (c *cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[cacheKey]cacheEntry)
}
