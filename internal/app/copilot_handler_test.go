package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// TestCmdKEdit_UsesNovelBinding 小说编辑器 Cmd+K 编辑应走 novel 功能绑定，
// 而不是全局 cfg.Model + 活跃引擎（P1 已知限制回归）。
func TestCmdKEdit_UsesNovelBinding(t *testing.T) {
	hit := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit <- struct{}{}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"绑定模型改写\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	c := &core{
		cfg:       &config.Config{Model: "stale-xai-model", XaiAPIBaseURL: "http://127.0.0.1:1"},
		engineMgr: modelengine.NewManager("", ""),
	}
	// herdsman → 测试服务器；xai 指向死地址（若错误走活跃引擎会失败）
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: srv.URL,
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}

	a := &App{core: c}
	a.client = ai.NewClient(a.cfg)
	a.client.SetEngineManager(a.engineMgr)

	out, err := a.CmdKEdit("选中文本", "重写", "")
	if err != nil {
		t.Fatalf("CmdKEdit: %v", err)
	}
	select {
	case <-hit:
	default:
		t.Fatal("请求未到达 novel 绑定引擎（仍走全局/活跃引擎）")
	}
	if edited, _ := out["edited"].(string); edited != "绑定模型改写" {
		t.Errorf("edited = %q", edited)
	}
}
