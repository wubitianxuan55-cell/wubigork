package app

// T5-3a/b/c 本地模型调度纵深单测：estimateModelSwitch 全分支、保活开关跳过与
// 探针行为、自动预载选择逻辑与启动路径、换模预计等待各档位。全部隔离在
// 内存构造（herdsmanCLI / keepWarmProbe / autoPreloadStartModel 可注入替身），
// 不依赖本机 herdsman 安装，不发真实 HTTP 请求。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// scheduleTestApp 构造带 cfg/engineMgr/officeState 的最小 App（测试替身）。
func scheduleTestApp(cfg *config.Config) *App {
	return &App{
		core:        &core{cfg: cfg, engineMgr: modelengine.NewManager("", "")},
		officeState: &officeState{},
	}
}

// catalogResult 构造 herdsmanCLI 替身：返回单条模型记录。
func catalogResult(name string, installed, running bool) string {
	return `{"ok":true,"result":[{"name":"` + name + `","installed":` + boolStr(installed) + `,"running":` + boolStr(running) + `}]}`
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ─── T5-3c estimateModelSwitch 全分支 ──────────────────────

func TestEstimateModelSwitch(t *testing.T) {
	cases := []struct {
		name       string
		installed  bool
		running    bool
		wantStatus string
		wantWait   int
	}{
		{"运行中→hot", true, true, "hot", 1},
		{"已安装未运行→cold", true, false, "cold", 20},
		{"未安装→download", false, false, "download", 0},
		{"未安装但运行中→hot（running 优先）", false, true, "hot", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, wait := estimateModelSwitch(c.installed, c.running)
			if status != c.wantStatus || wait != c.wantWait {
				t.Errorf("estimateModelSwitch(%v,%v) = (%q,%d), want (%q,%d)",
					c.installed, c.running, status, wait, c.wantStatus, c.wantWait)
			}
		})
	}
}

// ─── T5-3a keep-warm ───────────────────────────────────────

// TestKeepWarmDisabledSkipsRound 开关关闭：整轮跳过（不查目录、不发探针）。
func TestKeepWarmDisabledSkipsRound(t *testing.T) {
	a := scheduleTestApp(&config.Config{KeepWarmEnabled: false})

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		t.Fatal("开关关闭时不应调用 herdsman CLI 查模型目录")
		return nil, errors.New("unexpected")
	}

	var probed []string
	oldProbe := keepWarmProbe
	defer func() { keepWarmProbe = oldProbe }()
	keepWarmProbe = func(_ context.Context, _, model string) error {
		probed = append(probed, model)
		return nil
	}

	statuses := a.keepWarmRound()
	if len(statuses) != 0 {
		t.Errorf("开关关闭应整轮跳过，got %v", statuses)
	}
	if len(probed) != 0 {
		t.Errorf("开关关闭不应发起探针，got %v", probed)
	}
}

// TestKeepWarmProbesRunningModels 只探 catalog 中 Running 的模型，
// 成功后记录 lastKeepAliveAt。
func TestKeepWarmProbesRunningModels(t *testing.T) {
	a := scheduleTestApp(&config.Config{KeepWarmEnabled: true})

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(`{"ok":true,"result":[
			{"name":"m1","installed":true,"running":true},
			{"name":"m2","installed":true,"running":false}
		]}`), nil
	}

	var probed []string
	oldProbe := keepWarmProbe
	defer func() { keepWarmProbe = oldProbe }()
	keepWarmProbe = func(_ context.Context, baseURL, model string) error {
		probed = append(probed, model)
		return nil
	}

	statuses := a.keepWarmRound()
	if len(probed) != 1 || probed[0] != "m1" {
		t.Fatalf("只应探 running 模型 m1，实际探了 %v", probed)
	}
	if statuses["m1"] != "ok" {
		t.Errorf("m1 探针成功应为 ok，got %q", statuses["m1"])
	}
	if _, ok := statuses["m2"]; ok {
		t.Error("m2 未运行，不应出现在状态里")
	}
	if a.lastKeepAlive("m1") == "" {
		t.Error("探针成功后应记录 lastKeepAliveAt")
	}
	if a.lastKeepAlive("m2") != "" {
		t.Error("未运行模型不应记录 lastKeepAliveAt")
	}
}

// TestKeepWarmProbeFailureSkipped 探针失败：记 fail、不记 lastKeepAlive，
// 待下一轮 catalog 重新 Running 再探（不主动 start）。
func TestKeepWarmProbeFailureSkipped(t *testing.T) {
	a := scheduleTestApp(&config.Config{KeepWarmEnabled: true})

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(catalogResult("m1", true, true)), nil
	}

	oldProbe := keepWarmProbe
	defer func() { keepWarmProbe = oldProbe }()
	keepWarmProbe = func(_ context.Context, _, _ string) error {
		return errors.New("模型已卸载")
	}

	statuses := a.keepWarmRound()
	if statuses["m1"] != "fail" {
		t.Errorf("探针失败应为 fail，got %q", statuses["m1"])
	}
	if a.lastKeepAlive("m1") != "" {
		t.Error("探针失败不应记录 lastKeepAliveAt")
	}
}

