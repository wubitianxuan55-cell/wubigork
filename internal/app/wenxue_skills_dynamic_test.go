package app

// wenxue_skills_dynamic_test.go — 技能面动态终验（2026-11-15 众神殿首刀验收
// 合同四段式第四腿，pan_1794754879290：施工后动态各调一次 read_skill/run_skill）。
// 与单测（wenxue_skills_test / skill 包五例）互补：这里走真实 boot.Build 全链
// 装配——真实 wenxueSkillFilter/SkillExtras/workbench 正文，非测试桩。
// hermetic：chdir 临时目录 + mock provider，不写用户配置（同 TestGaeaHistoryGolden
// 纪律：不改 APPDATA，boot.Build 打开的全局库句柄才不锁 cleanup）。

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// wenxueDynamicTools 真实装配一次 boot.Build，返回按名索引的工具面。
func wenxueDynamicTools(t *testing.T, ws string, board bool) map[string]tool.Tool {
	t.Helper()

	opts := gaeaBoot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		SessionDir: filepath.Join(ws, ".gaea", "sessions"),
	}
	if board {
		// 图文缝（与 wenxueBuildController 同参）：allowlist+内嵌三件+工具 deny。
		opts.SkillFilter = wenxueSkillFilter
		opts.SkillExtras = wenxueSkillExtras()
		opts.ToolFilter = wenxueToolFilter
	}
	ctrl, err := gaeaBoot.Build(context.Background(), opts)
	if err != nil {
		t.Fatalf("boot.Build(board=%v): %v", board, err)
	}
	t.Cleanup(ctrl.Close)
	byName := make(map[string]tool.Tool)
	for _, tl := range ctrl.Tools() {
		byName[tl.Name()] = tl
	}
	return byName
}

func wenxueDynamicExec(t *testing.T, tl tool.Tool, args string) (string, error) {
	t.Helper()
	out, err := tl.Execute(context.Background(), json.RawMessage(args))
	return out, err
}

// TestWenxueSkillFacesDynamicAcceptance 动态终验：图文装配下 read_skill 与
// run_skill 各真实调一次（extras 可达）+ 真实 store 技能被滤名双工具都
// fail-closed；办公装配（无缝）run_skill 活读同一夹具成功——nil-filter 行为
// 不变的动态证明。
func TestWenxueSkillFacesDynamicAcceptance(t *testing.T) {
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	ws := t.TempDir()
	if err := os.Chdir(ws); err != nil {
		t.Fatal(err)
	}
	// 项目域 store 技能夹具：真实存在于 store、但被图文 allowlist 拒绝——
	// fail-closed 判据不依赖用户 home 的偶然内容。
	skillsDir := filepath.Join(ws, ".gaea", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "gongwen.md"),
		[]byte("---\ndescription: 公文纪律\n---\n公文正文（store 夹具）"), 0o644); err != nil {
		t.Fatal(err)
	}

	kind := testKind("test-mock-wenxue-dynamic")
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name: "mock", Kind: kind, Model: "grok-3", ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	t.Cleanup(func() { gaeaConfig.SetLoader(nil) })

	// ── 图文面：extras 经 run_skill/read_skill 双工具可达 ──
	wx := wenxueDynamicTools(t, ws, true)
	rs, ok := wx["run_skill"]
	if !ok {
		t.Fatal("图文装配缺 run_skill")
	}
	out, err := wenxueDynamicExec(t, rs, `{"name":"wenxue-workbench"}`)
	if err != nil {
		t.Fatalf("动态 run_skill(wenxue-workbench) 应可达: %v", err)
	}
	if !strings.Contains(out, "写完一章必须回写") {
		t.Fatalf("run_skill 应返回 workbench 真实正文: %s", out)
	}
	rd, ok := wx["read_skill"]
	if !ok {
		t.Fatal("图文装配缺 read_skill")
	}
	out, err = wenxueDynamicExec(t, rd, `{"name":"wenxue-workbench"}`)
	if err != nil {
		t.Fatalf("动态 read_skill(wenxue-workbench) 应可达: %v", err)
	}
	if !strings.Contains(out, "写完一章必须回写") {
		t.Fatalf("read_skill 应返回 workbench 真实正文: %s", out)
	}

	// ── 图文面：真实 store 技能被 allowlist 拒名——双工具 fail-closed ──
	if _, err = wenxueDynamicExec(t, rs, `{"name":"gongwen"}`); err == nil {
		t.Fatal("被滤 store 技能经 run_skill 应 unknown")
	} else if !strings.Contains(err.Error(), "unknown skill") {
		t.Fatalf("run_skill 错误语义不符: %v", err)
	}
	if _, err = wenxueDynamicExec(t, rd, `{"name":"gongwen"}`); err == nil {
		t.Fatal("被滤 store 技能经 read_skill 应 not found")
	}

	// ── 办公面（双 nil）：同一夹具 run_skill 活读成功——nil-filter 不变 ──
	office := wenxueDynamicTools(t, ws, false)
	ors, ok := office["run_skill"]
	if !ok {
		t.Fatal("办公装配缺 run_skill")
	}
	out, err = wenxueDynamicExec(t, ors, `{"name":"gongwen"}`)
	if err != nil {
		t.Fatalf("办公 run_skill 活读 store 夹具应成功: %v", err)
	}
	if !strings.Contains(out, "公文正文") {
		t.Fatalf("办公 run_skill 应返回夹具正文: %s", out)
	}
}
