package app

import (
	"encoding/json"
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

// ── 识图跟随聊天模型（2026-10-19：本地多模态主力在线时不另找第二个识图模型）──

// seedTianxuan 给天璇引擎植入双档目录（syncManagedCatalog 保存形状）。
func seedTianxuan(t *testing.T, c *core, running string) {
	t.Helper()
	models := []modelengine.ModelInfo{
		{ID: "qwen3.6-35b-a3b-uncensored", Kind: "llm", Status: "stopped"},
		{ID: "qwen3.8-flash-next-uncensored", Kind: "llm", Status: "stopped"},
	}
	if running != "" {
		for i := range models {
			if models[i].ID == running {
				models[i].Status = "running"
			}
		}
	}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "tianxuan", Enabled: true, Models: models}); err != nil {
		t.Fatalf("seedTianxuan: %v", err)
	}
}

// TestVisionBoundRuntime_FollowsServedTianxuanProfile 聊天引擎=天璇且 flash
// 档在服务：识图直接跟随该档（=聊天正用的多模态模型），压过指向别处的
// func_vision 独立绑定——单模型引擎只认当前服务档，绑另一档必 404。
func TestVisionBoundRuntime_FollowsServedTianxuanProfile(t *testing.T) {
	c := newTestCore(t)
	seedTianxuan(t, c, "qwen3.8-flash-next-uncensored")
	if err := c.SetFeatureModel("gaea", "tianxuan", "qwen3.8-flash-next-uncensored"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	// 识图独立绑定到 herdsman 另一模型：跟随优先，不另找第二个识图模型
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("天璇在服务档位时应跟随命中")
	}
	e, _ := c.engineMgr.GetEngine("tianxuan")
	if r.BaseURL != e.BaseURL {
		t.Errorf("BaseURL = %q, want 天璇端点 %q", r.BaseURL, e.BaseURL)
	}
	if r.Model != "qwen3.8-flash-next-uncensored" {
		t.Errorf("Model = %q, want 当前服务档 qwen3.8-flash-next-uncensored", r.Model)
	}
}