// ─── T5-3b 自动预载 ────────────────────────────────────────

// TestPreloadTargetPriority 优先级 gaea→office→chat，且只认 herdsman 引擎。
func TestPreloadTargetPriority(t *testing.T) {
	// gaea 优先：三域都绑 herdsman 时选 gaea。
	cfg := &config.Config{
		FuncGaeaEngine:    "herdsman",
		FuncGaeaModel:     "gaea-m",
		FuncOfficeEngine:  "herdsman",
		FuncOfficeModel:   "office-m",
		FuncChatEngine:    "herdsman",
		FuncChatModel:     "chat-m",
	}
	model, ok := preloadTarget(cfg)
	if !ok || model != "gaea-m" {
		t.Fatalf("preloadTarget = (%q,%v), want (gaea-m,true)", model, ok)
	}

	// gaea 非 herdsman → 回退 office。
	cfg = &config.Config{
		FuncGaeaEngine:   "xai",
		FuncGaeaModel:    "grok-4.20",
		FuncOfficeEngine: "herdsman",
		FuncOfficeModel:  "office-m",
	}
	model, ok = preloadTarget(cfg)
	if !ok || model != "office-m" {
		t.Fatalf("preloadTarget = (%q,%v), want (office-m,true)", model, ok)
	}

	// 只剩 chat 绑定 herdsman。
	cfg = &config.Config{
		FuncChatEngine: "herdsman",
		FuncChatModel:  "chat-m",
	}
	model, ok = preloadTarget(cfg)
	if !ok || model != "chat-m" {
		t.Fatalf("preloadTarget = (%q,%v), want (chat-m,true)", model, ok)
	}

	// 全部非 herdsman / 未绑定 → 不选。
	cfg = &config.Config{FuncGaeaEngine: "xai", FuncGaeaModel: "grok-4.20"}
	if model, ok = preloadTarget(cfg); ok {
		t.Errorf("非 herdsman 绑定不应选中，got %q", model)
	}
	if model, ok = preloadTarget(&config.Config{}); ok {
		t.Errorf("未绑定时不应选中，got %q", model)
	}
}

// TestAutoPreloadDecision 决策全分支：运行中/未安装/未找到 → 跳过，
// 已安装且未运行 → 预载。
func TestAutoPreloadDecision(t *testing.T) {
	catalog := HerdsmanCatalog{Models: []HerdsmanCatalogModel{
		{Name: "m-running", Installed: true, Running: true},
		{Name: "m-cold", Installed: true, Running: false},
		{Name: "m-missing-install", Installed: false, Running: false},
	}}
	cases := []struct {
		model      string
		wantStart  bool
		wantReason string
	}{
		{"m-cold", true, "已安装且未运行"},
		{"m-running", false, "已在运行"},
		{"m-missing-install", false, "未安装"},
		{"ghost-model", false, "未找到"},
		{"", false, "未绑定"},
	}
	for _, c := range cases {
		start, reason := autoPreloadDecision(catalog, c.model)
		if start != c.wantStart || !strings.Contains(reason, c.wantReason) {
			t.Errorf("autoPreloadDecision(%q) = (%v,%q), want (%v,含%q)",
				c.model, start, reason, c.wantStart, c.wantReason)
		}
	}
}

// TestAutoPreloadDisabledSkips 开关关闭：runAutoPreload 直接跳过，不碰 CLI。
func TestAutoPreloadDisabledSkips(t *testing.T) {
	a := scheduleTestApp(&config.Config{AutoPreload: false})

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		t.Fatal("开关关闭时不应调用 herdsman CLI")
		return nil, errors.New("unexpected")
	}
	a.runAutoPreload() // 不 panic、不启动即为通过
}

