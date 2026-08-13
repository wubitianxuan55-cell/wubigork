package modelengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ─── NewManager 与预置引擎 ───────────────────────────────────

func TestNewManager_Presets(t *testing.T) {
	m := NewManager("xai-token", "ds-key")
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	// 预置 6 引擎
	want := map[string]EngineType{
		"xai": EngineXAI, "ollama": EngineOllama,
		"herdsman": EngineHerdsman, "deepseek": EngineDeepseek,
		"opencode-go": EngineOpencodeGo, "opencode-zen": EngineOpencodeZen,
	}
	for id, typ := range want {
		e, ok := m.GetEngine(id)
		if !ok {
			t.Errorf("引擎 %s 未预置", id)
			continue
		}
		if e.Type != typ {
			t.Errorf("%s 类型 = %s, want %s", id, e.Type, typ)
		}
	}
}

func TestNewManager_Keys(t *testing.T) {
	m := NewManager("tok-abc", "key-def")
	// BuildChatURL 的 API key 应从传入 key 生效
	url, key, err := m.BuildChatURL("xai")
	if err != nil {
		t.Fatalf("BuildChatURL(xai): %v", err)
	}
	if key != "tok-abc" {
		t.Errorf("xAI key = %q, want tok-abc", key)
	}
	if !strings.HasSuffix(url, "/chat/completions") {
		t.Errorf("URL = %q, want suffix /chat/completions", url)
	}
}

// ─── GetEngine / GetEngines ──────────────────────────────────

func TestGetEngine_Missing(t *testing.T) {
	m := NewManager("", "")
	if _, ok := m.GetEngine("nope"); ok {
		t.Error("不存在的引擎应返回 ok=false")
	}
}

func TestGetEngine_ReturnsCopy(t *testing.T) {
	m := NewManager("", "")
	e, _ := m.GetEngine("ollama")
	e.DefaultModel = "hacked"
	// 内部不应被修改
	again, _ := m.GetEngine("ollama")
	if again.DefaultModel == "hacked" {
		t.Error("GetEngine 返回了内部指针，外部修改泄漏")
	}
}

func TestGetEngine_StripsAPIKey(t *testing.T) {
	m := NewManager("", "")
	// deepseek 引擎无 APIKey 字段在内存中；xAI 的 key 存于 Manager 不在此
	// 验证 GetEngine 返回的配置中 APIKey 被清空（敏感信息不暴露给前端）
	m.UpdateDeepseekKey("secret-ds")
	_ = m
	e, _ := m.GetEngine("deepseek")
	if e.APIKey != "" {
		t.Errorf("APIKey 应被清空, got %q", e.APIKey)
	}
}

func TestGetEngines_CountAndKeys(t *testing.T) {
	m := NewManager("", "")
	es := m.GetEngines()
	if len(es) != 7 {
		t.Fatalf("GetEngines 数量 = %d, want 7", len(es))
	}
	for _, e := range es {
		if e.APIKey != "" {
			t.Errorf("引擎 %s 泄漏 APIKey", e.ID)
		}
	}
}

// ─── SaveEngine ──────────────────────────────────────────────

