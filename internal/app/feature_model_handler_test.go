package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newTestCore 构造测试用 core：engineMgr 预置引擎 + 可写模型列表，配置写入临时目录。
func newTestCore(t *testing.T) *core {
	t.Helper()
	// config.Save 写 os.UserHomeDir() 下的 .gaea_config.json，重定向到临时目录
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	c := &core{cfg: &config.Config{Model: "grok-4.20"}, engineMgr: modelengine.NewManager("", "")}
	// 给 herdsman 引擎配模型列表，触发模型存在性校验分支
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "whisper-base"}, {ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	return c
}

// TestAppSetFeatureModel_GaeaBindingApplies 验证办公功能绑定经 App 层写入后，
// 配置正确更新；办公引擎未初始化时只注入 bridge、不重建、不 panic。
func TestAppSetFeatureModel_GaeaBindingApplies(t *testing.T) {
	c := newTestCore(t)
	a := &App{core: c}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:      "xai",
		Enabled: true,
		Models:  []modelengine.ModelInfo{{ID: "grok-4.6"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetFeatureModel("gaea", "xai", "grok-4.6"); err != nil {
		t.Fatalf("App.SetFeatureModel: %v", err)
	}
	eng, model := c.cfg.GetFeatureModel("gaea")
	if eng != "xai" || model != "grok-4.6" {
		t.Errorf("gaea 绑定 = (%q,%q), want (xai,grok-4.6)", eng, model)
	}
}

// TestSetFeatureModel_ValidationErrors 覆盖 SetFeatureModel 的校验分支
func TestSetFeatureModel_ValidationErrors(t *testing.T) {
	c := newTestCore(t)

	cases := []struct {
		name    string
		feature string
		engine  string
		model   string
		wantErr string
	}{
		{"未知功能", "foo", "herdsman", "qwen3-8b", "未知功能"},
		{"引擎不存在", "novel", "nope", "x", "引擎不存在"},
		{"模型不在列表", "novel", "herdsman", "grok-4.20", "不在引擎"},
	}
	// 禁用 xai 后测「引擎未启用」分支
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "xai", Enabled: false}); err != nil {
		t.Fatalf("禁用 xai: %v", err)
	}
	cases = append(cases, struct {
		name    string
		feature string
		engine  string
		model   string
		wantErr string
	}{"引擎未启用", "novel", "xai", "grok-4.20", "引擎未启用"})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.SetFeatureModel(tc.feature, tc.engine, tc.model)
			if err == nil {
				t.Fatalf("SetFeatureModel(%q,%q,%q) 期望报错，实际 nil", tc.feature, tc.engine, tc.model)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("错误 = %q，期望包含 %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestSetFeatureModel_RoundTrip 绑定 → 读回 → 持久化文件可查
func TestSetFeatureModel_RoundTrip(t *testing.T) {
	c := newTestCore(t)

	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}
	eng, model := c.featureModel("novel")
	if eng != "herdsman" || model != "qwen3-8b" {
		t.Errorf("featureModel = (%q,%q)，期望 (herdsman,qwen3-8b)", eng, model)
	}
	got := c.GetFeatureModel("novel")
	if got["engine"] != "herdsman" || got["model"] != "qwen3-8b" {
		t.Errorf("GetFeatureModel = %v", got)
	}

	// 未绑定功能保持空（走全局）
	if eng, model := c.featureModel("chat"); eng != "" || model != "" {
		t.Errorf("chat 未绑定却读到 (%q,%q)", eng, model)
	}
}

// TestSetFeatureModelEnabled_RoundTrip 功能级启停：停用 → 持久化 → 重新启用
func TestSetFeatureModelEnabled_RoundTrip(t *testing.T) {
	c := newTestCore(t)

	if err := c.SetFeatureModelEnabled("novel", false); err != nil {
		t.Fatalf("SetFeatureModelEnabled(false): %v", err)
	}
	if c.GetFeatureModelEnabled("novel") {
		t.Error("停用后 GetFeatureModelEnabled 应为 false")
	}
	// 持久化：临时 HOME 下配置文件可读回 false
	if cfg := config.Load(); cfg.GetFeatureModelEnabled("novel") {
		t.Error("持久化后配置文件应为停用")
	}

	if err := c.SetFeatureModelEnabled("novel", true); err != nil {
		t.Fatalf("SetFeatureModelEnabled(true): %v", err)
	}
	if !c.GetFeatureModelEnabled("novel") {
		t.Error("重新启用后应为 true")
	}
}

// TestSetFeatureModelEnabled_UnknownFeature 未知功能必须报错
func TestSetFeatureModelEnabled_UnknownFeature(t *testing.T) {
	c := newTestCore(t)
	if err := c.SetFeatureModelEnabled("foo", false); err == nil {
		t.Fatal("未知功能应报错，实际 nil")
	}
}

// TestSetFeatureModel_RebindReenables 重新绑定时功能应自动恢复启用
func TestSetFeatureModel_RebindReenables(t *testing.T) {
	c := newTestCore(t)
	if err := c.SetFeatureModelEnabled("novel", false); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFeatureModel("novel", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("重新绑定: %v", err)
	}
	if !c.GetFeatureModelEnabled("novel") {
		t.Error("重新绑定后功能应自动启用")
	}
}

// TestSetActiveASRModel_Validation 模型不在引擎列表时必须报错（防静默配置无效 ASR）
func TestSetActiveASRModel_Validation(t *testing.T) {
	c := newTestCore(t)
	m := &mediaState{core: c, app: &App{}}

	if err := m.SetActiveASRModel("herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("合法模型应通过: %v", err)
	}
	if err := m.SetActiveASRModel("herdsman", "not-a-model"); err == nil {
		t.Fatal("无效模型应报错，实际 nil")
	}
}

// TestGetModelMonitor 启用引擎进入监控列表，isLocal 标记本地/云端
func TestGetModelMonitor(t *testing.T) {
	c := newTestCore(t)
	a := &App{core: c}

	got := a.GetModelMonitor()
	engines, ok := got["engines"].([]map[string]interface{})
	if !ok {
		t.Fatalf("engines 类型异常: %T", got["engines"])
	}
	found := false
	for _, e := range engines {
		if e["engine"] == "herdsman" {
			found = true
			if e["isLocal"] != true {
				t.Errorf("herdsman isLocal = %v, want true（本地模型）", e["isLocal"])
			}
		}
		if e["engine"] == "xai" {
			if e["isLocal"] != false {
				t.Errorf("xai isLocal = %v, want false（云端 API 不占本机资源）", e["isLocal"])
			}
		}
	}
	if !found {
		t.Errorf("监控列表缺少 herdsman: %v", engines)
	}
	// comfyRunning 字段存在（测试环境 ComfyUI 未运行 → false）
	if _, exists := got["comfyRunning"]; !exists {
		t.Errorf("缺少 comfyRunning 字段")
	}
	if got["comfyRunning"] != false {
		t.Errorf("comfyRunning = %v, want false（测试环境未启动 ComfyUI）", got["comfyRunning"])
	}
}

// TestConfigSave_Concurrent 并发 Save 不同 key 不互相覆盖（lost update 防护）
func TestConfigSave_Concurrent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	const n = 20
	done := make(chan error, n*2)
	for i := 0; i < n; i++ {
		go func() {
			done <- config.Save(config.KeyFuncChatEngine, "herdsman")
			done <- config.Save(config.KeyFuncNovelModel, "qwen3-8b")
		}()
	}
	for i := 0; i < n*2; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	// 最终文件两个 key 都在（任一次 Save 都不该被覆盖丢字段）
	cfg := config.Load()
	if cfg.FuncChatEngine != "herdsman" {
		t.Errorf("FuncChatEngine = %q, want herdsman（并发 Save 丢字段）", cfg.FuncChatEngine)
	}
	if cfg.FuncNovelModel != "qwen3-8b" {
		t.Errorf("FuncNovelModel = %q, want qwen3-8b（并发 Save 丢字段）", cfg.FuncNovelModel)
	}
}
