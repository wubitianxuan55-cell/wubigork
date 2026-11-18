package boot_test

// 2026-11-18 众神殿 pan_1795007052552 v1.0 验收合同①②⑤：图文 executor 最终
// 串快照（work / space.mode=off 态取——play 下 wenxue 工具族被 AllowsSpace 全
// 滤属装配洞另案，不用 play 快照判文本失败）。ExecBoard="wenxue" 分叉断言：
// 图文执行常量在场、办公执行后缀与 officeLive 段不在场、导出根字面不进串。

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

func buildWithExecBoard(t *testing.T, execBoard, override string) string {
	t.Helper()
	oldAPPDATA := os.Getenv("APPDATA")
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("APPDATA", t.TempDir())
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		os.Setenv("APPDATA", oldAPPDATA)
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	})
	userDir := config.MemoryUserDir()
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	registerSpaceMockKind()
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Tools.Enabled = nil
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:        "mock",
		RequireKey:   false,
		Sink:         event.FuncSink(func(event.Event) {}),
		Stderr:       io.Discard,
		Cwd:          t.TempDir(),
		SessionDir:   t.TempDir(),
		SystemPrompt: override,
		ExecBoard:    execBoard,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(func() { ctrl.Close() })
	// executor 最终串=L1 身份+L2 执行纪律（ctrl.SystemPrompt() 只回 L1）：
	// 从 History 快照过滤 system 消息拼接（与 AgentRunner.systemPrompt 同构）。
	var b strings.Builder
	for _, m := range ctrl.History() {
		if m.Role == provider.RoleSystem {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(m.Content)
		}
	}
	return b.String()
}

// TestWenxueExecBoardAssembly 图文板块缝：executor 最终串换图文执行常量，
// 办公执行后缀/officeLive 段/固定导出根字面全部不在场（验收①②⑤——快照在
// space.mode=off 态取）。
func TestWenxueExecBoardAssembly(t *testing.T) {
	got := buildWithExecBoard(t, "wenxue", "你是图文板块测试人格 MARKER-WX-EXEC。")

	// 图文执行常量在场锚（含过程句 work 字面量——验收②：过程句应含
	// .gaea/work/，不得把安全过程路径误判违规）。
	for _, sub := range []string{
		"本会话文件落点",
		".gaea/work/<任务名>/",
		"wenxue_final_write",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("图文 executor 串缺执行常量锚 %q", sub)
		}
	}

	// 办公执行后缀独有锚（分叉前文学串吃的就是这段）。
	for _, officeOnly := range []string{
		"沉稳、坦诚、高效的通用办公助手", // SingleModelPrompt 角色行
		"事实底座（一稿多用）",
		"办公领域质量检查点",
		".gaea/exports/", // 固定导出根（办公落盘句）
	} {
		if strings.Contains(got, officeOnly) {
			t.Errorf("图文 executor 串含办公执行锚 %q（板块缝未生效）", officeOnly)
		}
	}

	// officeLive 段：本机装 Office/WPS 时办公链会追加，图文必须跳过（分叉
	// 与 COM 探测解耦——即使 ProbeSuiteAvailable=true 也不进文学串）。
	if strings.Contains(got, "Office 实时协同") || strings.Contains(got, "office_open") {
		t.Error("图文 executor 串含 officeLive 段（应跳过）")
	}
}

// TestExecBoardEmptyKeepsOfficeDiscipline 空板块=现状逐字节：办公/CLI/TUI
// 链路仍吃 SingleModelPromptFor（含 v1.0 增量：能力发现两行）。
func TestExecBoardEmptyKeepsOfficeDiscipline(t *testing.T) {
	got := buildWithExecBoard(t, "", "")
	for _, sub := range []string{
		"事实底座（一稿多用）",
		"办公领域质量检查点",
		"**能力发现**",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("办公 executor 串缺执行常量锚 %q（空调应保持现状）", sub)
		}
	}
}