func TestSaveEngine_UpdateFields(t *testing.T) {
	m := NewManager("", "")
	// 只更新 BaseURL + Enabled，不应清空其他字段
	err := m.SaveEngine(EngineConfig{
		ID:      "ollama",
		BaseURL: "http://new-host:11434/v1",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	e, _ := m.GetEngine("ollama")
	if e.BaseURL != "http://new-host:11434/v1" {
		t.Errorf("BaseURL = %q, want new-host", e.BaseURL)
	}
	if e.Enabled {
		t.Error("Enabled 应被置为 false")
	}
	if e.Name == "" {
		t.Error("Name 不应被清空")
	}
}

func TestSaveEngine_UpdateModels(t *testing.T) {
	m := NewManager("", "")
	models := []ModelInfo{{ID: "grok-4.20", Status: "running"}}
	if err := m.SaveEngine(EngineConfig{ID: "xai", Models: models}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	e, _ := m.GetEngine("xai")
	if len(e.Models) != 1 || e.Models[0].ID != "grok-4.20" {
		t.Errorf("Models 未更新: %+v", e.Models)
	}
}

func TestSaveEngine_Missing(t *testing.T) {
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "nope"}); err == nil {
		t.Error("保存不存在的引擎应报错")
	}
}

// ─── SetDefaultModel / GetDefaultModel ───────────────────────

func TestSetDefaultModel_Valid(t *testing.T) {
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "xai", Models: []ModelInfo{{ID: "grok-4.20"}, {ID: "grok-4.1"}}}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetDefaultModel("xai", "grok-4.1"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}
	got, _ := m.GetDefaultModel("xai")
	if got != "grok-4.1" {
		t.Errorf("默认模型 = %q, want grok-4.1", got)
	}
}

func TestSetDefaultModel_NotInList(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "xai", Models: []ModelInfo{{ID: "grok-4.20"}}})
	if err := m.SetDefaultModel("xai", "ghost-model"); err == nil {
		t.Error("设置不在列表中的模型应报错")
	}
}

func TestSetDefaultModel_EmptyModelsAllowsAny(t *testing.T) {
	m := NewManager("", "")
	// 模型列表为空时（如未拉取过），允许任意模型名
	if err := m.SetDefaultModel("ollama", "qwen3"); err != nil {
		t.Errorf("空模型列表应允许设置: %v", err)
	}
}

func TestSetDefaultModel_MissingEngine(t *testing.T) {
	m := NewManager("", "")
	if err := m.SetDefaultModel("nope", "m"); err == nil {
		t.Error("不存在的引擎应报错")
	}
}

func TestGetDefaultModel_MissingEngine(t *testing.T) {
	m := NewManager("", "")
	if _, err := m.GetDefaultModel("nope"); err == nil {
		t.Error("不存在的引擎应报错")
	}
}

// ─── BuildChatURL ────────────────────────────────────────────

func TestBuildChatURL_TrailingSlash(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: "http://localhost:11434/v1/", Enabled: true})
	url, _, err := m.BuildChatURL("ollama")
	if err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	if url != "http://localhost:11434/v1/chat/completions" {
		t.Errorf("URL = %q, want 无重复斜杠拼接", url)
	}
}

func TestBuildChatURL_Disabled(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "ollama", Enabled: false})
	if _, _, err := m.BuildChatURL("ollama"); err == nil {
		t.Error("未启用引擎应报错")
	}
}

func TestBuildChatURL_DeepseekKey(t *testing.T) {
	m := NewManager("", "ds-key-123")
	_, key, err := m.BuildChatURL("deepseek")
	if err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	if key != "ds-key-123" {
		t.Errorf("deepseek key = %q, want ds-key-123", key)
	}
}

// ─── UpdateXAIKey / UpdateDeepseekKey ────────────────────────

func TestUpdateKeys_AffectBuildChatURL(t *testing.T) {
	m := NewManager("old-xai", "old-ds")
	m.UpdateXAIKey("new-xai")
	m.UpdateDeepseekKey("new-ds")

	_, xaiKey, _ := m.BuildChatURL("xai")
	if xaiKey != "new-xai" {
		t.Errorf("xai key 未更新: %q", xaiKey)
	}
	_, dsKey, _ := m.BuildChatURL("deepseek")
	if dsKey != "new-ds" {
		t.Errorf("deepseek key 未更新: %q", dsKey)
	}
}

// ─── HTTP 路径：TestConnection / RefreshModels / fetchModels ─

