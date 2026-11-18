package boot_test

// 2026-11-18 众神殿余项·办公 schema 三态验收：办公身份稿瘦身后能力可见性
// 依赖「schema 完整描述 + 执行常量两行能力发现路由」双保险。三态=正常 /
// schema_budget 截断 / tool_search+deferred 隐藏——断言前两态的可见面形态，
// 以及兜底路由在任一态都仍随执行纪律注入（模型总能拿到「先看 schema→
// tool_search→如实说明」的路由）。暴露面=Controller.Schemas()（可见面，
// hide/budget 已套用）与 Tools()（注册面，藏而可按名调用）。

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
	"github.com/gaea/gaea/internal/gaea/tool"
)

// buildOfficeSchemaState 构建办公 mock 引擎并返回可见 schema 面与执行串。
func buildOfficeSchemaState(t *testing.T, tune func(*config.Config)) ([]provider.ToolSchema, []tool.Tool, string) {
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
	if tune != nil {
		tune(cfg)
	}
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := boot.Build(context.Background(), boot.Options{
		Model:      "mock",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		Stderr:     io.Discard,
		Cwd:        t.TempDir(),
		SessionDir: t.TempDir(),
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
	return ctrl.Schemas(), ctrl.Tools(), b.String()
}

func schemaRow(schemas []provider.ToolSchema, name string) (provider.ToolSchema, bool) {
	for _, s := range schemas {
		if s.Name == name {
			return s, true
		}
	}
	return provider.ToolSchema{}, false
}

// TestOfficeSchemaStateNormal 正常态：vision 的 compact 描述完整在场
// （CompactDescriptor 无条件替换——模型恒见 compact 变体，能力说明单源），
// 执行纪律带能力发现两行。
func TestOfficeSchemaStateNormal(t *testing.T) {
	schemas, _, exec := buildOfficeSchemaState(t, nil)

	row, ok := schemaRow(schemas, "vision")
	if !ok {
		t.Fatal("正常态 schema 面应含 vision")
	}
	if !strings.Contains(row.Description, "配合screen_capture理解截图") {
		t.Errorf("正常态 vision 应服务完整 compact 描述，got %.60q", row.Description)
	}
	if !strings.Contains(exec, "**能力发现**") {
		t.Error("执行纪律缺能力发现两行（兜底路由恒注入）")
	}
}

// TestOfficeSchemaStateBudget budget 态：小预算下描述截断或整行省略——完整
// 说明不可依赖，兜底路由仍随执行纪律在场（裁决 issue-4：不能只靠 schema）。
func TestOfficeSchemaStateBudget(t *testing.T) {
	schemas, _, exec := buildOfficeSchemaState(t, func(cfg *config.Config) {
		cfg.Tools.SchemaBudget.Enabled = true
		cfg.Tools.SchemaBudget.Bytes = 24
	})

	row, ok := schemaRow(schemas, "vision")
	if ok {
		// 行保留=compact 描述被截断：完整锚不得幸存（24 字预算装不下）。
		if strings.Contains(row.Description, "screen_capture") {
			t.Errorf("budget=24 下 vision 描述不应含完整锚，got %.60q", row.Description)
		}
	} // !ok = 整行省略（omissionReserve 挤出）——同样证明预算态不可依赖完整描述。
	if !strings.Contains(exec, "**能力发现**") || !strings.Contains(exec, "tool_search") {
		t.Error("budget 态执行纪律仍应带能力发现路由（兜底不受三态影响）")
	}
}

// TestOfficeSchemaStateDeferred deferred 态：名单工具出首轮 schema（藏而可
// 按名调用——注册面仍在），tool_search 注册进可见面。
func TestOfficeSchemaStateDeferred(t *testing.T) {
	schemas, tools, exec := buildOfficeSchemaState(t, func(cfg *config.Config) {
		cfg.Tools.Search.Enabled = true
		cfg.Tools.Deferred = []string{"vision"}
	})

	if _, ok := schemaRow(schemas, "vision"); ok {
		t.Error("deferred 态 vision 应藏出首轮 schema")
	}
	if _, ok := schemaRow(schemas, "tool_search"); !ok {
		t.Error("deferred 态 tool_search 应在可见面")
	}
	// 注册面保留（藏而可按名调用，tool_search 可取回）。
	found := false
	for _, tl := range tools {
		if tl.Name() == "vision" {
			found = true
		}
	}
	if !found {
		t.Error("deferred 工具应仍在注册面（可按名调用）")
	}
	if !strings.Contains(exec, "**能力发现**") {
		t.Error("deferred 态执行纪律仍应带能力发现路由")
	}
}
