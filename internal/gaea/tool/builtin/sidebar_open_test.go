package builtin

// sidebar_open_test.go — v4.25「模型主动打开」工具测试：注册/元信息、参数校验、
// 工作区未设置、防穿越（../ 与绝对路径逃逸）、file/directory 推断与显式 kind、
// envelope 结构（data.kind/path_abs/path_rel）。全部走 envelope，错误不走
// Go error 通道（与 browser_* 先例同口径）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// parseSidebarEnv 断言输出是合法 envelope。
func parseSidebarEnv(t *testing.T, out string) tool.ToolEnvelope {
	t.Helper()
	env, ok := tool.ParseEnvelope(out)
	if !ok {
		t.Fatalf("输出不是合法 envelope: %q", out)
	}
	return env
}

// TestSidebarOpenMeta 注册、元信息、只读分类、work 空间标签、compact 条目。
func TestSidebarOpenMeta(t *testing.T) {
	if _, ok := tool.LookupBuiltin("sidebar_open"); !ok {
		t.Fatal("未注册进 builtin 表")
	}
	var tv tool.Tool = sidebarOpen{}
	if got := tv.Name(); got != "sidebar_open" {
		t.Fatalf("Name = %q, want sidebar_open", got)
	}
	if strings.TrimSpace(tv.Description()) == "" {
		t.Fatal("Description 为空")
	}
	if !json.Valid(tv.Schema()) {
		t.Fatalf("Schema 非法: %s", tv.Schema())
	}
	if !tv.ReadOnly() {
		t.Fatal("ReadOnly = false, want true（纯 UI 动作，不走写类权限弹卡）")
	}
	if got := tool.SpaceTagOf(tv); got != "work" {
		t.Fatalf("SpaceTag = %q, want work", got)
	}
	cd, ok := tv.(tool.CompactDescriptor)
	if !ok {
		t.Fatal("未实现 CompactDescriptor")
	}
	if cd.CompactDescription() == "" || len(cd.CompactSchema()) == 0 {
		t.Fatal("compact 条目缺失（compact.go）")
	}
	if !json.Valid(cd.CompactSchema()) {
		t.Fatalf("CompactSchema 非法: %s", cd.CompactSchema())
	}
}

// TestSidebarOpenWorkspaceBinding 桌面端装配：Workspace.Tools() 返回绑定
// 工作区根的同名实例（按名替换 init 注册的零值）。
func TestSidebarOpenWorkspaceBinding(t *testing.T) {
	ws := Workspace{Dir: t.TempDir()}
	var found tool.Tool
	for _, tl := range ws.Tools() {
		if tl.Name() == "sidebar_open" {
			found = tl
			break
		}
	}
	if found == nil {
		t.Fatal("Workspace.Tools() 未包含 sidebar_open")
	}
	so, ok := found.(sidebarOpen)
	if !ok {
		t.Fatalf("类型 = %T, want builtin.sidebarOpen", found)
	}
	if so.root == "" {
		t.Fatal("绑定实例 root 为空（应绑定 Workspace.Dir）")
	}
}

