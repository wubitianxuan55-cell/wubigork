package bm25

import (
	"sync"
	"testing"
)

// TestCacheHitBuildsOnce 同一 key 命中缓存：build 只调用一次，Rank 结果复用。
func TestCacheHitBuildsOnce(t *testing.T) {
	c := NewCache()
	builds := 0
	r1 := c.Get("cost-lib", func() []Doc {
		builds++
		return []Doc{{ID: 0, Text: "振动锤 台班 租赁"}}
	})
	r2 := c.Get("cost-lib", func() []Doc {
		builds++
		return []Doc{{ID: 0, Text: "振动锤 台班 租赁"}}
	})
	if builds != 1 {
		t.Fatalf("build 应只调用一次，实际 %d", builds)
	}
	if r1 != r2 {
		t.Fatal("命中缓存应返回同一 Ranker 实例")
	}
	if len(r1.Rank("振动锤")) == 0 {
		t.Fatal("缓存 Ranker 应可正常打分")
	}
}

// TestCacheInvalidateRebuilds Invalidate 后重新构建。
func TestCacheInvalidateRebuilds(t *testing.T) {
	c := NewCache()
	builds := 0
	_ = c.Get("k", func() []Doc { builds++; return []Doc{{ID: 0, Text: "水泥 吨"}} })
	_ = c.Get("k", func() []Doc { builds++; return []Doc{{ID: 0, Text: "水泥 吨"}} })
	if builds != 1 {
		t.Fatalf("前置构建次数异常: %d", builds)
	}
	c.Invalidate("k")
	_ = c.Get("k", func() []Doc { builds++; return []Doc{{ID: 0, Text: "水泥 吨"}} })
	if builds != 2 {
		t.Fatalf("Invalidate 后应重新构建，实际 build 次数 %d", builds)
	}
}

// TestCacheDistinctKeys 不同 key 各自构建互不影响。
func TestCacheDistinctKeys(t *testing.T) {
	c := NewCache()
	builds := map[string]int{}
	get := func(key string) {
		_ = c.Get(key, func() []Doc {
			builds[key]++
			return []Doc{{ID: 0, Text: key}}
		})
	}
	get("project-a")
	get("project-b")
	get("project-a")
	if builds["project-a"] != 1 || builds["project-b"] != 1 {
		t.Fatalf("各 key 应各自构建一次: %+v", builds)
	}
}

// TestCacheConcurrentAccess 并发读写不 panic、不丢缓存（-race 下验证数据竞争）。
func TestCacheConcurrentAccess(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "k" + string(rune('0'+n%4))
			_ = c.Get(key, func() []Doc { return []Doc{{ID: n, Text: "振动锤 台班"}} })
			c.InvalidateAll()
			_ = c.Get(key, func() []Doc { return []Doc{{ID: n, Text: "振动锤 台班"}} })
		}(i)
	}
	wg.Wait()
}