// TestVisionBoundRuntime_TianxuanDownFallsToHerdsmanDefault 天璇未运行（无
// running 档）：herdsman 可用（启用+默认 llm）时以其默认模型兜底——聊天此刻
// 多半已故障转移到 herdsman，识图就近复用，不再去找未服务的档位。
func TestVisionBoundRuntime_TianxuanDownFallsToHerdsmanDefault(t *testing.T) {
	c := newTestCore(t)
	seedTianxuan(t, c, "")
	if err := c.SetFeatureModel("gaea", "tianxuan", "qwen3.8-flash-next-uncensored"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	if err := c.engineMgr.SetDefaultModel("herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("herdsman 主力可用时应兜底命中")
	}
	e, _ := c.engineMgr.GetEngine("herdsman")
	if r.BaseURL != e.BaseURL {
		t.Errorf("BaseURL = %q, want herdsman 端点", r.BaseURL)
	}
	if r.Model != "qwen3-8b" {
		t.Errorf("Model = %q, want herdsman 默认模型 qwen3-8b", r.Model)
	}
}

// TestVisionBoundRuntime_FollowsHerdsmanChatModel 聊天引擎=herdsman：识图
// 同引擎同模型（已加载零冷启），压过云端识图独立绑定；离线模式不拦本地跟随。
func TestVisionBoundRuntime_FollowsHerdsmanChatModel(t *testing.T) {
	c := newTestCore(t)
	id, err := c.engineMgr.AddCustomEngine("云端识图", "https://api.example.com/v1", "sk-c")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("vision", id, "glm-4.5v"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	if err := c.SetFeatureModel("gaea", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	c.cfg.SetOfflineMode(true)
	defer c.cfg.SetOfflineMode(false)
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("herdsman 聊天应跟随命中")
	}
	if r.Model != "qwen3-8b" || r.APIKey != "" {
		t.Errorf("跟随结果 = (model %q, key %q), want (qwen3-8b, 空)", r.Model, r.APIKey)
	}
	if r.BaseURL == "https://api.example.com/v1" {
		t.Error("不应再去云端识图绑定（本地多模态主力在线）")
	}
}

// TestVisionBoundRuntime_HerdsmanChatNoModelAbstains 聊天绑 herdsman 但模型
// 与引擎默认均空：跟随弃权，回落 func_vision 绑定链。
func TestVisionBoundRuntime_HerdsmanChatNoModelAbstains(t *testing.T) {
	c := newTestCore(t)
	c.cfg.SetFeatureModel("gaea", "herdsman", "")
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("跟随弃权后应由 func_vision 绑定命中")
	}
	if r.Model != "qwen3-8b" {
		t.Errorf("Model = %q, want 绑定模型 qwen3-8b", r.Model)
	}
}

// TestVisionBoundRuntime_CloudChatKeepsVisionBinding 聊天在云端引擎：不跟随，
// func_vision 绑定照常生效（云端聊天时识图独立绑定才有意义）。
func TestVisionBoundRuntime_CloudChatKeepsVisionBinding(t *testing.T) {
	c := newTestCore(t)
	chatID, err := c.engineMgr.AddCustomEngine("云端聊天", "https://api.example.com/v1", "sk-chat")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("gaea", chatID, "deepseek-v4-pro"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	r, ok := c.visionBoundRuntime()
	if !ok {
		t.Fatal("云端聊天时应由 func_vision 绑定命中")
	}
	if r.Model != "qwen3-8b" {
		t.Errorf("Model = %q, want 绑定模型 qwen3-8b", r.Model)
	}
}

// ── VisionRoute 展示绑定（模型中心识图卡「当前生效」行，只读不记账）──

// parseVisionRoute 解析 VisionRoute 的 JSON 返回。
func parseVisionRoute(t *testing.T, s string) (engine, model, source string) {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("VisionRoute JSON: %v", err)
	}
	return m["engine"], m["model"], m["source"]
}

// TestVisionRoute_MirrorsBoundRuntimeChain 三态口径与 visionBoundRuntime 同链
// 同判定：跟随（chat-follow）/ 聊天引擎停机兜底（chat-follow-fallback）/
// 功能绑定（feature）/ 静态兜底（static）。
func TestVisionRoute_MirrorsBoundRuntimeChain(t *testing.T) {
	c := newTestCore(t)
	a := &App{core: c}

	// 聊天在云端 + 未绑识图：不跟随不绑定 → static（engine 不可知报空）。
	chatID, err := c.engineMgr.AddCustomEngine("云端聊天", "https://api.example.com/v1", "sk-chat")
	if err != nil {
		t.Fatalf("AddCustomEngine: %v", err)
	}
	if err := c.SetFeatureModel("gaea", chatID, "deepseek-v4-pro"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	eng, model, source := parseVisionRoute(t, mustVisionRoute(t, a))
	if source != "static" || eng != "" || model == "" {
		t.Errorf("static 态 = (%q,%q,%q), want (空, 非空模型, static)", eng, model, source)
	}

	// 聊天在云端 + 绑定识图：feature 层接管（云端聊天时独立绑定才有意义）。
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(vision): %v", err)
	}
	eng, model, source = parseVisionRoute(t, mustVisionRoute(t, a))
	if source != "feature" || eng != "herdsman" || model != "qwen3-8b" {
		t.Errorf("feature 态 = (%q,%q,%q), want (herdsman,qwen3-8b,feature)", eng, model, source)
	}

	// 聊天切天璇且档位在服务：跟随压过绑定。
	seedTianxuan(t, c, "qwen3.8-flash-next-uncensored")
	if err := c.SetFeatureModel("gaea", "tianxuan", "qwen3.8-flash-next-uncensored"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	eng, model, source = parseVisionRoute(t, mustVisionRoute(t, a))
	if source != "chat-follow" || eng != "tianxuan" || model != "qwen3.8-flash-next-uncensored" {
		t.Errorf("chat-follow 态 = (%q,%q,%q), want (tianxuan,当前服务档,chat-follow)", eng, model, source)
	}

	// 天璇停机 + herdsman 主力可用：chat-follow-fallback。
	seedTianxuan(t, c, "")
	if err := c.engineMgr.SetDefaultModel("herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetDefaultModel: %v", err)
	}
	eng, model, source = parseVisionRoute(t, mustVisionRoute(t, a))
	if source != "chat-follow-fallback" || eng != "herdsman" || model != "qwen3-8b" {
		t.Errorf("chat-follow-fallback 态 = (%q,%q,%q), want (herdsman,qwen3-8b,chat-follow-fallback)", eng, model, source)
	}
}

func mustVisionRoute(t *testing.T, a *App) string {
	t.Helper()
	s, err := a.VisionRoute()
	if err != nil {
		t.Fatalf("VisionRoute: %v", err)
	}
	return s
}

// TestVisionRoute_DoesNotAccountUsage 展示路径不记账：VisionRoute 轮询不得
// TouchEngine——否则 model_idle_exit_minutes 空闲收割器被前端刷新永远喂饱
// （记账点=BuildChatURL/启动/真实识图调用，探活/展示刻意不记账）。
func TestVisionRoute_DoesNotAccountUsage(t *testing.T) {
	c := newTestCore(t)
	a := &App{core: c}
	seedTianxuan(t, c, "qwen3.8-flash-next-uncensored")
	if err := c.SetFeatureModel("gaea", "tianxuan", "qwen3.8-flash-next-uncensored"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := a.VisionRoute(); err != nil {
			t.Fatalf("VisionRoute: %v", err)
		}
	}
	if at := c.engineMgr.EngineLastActivity("tianxuan"); !at.IsZero() {
		t.Errorf("VisionRoute 不应记账，tianxuan lastActivity = %v", at)
	}
	// 对照：真实识图调用路径（visionBoundRuntime）才记账。
	if _, ok := c.visionBoundRuntime(); !ok {
		t.Fatal("跟随命中应成功")
	}
	if at := c.engineMgr.EngineLastActivity("tianxuan"); at.IsZero() {
		t.Error("visionBoundRuntime 解析即记账，lastActivity 应非零")
	}
}
