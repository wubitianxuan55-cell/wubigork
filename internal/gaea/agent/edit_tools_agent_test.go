package agent

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/cache"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin" // 注册 builtin 工具（含 S0.6 编辑系）
)

// S0.6 integration tests: the edit-tools layer (edit_file / move_file / grep)
// wired through the agent runner — precheck, stale guard, loop guard, cache
// invalidation, conflict keys. 2026-11-25 办公工具梳理批退役 multi_edit /
// edit_lines / replace，对应用例随之移除。

func lookupEditTool(t *testing.T, name string) tool.Tool {
	t.Helper()
	tl, ok := tool.LookupBuiltin(name)
	if !ok {
		t.Fatalf("builtin %q not registered — the tool layer must be implemented (S0.6)", name)
	}
	return tl
}

func editToolsRegistry(t *testing.T, names ...string) *tool.Registry {
	t.Helper()
	reg := tool.NewRegistry()
	for _, n := range names {
		reg.Add(lookupEditTool(t, n))
	}
	return reg
}

// TestRetiredEditToolsStayUnregistered 退役工具不得回潮：注册面、紧凑描述表、
// 常驻名单三处都没有它们的位子。
func TestRetiredEditToolsStayUnregistered(t *testing.T) {
	for _, name := range []string{"multi_edit", "edit_lines", "replace", "cost_compose"} {
		if _, ok := tool.LookupBuiltin(name); ok {
			t.Errorf("%s 已退役，不应再注册", name)
		}
		if a := (&AgentRunner{}).precheckTool(name, []byte(`{"path":"x"}`)); a != "" {
			t.Errorf("%s 已退役，precheck 不应再认它", name)
		}
	}
}

func TestEditToolsRegisteredWithReadOnlyClassification(t *testing.T) {
	for name, wantReadOnly := range map[string]bool{
		"edit_file": false, "move_file": false, "grep": true,
	} {
		tl := lookupEditTool(t, name)
		if got := tl.ReadOnly(); got != wantReadOnly {
			t.Errorf("%s ReadOnly() = %v, want %v", name, got, wantReadOnly)
		}
		// S0.6 risk 1: compact entries must exist or Schemas() exports empties.
		if cd, ok := tl.(tool.CompactDescriptor); ok {
			if cd.CompactDescription() == "" || len(cd.CompactSchema()) == 0 {
				t.Errorf("%s has empty compact desc/schema — missing compact.go entry", name)
			}
		}
	}
}

func TestPrecheckSilentForMoveFileGrep(t *testing.T) {
	// move_file / grep carry no anchor semantics — precheck must not
	// interfere (design §3), letting Execute produce its own errors.
	a := &AgentRunner{}
	cases := []struct {
		name string
		args string
	}{
		{"move_file", `{"source":"a.txt","destination":"b.txt"}`},
		{"grep", `{"pattern":"x"}`},
	}
	for _, tc := range cases {
		if msg := a.precheckTool(tc.name, []byte(tc.args)); msg != "" {
			t.Errorf("precheckTool(%s) = %q, want silent", tc.name, msg)
		}
	}
}

