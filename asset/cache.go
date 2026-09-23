package asset

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type loadKey struct {
	id     string
	typ    reflect.Type
	format string
}

func newLoadKey(id string, typ reflect.Type, format string) loadKey {
	return loadKey{
		id:     id,
		typ:    typ,
		format: format,
	}
}

func (k loadKey) String() string {
	typName := "<nil>"
	if k.typ != nil {
		typName = k.typ.PkgPath() + "." + k.typ.String()
	}
	return fmt.Sprintf(
		"%d:%s%d:%s%d:%s",
		len(k.id), k.id,
		len(typName), typName,
		len(k.format), k.format,
	)
}

type cacheEntry struct {
	value    any
	loadedAt time.Time
}

type cache struct {
	mu      sync.RWMutex
	entries map[loadKey]cacheEntry
}

func newCache() *cache {
	return &cache{
		entries: make(map[loadKey]cacheEntry),
	}
}

func (c *cache) get(key loadKey) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return entry.value, ok
}

func (c *cache) put(key loadKey, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, loadedAt: time.Now()}
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
	c.entries = make(map[loadKey]cacheEntry)
}
