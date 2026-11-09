package docmd

// OCR 结果进程内缓存：同一图片/扫描件 PDF 的 OCR 是分钟级慢操作（uocr2 GPU
// 6s/页、CPU 20s/页起），而消费方（@引用重发、format_convert 跟进、summarize、
// 预览、成本导入）常对同一文件反复转换。缓存语义：
//   - 键 = 内容 sha256 前缀 + 用途域 + 引擎序（GAEA_OCR_ENGINE 切换不串味）；
//     用内容哈希不用 mtime+size——Windows 同纳秒互覆有 mtime 失真前科，且转换
//     路径本就读全文件，哈希近免费；
//   - 只缓存成功结果——失败不缓存，保住「冷启动超预算放弃→引擎后台拉起→
//     下轮命中」的阶梯（2026-11-09 办公识图预算刀的语义依赖）；
//   - singleflight 合并并发同键调用：第二个调用方等第一个的结果，不重跑引擎
//     （预算放弃的调用方后台继续跑完，后来者直接等飞行结果）；
//   - LRU 双上限（条数 + 总字节），逐出最旧。
//
// 已知语义：uocr2 后端腿（llama/python auto 择优）在首次成功时被钉住；磁盘
// 持久化不做（挂拍板池）。单测必须经 seam 或直接打 newOCRCache/ocrCached，
// 不得真调引擎（测试语义管线地雷）。

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	ocrCacheMaxEntries = 32
	ocrCacheMaxBytes   = 64 << 20 // 正文总量上限 64MB
)

// engineOrderKey 引擎序键段（GAEA_OCR_ENGINE 解析结果；解析失败也给出稳定段，
// 让后续 fn 内部走 fail-closed 报错路径）。
func engineOrderKey() string {
	kinds, err := ocrEngineOrder()
	if err != nil {
		return "invalid"
	}
	return strings.Join(kinds, ",")
}

// ocrFileFingerprint 文件内容指纹（sha256 前 16 字节 hex，防碰撞够用且省内存）。
func ocrFileFingerprint(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)[:16]), nil
}

// ocrCacheEntry 单条缓存。
type ocrCacheEntry struct {
	key  string
	text string
}

// ocrCache LRU（条数+字节双上限），front=最新。
type ocrCache struct {
	mu    sync.Mutex
	ll    *list.List
	items map[string]*list.Element
	bytes int
}

var defaultOCRCache = newOCRCache()

func newOCRCache() *ocrCache {
	return &ocrCache{ll: list.New(), items: map[string]*list.Element{}}
}

func (c *ocrCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*ocrCacheEntry).text, true
	}
	return "", false
}

func (c *ocrCache) put(key, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		it := el.Value.(*ocrCacheEntry)
		c.bytes += len(text) - len(it.text)
		it.text = text
		c.ll.MoveToFront(el)
		c.evictLocked()
		return
	}
	c.items[key] = c.ll.PushFront(&ocrCacheEntry{key: key, text: text})
	c.bytes += len(text)
	c.evictLocked()
}

func (c *ocrCache) evictLocked() {
	for (len(c.items) > ocrCacheMaxEntries || c.bytes > ocrCacheMaxBytes) && c.ll.Len() > 0 {
		oldest := c.ll.Back()
		it := oldest.Value.(*ocrCacheEntry)
		c.bytes -= len(it.text)
		c.ll.Remove(oldest)
		delete(c.items, it.key)
	}
}

// ocrFlight 同键在途转换（singleflight）。
type ocrFlight struct {
	done chan struct{}
	text string
	err  error
}

var (
	ocrFlightsMu sync.Mutex
	ocrFlights   = map[string]*ocrFlight{}
)

// ocrCached 查缓存→命中直接返回；未命中合并同键在途调用，只有发起者真跑
// fn，其余等结果。fn 成功才入缓存（失败不缓存）。
func ocrCached(key string, fn func() (string, error)) (string, error) {
	if text, ok := defaultOCRCache.get(key); ok {
		return text, nil
	}
	ocrFlightsMu.Lock()
	if f, ok := ocrFlights[key]; ok {
		ocrFlightsMu.Unlock()
		<-f.done
		return f.text, f.err
	}
	f := &ocrFlight{done: make(chan struct{})}
	ocrFlights[key] = f
	ocrFlightsMu.Unlock()

	f.text, f.err = fn()
	if f.err == nil {
		defaultOCRCache.put(key, f.text)
	}
	close(f.done)
	ocrFlightsMu.Lock()
	delete(ocrFlights, key)
	ocrFlightsMu.Unlock()
	return f.text, f.err
}