// TestSidebarOpenValidation 参数校验：缺 path、kind 非法、坏 JSON。
func TestSidebarOpenValidation(t *testing.T) {
	root := t.TempDir()
	tv := sidebarOpen{root: root}
	cases := []struct {
		name string
		args string
	}{
		{"缺 path", `{}`},
		{"path 空串", `{"path":"  "}`},
		{"kind 非法", `{"path":"a.txt","kind":"folder"}`},
		{"坏 JSON", `{"path":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := parseSidebarEnv(t, mustSidebarExec(t, tv, json.RawMessage(tc.args)))
			if env.OK || env.Code != tool.CodeValidationError {
				t.Fatalf("want ok=false code=validation_error, got ok=%v code=%q err=%q", env.OK, env.Code, env.Error)
			}
			if strings.TrimSpace(env.Error) == "" {
				t.Fatal("error 信息为空")
			}
		})
	}
}

// TestSidebarOpenNoWorkspace 工作区未设置（零值实例）→ 结构化报错。
func TestSidebarOpenNoWorkspace(t *testing.T) {
	tv := sidebarOpen{}
	env := parseSidebarEnv(t, mustSidebarExec(t, tv, json.RawMessage(`{"path":"a.txt"}`)))
	if env.OK || env.Code != "no_workspace" {
		t.Fatalf("want ok=false code=no_workspace, got ok=%v code=%q", env.OK, env.Code)
	}
}

// TestSidebarOpenTraversal 防穿越：../ 回溯与绝对路径逃逸工作区均拒绝。
func TestSidebarOpenTraversal(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// 在工作区外放一个真实存在的文件，确保拒绝发生在 within 判定而非 not_found。
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tv := sidebarOpen{root: root}
	// 相对形式的绝对路径：windows 盘符路径 / posix 根路径按平台构造。
	cases := []struct {
		name string
		path string
	}{
		{"../ 回溯", filepath.Join("..", filepath.Base(outside), "secret.txt")},
		{"绝对路径逃逸", outsideFile},
		{"深层回溯", filepath.Join("a", "b", "..", "..", "..", filepath.Base(outside), "secret.txt")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(map[string]string{"path": tc.path})
			if err != nil {
				t.Fatal(err)
			}
			env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
			if env.OK || env.Code != tool.CodeValidationError {
				t.Fatalf("want ok=false code=validation_error, got ok=%v code=%q err=%q", env.OK, env.Code, env.Error)
			}
			if !strings.Contains(env.Error, "工作区外") {
				t.Fatalf("error 未说明越界: %q", env.Error)
			}
		})
	}
}

// TestSidebarOpenInference file/directory 推断与 envelope 结构。
func TestSidebarOpenInference(t *testing.T) {
	root := t.TempDir()
	fileRel := filepath.Join("docs", "报告.docx")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, fileRel), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirRel := "docs"
	if err := os.MkdirAll(filepath.Join(root, "out"), 0o755); err != nil {
		t.Fatal(err)
	}

	tv := sidebarOpen{root: root}
	t.Run("缺省推断 file（相对路径）", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": fileRel})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if !env.OK || env.Code != "ok" {
			t.Fatalf("want ok, got ok=%v code=%q err=%q", env.OK, env.Code, env.Error)
		}
		assertSidebarData(t, env, "file", filepath.Join(root, fileRel), fileRel)
	})
	t.Run("缺省推断 directory", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": dirRel})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if !env.OK {
			t.Fatalf("want ok, got code=%q err=%q", env.Code, env.Error)
		}
		assertSidebarData(t, env, "directory", filepath.Join(root, dirRel), dirRel)
	})
	t.Run("显式 kind=file 命中", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": fileRel, "kind": "file"})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if !env.OK {
			t.Fatalf("want ok, got code=%q err=%q", env.Code, env.Error)
		}
		assertSidebarData(t, env, "file", filepath.Join(root, fileRel), fileRel)
	})
	t.Run("显式 kind 与实际不符报错", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": dirRel, "kind": "file"})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if env.OK || env.Code != tool.CodeValidationError {
			t.Fatalf("want ok=false code=validation_error, got ok=%v code=%q", env.OK, env.Code)
		}
	})
	t.Run("绝对路径在工作区内", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": filepath.Join(root, fileRel)})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if !env.OK {
			t.Fatalf("want ok, got code=%q err=%q", env.Code, env.Error)
		}
		assertSidebarData(t, env, "file", filepath.Join(root, fileRel), fileRel)
	})
	t.Run("不存在的路径报 not_found（推断不出）", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": filepath.Join("nope", "ghost.txt")})
		env := parseSidebarEnv(t, mustSidebarExec(t, tv, args))
		if env.OK || env.Code != tool.CodeNotFound {
			t.Fatalf("want ok=false code=not_found, got ok=%v code=%q", env.OK, env.Code)
		}
	})
}

// assertSidebarData 钉住 envelope data 精确 schema：
// {"kind":"file|directory","path_abs":...,"path_rel":...}。
func assertSidebarData(t *testing.T, env tool.ToolEnvelope, wantKind, wantAbs, wantRel string) {
	t.Helper()
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Kind    string `json:"kind"`
		PathAbs string `json:"path_abs"`
		PathRel string `json:"path_rel"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("data 非结构化对象: %s", raw)
	}
	if data.Kind != wantKind {
		t.Fatalf("kind = %q, want %q", data.Kind, wantKind)
	}
	if data.PathAbs != wantAbs {
		t.Fatalf("path_abs = %q, want %q", data.PathAbs, wantAbs)
	}
	if data.PathRel != wantRel {
		t.Fatalf("path_rel = %q, want %q", data.PathRel, wantRel)
	}
}

// mustSidebarExec 执行工具；Go error 通道非空即失败（错误应走 envelope）。
func mustSidebarExec(t *testing.T, tv sidebarOpen, args json.RawMessage) string {
	t.Helper()
	out, err := tv.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute 返回 Go error: %v（错误应走 envelope）", err)
	}
	return out
}
