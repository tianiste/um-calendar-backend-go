package cache

import (
	"sync"
	"time"
)

type Entry struct {
	Content     []byte
	ContentType string
	ExpiresAt   time.Time
}

type InMemoryCache struct {
	mutex sync.RWMutex
	items map[string]Entry
	ttl   time.Duration
}

func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	return &InMemoryCache{
		items: make(map[string]Entry),
		ttl:   ttl,
	}
}

func (cache *InMemoryCache) Get(key string) (Entry, bool) {
	cache.mutex.RLock()
	entry, ok := cache.items[key]
	cache.mutex.RUnlock()

	if !ok || time.Now().After(entry.ExpiresAt) {
		return Entry{}, false
	}

	return entry, true
}

func (cache *InMemoryCache) Set(key string, content []byte, contentType string) {
	cache.mutex.Lock()
	cache.items[key] = Entry{
		Content:     content,
		ContentType: contentType,
		ExpiresAt:   time.Now().Add(cache.ttl),
	}
	cache.mutex.Unlock()
}
