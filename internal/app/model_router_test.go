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

// 功能级停用（FeatureModelBar「停用」语义）：绑定仍在但停用 → 路由不得用 feature 绑定。
func TestRouteModelFeatureDisabledFallsBack(t *testing.T) {
	c := newRouterTestCore(t)
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFeatureModelEnabled("novel", false); err != nil {
		t.Fatal(err)
	}
	eng, model, source := c.routeModel("novel")
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("停用后 route = (%q,%q,%q)，期望全局 (xai,grok-4.20,global)", eng, model, source)
	}
}

// 功能停用后重新绑定 → 立即恢复 feature 路由。
func TestRouteModelFeatureReenabledByRebind(t *testing.T) {
	c := newRouterTestCore(t)
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFeatureModelEnabled("novel", false); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatal(err)
	}
	_, model, source := c.routeModel("novel")
	if model != "qwen3-8b" || source != "feature" {
		t.Fatalf("重新绑定后 route = (%q,%q)，期望 (qwen3-8b,feature)", model, source)
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

// ── S2-4 敏感域本地化路由 ──────────────────────────────────

// enableHerdsmanBaseURL 给测试夹具的 herdsman 引擎补 BaseURL（routeSensitiveLocal 要求）。
func enableHerdsmanBaseURL(t *testing.T, c *core, enabled bool) {
	t.Helper()
	herd, ok := c.engineMgr.GetEngine("herdsman")
	if !ok {
		t.Fatal("herdsman engine missing")
	}
	herd.BaseURL = "http://127.0.0.1:8080"
	herd.Enabled = enabled
	if err := c.engineMgr.SaveEngine(*herd); err != nil {
		t.Fatal(err)
	}
}

// 开关开启 + herdsman 可用 → 强制本地（sensitive-local），无视常规功能绑定。
func TestRouteSensitiveLocalForcesHerdsman(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, true)
	c.cfg.SensitiveLocal = true
	// 常规路由先绑定 novel → herdsman/qwen3-8b（feature），敏感路由仍应覆盖。
	if err := c.SetFeatureModel("office", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	eng, model, source := c.routeSensitiveLocal("office")
	if eng != "herdsman" || model != "qwen3-8b" || source != "sensitive-local" {
		t.Fatalf("route = (%q,%q,%q)，期望 (herdsman,qwen3-8b,sensitive-local)", eng, model, source)
	}
}

// 开关开启但 herdsman 停用 → 回退常规路由（全局 xai）。
func TestRouteSensitiveLocalHerdsmanDisabledFallsBack(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, false)
	c.cfg.SensitiveLocal = true
	eng, model, source := c.routeSensitiveLocal("office")
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)，期望回退全局 (xai,grok-4.20,global)", eng, model, source)
	}
}

// 开关关闭 → 走常规路由（可回云端），source 非 sensitive-local。
func TestRouteSensitiveLocalOffUsesNormalRoute(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, true)
	c.cfg.SensitiveLocal = false
	eng, model, source := c.routeSensitiveLocal("office")
	if source == "sensitive-local" {
		t.Fatalf("开关关闭仍走了 sensitive-local: (%q,%q,%q)", eng, model, source)
	}
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)，期望常规全局", eng, model, source)
	}
}

// 默认值：未显式配置时 GetSensitiveLocal() 返回 true（D8 默认本地优先）。
// 完整持久化往返见 internal/config/config_test.go TestSave_SensitiveLocalRoundTrip。
func TestSensitiveLocalDefaultOn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := config.Load()
	if !cfg.GetSensitiveLocal() {
		t.Fatal("SensitiveLocal 默认应为 true")
	}
}

// ── v4.8 全局离线模式（offline_mode）路由门控 ──────────────────

// 离线开启：云端功能绑定/全局被跳过，落回首个本地引擎（注册序 ollama 在前）；
// 本地引擎全部停用则路由为空。
func TestRouteModelOfflineMode(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.OfflineMode = true

	// 全局活跃 = xai（云端）→ 应被跳过，落回首个本地引擎（ollama 注册序在前）
	eng, _, source := c.routeModel("chat")
	if eng != "ollama" || source != "fallback" {
		t.Fatalf("离线模式 route = (%q,%q)，期望落回本地 ollama/fallback", eng, source)
	}

	// 功能绑定云端引擎 → 同样被跳过
	if err := c.SetFeatureModel("office", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	eng, _, _ = c.routeModel("office")
	if eng != "ollama" {
		t.Fatalf("离线模式云端功能绑定应被跳过，实际 %q", eng)
	}

	// 停用全部本地引擎 → 无可用路由（调用方按模型不可用降级）
	for _, id := range []string{"ollama", "herdsman", "cosyvoice"} {
		if e, ok := c.engineMgr.GetEngine(id); ok {
			e.Enabled = false
			if err := c.engineMgr.SaveEngine(*e); err != nil {
				t.Fatal(err)
			}
		}
	}
	eng, model, source := c.routeModel("chat")
	if eng != "" || model != "" || source != "" {
		t.Fatalf("离线且无本地引擎应返回空，实际 (%q,%q,%q)", eng, model, source)
	}
}

// 离线关闭（默认）：云端路由行为与既往一致（回归保护）。
func TestRouteModelOfflineOffKeepsCloud(t *testing.T) {
	c := newRouterTestCore(t)
	c.cfg.OfflineMode = false
	eng, _, source := c.routeModel("chat")
	if eng != "xai" || source != "global" {
		t.Fatalf("默认（离线关）route = (%q,%q)，期望 (xai,global)", eng, source)
	}
}

// ── 2026-08-28 办公本地优先路由 ──────────────────────────────────

// 开关开启 + herdsman 可用 → 办公功能级调用强制本地（office-local），
// 无视常规功能绑定（与 sensitive-local 同构）。
func TestRouteOfficeLocalForcesHerdsman(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, true)
	c.cfg.OfficeLocal = true
	if err := c.SetFeatureModel("office", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	eng, model, source := c.routeOfficeLocal("office")
	if eng != "herdsman" || model != "qwen3-8b" || source != "office-local" {
		t.Fatalf("route = (%q,%q,%q)，期望 (herdsman,qwen3-8b,office-local)", eng, model, source)
	}
}

// 开关开启但 herdsman 停用 → 回退常规路由（全局 xai）。
func TestRouteOfficeLocalHerdsmanDisabledFallsBack(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, false)
	c.cfg.OfficeLocal = true
	eng, model, source := c.routeOfficeLocal("office")
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)，期望回退全局 (xai,grok-4.20,global)", eng, model, source)
	}
}

// 开关关闭 → 走常规路由（可回云端），source 非 office-local。
func TestRouteOfficeLocalOffUsesNormalRoute(t *testing.T) {
	c := newRouterTestCore(t)
	enableHerdsmanBaseURL(t, c, true)
	c.cfg.OfficeLocal = false
	eng, model, source := c.routeOfficeLocal("office")
	if source == "office-local" {
		t.Fatalf("开关关闭仍走了 office-local: (%q,%q,%q)", eng, model, source)
	}
	if eng != "xai" || model != "grok-4.20" || source != "global" {
		t.Fatalf("route = (%q,%q,%q)，期望常规全局", eng, model, source)
	}
}

// 默认值：未显式配置时 GetOfficeLocal() 返回 true（办公本地优先默认开启）。
func TestOfficeLocalDefaultOn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := config.Load()
	if !cfg.GetOfficeLocal() {
		t.Fatal("OfficeLocal 默认应为 true")
	}
}
