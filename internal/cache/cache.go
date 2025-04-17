package cache

import (
	"container/list"
	"sync"
)

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
}

type MemoryCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.Mutex
}

type cacheItem struct {
	key   string
	value []byte
}

func NewMemoryCache(capacity int) *MemoryCache {
	return &MemoryCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.cache[key]; exists {
		c.list.MoveToFront(elem)
		return elem.Value.(*cacheItem).value, true
	}
	return nil, false
}

func (c *MemoryCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.cache[key]; exists {
		c.list.MoveToFront(elem)
		elem.Value.(*cacheItem).value = value
		return
	}

	if c.list.Len() >= c.capacity {
		// Remove the least recently used item
		last := c.list.Back()
		if last != nil {
			delete(c.cache, last.Value.(*cacheItem).key)
			c.list.Remove(last)
		}
	}

	item := &cacheItem{key: key, value: value}
	elem := c.list.PushFront(item)
	c.cache[key] = elem
}
