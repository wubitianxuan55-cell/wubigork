package characterlib

// T7-2 可见性收口：剧照文件 ID 清洗/哈希——防路径穿越与不安全字符替换测试。

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tinyPNG 1x1 红色 PNG data URL（合法 base64）。
const tinyPNG = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// TestPortraitFileBase 清洗规则：安全 ID 原样；穿越/分隔符/空 → 哈希；
// 非安全字符 → 下划线。
func TestPortraitFileBase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"c1", "c1"},
		{"char-123_abc", "char-123_abc"},
		{"../../etc/passwd", "3754d6cb3a38e118"}, // 穿越 → SHA-256 短哈希
		{"a/b", "c14cddc033f64b9d"},              // 分隔符 → 哈希
		{"a\\b", "c62016d0f8ee3333"},             // 反斜杠 → 哈希
		{"..", "5ec1f7e700f37c3d"},               // 纯 ".." → 哈希
		{"", "e3b0c44298fc1c14"},                 // 空 ID → 哈希（SHA-256("") 前缀）
		{"c:1", "c_1"},                           // 冒号 → 下划线
		{"a b", "a_b"},                           // 空格 → 下划线
		{"...", "ab5df625bc76dbd4"},              // 全点号清洗后为空 → 哈希
	}
	for _, c := range cases {
		got := portraitFileBase(c.in)
		if got != c.want {
			t.Errorf("portraitFileBase(%q) = %q, want %q", c.in, got, c.want)
		}
		if strings.ContainsAny(got, "/\\") || strings.Contains(got, "..") {
			t.Errorf("portraitFileBase(%q) 输出仍含穿越特征: %q", c.in, got)
		}
	}
}

// TestPortraitIDHash_Deterministic 哈希确定性 + 长度 + 无分隔符。
func TestPortraitIDHash_Deterministic(t *testing.T) {
	h1 := portraitIDHash("../../evil")
	h2 := portraitIDHash("../../evil")
	if h1 != h2 || len(h1) != 16 {
		t.Errorf("哈希应确定且 16 位: %q vs %q", h1, h2)
	}
	if strings.ContainsAny(h1, "/\\") {
		t.Errorf("哈希不应含分隔符: %q", h1)
	}
	if portraitIDHash("a") == portraitIDHash("b") {
		t.Error("不同 ID 哈希不应相同")
	}
}

// TestSavePortraitFile_RejectsTraversal 穿越 ID（../evil）落盘后路径仍在
// portraits 目录内（哈希文件名），不逃逸到上级目录。
func TestSavePortraitFile_RejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	got := savePortraitFile(dir, "../../evil", tinyPNG)
	portraits := portraitFileDir(dir)
	if !strings.HasPrefix(got, portraits+string(filepath.Separator)) {
		t.Fatalf("剧照路径逃逸出 portraits 目录: %q", got)
	}
	if strings.Contains(got, "..") {
		t.Errorf("路径不应含 ..: %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("剧照文件不存在: %v", err)
	}
	// 目录外（dataDir 直接子级）不应产生文件。
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("dataDir 根下不应有剧照文件: %s", e.Name())
		}
	}
}

// TestSavePortraitFile_SafeIDKeepsName 安全 ID 保持原名落盘（c1 → c1.png）。
func TestSavePortraitFile_SafeIDKeepsName(t *testing.T) {
	dir := t.TempDir()
	got := savePortraitFile(dir, "c1", tinyPNG)
	want := filepath.Join(portraitFileDir(dir), "c1.png")
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("剧照文件不存在: %v", err)
	}
}

// TestSavePortraitFile_NonDataURLPassthrough 非 data URL 原样返回（不落盘）。
func TestSavePortraitFile_NonDataURLPassthrough(t *testing.T) {
	dir := t.TempDir()
	if got := savePortraitFile(dir, "c1", "http://example.com/p.png"); got != "http://example.com/p.png" {
		t.Errorf("远程 URL 应原样返回: %q", got)
	}
}

// TestSaveRemotePortrait_Downloads 远程剧照下载到本地 portraits 目录。
func TestSaveRemotePortrait_Downloads(t *testing.T) {
	tinyPNG := []byte("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(tinyPNG)
	}))
	defer srv.Close()

	dir := t.TempDir()
	got := saveRemotePortrait(dir, "c1", srv.URL+"/p.png")
	if got == srv.URL+"/p.png" {
		t.Fatalf("远程 URL 应被本地化，仍返回原 URL: %q", got)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("本地文件不存在: %v", err)
	}
	if string(data) != string(tinyPNG) {
		t.Fatalf("文件内容不一致")
	}
	if !strings.HasPrefix(got, portraitFileDir(dir)+string(filepath.Separator)) {
		t.Fatalf("落盘路径不在 portraits 目录: %q", got)
	}
}

// TestSaveRemotePortrait_FailureKeepsURL 下载失败保留原 URL（保存不阻塞）。
func TestSaveRemotePortrait_FailureKeepsURL(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if got := saveRemotePortrait(t.TempDir(), "c1", srv.URL+"/missing.png"); got != srv.URL+"/missing.png" {
		t.Fatalf("404 时应保留原 URL, got %q", got)
	}
}

func TestExtFromContentType(t *testing.T) {
	cases := map[string]string{
		"image/png":      ".png",
		"image/jpeg":     ".jpg",
		"image/webp;...": ".webp",
		"text/html":      "",
	}
	for ct, want := range cases {
		if got := extFromContentType(ct); got != want {
			t.Errorf("extFromContentType(%q) = %q, want %q", ct, got, want)
		}
	}
}
