package app

// wx_image_cache.go — 微信入站图片缓存（对话式改图第一环，v4.9）。
//
// 微信入站图片经 clawbot 识别管线解密落盘的是临时文件（识别完即删），而改图
// 意图需要在「发图 → 发指令」两条消息之间记住那张图：这里以 assistantID 为
// 键做缓存自持——Set 时把源文件复制进 whisperDataRoot/wx_edit_cache/，不依赖
// clawbot 临时文件的生命周期。同助手只留最新一张；TTL 10 分钟；单文件上限
// 10MiB（editImageFromCard 契约的 data URL 输入口径）。
//
// 纪律：缓存纯属旁路（OCR 识别主流程零依赖），Set/Get 任何失败只返回错误或
// 记日志，绝不影响入站消息处理与聊天管道。

import (
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 缓存参数：TTL 与单文件复制上限。
const (
	wxEditCacheTTL     = 10 * time.Minute
	wxEditCacheMaxFile = 10 << 20 // 10 MiB
)

// wxImageEntry 缓存条目：自持副本路径 + mime + 存入时间。
type wxImageEntry struct {
	Path   string
	MIME   string
	Stored time.Time
}

// wxImageCache 线程安全的「assistantID → 最新入站图片」缓存。now 为时钟
// seam（测试注入）；dir 为自持副本目录；maxFile 单文件复制上限。
type wxImageCache struct {
	mu      sync.Mutex
	entries map[string]wxImageEntry
	dir     string
	maxFile int64
	now     func() time.Time
}

// newWxImageCache 构造缓存（副本根 = dataRoot/wx_edit_cache/）。
func newWxImageCache(dataRoot string) *wxImageCache {
	return &wxImageCache{
		entries: make(map[string]wxImageEntry),
		dir:     filepath.Join(dataRoot, "wx_edit_cache"),
		maxFile: wxEditCacheMaxFile,
		now:     time.Now,
	}
}

// 进程级单例：whisperState（注入 OnInboundImage 钩子）与 App（execEditImage
// 执行层）共用同一份缓存——两处都拿不到 struct 字段（app.go 归并行线所有），
// 以惰性单例按 whisperDataRoot 定根。
var (
	wxEditCacheMu   sync.Mutex
	wxEditCacheInst *wxImageCache
)

// wxEditImageCache 返回进程级改图缓存单例（首次调用以 dataRoot 定根；进程内
// dataRoot 恒定，为空退系统临时目录兜底——测试直构 newWxImageCache 不走此路）。
func wxEditImageCache(dataRoot string) *wxImageCache {
	wxEditCacheMu.Lock()
	defer wxEditCacheMu.Unlock()
	if wxEditCacheInst == nil {
		if dataRoot == "" {
			dataRoot = os.TempDir()
		}
		wxEditCacheInst = newWxImageCache(dataRoot)
	}
	return wxEditCacheInst
}

// wxEditCache App 侧取缓存（whisperState 未装配返回 nil——执行层按未命中处理）。
func (a *App) wxEditCache() *wxImageCache {
	if a == nil || a.whisperState == nil {
		return nil
	}
	return wxEditImageCache(a.whisperState.whisperDataRoot)
}

// ─── 读写 ─────────────────────────────────────────────────────

// Set 记录助手最新收到的入站图片：把源文件复制进 wx_edit_cache/（缓存自持，
// 不依赖 clawbot 临时文件生命周期），同助手只留最新一张（旧副本即删），入口
// 统一先清理过期条目。源文件超上限（10MiB）报错不入缓存。
func (c *wxImageCache) Set(assistantID, srcPath string) (wxImageEntry, error) {
	if assistantID == "" {
		return wxImageEntry{}, fmt.Errorf("assistantID 为空，不入改图缓存")
	}
	st, err := os.Stat(srcPath)
	if err != nil {
		return wxImageEntry{}, err
	}
	if st.Size() > c.maxFile {
		return wxImageEntry{}, fmt.Errorf("图片 %d 字节超过改图缓存上限 %d", st.Size(), c.maxFile)
	}
	mime, err := sniffImageMIME(srcPath)
	if err != nil {
		return wxImageEntry{}, err
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return wxImageEntry{}, err
	}
	dst := filepath.Join(c.dir, wxEditCacheFileName(assistantID, c.now(), srcPath))
	if err := copyFileBounded(srcPath, dst, c.maxFile); err != nil {
		return wxImageEntry{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	c.purgeExpiredLocked(now)
	old := c.entries[assistantID]
	c.entries[assistantID] = wxImageEntry{Path: dst, MIME: mime, Stored: now}
	if old.Path != "" && old.Path != dst {
		os.Remove(old.Path) // 同助手只留最新一张：旧副本即删
	}
	slog.Debug("[wx-edit-cache] 入站图片已缓存", "assistant", assistantID, "file", filepath.Base(dst))
	return c.entries[assistantID], nil
}

// Get 取助手当前缓存的图片。命中策略：刷新命中时间——正在连续改图的图片不
// 中途过期（「调成黑白」之后两分钟再「加上帽子」仍在窗口内）；TTL 语义因此
// 是「10 分钟不活跃」而非「10 分钟生死线」。
func (c *wxImageCache) Get(assistantID string) (path, mime string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, hit := c.entries[assistantID]
	if !hit {
		return "", "", false
	}
	if c.now().Sub(e.Stored) > wxEditCacheTTL {
		c.removeLocked(assistantID, e)
		return "", "", false
	}
	// 自持副本被外部清走（用户手动清理目录等）：按未命中处理并顺手清账。
	if st, err := os.Stat(e.Path); err != nil || st.Size() == 0 {
		c.removeLocked(assistantID, e)
		return "", "", false
	}
	e.Stored = c.now()
	c.entries[assistantID] = e
	return e.Path, e.MIME, true
}

// Delete 清除单个助手的缓存（停用该助手时调用；副本文件一并删除）。
func (c *wxImageCache) Delete(assistantID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[assistantID]; ok {
		c.removeLocked(assistantID, e)
	}
}

// PurgeAll 清空全部缓存并删除副本文件（删除助手/停机全量清理用）。
func (c *wxImageCache) PurgeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, e := range c.entries {
		c.removeLocked(id, e)
	}
}

// removeLocked 删条目 + 副本文件（持锁调用；副本已不在视为幂等成功）。
func (c *wxImageCache) removeLocked(assistantID string, e wxImageEntry) {
	delete(c.entries, assistantID)
	if e.Path != "" {
		os.Remove(e.Path)
	}
}

// purgeExpiredLocked 清理过期条目（Set 入口统一触发；持锁调用）。
func (c *wxImageCache) purgeExpiredLocked(now time.Time) {
	for id, e := range c.entries {
		if now.Sub(e.Stored) > wxEditCacheTTL {
			c.removeLocked(id, e)
		}
	}
}

// ─── 工具 ─────────────────────────────────────────────────────

// wxEditCacheExts 自持副本扩展名白名单（微信侧图片常见格式；其余一律 .img）。
var wxEditCacheExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true,
}

// wxEditCacheFileName 自持副本文件名：助手 ID 哈希（防路径注入）+ 纳秒序号
// （同助手多次 Set 不冲突）+ 白名单扩展名。
func wxEditCacheFileName(assistantID string, at time.Time, srcPath string) string {
	h := fnv.New32a()
	h.Write([]byte(assistantID))
	ext := strings.ToLower(filepath.Ext(srcPath))
	if !wxEditCacheExts[ext] {
		ext = ".img"
	}
	return fmt.Sprintf("wx-edit-%08x-%d%s", h.Sum32(), at.UnixNano(), ext)
}

// copyFileBounded 有界复制：Stat 预检之外复制期间也限流（源文件被增长/置换
// 同样逃不掉），超限即停并删除半成品。
func copyFileBounded(src, dst string, max int64) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		if err != nil {
			os.Remove(dst) // 半成品不留盘
		}
	}()
	n, err := io.Copy(out, io.LimitReader(in, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return fmt.Errorf("图片复制时超过上限 %d 字节", max)
	}
	return out.Close()
}

// sniffImageMIME 魔数探测（http.DetectContentType，512 字节头）；探测不出按
// 扩展名兜底；仍不识别回 application/octet-stream。空文件报错不入缓存。
func sniffImageMIME(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("空文件不入改图缓存：%s", filepath.Base(path))
	}
	if ct := http.DetectContentType(head[:n]); ct != "application/octet-stream" {
		return ct, nil
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	case ".gif":
		return "image/gif", nil
	case ".bmp":
		return "image/bmp", nil
	}
	return "application/octet-stream", nil
}
