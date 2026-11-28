package agent

// stale_paths_test.go — 文件闸的路径口径与锚点判定（2026-11-27 AI 反馈 #5 根修）：
// 闸用工具自己的路径解析结果，write_file 之后紧跟的 edit_file 锚点仍匹配就放行。
// 正式环境的形态是「工具绑工作区 + 进程 cwd 在别处 + 模型写相对路径」——旧实现在
// 这个形态下按进程 cwd 读文件，读不到就当锚点失配，必拦；旧用例全用绝对路径，
// 一直没测出来。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	builtin "github.com/gaea/gaea/internal/gaea/tool/builtin"

	"github.com/gaea/gaea/internal/gaea/agent/cache"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// staleGuardKey 用例里手填 stale 记账表时的键：必须与闸的查找键同源。
func staleGuardKey(t *testing.T, name, args string) string {
	t.Helper()
	paths := staleGuardPaths(lookupEditTool(t, name), name, args)
	if len(paths) == 0 {
		t.Fatalf("%s 解析不出目标路径", name)
	}
	return paths[0]
}

func TestAnchorPresentToleratesWhitespaceDrift(t *testing.T) {
	content := "def run():\n    x = 1  \n\treturn x\n"
	for _, old := range []string{"    x = 1  \n", "    x = 1\n", "\treturn x"} {
		if !anchorPresent(content, old) {
			t.Errorf("锚点 %q 应视为仍在（空白漂移不算失配）", old)
		}
	}
	for _, old := range []string{"    y = 1", "", "return y"} {
		if anchorPresent(content, old) {
			t.Errorf("锚点 %q 不应视为仍在", old)
		}
	}
	if anchorPresent("", "x") {
		t.Error("空内容恒 false")
	}
}

// 行尾换算后精确命中的锚点也算在（2026-11-28 黄甲反馈）：CRLF 文件对裸 \n 的多行
// 锚点、LF 文件对 CRLF 锚点——edit_file 放行的，预检不得拦。锚点末行是行内片段
// （整行比对对不上）时，只有这条精确路径能命中。
func TestAnchorPresentEOLAgnostic(t *testing.T) {
	crlf := "### 4.1.1 环境管理组织\r\n\r\n项目部建立以项目负责人为第一责任人的环境保护管理组织：负责人审批。\r\n"
	if !anchorPresent(crlf, "### 4.1.1 环境管理组织\n\n项目部建立以项目负责人为第一责任人的环境保护管理组织") {
		t.Error("CRLF 文件对裸 \\n 的多行锚点应视为仍在")
	}
	lf := "### 4.1.1 环境管理组织\n\n项目部建立以项目负责人为第一责任人的环境保护管理组织：负责人审批。\n"
	if !anchorPresent(lf, "### 4.1.1 环境管理组织\r\n\r\n项目部建立以项目负责人为第一责任人的环境保护管理组织") {
		t.Error("LF 文件对 CRLF 锚点应视为仍在")
	}
	if anchorPresent(lf, "### 4.1.9 不存在的节\n\n正文") {
		t.Error("真不在的锚点不应视为仍在")
	}
}

// TestPrecheckDefaultReadThenMultiLineEdit 正式环境形态（2026-11-28 黄甲反馈）：
// 整读（不带 offset/limit）之后紧跟多行 edit_file。预检比对的是磁盘原文，不得拿
// read_file 的渲染缓存当文件内容——旧实现在这里必拦（ch04 连拦 3 次，锚点确在
// 原文偏移 37 处）。CRLF 文件（python 脚本在 Windows 文本模式写出来的形态）同样放行
// 且不混行尾。
func TestPrecheckDefaultReadThenMultiLineEdit(t *testing.T) {
	ws := t.TempDir()
	reg := tool.NewRegistry()
	wsTools := builtin.Workspace{Dir: ws}
	for _, tl := range wsTools.Tools("read_file", "write_file", "edit_file") {
		reg.Add(tl)
	}
	a := &AgentRunner{tools: reg, sink: event.Discard, session: NewSession("sys"), tc: cache.New(0)}
	t.Chdir(t.TempDir())
	call := func(id, name, args string) toolOutcome {
		return a.executeOne(context.Background(), provider.ToolCall{ID: id, Name: name, Arguments: args})
	}

	head := "### 4.1.1 环境管理组织"
	tail := "项目部建立以项目负责人为第一责任人的环境保护管理组织"
	for _, tc := range []struct {
		name, rel, eol string
	}{
		{"LF 文件", "ch04.md", "\n"},
		{"CRLF 文件", "ch06.md", "\r\n"},
	} {
		body := head + tc.eol + tc.eol + tail + "：负责人审批。" + tc.eol
		args := func(m map[string]string) string {
			b, _ := json.Marshal(m)
			return string(b)
		}
		if out := call("w-"+tc.rel, "write_file", args(map[string]string{"path": tc.rel, "content": body})); out.blocked || out.errMsg != "" {
			t.Fatalf("%s：write_file 应成功：%+v", tc.name, out)
		}
		if out := call("r-"+tc.rel, "read_file", args(map[string]string{"path": tc.rel})); out.blocked || out.errMsg != "" {
			t.Fatalf("%s：read_file 应成功：%+v", tc.name, out)
		}
		out := call("e-"+tc.rel, "edit_file", args(map[string]string{
			"path": tc.rel, "old_string": head + "\n\n" + tail, "new_string": "### 4.1.1 环境管理组织体系\n\n" + tail + "体系",
		}))
		if out.blocked || out.errMsg != "" {
			t.Fatalf("%s：整读之后的多行 edit_file 不该被拦：%+v", tc.name, out)
		}
		raw, _ := os.ReadFile(filepath.Join(ws, tc.rel))
		want := "### 4.1.1 环境管理组织体系" + tc.eol + tc.eol + tail + "体系：负责人审批。" + tc.eol
		if string(raw) != want {
			t.Fatalf("%s：替换块应沿用文件行尾\n got %q\nwant %q", tc.name, raw, want)
		}
	}
}

