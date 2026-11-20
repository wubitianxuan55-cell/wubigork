package app

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestGaeaRoutedContextWindowUsesDeepSeekCapability(t *testing.T) {
	a := modelsTestApp(t)
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "deepseek",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
		Models: []modelengine.ModelInfo{
			{ID: "deepseek-v4-pro", Kind: "llm", ContextLength: 1_000_000},
			{ID: "deepseek-v4-flash", Kind: "llm", ContextLength: 1_000_000},
		},
	}); err != nil {
		t.Fatal(err)
	}
	a.client.SetActiveEngine("deepseek")

	if got := a.gaeaRoutedContextWindow(256_000); got != 1_000_000 {
		t.Fatalf("DeepSeek office window = %d, want 1000000", got)
	}

	a.cfg.SetFeatureModel("gaea", "deepseek", "deepseek-v4-flash")
	a.cfg.SetFeatureModelEnabled("gaea", true)
	if got := a.gaeaRoutedContextWindow(256_000); got != 1_000_000 {
		t.Fatalf("DeepSeek feature-bound office window = %d, want 1000000", got)
	}
}

func TestGaeaRoutedContextWindowFallsBackWhenCapabilityUnknown(t *testing.T) {
	a := modelsTestApp(t)
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "deepseek",
		Enabled:      true,
		DefaultModel: "unknown",
		Models:       []modelengine.ModelInfo{{ID: "unknown", Kind: "llm"}},
	}); err != nil {
		t.Fatal(err)
	}
	a.client.SetActiveEngine("deepseek")

	if got := a.gaeaRoutedContextWindow(256_000); got != 256_000 {
		t.Fatalf("unknown capability window = %d, want fallback 256000", got)
	}
}

func TestGaeaRoutedContextWindowDeepSeekV4WithoutCatalogCapability(t *testing.T) {
	a := modelsTestApp(t)
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "deepseek",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
		Models:       []modelengine.ModelInfo{{ID: "deepseek-v4-pro", Kind: "llm"}},
	}); err != nil {
		t.Fatal(err)
	}
	a.client.SetActiveEngine("deepseek")

	if got := a.gaeaRoutedContextWindow(256_000); got != 1_000_000 {
		t.Fatalf("DeepSeek v4 fallback window = %d, want 1000000", got)
	}

	// 模型中心的功能绑定使用 provider alias（deepseek-flash），不是模型 ID。
	a.cfg.SetFeatureModel("gaea", "deepseek", "deepseek-flash")
	a.cfg.SetFeatureModelEnabled("gaea", true)
	if got := a.gaeaRoutedContextWindow(256_000); got != 1_000_000 {
		t.Fatalf("DeepSeek alias window = %d, want 1000000", got)
	}
}

func TestGaeaContextHealsStaleControllerWindow(t *testing.T) {
	a := modelsTestApp(t)
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "deepseek",
		Enabled:      true,
		DefaultModel: "deepseek-v4-pro",
		Models:       []modelengine.ModelInfo{{ID: "deepseek-v4-pro", Kind: "llm", ContextLength: 1_000_000}},
	}); err != nil {
		t.Fatal(err)
	}
	a.client.SetActiveEngine("deepseek")

	exec := agent.New(nil, nil, agent.NewSession("sys"), agent.Options{
		Compaction: agent.CompactionConfig{Window: 256_000, Ratio: 0.8, RecentKeep: 3},
	}, event.Discard)
	ctrl := control.New(control.Options{Executor: exec, Sink: event.Discard})
	ga.mu.Lock()
	old := ga.ctrl
	ga.ctrl = ctrl
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.ctrl = old
		ga.mu.Unlock()
	})

	if got := a.GaeaContext(); got.Window != 1_000_000 {
		t.Fatalf("GaeaContext window = %d, want 1000000", got.Window)
	}
	if window := exec.ContextWindow(); window != 1_000_000 {
		t.Fatalf("controller window after heal = %d, want 1000000", window)
	}
}