// newTestManager 返回 Manager + 指向测试服务器的引擎配置
func newTestManager(t *testing.T, handler http.Handler) *Manager {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	m := NewManager("xai-tok", "ds-key")
	// 把 ollama 引擎指向测试服务器（不启用认证）
	m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true})
	// 把 xai 引擎也指向测试服务器（验证 Bearer 头）
	m.SaveEngine(EngineConfig{ID: "xai", BaseURL: srv.URL, Enabled: true})
	return m
}

func TestRefreshModels_Success(t *testing.T) {
	var gotAuth string
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/models" {
			t.Errorf("path = %q, want /models", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "model-a", "owned_by": "gaea", "status": "running"},
				{"id": "model-b", "owned_by": "gaea"},
			},
		})
	}))
	models, err := m.RefreshModels(context.Background(), "xai")
	if err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("模型数 = %d, want 3（含内置 grok-tts）", len(models))
	}
	if models[0].ID != "model-a" || models[0].OwnedBy != "gaea" || models[0].Status != "running" {
		t.Errorf("模型解析错误: %+v", models[0])
	}
	if models[len(models)-1].ID != "grok-tts" {
		t.Errorf("xAI 刷新后应包含内置 grok-tts, got %q", models[len(models)-1].ID)
	}
	if gotAuth != "Bearer xai-tok" {
		t.Errorf("Authorization = %q, want Bearer xai-tok", gotAuth)
	}
	// 刷新后引擎的 Models 应更新
	e, _ := m.GetEngine("xai")
	if len(e.Models) != 3 {
		t.Errorf("引擎 Models 未更新, 长度 = %d", len(e.Models))
	}
}

func TestRefreshModels_401XAI(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	_, err := m.RefreshModels(context.Background(), "xai")
	if err == nil {
		t.Fatal("401 应报错")
	}
	if !strings.Contains(err.Error(), "未登录 xAI") {
		t.Errorf("错误信息 = %q, want 包含 xAI 登录提示", err.Error())
	}
}

func TestRefreshModels_401Deepseek(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	_, err := m.RefreshModels(context.Background(), "deepseek")
	if err == nil {
		t.Fatal("401 应报错")
	}
	if !strings.Contains(err.Error(), "DeepSeek API Key") {
		t.Errorf("错误信息 = %q, want 包含 DeepSeek Key 提示", err.Error())
	}
}

// ─── OpenCode Go 引擎 ───────────────────────────────────────

func TestOpencodeGo_DefaultConfig(t *testing.T) {
	m := NewManager("", "")
	e, ok := m.GetEngine("opencode-go")
	if !ok {
		t.Fatal("opencode-go 引擎未预置")
	}
	if e.Type != EngineOpencodeGo {
		t.Errorf("类型 = %s, want opencode-go", e.Type)
	}
	if e.BaseURL != "https://opencode.ai/zen/go/v1" {
		t.Errorf("BaseURL = %q, want https://opencode.ai/zen/go/v1", e.BaseURL)
	}
	if e.DefaultModel != "deepseek-v4-pro" {
		t.Errorf("默认模型 = %q, want deepseek-v4-pro", e.DefaultModel)
	}
}

func TestOpencodeGo_BuildChatURL(t *testing.T) {
	m := NewManager("", "")
	m.UpdateOpencodeKey("oc-key-123")
	url, key, err := m.BuildChatURL("opencode-go")
	if err != nil {
		t.Fatalf("BuildChatURL(opencode-go): %v", err)
	}
	if key != "oc-key-123" {
		t.Errorf("key = %q, want oc-key-123", key)
	}
	if url != "https://opencode.ai/zen/go/v1/chat/completions" {
		t.Errorf("URL = %q, want .../v1/chat/completions", url)
	}
}