// TestStaleGuardWorkspaceRelativePaths 正式环境形态：工作区绑定的工具、进程 cwd
// 在工作区之外、模型全程写相对路径。write_file → edit_file（锚点逐字一致）必须
// 放行；锚点真失配时预检查拦下并指出最接近的位置。
func TestStaleGuardWorkspaceRelativePaths(t *testing.T) {
	ws := t.TempDir()
	reg := tool.NewRegistry()
	wsTools := builtin.Workspace{Dir: ws}
	for _, tl := range wsTools.Tools("read_file", "write_file", "edit_file") {
		reg.Add(tl)
	}
	a := &AgentRunner{tools: reg, sink: event.Discard, session: NewSession("sys"), tc: cache.New(0)}
	t.Chdir(t.TempDir()) // 进程 cwd 不在工作区：相对路径按 cwd 读不到文件

	rel := ".gaea/work/plan.py"
	call := func(id, name, args string) toolOutcome {
		return a.executeOne(context.Background(), provider.ToolCall{ID: id, Name: name, Arguments: args})
	}
	out := call("c1", "write_file", `{"path":"`+rel+`","content":"def run():\n    x = 1\n    return x\n"}`)
	if out.blocked || out.errMsg != "" {
		t.Fatalf("write_file 应成功：%+v", out)
	}
	target := filepath.Join(ws, ".gaea", "work", "plan.py")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("相对路径应落在工作区：%v", err)
	}

	// 紧接着的 edit_file：锚点与刚写的内容逐字一致——放行并生效（旧实现在这里必拦）。
	out = call("c2", "edit_file", `{"path":"`+rel+`","old_string":"    x = 1","new_string":"    x = 2"}`)
	if out.blocked || out.errMsg != "" || strings.Contains(out.output, "stale") {
		t.Fatalf("write_file 之后锚点仍匹配的 edit_file 不应被拦：%+v", out)
	}
	if b, _ := os.ReadFile(target); !strings.Contains(string(b), "x = 2") {
		t.Fatalf("edit_file 应已生效：%q", b)
	}

	// 锚点真失配：预检查读得到文件，拦下并给出最接近的位置与现行文本。
	out = call("c3", "edit_file", `{"path":"`+rel+`","old_string":"    x = 999","new_string":"z"}`)
	if !out.blocked || !strings.Contains(out.output, "precheck blocked: old_string not found") {
		t.Fatalf("失配锚点应被预检查拦下：%+v", out)
	}
	for _, want := range []string{"最接近的位置：第 2 行起", "2→    x = 2", "按上面的现行文本重写 old_string 后直接重试"} {
		if !strings.Contains(out.output, want) {
			t.Errorf("失配诊断应含 %q：%s", want, out.output)
		}
	}
}

// TestStaleVersionGuardSeesRelativePaths 跨轮版本观察在相对路径上也生效：读过之后
// 文件被外部改过——整篇重写（未声明 overwrite）被拦；锚点仍匹配的定点 edit_file 放行。
func TestStaleVersionGuardSeesRelativePaths(t *testing.T) {
	ws := t.TempDir()
	reg := tool.NewRegistry()
	wsTools := builtin.Workspace{Dir: ws}
	for _, tl := range wsTools.Tools("read_file", "write_file", "edit_file") {
		reg.Add(tl)
	}
	a := &AgentRunner{tools: reg, sink: event.Discard, session: NewSession("sys")}
	t.Chdir(t.TempDir())
	rel := "notes.md"
	call := func(id, name, args string) toolOutcome {
		return a.executeOne(context.Background(), provider.ToolCall{ID: id, Name: name, Arguments: args})
	}
	if out := call("c1", "write_file", `{"path":"`+rel+`","content":"one\ntwo\n"}`); out.blocked || out.errMsg != "" {
		t.Fatalf("write_file 应成功：%+v", out)
	}
	if out := call("c2", "read_file", `{"path":"`+rel+`"}`); out.blocked || out.errMsg != "" {
		t.Fatalf("read_file 应成功：%+v", out)
	}
	// 外部改动（换一种 mtime）：观察到的版本与磁盘不一致。
	target := filepath.Join(ws, rel)
	if err := os.WriteFile(target, []byte("one\ntwo\nexternal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(target, future, future); err != nil {
		t.Fatal(err)
	}
	// 整篇重写未声明 overwrite：拦（防静默覆盖外部改动）。
	out := call("c3", "write_file", `{"path":"`+rel+`","content":"mine\n"}`)
	if !out.blocked || !strings.Contains(out.output, "stale version") {
		t.Fatalf("外部改动后的整篇重写应被拦：%+v", out)
	}
	// 锚点仍匹配的定点编辑：放行，外部改动原样保留。
	out = call("c4", "edit_file", `{"path":"`+rel+`","old_string":"two","new_string":"TWO"}`)
	if out.blocked || out.errMsg != "" {
		t.Fatalf("锚点仍匹配的 edit_file 不应被版本闸拦：%+v", out)
	}
	if b, _ := os.ReadFile(target); string(b) != "one\nTWO\nexternal\n" {
		t.Fatalf("定点编辑应生效且保留外部改动：%q", b)
	}
}
