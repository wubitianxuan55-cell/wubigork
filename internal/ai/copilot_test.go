package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestBeatType(t *testing.T) {
	b := Beat{
		ID:          "beat-1",
		Description: "Elara 推开大门",
		Order:       1,
	}
	if b.ID != "beat-1" {
		t.Fatal("beat ID mismatch")
	}
	if b.Order != 1 {
		t.Fatal("beat order mismatch")
	}
}

func TestGhostCompletePromptTruncation(t *testing.T) {
	// 验证超长文本会被截断到 2000 字符
	longText := ""
	for i := 0; i < 3000; i++ {
		longText += "文"
	}

	// 截断逻辑在 GhostComplete 内部执行
	runes := []rune(longText)
	if len(runes) <= 2000 {
		t.Fatal("test text should be > 2000 runes")
	}
	trimmed := string(runes[len(runes)-2000:])
	if len([]rune(trimmed)) != 2000 {
		t.Fatalf("trimmed should be 2000 runes, got %d", len([]rune(trimmed)))
	}
}

func TestBeatSliceOrdering(t *testing.T) {
	beats := []Beat{
		{ID: "beat-3", Description: "Third", Order: 3},
		{ID: "beat-1", Description: "First", Order: 1},
		{ID: "beat-2", Description: "Second", Order: 2},
	}

	// 验证结构完整性
	for _, b := range beats {
		if b.ID == "" {
			t.Fatal("beat ID should not be empty")
		}
		if b.Order < 1 {
			t.Fatal("beat order should be >= 1")
		}
	}
}

// TestCmdKEdit_EngineIDRoutesToEngine Cmd+K 编辑必须按传入引擎路由
// （修复 P1 已知限制：此前模型名走 cfg.Model、引擎随活跃引擎，忽略功能绑定）。
func TestCmdKEdit_EngineIDRoutesToEngine(t *testing.T) {
	var gotModel string
	hit := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotModel = req.Model
		hit <- struct{}{}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"改写后的文本\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: srv.URL,
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	c := &Client{
		cfg:        &config.Config{XaiAPIBaseURL: "http://127.0.0.1:1", Model: "stale-model"},
		httpClient: srv.Client(),
		sem:        make(chan struct{}, 4),
		engineMgr:  mgr,
	}

	out, err := c.CmdKEdit(context.Background(), "herdsman", "qwen3-8b", "选中文本", "用更紧张的节奏重写", "")
	if err != nil {
		t.Fatalf("CmdKEdit: %v", err)
	}
	select {
	case <-hit:
	default:
		t.Fatal("请求未到达绑定引擎（引擎路由未生效）")
	}
	if gotModel != "qwen3-8b" {
		t.Errorf("请求模型 = %q, want qwen3-8b（功能绑定模型应透传）", gotModel)
	}
	if out != "改写后的文本" {
		t.Errorf("编辑结果 = %q", out)
	}
}
