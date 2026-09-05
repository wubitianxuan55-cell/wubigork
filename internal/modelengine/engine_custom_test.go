package modelengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── A 刀：自定义引擎（OpenAI 兼容自定义服务商）─────────────────

// TestAddCustomEngine_FullChain 创建 → 引擎列表/单查 → 聊天 URL/Key 取用。
func TestAddCustomEngine_FullChain(t *testing.T) {
	m := NewManager("", "")
	id, err := m.AddCustomEngine("My LLM Relay", "https://relay.example.com/v1", "sk-relay-123")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if !strings.HasPrefix(id, "custom-") {
		t.Fatalf("engineID = %q, want custom- 前缀", id)
	}

	eng, ok := m.GetEngine(id)
	if !ok {
		t.Fatal("创建后 GetEngine 找不到自定义引擎")
	}
	if eng.Type != EngineCustom {
		t.Errorf("Type = %q, want custom", eng.Type)
	}
	if eng.Name != "custom" || eng.Label != "My LLM Relay" {
		t.Errorf("Name/Label = %q/%q, want custom/My LLM Relay", eng.Name, eng.Label)
	}
	if eng.BaseURL != "https://relay.example.com/v1" {
		t.Errorf("BaseURL = %q", eng.BaseURL)
	}
	if !eng.Enabled {
		t.Error("新自定义引擎应默认启用")
	}
	if eng.Type.IsLocal() {
		t.Error("EngineCustom.IsLocal() 应为 false（云端语义，离线模式下被门控）")
	}
	if eng.APIKey != "" {
		t.Error("EngineConfig.APIKey 必须恒为空（Key 只存 customKeys）")
	}

	// 挂在展示顺序末尾
	got := m.GetEngines()
	if got[len(got)-1].ID != id {
		t.Errorf("自定义引擎应追加在 order 末尾，实际末位 = %q", got[len(got)-1].ID)
	}

	// 聊天路径：BuildChatURL 从 customKeys 取 Key
	chatURL, apiKey, err := m.BuildChatURL(id)
	if err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	if chatURL != "https://relay.example.com/v1/chat/completions" {
		t.Errorf("chatURL = %q", chatURL)
	}
	if apiKey != "sk-relay-123" {
		t.Errorf("BuildChatURL apiKey = %q, want sk-relay-123", apiKey)
	}
	if m.CustomEngineKey(id) != "sk-relay-123" {
		t.Error("CustomEngineKey 应返回内存明文 Key")
	}
}

// TestAddCustomEngine_SlugAndIDConflict slug 生成 + id 冲突追加 -2/-3。
func TestAddCustomEngine_SlugAndIDConflict(t *testing.T) {
	m := NewManager("", "")
	id1, err := m.AddCustomEngine("My Engine!!", "https://a.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if id1 != "custom-my-engine" {
		t.Fatalf("id1 = %q, want custom-my-engine（小写化 + 去非法字符）", id1)
	}
	id2, err := m.AddCustomEngine("My Engine!!", "https://b.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine 冲突: %v", err)
	}
	if id2 != "custom-my-engine-2" {
		t.Fatalf("id2 = %q, want custom-my-engine-2", id2)
	}
	id3, err := m.AddCustomEngine("My Engine!!", "https://c.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine 二次冲突: %v", err)
	}
	if id3 != "custom-my-engine-3" {
		t.Fatalf("id3 = %q, want custom-my-engine-3", id3)
	}
	// 纯非法字符（中文/符号）→ 空 slug 回退 "engine"
	id4, err := m.AddCustomEngine("测试引擎", "https://d.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine 中文名: %v", err)
	}
	if id4 != "custom-engine" {
		t.Fatalf("id4 = %q, want custom-engine", id4)
	}
	// 与现有 id 冲突（上一条已占 custom-engine）
	id5, err := m.AddCustomEngine("中文二号", "https://e.example.com/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if id5 != "custom-engine-2" {
		t.Fatalf("id5 = %q, want custom-engine-2", id5)
	}
}

