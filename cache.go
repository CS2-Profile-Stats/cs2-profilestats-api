package main

import (
	"sync"
	"time"
)

type cacheItem struct {
	value   any
	expires time.Time
}

type Cache struct {
	mutex sync.RWMutex
	entries map[string]cacheItem
}

func NewCache() *Cache {
  c := &Cache{entries: make(map[string]cacheItem)}
  go c.cleanup()
  return c
}

func (c *Cache) Get(key string) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expires) {
    return nil, false
	}
	return entry.value, true
}

func (c *Cache) Set(key string, value any, ttl time.Duration) {
  c.mutex.Lock()
  defer c.mutex.Unlock()
  c.entries[key] = cacheItem{value: value, expires: time.Now().Add(ttl)}
}

func (c *Cache) cleanup() {
  ticker := time.NewTicker(5 * time.Minute)
  for range ticker.C {
    c.mutex.Lock()
    for k, v := range c.entries {
      if time.Now().After(v.expires) {
        delete(c.entries, k)
      }
    }
    c.mutex.Unlock()
  }
}
