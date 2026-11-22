//go:build !faker

package usecase

import (
	"sync"
	"time"
)

// ttlCache keeps the slow-changing data of the team board tiles (repository lists, counts, commit activity)
// for a while, so a wallboard refreshing every 10 seconds does not query Gitea for each of them every time.
type ttlCache struct {
	mutex   sync.Mutex
	entries map[string]ttlEntry
}

type ttlEntry struct {
	value   interface{}
	expires time.Time
}

func newTTLCache() *ttlCache {
	return &ttlCache{entries: make(map[string]ttlEntry)}
}

// get returns the cached value of key, or loads it and keeps it for ttl. Errors are not cached.
// A nil cache always loads.
func (c *ttlCache) get(key string, ttl time.Duration, now time.Time, load func() (interface{}, error)) (interface{}, error) {
	if c == nil {
		return load()
	}

	c.mutex.Lock()
	entry, ok := c.entries[key]
	c.mutex.Unlock()
	if ok && now.Before(entry.expires) {
		return entry.value, nil
	}

	value, err := load()
	if err != nil {
		return nil, err
	}

	c.mutex.Lock()
	c.entries[key] = ttlEntry{value: value, expires: now.Add(ttl)}
	c.mutex.Unlock()

	return value, nil
}