// TestAddCustomEngine_RejectsInvalidInput 校验：空名拒绝；非法地址拒绝
// （含「把 API Key 当地址粘进来」——v4.9.1 Key 粘错框防线的延伸）。
func TestAddCustomEngine_RejectsInvalidInput(t *testing.T) {
	m := NewManager("", "")
	cases := []struct {
		nm      string
		name    string
		baseURL string
	}{
		{"空名", "", "https://ok.example.com/v1"},
		{"纯空白名", "   ", "https://ok.example.com/v1"},
		{"Key当地址", "合法名", "sk-proj-abcdef1234567890"},
		{"Key带Bearer前缀当地址", "合法名", "Bearer sk-proj-abcdef1234567890"},
		{"非http scheme", "合法名", "ftp://files.example.com/v1"},
		{"无host", "合法名", "http://"},
		{"无scheme", "合法名", "example.com:8080/v1"},
		{"纯文字", "合法名", "我的中转站地址"},
	}
	for _, tc := range cases {
		if _, err := m.AddCustomEngine(tc.name, tc.baseURL, ""); err == nil {
			t.Errorf("[%s] 应被拒绝（name=%q baseURL=%q）", tc.nm, tc.name, tc.baseURL)
		}
	}
	if es := m.GetEngines(); len(es) != 9 {
		t.Errorf("全部拒绝后引擎数量 = %d, want 9（未产生半成品引擎）", len(es))
	}
}

// TestAddCustomEngine_KeyNeverLeaks Key 不出现在状态文件快照与 GetEngines 输出。
func TestAddCustomEngine_KeyNeverLeaks(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	const secret = "sk-super-secret-9999"
	id, err := m.AddCustomEngine("Leak Probe", "https://probe.example.com/v1", secret)
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Error("状态文件泄漏自定义引擎 Key 明文")
	}
	var f stateFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("状态文件不是合法 JSON: %v", err)
	}
	if ce, ok := f.Engines[id]; ok && ce.APIKey != "" {
		t.Errorf("状态文件快照中自定义引擎 api_key = %q, want 空", ce.APIKey)
	}
	for _, e := range m.GetEngines() {
		if e.APIKey != "" {
			t.Errorf("GetEngines 泄漏 Key（引擎 %s）", e.ID)
		}
	}
}

// TestUpdateCustomEngine 全链路：改名/改地址/改 Key；空 Key = 保留原 Key。
func TestUpdateCustomEngine(t *testing.T) {
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Old Name", "https://old.example.com/v1", "sk-old-key")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}

	// 空 apiKey = 保留原 Key
	if err := m.UpdateCustomEngine(id, "New Name", "https://new.example.com/v1", ""); err != nil {
		t.Fatalf("UpdateCustomEngine(空Key): %v", err)
	}
	eng, _ := m.GetEngine(id)
	if eng.Label != "New Name" || eng.BaseURL != "https://new.example.com/v1" {
		t.Errorf("名称/地址未更新: %q/%q", eng.Label, eng.BaseURL)
	}
	if _, key := mustChatURL(t, m, id); key != "sk-old-key" {
		t.Errorf("空 apiKey 应保留原 Key, got %q", key)
	}

	// 非空 apiKey = 替换
	if err := m.UpdateCustomEngine(id, "New Name", "https://new.example.com/v1", "sk-new-key"); err != nil {
		t.Fatalf("UpdateCustomEngine(新Key): %v", err)
	}
	if _, key := mustChatURL(t, m, id); key != "sk-new-key" {
		t.Errorf("新 Key 未生效, got %q", key)
	}

	// 校验失败路径
	if err := m.UpdateCustomEngine(id, "", "https://new.example.com/v1", ""); err == nil {
		t.Error("空名称应拒绝")
	}
	if err := m.UpdateCustomEngine(id, "New Name", "sk-key-pasted-as-url", ""); err == nil {
		t.Error("Key 当地址应拒绝")
	}
	eng, _ = m.GetEngine(id)
	if eng.BaseURL != "https://new.example.com/v1" {
		t.Errorf("校验失败后 BaseURL 被改动: %q", eng.BaseURL)
	}

	// 不存在 / 非自定义引擎
	if err := m.UpdateCustomEngine("custom-ghost", "X", "https://x.example.com/v1", ""); err == nil {
		t.Error("更新不存在的引擎应报错")
	}
	if err := m.UpdateCustomEngine("xai", "X", "https://x.example.com/v1", ""); err == nil {
		t.Error("更新内置引擎应报错")
	}
}

