package app

// wx_image_cache_test.go — 微信入站图片缓存（对话式改图 v4.9）：TTL 过期、
// 复制自持、同助手只留最新、10MiB 上限、Get 刷新命中时间、Delete/PurgeAll。
// 时钟经 wxImageCache.now seam 注入（表驱动拨针，不 sleep）。

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// wxEditTestPNG 最小 PNG 头（魔数嗅探只看签名，不解码）。
var wxEditTestPNG = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

// newTestWxImageCache 独立实例 + 可拨动时钟；返回 advance 供测试推进时间。
func newTestWxImageCache(t *testing.T) (*wxImageCache, func(time.Duration)) {
	t.Helper()
	now := time.Now()
	c := newWxImageCache(t.TempDir())
	c.now = func() time.Time { return now }
	return c, func(d time.Duration) { now = now.Add(d) }
}

func writeWxEditSrc(t *testing.T, data []byte, ext string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "inbound"+ext)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("写源图片: %v", err)
	}
	return p
}

// Set→Get 往返：命中返回路径+mime；副本内容与源一致。
func TestWxImageCache_SetGetRoundtrip(t *testing.T) {
	c, _ := newTestWxImageCache(t)
	src := writeWxEditSrc(t, wxEditTestPNG, ".png")

	e, err := c.Set("ast-rt", src)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if e.MIME != "image/png" {
		t.Errorf("mime 魔数探测 = %q, want image/png", e.MIME)
	}
	if !strings.HasPrefix(filepath.Base(e.Path), "wx-edit-") {
		t.Errorf("副本应落在 wx_edit_cache 管理命名下: %q", e.Path)
	}
	path, mime, ok := c.Get("ast-rt")
	if !ok || path != e.Path || mime != "image/png" {
		t.Fatalf("Get = (%q,%q,%v), want 命中且与 Set 一致", path, mime, ok)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读副本: %v", err)
	}
	if !bytes.Equal(got, wxEditTestPNG) {
		t.Error("副本内容应与源一致")
	}
	// 未知助手未命中。
	if _, _, ok := c.Get("ast-other"); ok {
		t.Error("未 Set 的助手不应命中")
	}
}

// 复制自持：源文件（clawbot 临时文件）删除后缓存仍可用。
func TestWxImageCache_CopySelfContained(t *testing.T) {
	c, _ := newTestWxImageCache(t)
	src := writeWxEditSrc(t, wxEditTestPNG, ".png")
	if _, err := c.Set("ast-self", src); err != nil {
		t.Fatalf("Set: %v", err)
	}
	os.Remove(src) // 模拟识别管线 cleanup 临时文件

	path, _, ok := c.Get("ast-self")
	if !ok {
		t.Fatal("源删除后缓存应仍命中（自持副本）")
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, wxEditTestPNG) {
		t.Fatalf("自持副本应完好: err=%v", err)
	}
}

