package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newProbeApp 构造带 herdsman 引擎（BaseURL=测试 server）的 App。
func newProbeApp(t *testing.T, srvURL string) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	c := &core{cfg: &config.Config{}, engineMgr: modelengine.NewManager("", "")}
	a := &App{core: c}
	eng := modelengine.EngineConfig{ID: "herdsman", Name: "herdsman", BaseURL: srvURL, Enabled: true}
	if err := c.engineMgr.SaveEngine(eng); err != nil {
		t.Fatal(err)
	}
	return a
}

// streamSSEServer 模拟 Herdsman 流式响应：3 个分块 + [DONE]。
func streamSSEServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		chunks := []string{"你好，", "我是本地", "大模型。"}
		for i, c := range chunks {
			body := `{"choices":[{"delta":{"content":"` + c + `"}}]}`
			_, _ = w.Write([]byte("data: " + body + "\n\n"))
			fl.Flush()
			time.Sleep(5 * time.Millisecond) // 制造分块间隔
			_ = i
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		fl.Flush()
	}))
}

func TestGaeaBenchmarkStreamProbe_OK(t *testing.T) {
	srv := streamSSEServer(t)
	defer srv.Close()
	a := newProbeApp(t, srv.URL)

	res, err := a.GaeaBenchmarkStreamProbe("qwen3-8b")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !res.Completed {
		t.Fatalf("probe = %+v, want ok+completed", res)
	}
	if res.Chunks != 3 {
		t.Fatalf("chunks = %d, want 3", res.Chunks)
	}
	if res.TTFTMS < 0 || res.DurationMS <= 0 {
		t.Fatalf("timing 异常: %+v", res)
	}
	if !strings.Contains(res.ResponseStart, "你好") {
		t.Fatalf("response_start = %q", res.ResponseStart)
	}
	if res.MaxGapMS < 0 {
		t.Fatalf("max_gap = %d", res.MaxGapMS)
	}
}

func TestGaeaBenchmarkStreamProbe_Errors(t *testing.T) {
	// 空模型
	a := newProbeApp(t, "http://127.0.0.1:1")
	if _, err := a.GaeaBenchmarkStreamProbe(""); err == nil {
		t.Fatal("空模型应报错")
	}
	// 无 herdsman 引擎
	a2 := &App{core: &core{cfg: &config.Config{}}}
	if _, err := a2.GaeaBenchmarkStreamProbe("qwen3-8b"); err == nil {
		t.Fatal("未配置 herdsman 引擎应报错")
	}
}

func TestGaeaBenchmarkStreamProbe_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not running", http.StatusBadRequest)
	}))
	defer srv.Close()
	a := newProbeApp(t, srv.URL)
	_, err := a.GaeaBenchmarkStreamProbe("qwen3-8b")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v, want 400", err)
	}
}
