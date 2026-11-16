package cache

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Cache caches read-only tool results (file reads) to avoid redundant disk IO
// within a turn. Write operations invalidate related entries. Thread-safe.
// Cache keys include path + offset so different read ranges have separate entries.
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	pathKeys map[string]map[string]struct{} // path → set of cache keys (O(1) invalidate)
	ttl      time.Duration
}

type cacheItem struct {
	content string
	digest  [sha256.Size]byte
	cached  time.Time
}

// newToolCache creates a cache with the given TTL. Zero or negative TTL means
// no expiry (entries live until invalidated by a write, content change or clear).
func New(ttl time.Duration) *Cache {
	return &Cache{
		items:    make(map[string]*cacheItem),
		pathKeys: make(map[string]map[string]struct{}),
		ttl:      ttl,
	}
}

func cachePathKey(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func fileDigest(path string) ([sha256.Size]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [sha256.Size]byte{}, err
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

// readKey builds a collision-free cache key from normalized path, offset and
// limit. limit 进键（刀0，dsh-ops 第二波 2026-11-10）：同 path+offset 不同
// limit 是不同的页。NUL separators prevent paths containing "@" from colliding
// with numeric suffixes.
func readKey(path string, offset, limit int) string {
	return fmt.Sprintf("%s\x00%d\x00%d", path, offset, limit)
}

// Get returns the cached content for a read_file(path, offset, limit). Returns
// ("", false) on miss or stale entry.
func (c *Cache) Get(path string, offset, limit int) (string, bool) {
	path = cachePathKey(path)
	key := readKey(path, offset, limit)
	c.mu.RLock()
	ci, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}

	if c.ttl > 0 && time.Since(ci.cached) > c.ttl {
		c.deleteIfSame(path, key, ci)
		return "", false
	}

	// A cache hit is valid only while the bytes on disk match the bytes that
	// produced this result. mtime alone misses same-timestamp writes on Windows,
	// so verify a content digest on every hit.
	digest, err := fileDigest(path)
	if err != nil || digest != ci.digest {
		c.deleteIfSame(path, key, ci)
		return "", false
	}
	return ci.content, true
}

// Set caches content for a read_file(path, offset, limit).
func (c *Cache) Set(path string, offset, limit int, content string) {
	path = cachePathKey(path)
	digest, err := fileDigest(path)
	if err != nil {
		return
	}
	key := readKey(path, offset, limit)
	c.mu.Lock()
	c.items[key] = &cacheItem{
		content: content,
		digest:  digest,
		cached:  time.Now(),
	}
	if c.pathKeys[path] == nil {
		c.pathKeys[path] = make(map[string]struct{})
	}
	c.pathKeys[path][key] = struct{}{}
	c.mu.Unlock()
}

// invalidatePath removes all cache entries for a given file path. O(keys-per-path).
func (c *Cache) InvalidatePath(path string) {
	path = cachePathKey(path)
	c.mu.Lock()
	for k := range c.pathKeys[path] {
		delete(c.items, k)
	}
	delete(c.pathKeys, path)
	c.mu.Unlock()
}

// InvalidateUnder removes every cached file under a file or directory path.
// This is used by domain writers that update several files beneath a workspace.
func (c *Cache) InvalidateUnder(path string) {
	root := cachePathKey(path)
	c.mu.Lock()
	for cachedPath, keys := range c.pathKeys {
		rel, err := filepath.Rel(root, cachedPath)
		if err != nil || (rel != "." && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
			continue
		}
		for k := range keys {
			delete(c.items, k)
		}
		delete(c.pathKeys, cachedPath)
	}
	c.mu.Unlock()
}

func (c *Cache) deleteIfSame(path, key string, want *cacheItem) {
	c.mu.Lock()
	if cur, ok := c.items[key]; ok && cur == want {
		delete(c.items, key)
		if keys := c.pathKeys[path]; keys != nil {
			delete(keys, key)
			if len(keys) == 0 {
				delete(c.pathKeys, path)
			}
		}
	}
	c.mu.Unlock()
}

// clear removes all cache entries. Called at the start of each turn.
func (c *Cache) Clear() {
	c.mu.Lock()
	clear(c.items)
	clear(c.pathKeys)
	c.mu.Unlock()
}