// TestRemoveCustomEngine 删除链路：仅 custom- 前缀可删，删除后引擎/Key/顺序全清。
func TestRemoveCustomEngine(t *testing.T) {
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Doomed", "https://doom.example.com/v1", "sk-doom")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}

	if err := m.RemoveCustomEngine("xai"); err == nil {
		t.Error("删除内置引擎应拒绝")
	}
	if err := m.RemoveCustomEngine("custom-ghost"); err == nil {
		t.Error("删除不存在的自定义引擎应报错")
	}
	if err := m.RemoveCustomEngine(id); err != nil {
		t.Fatalf("RemoveCustomEngine: %v", err)
	}
	if _, ok := m.GetEngine(id); ok {
		t.Error("删除后 GetEngine 仍返回引擎")
	}
	for _, e := range m.GetEngines() {
		if e.ID == id {
			t.Error("删除后 GetEngines 仍包含引擎")
		}
	}
	if m.CustomEngineKey(id) != "" {
		t.Error("删除后内存 Key 未清")
	}
	if _, _, err := m.BuildChatURL(id); err == nil {
		t.Error("删除后 BuildChatURL 应报「引擎不存在」")
	}

	// 删除后可重建同名（id 不再冲突）
	reborn, err := m.AddCustomEngine("Doomed", "https://doom.example.com/v1", "")
	if err != nil {
		t.Fatalf("删除后重建: %v", err)
	}
	if reborn != id {
		t.Errorf("删除后重建 id = %q, want %q", reborn, id)
	}
}

// TestBuildChatURL_CustomEmptyKey 空 Key：URL 正常、Key 为空串
// （ai.Client 对空 Key 省略 Authorization 头，兼容无鉴权本地服务）。
func TestBuildChatURL_CustomEmptyKey(t *testing.T) {
	m := NewManager("", "")
	id, err := m.AddCustomEngine("Keyless", "http://127.0.0.1:9101/v1", "")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	chatURL, apiKey, err := m.BuildChatURL(id)
	if err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	if chatURL != "http://127.0.0.1:9101/v1/chat/completions" || apiKey != "" {
		t.Errorf("got (%q, %q), want URL 不变 + 空 Key", chatURL, apiKey)
	}
}

// TestRefreshModels_CustomEngineUsesKey 刷新模型走 OpenAI 兼容 /models，
// 带内存 Key；空 Key 不带 Authorization 头。
func TestRefreshModels_CustomEngineUsesKey(t *testing.T) {
	var lastAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		lastAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "chat-pro"}, {"id": "voice-tts"}},
		})
	}))
	t.Cleanup(srv.Close)

	m := NewManager("", "")
	id, err := m.AddCustomEngine("Relay", srv.URL, "sk-custom-abc")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	models, err := m.RefreshModels(context.Background(), id)
	if err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	if lastAuth != "Bearer sk-custom-abc" {
		t.Errorf("Authorization = %q, want Bearer sk-custom-abc", lastAuth)
	}
	if len(models) != 2 {
		t.Fatalf("模型数 = %d, want 2", len(models))
	}
	if models[0].Kind != "llm" || models[1].Kind != "tts" {
		t.Errorf("custom 引擎模型判型 = %q/%q, want llm/tts", models[0].Kind, models[1].Kind)
	}

	// 空 Key：请求不带 Authorization 头
	id2, err := m.AddCustomEngine("Keyless Relay", srv.URL, "")
	if err != nil {
		t.Fatalf("AddCustomEngine(无Key): %v", err)
	}
	if _, err := m.RefreshModels(context.Background(), id2); err != nil {
		t.Fatalf("RefreshModels(无Key): %v", err)
	}
	if lastAuth != "" {
		t.Errorf("空 Key 不应带 Authorization 头, got %q", lastAuth)
	}
}

