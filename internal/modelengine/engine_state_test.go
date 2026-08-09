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

// ─── 稳定顺序 ────────────────────────────────────────────────

func TestGetEngines_StableOrder(t *testing.T) {
	m := NewManager("", "")
	want := []string{"xai", "ollama", "herdsman", "deepseek", "cosyvoice", "opencode-go", "opencode-zen"}
	for i := 0; i < 5; i++ {
		got := m.GetEngines()
		if len(got) != len(want) {
			t.Fatalf("第 %d 次 GetEngines 数量 = %d, want %d", i+1, len(got), len(want))
		}
		for j, id := range want {
			if got[j].ID != id {
				t.Fatalf("第 %d 次顺序不稳定: index %d = %q, want %q（全序列 %+v）",
					i+1, j, got[j].ID, id, idsOf(got))
			}
		}
	}
}

func idsOf(es []EngineConfig) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.ID
	}
	return out
}

// ─── 状态持久化 ──────────────────────────────────────────────

// TestStatePersistence_RoundTrip 验证：变更 → 落盘 → 新 Manager 恢复。
func TestStatePersistence_RoundTrip(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState(空文件) = %v, want nil", err)
	}

	// 变更：停用 ollama、改 BaseURL、设默认模型、带模型列表
	if err := m.SaveEngine(EngineConfig{
		ID:      "ollama",
		BaseURL: "http://192.168.1.10:11434/v1",
		Enabled: false,
		Models:  []ModelInfo{{ID: "qwen3", Status: "running"}, {ID: "llama3"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := m.SetDefaultModel("ollama", "qwen3"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}

	// 新 Manager 从同一文件恢复
	m2 := NewManager("", "")
	if err := m2.LoadState(statePath); err != nil {
		t.Fatalf("m2.LoadState = %v, want nil", err)
	}
	e, ok := m2.GetEngine("ollama")
	if !ok {
		t.Fatal("恢复后 ollama 引擎缺失")
	}
	if e.Enabled {
		t.Error("恢复后 Enabled 应为 false（已停用）")
	}
	if e.BaseURL != "http://192.168.1.10:11434/v1" {
		t.Errorf("恢复后 BaseURL = %q", e.BaseURL)
	}
	if e.DefaultModel != "qwen3" {
		t.Errorf("恢复后 DefaultModel = %q, want qwen3", e.DefaultModel)
	}
	if len(e.Models) != 2 || e.Models[0].ID != "qwen3" {
		t.Errorf("恢复后模型列表未还原: %+v", e.Models)
	}
}

// TestStatePersistence_ExcludesKeys 状态文件不得包含任何 API Key 明文。
func TestStatePersistence_ExcludesKeys(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	m := NewManager("", "")
	_ = m.LoadState(statePath)
	_ = m.SaveEngine(EngineConfig{ID: "ollama", Enabled: false})

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}
	for _, secret := range []string{"xai-tok", "ds-key-123"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("状态文件泄漏敏感串 %q", secret)
		}
	}
}

// TestLoadState_MissingFile 文件不存在时静默降级（首次启动）。
func TestLoadState_MissingFile(t *testing.T) {
	m := NewManager("", "")
	if err := m.LoadState(filepath.Join(t.TempDir(), "nope.json")); err != nil {
		t.Fatalf("LoadState(不存在) = %v, want nil", err)
	}
	// 预置引擎不受影响
	es := m.GetEngines()
	if len(es) != 7 {
		t.Fatalf("引擎数量 = %d, want 7", len(es))
	}
}

// TestLoadState_UnknownEngineIgnored 状态文件中的未知引擎 ID 不创建新引擎。
func TestLoadState_UnknownEngineIgnored(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	state := map[string]any{
		"engines": map[string]any{
			"ghost": map[string]any{"id": "ghost", "enabled": true},
		},
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	m := NewManager("", "")
	if err := m.LoadState(statePath); err != nil {
		t.Fatalf("LoadState = %v", err)
	}
	if _, ok := m.GetEngine("ghost"); ok {
		t.Error("未知引擎 ghost 不应被创建")
	}
}

// ─── 连接状态缓存 ────────────────────────────────────────────

// TestRefreshModels_UpdatesStatusCache 刷新模型后，GetEngines 应携带最近连接状态。
func TestRefreshModels_UpdatesStatusCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "m1"}, {"id": "m2"}},
		})
	}))
	t.Cleanup(srv.Close)

	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RefreshModels(context.Background(), "ollama"); err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}

	es := m.GetEngines()
	for _, e := range es {
		if e.ID != "ollama" {
			continue
		}
		if !e.Status.Connected {
			t.Error("刷新成功后 Status.Connected 应为 true")
		}
		if e.Status.ModelCount != 2 {
			t.Errorf("Status.ModelCount = %d, want 2", e.Status.ModelCount)
		}
		if e.Status.LastChecked == "" {
			t.Error("Status.LastChecked 应为非空")
		}
		return
	}
	t.Fatal("GetEngines 未返回 ollama")
}

// TestStatusCache_Persisted 连接状态随状态文件恢复。
func TestStatusCache_Persisted(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "engines.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"id": "ok"}}})
	}))
	t.Cleanup(srv.Close)

	m := NewManager("", "")
	_ = m.LoadState(statePath)
	_ = m.SaveEngine(EngineConfig{ID: "herdsman", BaseURL: srv.URL, Enabled: true})
	if _, err := m.TestConnection(context.Background(), "herdsman"); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}

	m2 := NewManager("", "")
	if err := m2.LoadState(statePath); err != nil {
		t.Fatalf("m2.LoadState: %v", err)
	}
	e, _ := m2.GetEngine("herdsman")
	if !e.Status.Connected {
		t.Error("恢复后 Status.Connected 应为 true")
	}
	if e.Status.ModelCount != 1 {
		t.Errorf("恢复后 ModelCount = %d, want 1", e.Status.ModelCount)
	}
}
