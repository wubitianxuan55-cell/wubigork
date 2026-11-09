package docmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ── 单元：LRU 双上限 ─────────────────────────────────────────────

// TestOCRCache_LRUEviction 条数上限逐出最旧；字节上限逐出大条目；访问刷新新鲜度。
func TestOCRCache_LRUEviction(t *testing.T) {
	c := newOCRCache()
	for i := 0; i < ocrCacheMaxEntries+1; i++ {
		c.put(fmt.Sprintf("k%d", i), "x")
	}
	if _, ok := c.get("k0"); ok {
		t.Fatal("最旧条目应被条数上限逐出")
	}
	if _, ok := c.get("k1"); !ok {
		t.Fatal("次旧条目应保留")
	}

	c2 := newOCRCache()
	c2.put("big", strings.Repeat("a", ocrCacheMaxBytes))
	c2.put("big2", strings.Repeat("b", 1024))
	if _, ok := c2.get("big"); ok {
		t.Fatal("超字节上限应逐出旧条目")
	}
	if _, ok := c2.get("big2"); !ok {
		t.Fatal("新条目应保留")
	}

	// 访问刷新：k1 被访问后，再灌满时逐出的是 k2 而非 k1。
	c3 := newOCRCache()
	for i := 0; i < ocrCacheMaxEntries; i++ {
		c3.put(fmt.Sprintf("m%d", i), "x")
	}
	if _, ok := c3.get("m0"); !ok {
		t.Fatal("前置：m0 应在缓存")
	}
	c3.put("fresh", "y") // 超条数上限，逐出最久未访问的 m1（m0 刚被 get 刷新）
	if _, ok := c3.get("m0"); !ok {
		t.Fatal("被访问过的 m0 不应被逐出")
	}
	if _, ok := c3.get("m1"); ok {
		t.Fatal("最久未访问的 m1 应被逐出")
	}
}

// ── ocrCached 行为：命中/失败不缓存/singleflight ──────────────────

// TestOCRCached_HitDoesNotRerun 命中不重跑 fn（同名键第二次拿首跑结果）。
func TestOCRCached_HitDoesNotRerun(t *testing.T) {
	var calls int
	key := fmt.Sprintf("unit-hit-%d", time.Now().UnixNano())
	fn := func() (string, error) {
		calls++
		return fmt.Sprintf("v%d", calls), nil
	}
	v1, err := ocrCached(key, fn)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	v2, err := ocrCached(key, fn)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn 被跑 %d 次，命中不应重跑", calls)
	}
	if v1 != "v1" || v2 != "v1" {
		t.Fatalf("v1=%q v2=%q, 第二次应拿缓存的首跑结果", v1, v2)
	}
}

// TestOCRCached_ErrorsNotCached 失败不缓存：第二次调用重跑 fn（保住冷启动
// 预算超时→引擎拉起→下轮成功的阶梯）。
func TestOCRCached_ErrorsNotCached(t *testing.T) {
	var calls int
	key := fmt.Sprintf("unit-err-%d", time.Now().UnixNano())
	fn := func() (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("引擎冷启动中")
		}
		return "第二次成功", nil
	}
	if _, err := ocrCached(key, fn); err == nil {
		t.Fatal("首跑应失败")
	}
	v, err := ocrCached(key, fn)
	if err != nil {
		t.Fatalf("失败不应入缓存，重跑应成功: %v", err)
	}
	if v != "第二次成功" || calls != 2 {
		t.Fatalf("v=%q calls=%d, 失败结果不应被缓存", v, calls)
	}
}

// TestOCRCached_Singleflight 并发同键只跑一次 fn，两个调用方拿同一结果。
func TestOCRCached_Singleflight(t *testing.T) {
	key := fmt.Sprintf("unit-sf-%d", time.Now().UnixNano())
	started := make(chan struct{})
	release := make(chan struct{})
	var calls int32
	fn := func() (string, error) {
		atomic.AddInt32(&calls, 1)
		close(started) // 首个调用方已注册飞行
		<-release
		return "sf 结果", nil
	}

	var wg sync.WaitGroup
	results := make([]string, 2)
	wg.Add(2)
	go func() { defer wg.Done(); v, _ := ocrCached(key, fn); results[0] = v }()
	<-started
	go func() { defer wg.Done(); v, _ := ocrCached(key, fn); results[1] = v }()
	time.Sleep(50 * time.Millisecond) // 给第二个调用方入飞行的调度窗口
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("fn 被并发跑 %d 次，singleflight 应合并为 1", got)
	}
	for i, r := range results {
		if r != "sf 结果" {
			t.Fatalf("调用方 %d 拿到 %q", i, r)
		}
	}
}

// ── 集成：OCRImageText 按内容缓存 ────────────────────────────────

// TestOCRImageText_CachesByContent 同内容二次调用不重跑引擎（tesseract seam
// 计数验证）；内容变更→新指纹→重跑。内容带时间戳避免与既有测试缓存串味。
func TestOCRImageText_CachesByContent(t *testing.T) {
	t.Setenv("GAEA_OCR_ENGINE", "tesseract")
	t.Setenv("GAEA_OCR_DIR", filepath.Join(t.TempDir(), "no-such-ocr"))

	restoreTesseractVars(t)
	var calls int32
	tesseractLookPath = func(string) (string, error) { return `C:\fake\tesseract.exe`, nil }
	tesseractImage = func(_, _ string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "缓存集成结果", nil
	}

	img := filepath.Join(t.TempDir(), "img.png")
	content := fmt.Sprintf("unique-%d", time.Now().UnixNano())
	if err := os.WriteFile(img, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := OCRImageText(img); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := OCRImageText(img); err != nil {
		t.Fatalf("second (cached): %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("同内容引擎被调 %d 次，缓存命中不应重跑", got)
	}

	if err := os.WriteFile(img, []byte(content+"-v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OCRImageText(img); err != nil {
		t.Fatalf("after rewrite: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("内容变更后引擎应重跑，calls=%d", got)
	}
}
