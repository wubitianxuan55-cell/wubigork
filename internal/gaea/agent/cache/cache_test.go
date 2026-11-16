package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBasicGetSet(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	c.Set(f, 0, 0, "hello")
	val, ok := c.Get(f, 0, 0)
	if !ok {
		t.Fatal("cache miss after set")
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %q", val)
	}
}

func TestInvalidateOnWrite(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("v1"), 0644)

	c.Set(f, 0, 0, "v1")
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(f, []byte("v2"), 0644)

	_, ok := c.Get(f, 0, 0)
	if ok {
		t.Fatal("cache should be invalidated after file modification")
	}
}

func TestTOCTOUConcurrentGetSet(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("initial"), 0644)

	c.Set(f, 0, 0, "initial")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			c.Get(f, 0, 0)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			c.Set(f, 0, 0, fmt.Sprintf("v%d", i))
		}
	}()

	wg.Wait()

	val, ok := c.Get(f, 0, 0)
	if !ok {
		t.Fatal("缓存条目被 TOCTOU 竞态错误删除")
	}
	if val == "" {
		t.Fatal("缓存条目存在但内容为空")
	}
}

func TestClear(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("data"), 0644)

	c.Set(f, 0, 0, "data")
	c.Clear()

	_, ok := c.Get(f, 0, 0)
	if ok {
		t.Fatal("cache should be empty after clear")
	}
}

func TestInvalidatePath(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "a.txt")
	f2 := filepath.Join(tmp, "b.txt")
	os.WriteFile(f1, []byte("a"), 0644)
	os.WriteFile(f2, []byte("b"), 0644)

	c.Set(f1, 0, 0, "a")
	c.Set(f2, 0, 0, "b")
	c.InvalidatePath(f1)

	_, ok := c.Get(f1, 0, 0)
	if ok {
		t.Fatal("f1 should be invalidated")
	}
	val, ok := c.Get(f2, 0, 0)
	if !ok || val != "b" {
		t.Fatal("f2 should still be cached")
	}
}

func TestOffsetKeys(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("0123456789"), 0644)

	c.Set(f, 0, 0, "0123456789")
	c.Set(f, 5, 0, "56789")

	v0, ok0 := c.Get(f, 0, 0)
	v5, ok5 := c.Get(f, 5, 0)
	if !ok0 || v0 != "0123456789" {
		t.Fatal("offset 0 cache miss")
	}
	if !ok5 || v5 != "56789" {
		t.Fatal("offset 5 cache miss")
	}
}

func TestReadKeyDoesNotCollideWhenPathContainsSuffixLikeText(t *testing.T) {
	root := t.TempDir()
	f1 := filepath.Join(root, "a@5")
	f2 := filepath.Join(root, "a")
	for _, f := range []string{f1, f2} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := New(-1)
	c.Set(f1, 0, 0, "at-file")
	c.Set(f2, 5, 0, "offset-five")
	if got, ok := c.Get(f1, 0, 0); !ok || got != "at-file" {
		t.Fatalf("a@5 entry was overwritten: %q %v", got, ok)
	}
	if got, ok := c.Get(f2, 5, 0); !ok || got != "offset-five" {
		t.Fatalf("a offset=5 entry was overwritten: %q %v", got, ok)
	}
}

func TestContentDigestCatchesSameMtimeWrite(t *testing.T) {
	c := New(-1)
	tmp := t.TempDir()
	f := filepath.Join(tmp, "f.txt")
	os.WriteFile(f, []byte("v1"), 0644)

	c.Set(f, 0, 0, "v1")
	info, err := os.Stat(f)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	os.WriteFile(f, []byte("v2"), 0644)
	if err := os.Chtimes(f, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get(f, 0, 0)
	if ok {
		t.Fatal("same-mtime content change must invalidate the cache")
	}
}

// 刀0（dsh-ops 第二波 2026-11-10）：limit 进缓存键——同 path+offset 不同
// limit 是不同的页，旧键撞页会把缓存的 2000 行页错发给 limit=100 的调用。
func TestCacheLimitDistinctPages(t *testing.T) {
	f := filepath.Join(t.TempDir(), "page.txt")
	if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	c := New(0)
	c.Set(f, 0, 100, "first-100-lines")
	// 显式 limit=100 命中；缺省页（limit=0）与 limit=200 都是未命中。
	if v, ok := c.Get(f, 0, 100); !ok || v != "first-100-lines" {
		t.Fatalf("limit=100 应命中: %q %v", v, ok)
	}
	if _, ok := c.Get(f, 0, 0); ok {
		t.Fatal("缺省页不应命中 limit=100 的缓存")
	}
	if _, ok := c.Get(f, 0, 200); ok {
		t.Fatal("limit=200 不应命中 limit=100 的缓存")
	}
	// InvalidatePath 仍按 path 全键失效。
	c.Set(f, 10, 0, "offset-10-default")
	c.InvalidatePath(f)
	if _, ok := c.Get(f, 0, 100); ok {
		t.Fatal("失效后不得命中")
	}
}

func TestInvalidateUnder(t *testing.T) {
	root := t.TempDir()
	f1 := filepath.Join(root, "章节", "001.md")
	f2 := filepath.Join(root, "角色", "林晚.md")
	for _, f := range []string{f1, f2} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := New(-1)
	c.Set(f1, 0, 0, "chapter")
	c.Set(f2, 0, 0, "character")

	c.InvalidateUnder(root)
	if _, ok := c.Get(f1, 0, 0); ok {
		t.Fatal("章节缓存应随小说工作区写入失效")
	}
	if _, ok := c.Get(f2, 0, 0); ok {
		t.Fatal("角色缓存应随小说工作区写入失效")
	}
}
