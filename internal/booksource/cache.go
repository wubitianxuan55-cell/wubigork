package booksource

import (
	"sync"
	"time"
)

// ── 进程内 TTL 缓存（owllook Redis 缓存的桌面单机对应物，规格 §2.5/§4 D6）────

type ttlItem[V any] struct {
	value   V
	expires time.Time
}

// TTLCache 泛型 TTL 缓存（互斥 map；时钟可注入供测试）。
type TTLCache[V any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string]ttlItem[V]
}

func NewTTLCache[V any](ttl time.Duration, now func() time.Time) *TTLCache[V] {
	if now == nil {
		now = time.Now
	}
	return &TTLCache[V]{ttl: ttl, now: now, items: map[string]ttlItem[V]{}}
}

// Get 命中且未过期返回值；过期即删除并报未命中。
func (c *TTLCache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	it, ok := c.items[key]
	if !ok {
		return zero, false
	}
	if c.now().After(it.expires) {
		delete(c.items, key)
		return zero, false
	}
	return it.value, true
}

// Set 写入（拷贝引用类型由调用方负责不可变）。
func (c *TTLCache[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = ttlItem[V]{value: value, expires: c.now().Add(c.ttl)}
}
