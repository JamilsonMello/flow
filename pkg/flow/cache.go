package flow

import (
	"sync"
)

type flowCache struct {
	enabled bool
	maxSize int
	entries map[string]*Flow
	mu      sync.RWMutex
}

func newFlowCache(enabled bool, maxSize int) *flowCache {
	c := &flowCache{
		enabled: enabled,
		maxSize: maxSize,
	}
	if enabled {
		c.entries = make(map[string]*Flow, maxSize)
	}
	return c
}

func cacheKey(name, identifier string) string {
	return name + "::" + identifier
}

func (c *flowCache) Get(name, identifier string) (*Flow, bool) {
	if !c.enabled || c.entries == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.entries[cacheKey(name, identifier)]
	return f, ok
}

func (c *flowCache) Set(name, identifier string, f *Flow) {
	if !c.enabled || c.entries == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	c.entries[cacheKey(name, identifier)] = f
}

func (c *flowCache) Delete(name, identifier string) {
	if !c.enabled || c.entries == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, cacheKey(name, identifier))
}

func (c *flowCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = nil
}