// TestTestConnection_CustomEngine401 401 时给自定义引擎专属提示（不 panic）。
func TestTestConnection_CustomEngine401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	m := NewManager("", "")
	id, err := m.AddCustomEngine("BadKey", srv.URL, "sk-wrong")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	status, err := m.TestConnection(context.Background(), id)
	if err != nil {
		t.Fatalf("TestConnection 应把失败写进 status 而非返回 error: %v", err)
	}
	if status.Connected {
		t.Error("401 时 Connected 应为 false")
	}
	if !strings.Contains(status.Error, "自定义引擎") {
		t.Errorf("错误提示 = %q, want 含「自定义引擎」", status.Error)
	}
}

// TestClassifyModelKind_Custom custom 引擎复用通用关键词判型、默认 llm
// （刻意不加类型分支——行为锚定，防止未来重构把 custom 当特型）。
func TestClassifyModelKind_Custom(t *testing.T) {
	cases := map[string]string{
		"deepseek-v4-pro":   "llm", // 默认 llm
		"qwen3-32b":         "llm",
		"my-tts-service":    "tts",
		"text-embedding-v3": "embedding",
		"whisper-large":     "stt",
		"doc-ocr":           "ocr",
	}
	for modelID, want := range cases {
		if got := ClassifyModelKind(EngineCustom, modelID); got != want {
			t.Errorf("ClassifyModelKind(custom, %q) = %q, want %q", modelID, got, want)
		}
	}
}

// TestLoadState_RestoresCustomEngines 状态文件里的 custom- 条目重启后恢复；
// Key 与引擎分离：引擎从状态文件恢复，Key 由 app 层从 config 解密注入。
func TestLoadState_RestoresCustomEngines(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	id, err := m.AddCustomEngine("Persisted Engine", "https://persist.example.com/v1", "sk-persist")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := m.SetDefaultModel(id, "chat-pro"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}

	// 模拟重启：新 Manager（只种子内置引擎）+ 状态文件 + config 注入
	m2 := NewManager("", "")
	if err := m2.LoadState(statePath); err != nil {
		t.Fatalf("m2.LoadState: %v", err)
	}
	eng, ok := m2.GetEngine(id)
	if !ok {
		t.Fatal("重启后自定义引擎丢失")
	}
	if eng.Type != EngineCustom || eng.Name != "custom" || eng.Label != "Persisted Engine" {
		t.Errorf("恢复字段不全: %+v", eng)
	}
	if eng.BaseURL != "https://persist.example.com/v1" || !eng.Enabled || eng.DefaultModel != "chat-pro" {
		t.Errorf("恢复字段不全: %+v", eng)
	}
	// Key 未注入前：URL 正常、Key 为空
	if _, key, _ := m2.BuildChatURL(id); key != "" {
		t.Errorf("注入前 Key 应为空, got %q", key)
	}
	// app 层从 config 解密注入（此处模拟明文注入）
	m2.SetCustomEngineKeys(map[string]string{id: "sk-persist"})
	if _, key := mustChatURL(t, m2, id); key != "sk-persist" {
		t.Errorf("注入后 Key = %q, want sk-persist", key)
	}
	// 注入 nil = 清空（防串扰）
	m2.SetCustomEngineKeys(nil)
	if m2.CustomEngineKey(id) != "" {
		t.Error("SetCustomEngineKeys(nil) 应清空 customKeys")
	}
}

// TestLoadState_RejectsFakeCustomEntry 手改状态文件伪造的 custom- 条目不采纳。
func TestLoadState_RejectsFakeCustomEntry(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	state := map[string]any{
		"engines": map[string]any{
			"custom-fake-xai":   map[string]any{"id": "custom-fake-xai", "type": "xai", "enabled": true, "base_url": "https://x.example.com/v1"},
			"custom-fake-dirty": map[string]any{"id": "custom-fake-dirty", "type": "custom", "enabled": true, "base_url": "sk-key-as-url"},
		},
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	for _, ghost := range []string{"custom-fake-xai", "custom-fake-dirty"} {
		if _, ok := m.GetEngine(ghost); ok {
			t.Errorf("伪造条目 %s 不应被恢复", ghost)
		}
	}
}

func mustChatURL(t *testing.T, m *Manager, id string) (string, string) {
	t.Helper()
	url, key, err := m.BuildChatURL(id)
	if err != nil {
		t.Fatalf("BuildChatURL: %v", err)
	}
	return url, key
}
