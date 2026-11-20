package builtin

// W3 注册侧验收：RegisterComputerTools 幂等、白名单闸、ReadOnly 契约、
// state 参数校验与未知窗口语义。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func computerToolByName(name string) tool.Tool {
	for _, tl := range tool.Builtins() {
		if tl.Name() == name {
			return tl
		}
	}
	return nil
}

func TestRegisterComputerToolsIdempotent(t *testing.T) {
	RegisterComputerTools(nil, 0)
	RegisterComputerTools(nil, 0) // 二次装配不得 panic（RegisterBuiltin 重复注册会炸）
	for _, want := range []string{"computer_windows", "computer_state", "computer_launch"} {
		if computerToolByName(want) == nil {
			t.Fatalf("注册后 Builtins 缺 %s", want)
		}
	}
}

func TestComputerToolsReadOnlyContract(t *testing.T) {
	cases := map[string]bool{"computer_windows": true, "computer_state": true, "computer_launch": false}
	for name, want := range cases {
		tl := computerToolByName(name)
		if tl == nil {
			t.Fatalf("缺 %s（先注册）", name)
		}
		if tl.ReadOnly() != want {
			t.Fatalf("%s ReadOnly()=%v want %v", name, tl.ReadOnly(), want)
		}
		if tool.SpaceTagOf(tl) != "work" {
			t.Fatalf("%s SpaceTag=%q want work", name, tool.SpaceTagOf(tl))
		}
	}
}

func TestComputerAppsAllowed(t *testing.T) {
	RegisterComputerTools([]string{`process:C:\apps\notepad.exe`}, 0)
	if !computerAppsAllowed(`process:C:\apps\notepad.exe`) {
		t.Fatal("白名单内应放行")
	}
	if computerAppsAllowed(`process:C:\apps\cmd.exe`) {
		t.Fatal("白名单外必须拒")
	}
	RegisterComputerTools(nil, 0) // 空=不限制
	if !computerAppsAllowed(`process:C:\apps\cmd.exe`) {
		t.Fatal("空白名单=不限制")
	}
}

func TestComputerStateValidationAndUnknownWindow(t *testing.T) {
	RegisterComputerTools(nil, 0)
	tl := computerToolByName("computer_state")
	if tl == nil {
		t.Fatal("缺 computer_state")
	}
	// 缺 window_id → 校验信封。
	out, _ := tl.Execute(context.Background(), json.RawMessage(`{}`))
	if !strings.Contains(out, "window_id") {
		t.Fatalf("缺参应回校验信封: %.120s", out)
	}
	// 未知窗口 → not_found 信封。
	out, _ = tl.Execute(context.Background(), json.RawMessage(`{"window_id":2147480000}`))
	if !strings.Contains(out, `"not_found"`) {
		t.Fatalf("未知窗口应回 not_found: %.200s", out)
	}
}

func TestComputerLaunchValidationAndWhitelist(t *testing.T) {
	RegisterComputerTools([]string{`process:C:\apps\allowed.exe`}, 0)
	tl := computerToolByName("computer_launch")
	// 缺 app → 校验信封。
	out, _ := tl.Execute(context.Background(), json.RawMessage(`{}`))
	if !strings.Contains(out, "app") {
		t.Fatalf("缺参应回校验信封: %.120s", out)
	}
	// 白名单外 → 校验信封（不得真启动）。
	out, _ = tl.Execute(context.Background(), json.RawMessage(`{"app":"process:C:\\windows\\system32\\cmd.exe"}`))
	if !strings.Contains(out, "白名单") {
		t.Fatalf("白名单外必须拒: %.160s", out)
	}
	RegisterComputerTools(nil, 0)
}
