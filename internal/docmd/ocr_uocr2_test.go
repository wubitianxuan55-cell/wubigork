package docmd

// ocr_uocr2_test.go — uocr2 引擎（Unlimited-OCR-2）单测：结构标记清洗、
// /ocr 客户端解码与截断 fail-closed、服务拉起超时杀进程树/成功保留存活。
// 全部为纯单测，不依赖真实 python 服务（拉起用 ocr_test.go 的 helper 假进程）。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUocr2CleanText(t *testing.T) {
	raw := "<PAGE><BLOCK>header[42, 16, 143, 27]<FORK>2026/9/20 21:35\n" +
		"<BLOCK>header[490, 16, 654, 29]<FORK>福建省环境监测行业指导价\n" +
		"<BLOCK>table[133, 36, 868, 874]<FORK><table><tr><td>序号</td></tr></table>\n" +
		"<BLOCK>footer[42, 971, 958, 982]<FORK>https://mp.weixin.qq.com/s?src=11 2/15"
	want := "2026/9/20 21:35\n福建省环境监测行业指导价\n<table><tr><td>序号</td></tr></table>\n" +
		"https://mp.weixin.qq.com/s?src=11 2/15"
	if got := uocr2CleanText(raw); got != want {
		t.Errorf("uocr2CleanText = %q, want %q", got, want)
	}
	if got := uocr2CleanText("  纯文本无标记  "); got != "纯文本无标记" {
		t.Errorf("无标记输入应原样去空白: %q", got)
	}
}

// fakeUocr2Server 起 httptest 服务模拟 server.py：/health 探活 + /ocr 返回注入文本。
func fakeUocr2Server(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ocr", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// mustTempPNG 写一个内容任意的临时「PNG」（客户端只读字节转发，服务端才解码）。
func mustTempPNG(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "page.png")
	if err := os.WriteFile(p, []byte("fake-png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUocr2PageOCRDecodesAndCleans(t *testing.T) {
	raw := "<PAGE><BLOCK>table[1, 2, 3, 4]<FORK>水温\n<BLOCK>header[5, 6, 7, 8]<FORK>pH 值"
	srv := fakeUocr2Server(t, http.StatusOK, mustJSON(t, map[string]string{"text": raw}))
	got, err := uocr2PageOCR(srv.URL, mustTempPNG(t))
	if err != nil {
		t.Fatalf("uocr2PageOCR: %v", err)
	}
	if got != "水温\npH 值" {
		t.Errorf("text = %q, want 水温\\npH 值", got)
	}
}

func TestUocr2PageOCRTruncationFailsClosed(t *testing.T) {
	srv := fakeUocr2Server(t, http.StatusUnprocessableEntity,
		mustJSON(t, map[string]string{"error": "输出被 max_tokens 截断（finish_reason=length）"}))
	_, err := uocr2PageOCR(srv.URL, mustTempPNG(t))
	if err == nil {
		t.Fatal("422 截断应报错（fail-closed，不把残缺文本当完整结果）")
	}
	if !strings.Contains(err.Error(), "截断") {
		t.Errorf("错误应说明截断: %v", err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestStartUocr2ServerTimeoutKillsProcessTree 镜像 ovis 同名测试：永不健康探针 +
// 快速假进程 → 超时返回 false 且进程被杀（无孤儿残留）。
func TestStartUocr2ServerTimeoutKillsProcessTree(t *testing.T) {
	restoreUocr2Vars(t)

	var started *exec.Cmd
	uocr2BuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer() // 同一 helper 假进程（长时间阻塞）
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return false }
	uocr2StartWait = 2 * time.Second

	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if startUocr2Server(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2Server 超时后应返回 false")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	waitProcessDead(t, started.Process.Pid)
}

// TestStartUocr2ServerSuccessKeepsProcessAlive 立即健康探针 → 返回 true 且进程保留存活。
func TestStartUocr2ServerSuccessKeepsProcessAlive(t *testing.T) {
	restoreUocr2Vars(t)

	var started *exec.Cmd
	uocr2BuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer()
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return true }
	uocr2StartWait = 2 * time.Second

	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if !startUocr2Server(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2Server 就绪后应返回 true")
	}
	if !processAlive(started.Process.Pid) {
		t.Fatal("启动成功应保留进程存活")
	}
}

// restoreUocr2Vars 记录并恢复 startUocr2Server 的三个注入点。
func restoreUocr2Vars(t *testing.T) {
	t.Helper()
	origWait, origHealthy, origBuild := uocr2StartWait, uocr2Healthy, uocr2BuildCmd
	t.Cleanup(func() {
		uocr2StartWait, uocr2Healthy, uocr2BuildCmd = origWait, origHealthy, origBuild
	})
}
