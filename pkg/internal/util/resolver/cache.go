package resolver

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gost/gost/pkg/logger"
	"github.com/miekg/dns"
)

type CacheKey string

// NewCacheKey generates resolver cache key from question of dns query.
func NewCacheKey(q *dns.Question) CacheKey {
	if q == nil {
		return ""
	}
	key := fmt.Sprintf("%s%s.%s", q.Name, dns.Class(q.Qclass).String(), dns.Type(q.Qtype).String())
	return CacheKey(key)
}

type cacheItem struct {
	msg *dns.Msg
	ts  time.Time
	ttl time.Duration
}

type Cache struct {
	m      sync.Map
	logger logger.Logger
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) WithLogger(logger logger.Logger) *Cache {
	c.logger = logger
	return c
}

func (c *Cache) Load(key CacheKey) *dns.Msg {
	v, ok := c.m.Load(key)
	if !ok {
		return nil
	}

	item, ok := v.(*cacheItem)
	if !ok {
		return nil
	}

	if time.Since(item.ts) > item.ttl {
		c.m.Delete(key)
		return nil
	}

	c.logger.Debugf("resolver cache hit: %s", key)

	return item.msg.Copy()
}

func (c *Cache) Store(key CacheKey, mr *dns.Msg, ttl time.Duration) {
	if key == "" || mr == nil || ttl < 0 {
		return
	}

	if ttl == 0 {
		for _, answer := range mr.Answer {
			v := time.Duration(answer.Header().Ttl) * time.Second
			if ttl == 0 || ttl > v {
				ttl = v
			}
		}
	}
	if ttl == 0 {
		ttl = 30 * time.Second
	}

	c.m.Store(key, &cacheItem{
		msg: mr.Copy(),
		ts:  time.Now(),
		ttl: ttl,
	})

	c.logger.Debugf("resolver cache store: %s, ttl: %v", key, ttl)
}