func TestGetConflictKeyEditTools(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{"move_file", `{"source":"a.txt","destination":"b.txt"}`, "!write"},
		{"grep", `{"pattern":"x","path":"src"}`, "file:src"}, // 刀1：与写工具统一资源键，同路径读写保序
		{"grep", `{"pattern":"x"}`, ""},
	}
	for _, tc := range cases {
		if got := getConflictKey(provider.ToolCall{Name: tc.name, Arguments: tc.args}); got != tc.want {
			t.Errorf("getConflictKey(%s, %s) = %q, want %q", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestRepeatSuccessSignatureCoversEditTools(t *testing.T) {
	for _, name := range []string{"move_file"} {
		tl := lookupEditTool(t, name)
		sig, ok := repeatSuccessSignature(provider.ToolCall{
			Name: name, Arguments: `{"path":"a.txt","source":"a.txt","destination":"b.txt"}`,
		}, tl)
		if !ok || !strings.HasPrefix(sig, name+"\x00") {
			t.Errorf("repeatSuccessSignature(%s) = %q ok=%v, want tool-prefixed signature", name, sig, ok)
		}
	}
	// grep is read-only → never signed (loop guard does not apply).
	grep := lookupEditTool(t, "grep")
	if _, ok := repeatSuccessSignature(provider.ToolCall{Name: "grep", Arguments: `{"pattern":"x"}`}, grep); ok {
		t.Error("grep must not participate in repeat-success signatures (ReadOnly)")
	}
}

func TestMoveFileLoopGuardAfterTwoSuccesses(t *testing.T) {
	tl := lookupEditTool(t, "move_file")
	a := &AgentRunner{sink: event.Discard, session: NewSession("sys")}
	call := provider.ToolCall{ID: "c1", Name: "move_file",
		Arguments: `{"source":"a.txt","destination":"b.txt"}`}
	for i := 0; i < repeatSuccessAllowed; i++ {
		a.recordRepeatSuccess(call, tl)
	}
	// v4.376 降级裁决：硬阻断退役——越线前既不 advisory 也不阻止。
	if a.advisoryRepeatSuccess() {
		t.Fatal("below the threshold no advisory may fire")
	}
	a.recordRepeatSuccess(call, tl)
	if !a.advisoryRepeatSuccess() {
		t.Fatal("crossing the threshold must inject the advisory")
	}
	// advisory 是合成 user 提醒，不是 blocked 结果。
	var adv string
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, "succeeded 3 times with identical arguments") {
			adv = m.Content
		}
	}
	if adv == "" || !strings.Contains(adv, "move_file") {
		t.Fatalf("advisory must name the tool and count, got %q", adv)
	}
	// 每签名每轮至多一条。
	if a.advisoryRepeatSuccess() {
		t.Fatal("advisory must fire once per signature per turn")
	}
}

func TestMoveFileInvalidatesBothCachePaths(t *testing.T) {
	dir := t.TempDir()
	src := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	dst := filepath.ToSlash(filepath.Join(dir, "b.txt"))
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools: editToolsRegistry(t, "move_file"),
		tc:    cache.New(0),
	}
	a.tc.Set(src, 0, 0, "cached-src")
	a.tc.Set(dst, 0, 0, "cached-dst")

	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + src + `","destination":"` + dst + `"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("move_file should execute cleanly, got blocked=%v err=%q out=%q", out.blocked, out.errMsg, out.output)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be gone, stat err=%v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination should exist, stat err=%v", err)
	}
	// The S0.6 risk-2 assertion: BOTH endpoints' cache entries are gone —
	// move_file args carry no "path" key, so the generic branch would miss them.
	if _, hit := a.tc.Get(src, 0, 0); hit {
		t.Error("cache entry for source must be invalidated after move_file")
	}
	if _, hit := a.tc.Get(dst, 0, 0); hit {
		t.Error("cache entry for destination must be invalidated after move_file")
	}
}

// TestEditFileCacheInvalidation 写后读缓存失效（edit_file 的 path 键路径）。
func TestEditFileCacheInvalidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools: editToolsRegistry(t, "edit_file"),
		tc:    cache.New(0),
	}
	// 缓存里放真实内容：edit_file 的 precheck 先查缓存核 old_string。
	a.tc.Set(path, 0, 0, "one\ntwo\n")

	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "edit_file",
		Arguments: `{"path":"` + path + `","old_string":"one","new_string":"ONE"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("edit_file should execute cleanly, got blocked=%v err=%q out=%q", out.blocked, out.errMsg, out.output)
	}
	if _, hit := a.tc.Get(path, 0, 0); hit {
		t.Error("cache entry must be invalidated after edit_file")
	}
	if content, _ := os.ReadFile(path); string(content) != "ONE\ntwo\n" {
		t.Errorf("file content = %q", content)
	}
}

