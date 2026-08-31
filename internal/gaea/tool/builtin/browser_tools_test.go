package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/browser"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestBrowserToolsMeta 10 个 browser_* 工具的元信息：注册、Schema/CompactSchema
// JSON 合法、ReadOnly 分类、空间标签自声明 work、compact 条目非空。
func TestBrowserToolsMeta(t *testing.T) {
	cases := []struct {
		tool         tool.Tool
		name         string
		wantReadOnly bool
	}{
		{browserNavigate{}, "browser_navigate", false},
		{browserRead{}, "browser_read", true},
		{browserSnapshot{}, "browser_snapshot", true},
		{browserClick{}, "browser_click", false},
		{browserType{}, "browser_type", false},
		{browserScroll{}, "browser_scroll", false},
		{browserTabs{}, "browser_tabs", true},
		{browserNewTab{}, "browser_new_tab", false},
		{browserSwitchTab{}, "browser_switch_tab", false},
		{browserClose{}, "browser_close", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tool.LookupBuiltin(tc.name); !ok {
				t.Fatal("未注册进 builtin 表")
			}
			if got := tc.tool.Name(); got != tc.name {
				t.Fatalf("Name = %q, want %q", got, tc.name)
			}
			if strings.TrimSpace(tc.tool.Description()) == "" {
				t.Fatal("Description 为空")
			}
			if !json.Valid(tc.tool.Schema()) {
				t.Fatalf("Schema 非法: %s", tc.tool.Schema())
			}
			if got := tc.tool.ReadOnly(); got != tc.wantReadOnly {
				t.Fatalf("ReadOnly = %v, want %v", got, tc.wantReadOnly)
			}
			if got := tool.SpaceTagOf(tc.tool); got != "work" {
				t.Fatalf("SpaceTag = %q, want work", got)
			}
			cd, ok := tc.tool.(tool.CompactDescriptor)
			if !ok {
				t.Fatal("未实现 CompactDescriptor")
			}
			if cd.CompactDescription() == "" || len(cd.CompactSchema()) == 0 {
				t.Fatal("compact 条目缺失（compact.go）")
			}
			if !json.Valid(cd.CompactSchema()) {
				t.Fatalf("CompactSchema 非法: %s", cd.CompactSchema())
			}
		})
	}
}

// parseEnv 断言工具经 envelope 返回（错误不走 Go error 通道）。
func parseBrowserEnv(t *testing.T, out string, err error) tool.ToolEnvelope {
	t.Helper()
	if err != nil {
		t.Fatalf("Execute 返回 Go error: %v（错误应走 envelope）", err)
	}
	env, ok := tool.ParseEnvelope(out)
	if !ok {
		t.Fatalf("输出不是合法 envelope: %q", out)
	}
	return env
}

// TestBrowserToolsValidationEnvelopes 校验类错误在任何浏览器拉起之前就被
// envelope 拒绝（code=validation_error，不触碰浏览器进程）。
func TestBrowserToolsValidationEnvelopes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		exec func() (string, error)
	}{
		{"navigate 缺 url", func() (string, error) { return browserNavigate{}.Execute(ctx, json.RawMessage(`{}`)) }},
		{"navigate 非法 scheme", func() (string, error) {
			return browserNavigate{}.Execute(ctx, json.RawMessage(`{"url":"javascript:alert(1)"}`))
		}},
		{"navigate file scheme", func() (string, error) {
			return browserNavigate{}.Execute(ctx, json.RawMessage(`{"url":"file:///C:/x"}`))
		}},
		{"click 缺 ref/selector", func() (string, error) { return browserClick{}.Execute(ctx, json.RawMessage(`{}`)) }},
		{"type 缺 text", func() (string, error) { return browserType{}.Execute(ctx, json.RawMessage(`{"ref":1}`)) }},
		{"scroll 非法方向", func() (string, error) {
			return browserScroll{}.Execute(ctx, json.RawMessage(`{"direction":"left"}`))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := c.exec()
			env := parseBrowserEnv(t, out, err)
			if env.OK || env.Code != tool.CodeValidationError {
				t.Fatalf("envelope = ok=%v code=%q, want validation_error", env.OK, env.Code)
			}
		})
	}
}

// TestBrowserToolsRouteThroughDefault 工具经 Default()/SetForTest 注入点走
// browser 包：假端点 Ensure 失败 → exec_error envelope；未启动时 close 幂等成功。
func TestBrowserToolsRouteThroughDefault(t *testing.T) {
	prev := browser.SetForTest(browser.NewManager(browser.Options{InjectHTTPBase: "http://127.0.0.1:1"}))
	defer browser.SetForTest(prev)

	out, err := browserRead{}.Execute(context.Background(), nil)
	env := parseBrowserEnv(t, out, err)
	if env.OK || env.Code != tool.CodeExecError {
		t.Fatalf("read envelope = ok=%v code=%q, want exec_error（假端点 Ensure 失败）", env.OK, env.Code)
	}

	out, err = browserClose{}.Execute(context.Background(), nil)
	env = parseBrowserEnv(t, out, err)
	if !env.OK {
		t.Fatalf("未启动时 close 应幂等成功, got %q", env.Error)
	}
}

// TestBrowserTabToolsValidationEnvelopes 新工具的校验类错误在浏览器拉起之前
// 就被 envelope 拒绝（code=validation_error）。
func TestBrowserTabToolsValidationEnvelopes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		exec func() (string, error)
	}{
		{"new_tab 缺 url", func() (string, error) {
			return browserNewTab{}.Execute(ctx, json.RawMessage(`{}`))
		}},
		{"new_tab 非法 scheme", func() (string, error) {
			return browserNewTab{}.Execute(ctx, json.RawMessage(`{"url":"javascript:alert(1)"}`))
		}},
		{"switch_tab 缺 tab_id", func() (string, error) {
			return browserSwitchTab{}.Execute(ctx, json.RawMessage(`{}`))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := c.exec()
			env := parseBrowserEnv(t, out, err)
			if env.OK || env.Code != tool.CodeValidationError {
				t.Fatalf("envelope = ok=%v code=%q, want validation_error", env.OK, env.Code)
			}
		})
	}
}

// TestBrowserTabToolsDeadEndpoint 新工具经 SetForTest 假端点走 browser 包：
// Ensure 失败 → exec_error envelope（不真拉起浏览器）。
func TestBrowserTabToolsDeadEndpoint(t *testing.T) {
	prev := browser.SetForTest(browser.NewManager(browser.Options{InjectHTTPBase: "http://127.0.0.1:1"}))
	defer browser.SetForTest(prev)
	ctx := context.Background()
	cases := []struct {
		name string
		exec func() (string, error)
	}{
		{"tabs", func() (string, error) { return browserTabs{}.Execute(ctx, nil) }},
		{"new_tab", func() (string, error) {
			return browserNewTab{}.Execute(ctx, json.RawMessage(`{"url":"http://example.local/x"}`))
		}},
		{"switch_tab", func() (string, error) {
			return browserSwitchTab{}.Execute(ctx, json.RawMessage(`{"tab_id":"page-1"}`))
		}},
		{"close(tab_id)", func() (string, error) {
			return browserClose{}.Execute(ctx, json.RawMessage(`{"tab_id":"page-1"}`))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := c.exec()
			env := parseBrowserEnv(t, out, err)
			if env.OK || env.Code != tool.CodeExecError {
				t.Fatalf("envelope = ok=%v code=%q, want exec_error（假端点 Ensure 失败）", env.OK, env.Code)
			}
		})
	}
}
