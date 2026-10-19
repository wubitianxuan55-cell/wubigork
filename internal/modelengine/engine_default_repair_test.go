package modelengine

// engine_default_repair_test.go — 悬空默认模型自愈（2026-10-19 办公 404 事故）：
// 模型清单整体替换的三条路径（TestConnection / RefreshModels / SaveEngine 目录
// 覆盖）都必须在默认模型不在新清单时接班首个 llm 档、无 llm 则清空；清单内的
// 用户选择（含 stopped 档位）不动。failover 候选放行清单内默认、拦下悬空默认。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// danglingModelsServer 返回 /v1/models 只含给定 id 的假引擎。
func danglingModelsServer(t *testing.T, ids ...string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		data := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			data = append(data, map[string]any{"id": id})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// 事故原值：herdsman 换装 MTP-r2 权重后退役的旧默认名。
const danglingStaleModel = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"

// TestRefreshModels_RepairsDanglingDefault 刷新后默认不在清单 → 接班首个 llm
// 档（清单首个是 embedding 也必须跳过）。
func TestRefreshModels_RepairsDanglingDefault(t *testing.T) {
	srv := danglingModelsServer(t, "bge-m3", "llm-a")
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true, DefaultModel: danglingStaleModel}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RefreshModels(context.Background(), "ollama"); err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	e, _ := m.GetEngine("ollama")
	if e.DefaultModel != "llm-a" {
		t.Errorf("悬空默认接班 = %q, want llm-a（首个 llm 档，跳过 bge-m3）", e.DefaultModel)
	}
}

// TestRefreshModels_KeepsDefaultInList 默认仍在清单 → 用户选择不动。
func TestRefreshModels_KeepsDefaultInList(t *testing.T) {
	srv := danglingModelsServer(t, "llm-a", "llm-b")
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true, DefaultModel: "llm-b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RefreshModels(context.Background(), "ollama"); err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	e, _ := m.GetEngine("ollama")
	if e.DefaultModel != "llm-b" {
		t.Errorf("清单内默认被改写 = %q, want llm-b", e.DefaultModel)
	}
}

// TestRefreshModels_ClearsDefaultWhenNoLLM 新清单无 llm 档 → 清空（与
// RemoveModel 防悬空口径一致，宁空勿错）。
func TestRefreshModels_ClearsDefaultWhenNoLLM(t *testing.T) {
	srv := danglingModelsServer(t, "bge-m3")
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "ollama", BaseURL: srv.URL, Enabled: true, DefaultModel: "llm-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RefreshModels(context.Background(), "ollama"); err != nil {
		t.Fatalf("RefreshModels: %v", err)
	}
	e, _ := m.GetEngine("ollama")
	if e.DefaultModel != "" {
		t.Errorf("无 llm 可接班时默认 = %q, want 空", e.DefaultModel)
	}
}

// TestConnection_RepairsDanglingDefault 连接测试同样整体替换清单，同口径修复。
func TestConnection_RepairsDanglingDefault(t *testing.T) {
	srv := danglingModelsServer(t, "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2")
	m := NewManager("", "")
	if err := m.SaveEngine(EngineConfig{ID: "herdsman", BaseURL: srv.URL, Enabled: true, DefaultModel: danglingStaleModel}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.TestConnection(context.Background(), "herdsman"); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	e, _ := m.GetEngine("herdsman")
	if e.DefaultModel != "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2" {
		t.Errorf("悬空默认接班 = %q, want MTP-r2", e.DefaultModel)
	}
}

// TestSaveEngine_CatalogReplacementRepairsDangling 目录同步整体覆盖清单
// （app.syncManagedEngineCatalog 的保存形状：只带 ID/Enabled/Models）后：
// 悬空默认接班；清单内默认（含 stopped 档位，如天璇未启动的另一档）保留。
func TestSaveEngine_CatalogReplacementRepairsDangling(t *testing.T) {
	m := NewManager("", "")
	// Strata 收窄前：flash iq2_xs 档在册且为默认。
	if err := m.SaveEngine(EngineConfig{ID: "strata", Enabled: true,
		Models:       []ModelInfo{{ID: "qwen3.8-flash-next-iq2_xs", Kind: "llm"}},
		DefaultModel: "qwen3.8-flash-next-iq2_xs"}); err != nil {
		t.Fatal(err)
	}
	// 目录收窄：只留 uncensored 档（同步保存不带 DefaultModel）。
	if err := m.SaveEngine(EngineConfig{ID: "strata", Enabled: true,
		Models: []ModelInfo{{ID: "qwen3.6-35b-a3b-uncensored-q4_k_p", Kind: "llm", Status: "stopped"}}}); err != nil {
		t.Fatal(err)
	}
	e, _ := m.GetEngine("strata")
	if e.DefaultModel != "qwen3.6-35b-a3b-uncensored-q4_k_p" {
		t.Errorf("收窄后悬空默认 = %q, want qwen3.6-35b-a3b-uncensored-q4_k_p", e.DefaultModel)
	}

	// 天璇双档目录：默认指向未启动档（stopped）必须保留——启动档位≠改绑定。
	if err := m.SaveEngine(EngineConfig{ID: "tianxuan", Enabled: true,
		Models: []ModelInfo{
			{ID: "qwen3.6-35b-a3b-uncensored", Kind: "llm", Status: "running"},
			{ID: "qwen3.8-flash-next-uncensored", Kind: "llm", Status: "stopped"},
		},
		DefaultModel: "qwen3.8-flash-next-uncensored"}); err != nil {
		t.Fatal(err)
	}
	e, _ = m.GetEngine("tianxuan")
	if e.DefaultModel != "qwen3.8-flash-next-uncensored" {
		t.Errorf("清单内 stopped 档默认被改写 = %q, want qwen3.8-flash-next-uncensored", e.DefaultModel)
	}
}

// TestFailoverCandidates_SkipsDanglingDefault 悬空默认的引擎不做故障转移
// 候选（直发必 404 model_not_found）；修好默认或清单为空（无从校验）恢复旧口径。
func TestFailoverCandidates_SkipsDanglingDefault(t *testing.T) {
	m := NewManager("", "")
	m.mu.Lock()
	h := m.engines["herdsman"]
	h.Status.Connected = true
	h.DefaultModel = danglingStaleModel
	h.Models = []ModelInfo{{ID: "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2", Kind: "llm"}}
	o := m.engines["ollama"]
	o.Status.Connected = true
	o.DefaultModel = "llm-x" // 清单为空：无从校验，维持旧口径放行
	m.mu.Unlock()

	if got := m.FailoverCandidates(""); len(got) != 1 || got[0].ID != "ollama" {
		t.Fatalf("候选 = %v, want 仅 ollama（herdsman 悬空默认应被拦下）", idsOf(got))
	}

	m.mu.Lock()
	m.engines["herdsman"].DefaultModel = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2"
	m.mu.Unlock()
	got := m.FailoverCandidates("")
	if len(got) != 2 || got[0].ID != "ollama" || got[1].ID != "herdsman" {
		t.Fatalf("默认回清单后候选 = %v, want [ollama herdsman]（按 order 序）", idsOf(got))
	}
}