func TestStaleGuardWriteFileOverwriteEscape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	newRunner := func() *AgentRunner {
		return &AgentRunner{
			tools:             editToolsRegistry(t, "write_file"),
			staleWrittenFiles: map[string]bool{path: true},
			sink:              event.Discard,
			session:           NewSession("sys"),
		}
	}
	// 未声明 overwrite：同回合重写仍被拦，提示必须给出分支出路。
	blocked := newRunner().executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "write_file",
		Arguments: `{"path":` + strconv.Quote(path) + `,"content":"two\n"}`,
	})
	if !blocked.blocked || !strings.Contains(blocked.output, `"overwrite": true`) ||
		!strings.Contains(blocked.output, "edit_file") {
		t.Fatalf("stale guard should block and name both exits: %+v", blocked)
	}

	// 显式整篇重写：跳过同回合闸，不要求无意义的旧内容重读。
	out := newRunner().executeOne(context.Background(), provider.ToolCall{
		ID: "c2", Name: "write_file",
		Arguments: `{"path":` + strconv.Quote(path) + `,"content":"two\n","overwrite":true}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("overwrite:true should bypass same-turn guard, got %+v", out)
	}
	if content, _ := os.ReadFile(path); string(content) != "two\n" {
		t.Fatalf("overwrite content = %q", content)
	}
}

// P2-10（2026-11-21 沿江反馈）：同回合写后 edit_file 的 old_string 仍锚在
// 当前磁盘内容上=新鲜锚点，放行——改一行不再被迫重读整篇（实测被拒 6 次）；
// 锚点失配仍拦（闸的本意）。
func TestStaleGuardEditFileAnchorFreshPass(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	newRunner := func() *AgentRunner {
		return &AgentRunner{
			tools:             editToolsRegistry(t, "edit_file"),
			staleWrittenFiles: map[string]bool{path: true},
			sink:              event.Discard,
			session:           NewSession("sys"),
		}
	}
	// 锚点匹配当前内容：放行且编辑生效。
	out := newRunner().executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "edit_file",
		Arguments: `{"path":` + strconv.Quote(path) + `,"old_string":"two","new_string":"TWO"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("fresh anchor should bypass same-turn guard, got %+v", out)
	}
	if content, _ := os.ReadFile(path); string(content) != "one\nTWO\n" {
		t.Fatalf("edit should apply, got %q", content)
	}
	// 锚点失配（内容已被同回合其他写改变）：仍拦（precheck 的 old_string 存在性
	// 诊断更精准，先于 stale 闸；两路任一即符合「失配不放行」）。
	out2 := newRunner().executeOne(context.Background(), provider.ToolCall{
		ID: "c2", Name: "edit_file",
		Arguments: `{"path":` + strconv.Quote(path) + `,"old_string":"不存在的锚点","new_string":"x"}`,
	})
	if !out2.blocked ||
		(!strings.Contains(out2.output, "[stale content]") && !strings.Contains(out2.output, "old_string not found")) {
		t.Fatalf("stale anchor must still be blocked, got %+v", out2)
	}
}

func TestStaleGuardBlocksMoveFileWithoutRead(t *testing.T) {
	dir := t.TempDir()
	src := filepath.ToSlash(filepath.Join(dir, "a.txt"))
	if err := os.WriteFile(src, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{
		tools:             editToolsRegistry(t, "move_file"),
		staleWrittenFiles: map[string]bool{src: true},
	}
	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + src + `","destination":"` + filepath.ToSlash(filepath.Join(dir, "b.txt")) + `"}`,
	})
	if !out.blocked || !strings.Contains(out.output, "[stale content]") {
		t.Fatalf("stale guard must block move_file on a stale source, got %+v", out)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("blocked move must not remove the source, stat err=%v", err)
	}
}

func TestMoveFileEndToEndViaRunner(t *testing.T) {
	// Full executeOne pass for move_file with a name-list check: the result
	// is the moved contract output, not "unknown tool".
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "sub", "b.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &AgentRunner{tools: editToolsRegistry(t, "move_file")}
	out := a.executeOne(context.Background(), provider.ToolCall{
		ID: "c1", Name: "move_file",
		Arguments: `{"source":"` + filepath.ToSlash(src) + `","destination":"` + filepath.ToSlash(dst) + `"}`,
	})
	if out.blocked || out.errMsg != "" {
		t.Fatalf("move_file failed: %+v", out)
	}
	if !strings.Contains(out.output, "moved") || !strings.Contains(out.output, "→") {
		t.Errorf("output = %q, want `moved <src> → <dst>`", out.output)
	}
	if content, err := os.ReadFile(dst); err != nil || string(content) != "payload" {
		t.Errorf("dst = %q err=%v", content, err)
	}
	// turnFilesModified bookkeeping: move_file is already on the list; the
	// extract helper must keep exposing the source for it.
	if p := extractFilePath("move_file", `{"source":"a.txt","destination":"b.txt"}`); p != "a.txt" {
		t.Errorf("extractFilePath(move_file) = %q, want source", p)
	}
}