func TestOpencodeGo_RefreshModels_FiltersIncompatible(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "kimi-k2.7-code", "owned_by": "opencode-go"},
				{"id": "deepseek-v4-pro", "owned_by": "opencode-go"},
				{"id": "qwen3.7-max", "owned_by": "opencode-go"},  // Anthropic /messages，应过滤
				{"id": "minimax-m3", "owned_by": "opencode-go"},   // Anthropic /messages，应过滤
				{"id": "gpt-5.6-luna", "owned_by": "opencode-go"}, // /responses，应过滤
				{"id": "glm-5.2", "owned_by": "opencode-go"},
			},
		})
	}))
	t.Cleanup(srv.Close)
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "opencode-go", BaseURL: srv.URL, Enabled: true})
	m.UpdateOpencodeKey("oc-key")
	models, err := m.RefreshModels(context.Background(), "opencode-go")
	if err != nil {
		t.Fatalf("RefreshModels(opencode-go): %v", err)
	}
	if gotAuth != "Bearer oc-key" {
		t.Errorf("Authorization = %q, want Bearer oc-key", gotAuth)
	}
	want := []string{"kimi-k2.7-code", "deepseek-v4-pro", "glm-5.2"}
	if len(models) != len(want) {
		t.Fatalf("过滤后模型数 = %d, want %d: %+v", len(models), len(want), models)
	}
	for i, id := range want {
		if models[i].ID != id {
			t.Errorf("models[%d].ID = %q, want %q", i, models[i].ID, id)
		}
	}
}

func TestOpencodeGo_RefreshModels_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "opencode-go", BaseURL: srv.URL, Enabled: true})
	m.UpdateOpencodeKey("oc-key")
	_, err := m.RefreshModels(context.Background(), "opencode-go")
	if err == nil {
		t.Fatal("401 应报错")
	}
	if !strings.Contains(err.Error(), "OpenCode Go API Key") {
		t.Errorf("错误信息 = %q, want 包含 OpenCode Go Key 提示", err.Error())
	}
}

// ─── OpenCode Zen 引擎 ───────────────────────────────────────

func TestOpencodeZen_DefaultConfig(t *testing.T) {
	m := NewManager("", "")
	e, ok := m.GetEngine("opencode-zen")
	if !ok {
		t.Fatal("opencode-zen 引擎未预置")
	}
	if e.Type != EngineOpencodeZen {
		t.Errorf("类型 = %s, want opencode-zen", e.Type)
	}
	if e.BaseURL != "https://opencode.ai/zen/v1" {
		t.Errorf("BaseURL = %q, want https://opencode.ai/zen/v1", e.BaseURL)
	}
}

func TestOpencodeZen_BuildChatURL(t *testing.T) {
	m := NewManager("", "")
	m.UpdateOpencodeZenKey("zen-key-456")
	url, key, err := m.BuildChatURL("opencode-zen")
	if err != nil {
		t.Fatalf("BuildChatURL(opencode-zen): %v", err)
	}
	if key != "zen-key-456" {
		t.Errorf("key = %q, want zen-key-456", key)
	}
	if url != "https://opencode.ai/zen/v1/chat/completions" {
		t.Errorf("URL = %q, want .../v1/chat/completions", url)
	}
}

func TestOpencodeZen_RefreshModels_FiltersIncompatible(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "deepseek-v4-pro", "owned_by": "opencode"},
				{"id": "glm-5.2", "owned_by": "opencode"},
				{"id": "minimax-m3", "owned_by": "opencode"},       // Zen 走 chat/completions，应保留
				{"id": "gpt-5.5", "owned_by": "opencode"},          // /responses，应过滤
				{"id": "claude-opus-4-5", "owned_by": "opencode"},  // /messages，应过滤
				{"id": "gemini-3.5-flash", "owned_by": "opencode"}, // 专用端点，应过滤
				{"id": "qwen3.7-max", "owned_by": "opencode"},      // /messages，应过滤
				{"id": "grok-4.5", "owned_by": "opencode"},         // Zen 走 /responses，应过滤
			},
		})
	}))
	t.Cleanup(srv.Close)
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "opencode-zen", BaseURL: srv.URL, Enabled: true})
	m.UpdateOpencodeZenKey("zen-key")
	models, err := m.RefreshModels(context.Background(), "opencode-zen")
	if err != nil {
		t.Fatalf("RefreshModels(opencode-zen): %v", err)
	}
	if gotAuth != "Bearer zen-key" {
		t.Errorf("Authorization = %q, want Bearer zen-key", gotAuth)
	}
	want := []string{"deepseek-v4-pro", "glm-5.2", "minimax-m3"}
	if len(models) != len(want) {
		t.Fatalf("过滤后模型数 = %d, want %d: %+v", len(models), len(want), models)
	}
	for i, id := range want {
		if models[i].ID != id {
			t.Errorf("models[%d].ID = %q, want %q", i, models[i].ID, id)
		}
	}
}

