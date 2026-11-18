package boot_test

// 2026-11-18 众神殿余项·文学 play 装配洞根修验收：全局 session.space=play
// 时，文学引擎（ExecBoard=wenxue + SpaceAnchor=work）的基础读写工具不再被
// AllowsSpace 全滤（SpaceTag 恒 work 的工具族按 work 装配）；对照办公链
// （无锚）play 下照常滤出 play 面。census 口径=注册面（Tools）。

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

func buildUnderGlobalPlay(t *testing.T, execBoard, spaceAnchor string) ([]interface{ Name() string }, string) {
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
	cfg.Session.Space = "play"
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: "test-mock-space-assembly", Model: "m1", ContextWindow: 1_000_000,
	}}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:       "mock",
		RequireKey:  false,
		Sink:        event.FuncSink(func(event.Event) {}),
		Stderr:      io.Discard,
		Cwd:         t.TempDir(),
		SessionDir:  t.TempDir(),
		ExecBoard:   execBoard,
		SpaceAnchor: spaceAnchor,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(func() { ctrl.Close() })

	var b strings.Builder
	for _, m := range ctrl.History() {
		if m.Role == provider.RoleSystem {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(m.Content)
		}
	}
	names := make([]interface{ Name() string }, 0)
	for _, tl := range ctrl.Tools() {
		names = append(names, tl)
	}
	return names, b.String()
}

func hasTool(names []interface{ Name() string }, want string) bool {
	for _, n := range names {
		if n.Name() == want {
			return true
		}
	}
	return false
}

// TestWenxuePlayAnchorKeepsWorkTools 文学锚定：play 全局下 write_file/read_file
// 等基础工具与 wenxue 执行纪律照常装配（洞修复）；办公执行后缀仍不在场。
func TestWenxuePlayAnchorKeepsWorkTools(t *testing.T) {
	names, exec := buildUnderGlobalPlay(t, "wenxue", "work")

	// census：基础读写工具在场（修复前被 AllowsSpace 全滤）。
	for _, want := range []string{"write_file", "read_file", "ls"} {
		if !hasTool(names, want) {
			t.Errorf("play 全局下文学引擎缺基础工具 %q（SpaceAnchor=work 应保住 work 标签工具）", want)
		}
	}
	// 板块缝仍生效：文学执行常量在场、办公执行锚不在场。
	if !strings.Contains(exec, "本会话文件落点") {
		t.Error("文学执行常量应在场")
	}
	if strings.Contains(exec, "事实底座（一稿多用）") {
		t.Error("办公执行后缀不应进文学串")
	}
}

// TestOfficePlayUnanchoredStaysFiltered 对照组：办公链（无锚）play 全局下
// work 标签基础工具照常被滤（现状行为不被 SpaceAnchor 改动）。
func TestOfficePlayUnanchoredStaysFiltered(t *testing.T) {
	names, exec := buildUnderGlobalPlay(t, "", "")

	if hasTool(names, "write_file") {
		t.Error("办公链 play 下 write_file 应被滤（现状双态行为）")
	}
	// 办公执行纪律按 play 渲染（S4 根段替换）。
	if !strings.Contains(exec, ".gaea/play/exports/") {
		t.Error("办公 play 态执行纪律应含 play 根段")
	}
}