// TestRunAutoPreloadStartsTarget 命中「已安装且未运行」→ 后台启动目标模型。
func TestRunAutoPreloadStartsTarget(t *testing.T) {
	a := scheduleTestApp(&config.Config{
		AutoPreload:     true,
		FuncGaeaEngine:  "herdsman",
		FuncGaeaModel:   "gaea-m",
		FuncOfficeEngine: "herdsman",
		FuncOfficeModel: "office-m",
	})

	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(catalogResult("gaea-m", true, false)), nil
	}

	started := make(chan string, 1)
	oldStart := autoPreloadStartModel
	defer func() { autoPreloadStartModel = oldStart }()
	autoPreloadStartModel = func(_ *App, model string) error {
		started <- model
		return nil
	}

	a.runAutoPreload()
	select {
	case m := <-started:
		if m != "gaea-m" {
			t.Errorf("应预载 gaea-m（优先级 gaea 优先），got %q", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("应触发预载启动")
	}
}

// TestRunAutoPreloadRunningSkips 目标已在运行：不启动。
func TestRunAutoPreloadRunningSkips(t *testing.T) {
	a := scheduleTestApp(&config.Config{AutoPreload: true, FuncGaeaEngine: "herdsman", FuncGaeaModel: "gaea-m"})
	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(catalogResult("gaea-m", true, true)), nil
	}
	oldStart := autoPreloadStartModel
	defer func() { autoPreloadStartModel = oldStart }()
	autoPreloadStartModel = func(_ *App, model string) error {
		t.Fatalf("已在运行不应再启动，got %q", model)
		return nil
	}
	a.runAutoPreload()
}

// ─── T5-3c 换模预计等待 ────────────────────────────────────

// TestGaeaModelSwitchEstimateNonHerdsman 非 herdsman 引擎恒为 hot。
func TestGaeaModelSwitchEstimateNonHerdsman(t *testing.T) {
	a := scheduleTestApp(&config.Config{})
	e := a.GaeaModelSwitchEstimate("xai")
	if e.Status != "hot" || e.WaitSeconds != 1 {
		t.Errorf("xai 应为 hot/1s，got %q/%d", e.Status, e.WaitSeconds)
	}
	if e.Note != "引擎已就绪" {
		t.Errorf("note = %q", e.Note)
	}
}

// TestGaeaModelSwitchEstimateHerdsman 按目录状态给 hot/cold/download。
func TestGaeaModelSwitchEstimateHerdsman(t *testing.T) {
	cases := []struct {
		name       string
		installed  bool
		running    bool
		wantStatus string
		wantWait   int
	}{
		{"运行中→hot", true, true, "hot", 1},
		{"已安装未运行→cold", true, false, "cold", 20},
		{"未安装→download", false, false, "download", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := scheduleTestApp(&config.Config{})
			if err := a.engineMgr.SetDefaultModel("herdsman", "m1"); err != nil {
				t.Fatalf("SetDefaultModel: %v", err)
			}
			oldCLI := herdsmanCLI
			defer func() { herdsmanCLI = oldCLI }()
			herdsmanCLI = func(args ...string) ([]byte, error) {
				return []byte(catalogResult("m1", c.installed, c.running)), nil
			}
			e := a.GaeaModelSwitchEstimate("herdsman")
			if e.Status != c.wantStatus || e.WaitSeconds != c.wantWait {
				t.Errorf("Status/Wait = (%q,%d), want (%q,%d)", e.Status, e.WaitSeconds, c.wantStatus, c.wantWait)
			}
			if e.Model != "m1" {
				t.Errorf("Model = %q, want m1", e.Model)
			}
		})
	}
}

// TestGaeaModelSwitchEstimateUnknown 目录不可用/引擎未配置 → unknown。
func TestGaeaModelSwitchEstimateUnknown(t *testing.T) {
	// catalog 不可用。
	a := scheduleTestApp(&config.Config{})
	if err := a.engineMgr.SetDefaultModel("herdsman", "m1"); err != nil {
		t.Fatal(err)
	}
	oldCLI := herdsmanCLI
	defer func() { herdsmanCLI = oldCLI }()
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return nil, errors.New("herdsman 未运行")
	}
	e := a.GaeaModelSwitchEstimate("herdsman")
	if e.Status != "unknown" || e.WaitSeconds != 0 {
		t.Errorf("目录不可用应为 unknown/0s，got %q/%d", e.Status, e.WaitSeconds)
	}

	// 引擎管理器未初始化。
	b := &App{core: &core{cfg: &config.Config{}}}
	e = b.GaeaModelSwitchEstimate("herdsman")
	if e.Status != "unknown" {
		t.Errorf("引擎未初始化应为 unknown，got %q", e.Status)
	}
}

// ─── 绑定开关（T5-3a/b） ───────────────────────────────────

// TestScheduleConfigBindings 开关读绑定走配置（默认 true 由 config.Load 保证）。
func TestScheduleConfigBindings(t *testing.T) {
	c := &core{cfg: &config.Config{KeepWarmEnabled: true, AutoPreload: false}}
	if !c.GetKeepWarm() {
		t.Error("GetKeepWarm 应读配置值 true")
	}
	if c.GetPreloadPlan() {
		t.Error("GetPreloadPlan 应读配置值 false")
	}

	// 配置缺失按默认开启处理。
	nilCore := &core{}
	if !nilCore.GetKeepWarm() || !nilCore.GetPreloadPlan() {
		t.Error("配置缺失时开关应默认开启")
	}
}

