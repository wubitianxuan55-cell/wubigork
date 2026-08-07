package app

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

func newRouterTestCore(t *testing.T) *core {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	c := &core{cfg: &config.Config{Model: "grok-4.20"}, engineMgr: modelengine.NewManager("", "")}
	engines := []modelengine.EngineConfig{
		{ID: "herdsman", Enabled: true, Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}}},
		{ID: "xai", Enabled: true, Models: []modelengine.ModelInfo{{ID: "grok-4.20"}}},
	}
	for _, e := range engines {
		if err := c.engineMgr.SaveEngine(e); err != nil {
			t.Fatalf("SaveEngine(%s): %v", e.ID, err)
		}
	}
	return c
}

// 功能绑定优先：绑定 novel → herdsman/qwen3-8b，即使 cfg.Model 是陈旧模型名。
func TestRouteModelFeatureBindingWins(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.Model = "stale-xai-model"
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	eng, model, source := c.routeModel("novel")
	if eng != "herdsman" || model != "qwen3-8b" || source != "feature" {
		t.Fatalf("route = (%q,%q,%q)", eng, model, source)
	}
}

// 未绑定 → 全局活跃（client 为空时 = xai + cfg.Model）。
func TestRouteModelGlobalFallback(t *testing.T) {
	c := newRouterTestCore(t)
	eng, model, source := c.routeModel("novel")
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)", eng, model, source)
	}
}

// 绑定引擎被禁用 → 降级全局。
func TestRouteModelDisabledBindingFallsBack(t *testing.T) {
	c := newRouterTestCore(t)
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	herd, _ := c.engineMgr.GetEngine("herdsman")
	herd.Enabled = false
	if err := c.engineMgr.SaveEngine(*herd); err != nil {
		t.Fatal(err)
	}
	eng, model, source := c.routeModel("novel")
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)", eng, model, source)
	}
}

// 全局活跃引擎也不可用 → 首个启用引擎兜底。
func TestRouteModelFirstEnabledFallback(t *testing.T) {
	c := newRouterTestCore(t)
	// NewManager 自带默认引擎（如 ollama 启用）：只保留 herdsman，验证"首个可用"兜底。
	for _, e := range c.engineMgr.GetEngines() {
		if e.ID == "herdsman" {
			continue
		}
		e.Enabled = false
		if err := c.engineMgr.SaveEngine(e); err != nil {
			t.Fatal(err)
		}
	}
	eng, _, source := c.routeModel("whisper")
	if eng != "herdsman" || source != "fallback" {
		t.Fatalf("route = (%q,%q)", eng, source)
	}
}

// 未知功能域：不 panic，降级链同全局。
func TestRouteModelUnknownFeatureFallsBack(t *testing.T) {
	c := newRouterTestCore(t)
	if _, _, source := c.routeModel("does-not-exist"); source != "global" {
		t.Fatalf("source = %q", source)
	}
}
