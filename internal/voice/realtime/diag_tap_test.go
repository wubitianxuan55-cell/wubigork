// diag_tap_test.go — 播放面诊断落盘端点测试钉（2026-11-07 重叠线复开）。
package realtime

import (
	"bytes"
	"encoding/binary"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func postTap(t *testing.T, srv *Server, file string, rate string, truncate string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/diag/playback", bytes.NewReader(body))
	req.Header.Set("X-Gaea-Diag-File", file)
	req.Header.Set("X-Gaea-Diag-Rate", rate)
	if truncate != "" {
		req.Header.Set("X-Gaea-Diag-Truncate", truncate)
	}
	rec := httptest.NewRecorder()
	srv.handleDiagPlayback(rec, req)
	return rec
}

func TestDiagPlaybackTapAppendTruncateAndPrune(t *testing.T) {
	srv := &Server{}
	dir := t.TempDir()
	prevCfg := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = prevCfg })

	// 首块 truncate 开新文件 + 两块 append 同文件
	if rec := postTap(t, srv, "abc123", "48000", "1", []byte{1, 0, 2, 0}); rec.Code != 200 {
		t.Fatalf("truncate post: %d %s", rec.Code, rec.Body.String())
	}
	if rec := postTap(t, srv, "abc123", "48000", "", []byte{3, 0, 4, 0}); rec.Code != 200 {
		t.Fatalf("append post: %d %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "gaea", "diag", "voice-playback-tap-abc123.pcm"))
	if err != nil {
		t.Fatal(err)
	}
	want := []int16{1, 2, 3, 4}
	if len(raw) != 8 {
		t.Fatalf("bytes=%d", len(raw))
	}
	for i, v := range want {
		if got := binary.LittleEndian.Uint16(raw[i*2:]); got != uint16(v) {
			t.Fatalf("sample[%d]=%d want %d", i, got, v)
		}
	}

	// 守卫：坏 file id / 坏 rate / GET
	if rec := postTap(t, srv, "../evil", "48000", "", nil); rec.Code != 400 {
		t.Fatalf("path traversal guard: %d", rec.Code)
	}
	if rec := postTap(t, srv, "abc123", "not-a-rate", "", nil); rec.Code != 400 {
		t.Fatalf("bad rate guard: %d", rec.Code)
	}
	rec := httptest.NewRecorder()
	srv.handleDiagPlayback(rec, httptest.NewRequest(http.MethodGet, "/v1/diag/playback", nil))
	if rec.Code != 405 {
		t.Fatalf("GET guard: %d", rec.Code)
	}
	// OPTIONS 预检放行+CORS 头（浏览器跨源自定义头必预检，2026-11-07 二层根修钉）
	rec = httptest.NewRecorder()
	srv.handleDiagPlayback(rec, httptest.NewRequest(http.MethodOptions, "/v1/diag/playback", nil))
	if rec.Code != 200 {
		t.Fatalf("OPTIONS preflight: %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("预检缺 CORS 头")
	}

	// 淘汰：预置 7 个旧 tap 文件 + 1 个非 tap 文件 → 保留最近 5 个 tap，非 tap 不动
	for i := 0; i < 7; i++ {
		name := filepath.Join(dir, "gaea", "diag", "voice-playback-tap-old"+string(rune('a'+i))+".pcm")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(name, time.Now().AddDate(0, 0, -i-1), time.Now().AddDate(0, 0, -i-1)); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(dir, "gaea", "diag", "unrelated.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rec := postTap(t, srv, "fresh", "16000", "1", []byte{9, 0}); rec.Code != 200 {
		t.Fatalf("prune trigger post: %d", rec.Code)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "gaea", "diag", "voice-playback-tap-*.pcm"))
	// abc123 + fresh 本轮 2 个 + 保留窗内 3 个旧文件（上限 5）
	if len(matches) != 5 {
		t.Fatalf("tap files=%d want 5: %v", len(matches), matches)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("非 tap 文件被误删: %v", err)
	}
}

// TestDiagPlaybackTapOddBytes 奇数字节守卫：尾字节丢弃不崩。
func TestDiagPlaybackTapOddBytes(t *testing.T) {
	srv := &Server{}
	dir := t.TempDir()
	prevCfg := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = prevCfg })

	if rec := postTap(t, srv, "odd", "16000", "1", []byte{1, 0, 2}); rec.Code != 200 {
		t.Fatalf("odd post: %d %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "gaea", "diag", "voice-playback-tap-odd.pcm"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2 {
		t.Fatalf("bytes=%d want 2（尾字节截断）", len(raw))
	}
}

// TestDiagPlaybackTapMaxBody 超长 body 守卫：4MiB 限长。
func TestDiagPlaybackTapMaxBody(t *testing.T) {
	srv := &Server{}
	dir := t.TempDir()
	prevCfg := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = prevCfg })

	if rec := postTap(t, srv, "big", "16000", "1", make([]byte, 5<<20)); rec.Code == 200 {
		t.Fatalf("5MiB body 应被限长拒绝")
	}
	// fs 接口兜底引用（Windows 时区面无关，防 import fs 漂移）
	_ = fs.ModeDir
}