// TTL 过期：10 分钟内命中、超时未命中且副本清理；Set 入口顺带清过期条目。
// 注意 Get 刷新命中时间（见 GetRefreshesHitTime）——过期用例不得先 Get。
func TestWxImageCache_TTLExpiry(t *testing.T) {
	c, advance := newTestWxImageCache(t)
	if _, err := c.Set("ast-ttl", writeWxEditSrc(t, wxEditTestPNG, ".png")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	advance(9 * time.Minute)
	if _, _, ok := c.Get("ast-ttl"); !ok {
		t.Fatal("9 分钟应仍在 TTL 窗口内")
	}

	// 过期 + 副本清理（独立时钟：Set 后直落 11 分钟，中间无 Get）。
	c2, advance2 := newTestWxImageCache(t)
	eA, err := c2.Set("ast-a", writeWxEditSrc(t, wxEditTestPNG, ".png"))
	if err != nil {
		t.Fatalf("Set a: %v", err)
	}
	advance2(11 * time.Minute)
	if _, _, ok := c2.Get("ast-a"); ok {
		t.Fatal("超 10 分钟应过期未命中")
	}
	if _, err := os.Stat(eA.Path); !os.IsNotExist(err) {
		t.Error("过期条目的副本文件应被清理")
	}

	// Set 入口统一清理过期：新 Set 后过期条目不复活。
	if _, err := c2.Set("ast-b", writeWxEditSrc(t, wxEditTestPNG, ".png")); err != nil {
		t.Fatalf("Set b: %v", err)
	}
	if _, _, ok := c2.Get("ast-a"); ok {
		t.Error("Set 入口应清掉 ast-a 的过期条目")
	}
}

// Get 刷新命中时间（命中策略决策）：连续改图不中途过期；闲置超窗才失效。
func TestWxImageCache_GetRefreshesHitTime(t *testing.T) {
	c, advance := newTestWxImageCache(t)
	if _, err := c.Set("ast-fresh", writeWxEditSrc(t, wxEditTestPNG, ".png")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	advance(9 * time.Minute)
	if _, _, ok := c.Get("ast-fresh"); !ok {
		t.Fatal("第 9 分钟首次命中应成功")
	}
	advance(9 * time.Minute) // 距存入 18 分钟，但距上次命中仅 9 分钟
	if _, _, ok := c.Get("ast-fresh"); !ok {
		t.Fatal("Get 应刷新命中时间：连续使用不过期")
	}
	advance(11 * time.Minute) // 闲置超窗
	if _, _, ok := c.Get("ast-fresh"); ok {
		t.Fatal("闲置超 10 分钟应过期")
	}
}

// 同助手只留最新一张：旧副本即删，其他助手互不影响。
func TestWxImageCache_KeepLatestOnly(t *testing.T) {
	c, advance := newTestWxImageCache(t)
	e1, err := c.Set("ast-keep", writeWxEditSrc(t, wxEditTestPNG, ".png"))
	if err != nil {
		t.Fatalf("Set v1: %v", err)
	}
	v2 := append([]byte(nil), wxEditTestPNG...)
	v2[len(v2)-1] = 0xAB
	advance(time.Minute) // 推进时钟：副本文件名含纳秒序，两次 Set 不撞名
	e2, err := c.Set("ast-keep", writeWxEditSrc(t, v2, ".png"))
	if err != nil {
		t.Fatalf("Set v2: %v", err)
	}
	if e2.Path == e1.Path {
		t.Fatal("两次 Set 应产生不同副本文件")
	}
	if _, err := os.Stat(e1.Path); !os.IsNotExist(err) {
		t.Error("旧副本文件应即删（同助手只留最新一张）")
	}
	path, _, ok := c.Get("ast-keep")
	if !ok || path != e2.Path {
		t.Fatalf("Get 应返回最新副本: got (%q,%v)", path, ok)
	}
	// 另一助手不受波及。
	if _, err := c.Set("ast-keep2", writeWxEditSrc(t, wxEditTestPNG, ".jpg")); err != nil {
		t.Fatalf("Set other: %v", err)
	}
	if _, _, ok := c.Get("ast-keep2"); !ok {
		t.Error("其他助手的缓存不应被波及")
	}
}

// 10MiB 上限：超限 Set 报错不入缓存、不留半成品。
func TestWxImageCache_MaxFileSize(t *testing.T) {
	c, _ := newTestWxImageCache(t)
	p := filepath.Join(t.TempDir(), "huge.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	// 稀疏文件：定位到上限外写 2 字节 → 尺寸 = 上限+2，不真占 10MiB 内存/时间。
	if _, err := f.Seek(int64(wxEditCacheMaxFile), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{0, 0}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := c.Set("ast-big", p); err == nil {
		t.Fatal("超上限 Set 应报错")
	}
	if _, _, ok := c.Get("ast-big"); ok {
		t.Error("超上限不应入缓存")
	}
	// 目录可能尚未创建（Set 在 MkdirAll 前即拒绝）；存在则不得留半成品。
	if entries, err := os.ReadDir(c.dir); err == nil && len(entries) != 0 {
		t.Errorf("半成品不应留盘: entries=%d", len(entries))
	}
}

// Delete / PurgeAll：单助手精确清理与全量清理，副本文件一并删除。
func TestWxImageCache_DeleteAndPurgeAll(t *testing.T) {
	c, _ := newTestWxImageCache(t)
	eA, err := c.Set("ast-del", writeWxEditSrc(t, wxEditTestPNG, ".png"))
	if err != nil {
		t.Fatalf("Set a: %v", err)
	}
	eB, err := c.Set("ast-keep3", writeWxEditSrc(t, wxEditTestPNG, ".png"))
	if err != nil {
		t.Fatalf("Set b: %v", err)
	}

	c.Delete("ast-del")
	if _, _, ok := c.Get("ast-del"); ok {
		t.Error("Delete 后不应命中")
	}
	if _, err := os.Stat(eA.Path); !os.IsNotExist(err) {
		t.Error("Delete 应删副本文件")
	}
	if _, _, ok := c.Get("ast-keep3"); !ok {
		t.Error("Delete 单助手不应波及其他助手")
	}

	c.PurgeAll()
	if _, _, ok := c.Get("ast-keep3"); ok {
		t.Error("PurgeAll 后不应命中")
	}
	if _, err := os.Stat(eB.Path); !os.IsNotExist(err) {
		t.Error("PurgeAll 应删全部副本文件")
	}
}

// 文件名卫生：助手 ID 经哈希进文件名，特殊字符不产生路径注入。
func TestWxImageCache_FileNameSanitized(t *testing.T) {
	name := wxEditCacheFileName(`../evil/..`, time.Unix(0, 123), "x.png")
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		t.Errorf("文件名不应含路径成分: %q", name)
	}
	if !strings.HasSuffix(name, ".png") {
		t.Errorf("白名单扩展名应保留: %q", name)
	}
	if got := wxEditCacheFileName("a", time.Unix(0, 1), "x.exe"); !strings.HasSuffix(got, ".img") {
		t.Errorf("非白名单扩展名应落 .img: %q", got)
	}
}
