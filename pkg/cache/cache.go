package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/brunexgeek/hugye/pkg/domain"
)

const CACHE_TTL int64 = 1 * 60 * 1000 // 1 minute

type cache_entry struct {
	response []byte
	ttl      int64
}

type cache struct {
	entries map[string]*cache_entry
	lock    sync.Mutex
}

func (c *cache) Set(name string, typ uint16, buf []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	name = fmt.Sprintf("%s_%d", name, typ)

	limit := time.Now().UnixMilli() + CACHE_TTL
	entry := c.entries[name]
	if entry != nil {
		entry.response = buf
		entry.ttl = limit + CACHE_TTL
	} else {
		c.entries[name] = &cache_entry{response: buf, ttl: limit}
	}
}

func (c *cache) Get(name string, typ uint16) []byte {
	c.lock.Lock()
	defer c.lock.Unlock()

	name = fmt.Sprintf("%s_%d", name, typ)

	entry := c.entries[name]
	if entry != nil {
		limit := time.Now().UnixMilli()
		if entry.ttl > limit {
			// TODO: get the smaller TTL from DNS response
			entry.ttl = limit + CACHE_TTL
			return entry.response
		}
		delete(c.entries, name)
	}
	return nil
}

func NewCache() domain.Cache {
	return &cache{entries: make(map[string]*cache_entry, 0)}
}
