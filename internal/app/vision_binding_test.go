package app

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/modelengine"
)

// TestVisionBoundRuntime_UnboundFallsThrough 未绑定时解析器弃权：识图回落
// gaea.toml [vision] 静态注入（默认本机 herdsman 视觉模型），既有口径零变化。
func TestVisionBoundRuntime_UnboundFallsThrough(t *testing.T) {
	c := newTestCore(t)
	if _, ok := c.visionBoundRuntime(); ok {
		t.Error("未绑定应弃权（ok=false），实际命中")
	}
}

// TestVisionBoundRuntime_BoundResolves 绑定生效：解析器返回绑定引擎的
// OpenAI 兼容端点 + 绑定模型；本地引擎无 Key。
func TestVisionBoundRuntime_BoundResolves(t *testing.T) {
	c := newTestCore(t)
	// newTestCore 的 herdsman 种子（恒 Enabled，本地 8080 端点）+ 模型清单
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("绑定后应命中（ok=true）")
	}
	if r.BaseURL == "" {
		t.Error("BaseURL 应取绑定引擎地址")
	}
	if r.Model != "qwen3-8b" {
		t.Errorf("Model = %q, want qwen3-8b（绑定模型）", r.Model)
	}
	if r.APIKey != "" {
		t.Errorf("本地引擎 APIKey = %q, want 空", r.APIKey)
	}
	if r.Kind != vision.VisionProviderKindOpenAI {
		t.Errorf("Kind = %q, want openai（App 侧显式给 kind）", r.Kind)
	}
}

// TestVisionBoundRuntime_CloudEngineCarriesKey 云端自定义引擎绑定：Key 经
// EngineAPIKey 同源取用（vision.Provider 据此带 Bearer 头）。
func TestVisionBoundRuntime_CloudEngineCarriesKey(t *testing.T) {
	c := newTestCore(t)
	id, err := c.engineMgr.AddCustomEngine("云端识图", "https://api.example.com/v1", "sk-cloud-1")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("vision", id, "glm-4.5v"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("云端绑定应命中")
	}
	if r.APIKey != "sk-cloud-1" {
		t.Errorf("APIKey = %q, want sk-cloud-1", r.APIKey)
	}
	if r.BaseURL != "https://api.example.com/v1" {
		t.Errorf("BaseURL = %q", r.BaseURL)
	}
}

// TestVisionBoundRuntime_DisabledAbstains 停用即回退：与未绑定同口径。
func TestVisionBoundRuntime_DisabledAbstains(t *testing.T) {
	c := newTestCore(t)
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("前置绑定: %v", err)
	}
	if err := c.SetFeatureModelEnabled("vision", false); err != nil {
		t.Fatalf("停用: %v", err)
	}
	if _, ok := c.visionBoundRuntime(); ok {
		t.Error("停用后应弃权（回退静态注入）")
	}
}

// TestVisionBoundRuntime_MissingOrDisabledEngineAbstains 绑定的引擎消失/停用：
// 弃权回退，不用半失效端点亮红识图链路。
func TestVisionBoundRuntime_MissingOrDisabledEngineAbstains(t *testing.T) {
	c := newTestCore(t)
	id, err := c.engineMgr.AddCustomEngine("临时云端", "https://api.example.com/v1", "sk-x")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("vision", id, "m1"); err != nil {
		t.Fatalf("前置绑定: %v", err)
	}
	// 引擎停用
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{ID: id, Enabled: false}); err != nil {
		t.Fatalf("停用引擎: %v", err)
	}
	if _, ok := c.visionBoundRuntime(); ok {
		t.Error("引擎停用后应弃权")
	}
	// 引擎删除
	if err := c.engineMgr.RemoveCustomEngine(id); err != nil {
		t.Fatalf("RemoveCustomEngine: %v", err)
	}
	if _, ok := c.visionBoundRuntime(); ok {
		t.Error("引擎删除后应弃权")
	}
}

// TestVisionBoundRuntime_OfflineModeBlocksCloud 全局离线模式（数据不出本机
// 总闸）：云端绑定被拦（弃权回退本地静态注入），本地绑定不受影响——与
// routeModel 同一口径。
func TestVisionBoundRuntime_OfflineModeBlocksCloud(t *testing.T) {
	c := newTestCore(t)
	id, err := c.engineMgr.AddCustomEngine("云端识图", "https://api.example.com/v1", "sk-c")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("vision", id, "m1"); err != nil {
		t.Fatalf("前置绑定: %v", err)
	}
	c.cfg.SetOfflineMode(true)
	if _, ok := c.visionBoundRuntime(); ok {
		t.Error("离线模式下云端绑定应弃权")
	}
	// 换绑本地引擎：离线模式不拦本地
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("换绑本地: %v", err)
	}
	if _, ok := c.visionBoundRuntime(); !ok {
		t.Error("离线模式下本地绑定应命中")
	}
	c.cfg.SetOfflineMode(false)
}

// TestVisionBoundRuntime_FeatureModelKeysRegistered 功能注册表闭环：
// featureModelKeys/enabledKey 认识 vision，GetModelRoute 可解析（前端绑定卡
// 的「当前生效」行依赖它）。
func TestVisionBoundRuntime_FeatureModelKeysRegistered(t *testing.T) {
	if engineKey, modelKey, ok := featureModelKeys("vision"); !ok ||
		engineKey != config.KeyFuncVisionEngine || modelKey != config.KeyFuncVisionModel {
		t.Errorf("featureModelKeys(vision) = (%q,%q,%v)", engineKey, modelKey, ok)
	}
	if key, ok := featureModelEnabledKey("vision"); !ok || key != config.KeyFuncVisionEnabled {
		t.Errorf("featureModelEnabledKey(vision) = (%q,%v)", key, ok)
	}
}