func TestOpencodeZen_RefreshModels_401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{ID: "opencode-zen", BaseURL: srv.URL, Enabled: true})
	m.UpdateOpencodeZenKey("zen-key")
	_, err := m.RefreshModels(context.Background(), "opencode-zen")
	if err == nil {
		t.Fatal("401 应报错")
	}
	if !strings.Contains(err.Error(), "OpenCode Zen API Key") {
		t.Errorf("错误信息 = %q, want 包含 OpenCode Zen Key 提示", err.Error())
	}
}

func TestRefreshModels_ServerError(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	if _, err := m.RefreshModels(context.Background(), "ollama"); err == nil {
		t.Error("500 应报错")
	}
}

func TestRefreshModels_BadJSON(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not-json{"))
	}))
	if _, err := m.RefreshModels(context.Background(), "ollama"); err == nil {
		t.Error("非法 JSON 应报错")
	}
}

func TestRefreshModels_MissingEngine(t *testing.T) {
	m := NewManager("", "")
	if _, err := m.RefreshModels(context.Background(), "nope"); err == nil {
		t.Error("不存在的引擎应报错")
	}
}

func TestTestConnection_Success(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "m1"}, {"id": "m2"}, {"id": "m3"}},
		})
	}))
	status, err := m.TestConnection(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if !status.Connected {
		t.Error("Connected 应为 true")
	}
	if status.ModelCount != 3 {
		t.Errorf("ModelCount = %d, want 3", status.ModelCount)
	}
	if status.Error != "" {
		t.Errorf("Error = %q, want 空", status.Error)
	}
	if status.LatencyMs <= 0 {
		t.Errorf("LatencyMs = %d, want > 0", status.LatencyMs)
	}
}

func TestTestConnection_Failure(t *testing.T) {
	m := newTestManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	// TestConnection 失败时不返回 error，而是把错误放进 status.Error（前端展示）
	status, err := m.TestConnection(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("TestConnection 不应返回 error: %v", err)
	}
	if status.Connected {
		t.Error("连接失败时 Connected 应为 false")
	}
	if status.Error == "" {
		t.Error("连接失败时 Error 应非空")
	}
}

func TestTestConnection_MissingEngine(t *testing.T) {
	m := NewManager("", "")
	if _, err := m.TestConnection(context.Background(), "nope"); err == nil {
		t.Error("不存在的引擎应报错")
	}
}

func TestClassifyModelKind_HerdsmanSpecialized(t *testing.T) {
	cases := []struct {
		model string
		want  string
	}{
		{"qwen3-tts-voiceclone", "tts"},
		{"edge-tts", "tts"},
		{"voxcpm2", "tts"},
		{"sherpa-onnx-streaming-zipformer-zh-14m", "stt"},
		{"funasr", "stt"},
		{"paddleocr-ppocrv5-server", "ocr"},
		{"minerU", "ocr"},
		{"bge-reranker-v2-m3", "rerank"},
		{"bge-m3", "embedding"},
	}
	for _, c := range cases {
		if got := classifyModelKind(EngineHerdsman, c.model); got != c.want {
			t.Errorf("classifyModelKind(herdsman, %q) = %q, want %q", c.model, got, c.want)
		}
	}
}
