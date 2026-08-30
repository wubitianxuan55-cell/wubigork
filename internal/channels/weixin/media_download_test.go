package weixin

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

// pngMagic 最小 PNG 头（魔数嗅探只看签名，不解码）。
var pngMagic = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

// allowLoopback 测试期放行回环（httptest 监听 127.0.0.1），返回恢复函数。
func allowLoopback(t *testing.T) func() {
	t.Helper()
	old := mediaAllowLoopback
	mediaAllowLoopback = true
	return func() { mediaAllowLoopback = old }
}

// zeroReader 按需生成零字节流（不占 21MiB 内存）。
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// TestDownloadImage_OK_PNG 正常 PNG：落盘成功、cleanup 幂等删除。
func TestDownloadImage_OK_PNG(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngMagic)
	}))
	defer srv.Close()

	path, cleanup, err := DownloadImage(srv.URL + "/img.png?sig=ab&x=1")
	if err != nil {
		t.Fatalf("DownloadImage: %v", err)
	}
	if path == "" {
		t.Fatal("应返回本地临时路径")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("临时文件应存在: %v", err)
	}
	cleanup()
	cleanup() // 幂等：二次清理不 panic
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("cleanup 后临时文件应被删除")
	}
}

// TestDownloadImage_HTTP404 非 200 显式报错。
func TestDownloadImage_HTTP404(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if _, _, err := DownloadImage(srv.URL + "/missing.png"); err == nil {
		t.Fatal("404 应显式报错")
	}
}

// TestDownloadImage_TooLarge_CLPrecheck Content-Length 预检拒绝 21MiB。
func TestDownloadImage_TooLarge_CLPrecheck(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	const size = 21 << 20
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(size))
		io.Copy(w, io.MultiReader(bytes.NewReader(pngMagic), io.LimitReader(zeroReader{}, size)))
	}))
	defer srv.Close()

	_, cleanup, err := DownloadImage(srv.URL + "/big.png")
	if err == nil {
		cleanup()
		t.Fatal("21MiB 应被拒绝")
	}
	if !strings.Contains(err.Error(), "尺寸上限") {
		t.Fatalf("应报尺寸上限错误: %v", err)
	}
}

// TestDownloadImage_TooLarge_LimitReader 无 Content-Length（chunked 流）时
// 由 io.LimitReader 兜底拒绝。
func TestDownloadImage_TooLarge_LimitReader(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	const size = 21 << 20
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// 不设 Content-Length → chunked 编码，预检拿不到长度
		io.Copy(w, io.MultiReader(bytes.NewReader(pngMagic), io.LimitReader(zeroReader{}, size)))
	}))
	defer srv.Close()

	_, cleanup, err := DownloadImage(srv.URL + "/big.png")
	if err == nil {
		cleanup()
		t.Fatal("chunked 21MiB 应被拒绝")
	}
	if !strings.Contains(err.Error(), "尺寸上限") {
		t.Fatalf("应报尺寸上限错误: %v", err)
	}
}

// TestDownloadImage_PrivateNetRefused dial-time SSRF：回环/私网/链路本地/
// CGNAT 一律拒绝（生产 mediaAllowLoopback=false；无需真实监听——防线在
// 拨号前拦截）。文件不落盘、cleanup 为 nil。
func TestDownloadImage_PrivateNetRefused(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:1/x.png",      // 回环
		"http://10.1.2.3/x.png",         // RFC1918
		"http://192.168.1.1/x.png",      // RFC1918
		"http://169.254.169.254/latest", // 云 metadata
		"http://100.100.100.200/x.png",  // CGNAT（阿里云 metadata）
		"http://[::1]/x.png",            // IPv6 回环
	} {
		path, cleanup, err := DownloadImage(u)
		if err == nil {
			if cleanup != nil {
				cleanup()
			}
			t.Fatalf("%s 应被拒绝", u)
		}
		if path != "" || cleanup != nil {
			t.Fatalf("%s 拒绝时不应产生临时文件/清理函数", u)
		}
	}
}

// TestDownloadImage_TextPlainRejected text/plain 拒绝（Content-Type + 魔数双闸）。
func TestDownloadImage_TextPlainRejected(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello, not an image"))
	}))
	defer srv.Close()

	if _, _, err := DownloadImage(srv.URL + "/fake.png"); err == nil {
		t.Fatal("text/plain 应被拒绝")
	}
}

// TestDownloadImage_BadMagicRejected 伪报 Content-Type=image/png 但内容
// 非图片：魔数终审拒绝。
func TestDownloadImage_BadMagicRejected(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("definitely not an image"))
	}))
	defer srv.Close()

	_, cleanup, err := DownloadImage(srv.URL + "/evil.png")
	if err == nil {
		cleanup()
		t.Fatal("魔数不在白名单应被拒绝")
	}
	if !strings.Contains(err.Error(), "魔数") {
		t.Fatalf("应报魔数白名单错误: %v", err)
	}
}

// TestDownloadImage_BadURL 非绝对 http(s) URL 拒绝。
func TestDownloadImage_BadURL(t *testing.T) {
	for _, u := range []string{"", "ftp://x/a.png", "/local/path.png", "https://"} {
		if _, _, err := DownloadImage(u); err == nil {
			t.Fatalf("非法 URL %q 应报错", u)
		}
	}
}

// TestBlockedMediaIP 防线单元表：回环受 mediaAllowLoopback 开关控制，其余恒拒。
func TestBlockedMediaIP(t *testing.T) {
	cases := []struct {
		ip  string
		bad bool
	}{
		{"127.0.0.1", true},
		{"10.1.2.3", true},
		{"172.16.0.9", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"100.100.100.200", true},
		{"0.0.0.0", true},
		{"::1", true},
		{"fc00::1", true},
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"114.114.114.114", false},
	}
	for _, tc := range cases {
		if got := blockedMediaIP(net.ParseIP(tc.ip)); got != tc.bad {
			t.Errorf("blockedMediaIP(%s) = %v, want %v", tc.ip, got, tc.bad)
		}
	}

	// 测试放行回环后：127.0.0.1 通过、私网仍拒绝
	restore := allowLoopback(t)
	defer restore()
	if blockedMediaIP(net.ParseIP("127.0.0.1")) {
		t.Error("放行回环后 127.0.0.1 不应被拒")
	}
	if !blockedMediaIP(net.ParseIP("10.0.0.1")) {
		t.Error("放行回环后私网仍应被拒")
	}
}
