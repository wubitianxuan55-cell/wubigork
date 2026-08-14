package app

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/httpbridge"
)

// coreWithCfg 构造最小 core 供 TTS 服务测试（不触碰真实配置/不拉起真实进程）。
func coreWithCfg(cfg *config.Config) *core {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &core{cfg: cfg}
}

// TestTTSURLDerivation 配置推导（T6-9.5）：探活 URL 由端口推导；
// 零配置回退默认 8010（与历史硬编码一致）。
func TestTTSURLDerivation(t *testing.T) {
	if got := coreWithCfg(nil).ttsURL(); got != "http://127.0.0.1:8010/v1/models" {
		t.Errorf("零配置 ttsURL = %q, want 默认 8010（历史一致）", got)
	}
	if got := coreWithCfg(&config.Config{CosyVoicePort: 9020}).ttsURL(); got != "http://127.0.0.1:9020/v1/models" {
		t.Errorf("配置端口 9020 后 ttsURL = %q", got)
	}
}

// TestTTSCosyVoiceCmdFromConfig 命令构造（T6-9.5）：python/script 由配置目录推导；
// 零配置回退默认 C:\AI\cosyvoice（与历史硬编码一致）。
func TestTTSCosyVoiceCmdFromConfig(t *testing.T) {
	defDir := filepath.FromSlash("C:/AI/cosyvoice")

	// 零配置 → 默认目录
	cmd := ttsCosyVoiceCmdFor(&config.Config{})
	if cmd.Dir != defDir {
		t.Errorf("默认 cmd.Dir = %q, want %q", cmd.Dir, defDir)
	}
	wantArgs := []string{
		filepath.Join(defDir, "venv", "Scripts", "python.exe"),
		filepath.Join(defDir, "server.py"),
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != wantArgs[0] || cmd.Args[1] != wantArgs[1] {
		t.Errorf("默认 cmd.Args = %v, want %v", cmd.Args, wantArgs)
	}

	// 自定义目录 → 推导路径
	cfgDir := filepath.FromSlash("D:/voice/cosy")
	cmd = ttsCosyVoiceCmdFor(&config.Config{CosyVoiceDir: cfgDir})
	if cmd.Dir != cfgDir {
		t.Errorf("自定义 cmd.Dir = %q, want %q", cmd.Dir, cfgDir)
	}
	wantArgs = []string{
		filepath.Join(cfgDir, "venv", "Scripts", "python.exe"),
		filepath.Join(cfgDir, "server.py"),
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != wantArgs[0] || cmd.Args[1] != wantArgs[1] {
		t.Errorf("自定义 cmd.Args = %v, want %v", cmd.Args, wantArgs)
	}
}

// TestStartTTSService_RetryBackoff 退避重试（T6-9.5）：注入必然失败的 cmd 工厂，
// startTTSServiceWithRetry 初始 1 次 + 最多 3 次退避重试（≥2 次重试），
// 全部失败后返回错误。
func TestStartTTSService_RetryBackoff(t *testing.T) {
	var attempts int32
	origFactory := ttsCmdFactory
	origBackoff := ttsStartBackoff
	origMax := ttsStartMaxRetries
	missing := filepath.Join(t.TempDir(), "missing-python.exe")
	ttsCmdFactory = func(cfg *config.Config) *exec.Cmd {
		atomic.AddInt32(&attempts, 1)
		return exec.Command(missing, "server.py")
	}
	ttsStartBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	ttsStartMaxRetries = 3
	t.Cleanup(func() {
		ttsCmdFactory = origFactory
		ttsStartBackoff = origBackoff
		ttsStartMaxRetries = origMax
	})

	err := coreWithCfg(nil).startTTSServiceWithRetry("cosyvoice")
	if err == nil {
		t.Fatal("全部启动失败后应返回 error")
	}
	if n := atomic.LoadInt32(&attempts); n < 2 {
		t.Errorf("启动尝试次数 = %d, want ≥2（至少重试 2 次）", n)
	}
	if n := atomic.LoadInt32(&attempts); n != 4 {
		t.Errorf("启动尝试次数 = %d, want 4（初始 + 3 次重试）", n)
	}
}

// TestEnsureLocalTTSService_RetriesThenEmitsError 完整异步路径（T6-9.5）：
// 注入必然失败的 cmd 工厂，goroutine 退避重试后 emit 错误事件
// （ready=false 且带 error），并正确清理 ttsEnsuring（幂等防重入复原）。
func TestEnsureLocalTTSService_RetriesThenEmitsError(t *testing.T) {
	var attempts int32
	origFactory := ttsCmdFactory
	origBackoff := ttsStartBackoff
	origMax := ttsStartMaxRetries
	missing := filepath.Join(t.TempDir(), "missing-python.exe")
	ttsCmdFactory = func(cfg *config.Config) *exec.Cmd {
		atomic.AddInt32(&attempts, 1)
		return exec.Command(missing, "server.py")
	}
	ttsStartBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	ttsStartMaxRetries = 3
	t.Cleanup(func() {
		ttsCmdFactory = origFactory
		ttsStartBackoff = origBackoff
		ttsStartMaxRetries = origMax
	})

	// 用空闲端口构造 core：本机可能恰好有真实 CosyVoice 跑在默认 8010，
	// 测试不依赖机器端口状态（探活走真实 HTTP，空闲端口必然 connection refused）。
	c := coreWithCfg(&config.Config{CosyVoicePort: freePort(t)})

	// 订阅 tts-service-status（httpbridge 全局 hub），验证最终 emit 错误事件
	ts := httptest.NewServer(httpbridge.New(nil).Handler())
	t.Cleanup(ts.Close)
	eventCh := make(chan string, 32)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel) // LIFO：先 cancel 关闭 SSE，再关 server
	go func() {
		req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/stream?id=tts-service-status", nil)
		if err != nil {
			eventCh <- "ERR:" + err.Error()
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			eventCh <- "ERR:" + err.Error()
			return
		}
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data: ") {
				eventCh <- strings.TrimPrefix(line, "data: ")
			}
		}
	}()

	// 先等 SSE 连接建立（connected 事件），再触发启动，避免错过 emit
	waitStreamEvent(t, eventCh, func(m map[string]interface{}) bool {
		_, ok := m["id"]
		return ok
	}, "SSE connected")

	res := c.ensureLocalTTSService("cosyvoice")
	if res["starting"] != true {
		t.Errorf("ensure 应返回 starting=true, got %v", res)
	}

	// 重试全部失败后 emit 错误事件
	status := waitStreamEvent(t, eventCh, func(m map[string]interface{}) bool {
		_, ok := m["ready"]
		return ok
	}, "tts-service-status 错误事件")
	if status["ready"] != false {
		t.Errorf("错误事件 ready = %v, want false", status["ready"])
	}
	if status["error"] == nil || status["error"] == "" {
		t.Errorf("错误事件应带 error 信息, got %v", status)
	}

	// ttsEnsuring 正确清理（防重入标记复原）
	deadline := time.Now().Add(5 * time.Second)
	for {
		ttsEnsureMu.Lock()
		ensuring := ttsEnsuring["cosyvoice"]
		ttsEnsureMu.Unlock()
		if !ensuring {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("失败后 ttsEnsuring 未清理")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if n := atomic.LoadInt32(&attempts); n != 4 {
		t.Errorf("启动尝试次数 = %d, want 4（初始 + 3 次重试）", n)
	}
}

// freePort 返回一个当前空闲的本地端口（测试探活用，避免依赖机器端口状态）。
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// waitStreamEvent 从 SSE 事件流中等待满足 match 的事件（JSON payload）。
func waitStreamEvent(t *testing.T, ch <-chan string, match func(map[string]interface{}) bool, desc string) map[string]interface{} {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev := <-ch:
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(ev), &m); err != nil {
				continue
			}
			if match(m) {
				return m
			}
		case <-timeout:
			t.Fatalf("等待 %s 超时", desc)
			return nil
		}
	}
}
