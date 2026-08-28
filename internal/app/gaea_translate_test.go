package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

func translateTestApp(t *testing.T, models []modelengine.ModelInfo, handler http.HandlerFunc) (*App, string) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		BaseURL: srv.URL + "/v1",
		Enabled: true,
		Models:  models,
	}); err != nil {
		t.Fatal(err)
	}
	return &App{core: &core{engineMgr: mgr}}, srv.URL
}

func TestLocalTranslate_UsesTranslationModel(t *testing.T) {
	a, _ := translateTestApp(t, []modelengine.ModelInfo{
		{ID: "Hy-MT2:7B"},
		{ID: "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"},
	}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode: %v", err)
		}
		if req["model"] != "Hy-MT2:7B" {
			t.Errorf("model = %v, want Hy-MT2:7B", req["model"])
		}
		msgs := req["messages"].([]any)
		sys := msgs[0].(map[string]any)["content"].(string)
		user := msgs[1].(map[string]any)["content"].(string)
		if !strings.Contains(sys, "专业翻译") {
			t.Errorf("system 提示缺少翻译约束: %q", sys)
		}
		if !strings.Contains(user, "目标语言：en") {
			t.Errorf("user 提示缺少目标语言: %q", user)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello world"}}]}`))
	})

	res, err := a.LocalTranslate(LocalTranslateRequest{Text: "你好世界", TargetLang: "en"})
	if err != nil {
		t.Fatalf("LocalTranslate: %v", err)
	}
	if res.Text != "Hello world" || res.Model != "Hy-MT2:7B" || res.UsedFallback {
		t.Fatalf("结果异常: %+v", res)
	}
}

func TestLocalTranslate_ExplicitModel(t *testing.T) {
	a, _ := translateTestApp(t, nil, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["model"] != "Hunyuan-MT:7B" {
			t.Errorf("model = %v, want Hunyuan-MT:7B", req["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"bonjour"}}]}`))
	})
	res, err := a.LocalTranslate(LocalTranslateRequest{Text: "你好", TargetLang: "fr", Model: "Hunyuan-MT:7B"})
	if err != nil || res.Text != "bonjour" {
		t.Fatalf("结果异常: %+v err=%v", res, err)
	}
}

func TestLocalTranslate_NoTranslationModel(t *testing.T) {
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"}}, nil)
	_, err := a.LocalTranslate(LocalTranslateRequest{Text: "你好", TargetLang: "en"})
	if err == nil || !strings.Contains(err.Error(), "未安装翻译模型") {
		t.Fatalf("应提示未安装翻译模型, got %v", err)
	}
}

func TestLocalTranslate_EmptyText(t *testing.T) {
	a, _ := translateTestApp(t, nil, nil)
	if _, err := a.LocalTranslate(LocalTranslateRequest{Text: "  ", TargetLang: "en"}); err == nil {
		t.Fatal("空文本应报错")
	}
}

func TestResolveTranslationTarget(t *testing.T) {
	// 无 herdsman 引擎 → 未找到。
	c := &core{engineMgr: modelengine.NewManager("", "")}
	if _, _, found, err := c.resolveTranslationTarget(""); err != nil || found {
		t.Fatalf("无引擎应 found=false: found=%v err=%v", found, err)
	}
	// herdsman 含翻译模型 → 命中。
	mgr := modelengine.NewManager("", "")
	_ = mgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", BaseURL: "http://127.0.0.1:8080", Enabled: true,
		Models: []modelengine.ModelInfo{
			{ID: "Hunyuan-MT:7B"},
			{ID: "bge-m3"},
		},
	})
	c = &core{engineMgr: mgr}
	eng, model, found, err := c.resolveTranslationTarget("")
	if err != nil || !found || eng != "herdsman" || model != "Hunyuan-MT:7B" {
		t.Fatalf("应命中翻译模型: eng=%s model=%s found=%v err=%v", eng, model, found, err)
	}
	// 显式模型优先。
	eng, model, found, err = c.resolveTranslationTarget("Hy-MT1.5:1.8B")
	if err != nil || !found || model != "Hy-MT1.5:1.8B" {
		t.Fatalf("显式模型应优先: eng=%s model=%s found=%v err=%v", eng, model, found, err)
	}
}

func TestTranslateTextTool(t *testing.T) {
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello"}}]}`))
	})
	tool := translateTextTool{a: a}
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"text":"你好","target_lang":"en"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "翻译模型") {
		t.Fatalf("输出异常: %q", out)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("缺 text 应报错")
	}
}
