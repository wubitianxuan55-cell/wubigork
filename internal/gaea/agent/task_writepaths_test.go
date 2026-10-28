package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A10 write_paths 声明（Reasonix 2.x fleet 预检蒸馏）测试 ──────────────

func TestNormalizeWritePathsRejectsGlobsAndEscapes(t *testing.T) {
	for _, bad := range [][]string{
		{"build/*"},
		{"src/**"},
		{"../outside"},
		{"a[0]"},
		{""},
		{"C:\\"},
		{"/"},
	} {
		if _, err := normalizeWritePaths(bad); err == nil {
			t.Fatalf("应拒绝 %v", bad)
		}
	}
	got, err := normalizeWritePaths([]string{"./build", "build", "src\\gen"}) // 去重+归一
	if err != nil {
		t.Fatalf("合法声明不应报错: %v", err)
	}
	if len(got) != 2 { // ./build 与 build 归一同路径去重；src\gen 保留
		t.Fatalf("归一去重后应 2 条: %v", got)
	}
}

func TestPathsOverlapDirSemantics(t *testing.T) {
	yes := [][2][]string{
		{{"/ws/a"}, {"/ws/a"}},              // 相同
		{{"/ws/a"}, {"/ws/a/b"}},            // 目录包含
		{{"/ws/a/b"}, {"/ws/a"}},            // 反向包含
		{{"/ws/x", "/ws/a"}, {"/ws/a/sub"}}, // 集合内任一命中
	}
	for i, c := range yes {
		if !pathsOverlap(c[0], c[1]) {
			t.Fatalf("case %d 应重叠: %v vs %v", i, c[0], c[1])
		}
	}
	no := [][2][]string{
		{{"/ws/a"}, {"/ws/ab"}}, // 前缀字符串但非路径边界
		{{"/ws/a"}, {"/ws/b"}},
	}
	for i, c := range no {
		if pathsOverlap(c[0], c[1]) {
			t.Fatalf("case %d 不应重叠: %v vs %v", i, c[0], c[1])
		}
	}
}

func TestWriteClaimRegistry(t *testing.T) {
	tt := &TaskTool{}
	if err := tt.acquireWriteClaim("sa_1", []string{`C:\ws\build`}); err != nil {
		t.Fatalf("首登记应成功: %v", err)
	}
	// 不相交可共存
	if err := tt.acquireWriteClaim("sa_2", []string{`C:\ws\docs`}); err != nil {
		t.Fatalf("不相交声明应共存: %v", err)
	}
	// 重叠 fail-fast（含目录包含）
	err := tt.acquireWriteClaim("sa_3", []string{`c:\WS\build\sub`})
	if err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("重叠声明应拒绝: %v", err)
	}
	// 释放后可再登记
	tt.releaseWriteClaim("sa_1")
	if err := tt.acquireWriteClaim("sa_3", []string{`C:\ws\build`}); err != nil {
		t.Fatalf("释放后应可登记: %v", err)
	}
}

func TestTaskConflictKeyDeclaredPaths(t *testing.T) {
	key := func(args string) string {
		return getConflictKey(provider.ToolCall{ID: "c1", Name: "task", Arguments: args})
	}
	a := key(`{"prompt":"x","write_paths":["C:\\ws\\a","C:\\ws\\b"]}`)
	b := key(`{"prompt":"y","write_paths":["C:\\ws\\a","C:\\ws\\b"]}`)
	c := key(`{"prompt":"z","write_paths":["C:\\ws\\c"]}`)
	if a != b {
		t.Fatal("同声明集应同键（同批必串行）")
	}
	if a == c {
		t.Fatal("异声明集应异键（可并行）")
	}
	if !strings.HasPrefix(a, "wpaths:") {
		t.Fatalf("声明键应带 wpaths 前缀: %s", a)
	}
	// 未声明维持旧行为
	u1 := key(`{"prompt":"x"}`)
	u2 := getConflictKey(provider.ToolCall{ID: "c2", Name: "task", Arguments: `{"prompt":"x"}`})
	if u1 != "spawn:c1" || u2 != "spawn:c2" {
		t.Fatalf("未声明应维持唯一键: %q %q", u1, u2)
	}
}

// ── A11 完成裁决（Reasonix 2.x complete_subtask 蒸馏）测试 ───────────────

func childSessionWithReceipts() *Session {
	s := NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "fix the build"})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "t1", Name: "bash", Arguments: `{"command":"go test ./..."}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "t1", Name: "bash", Content: "ok"})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "t2", Name: "edit_file", Arguments: `{"path":"C:\\ws\\main.go","content":"x"}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "t2", Name: "edit_file", Content: "written"})
	return s
}

func TestAdjudicateCompleteBackedByReceipts(t *testing.T) {
	sess := childSessionWithReceipts()
	report := completionReport{
		Status: "complete", Summary: "done",
		Criteria: []completionCriterion{
			{ID: "tests", Status: "satisfied", Evidence: []completionEvidence{{Kind: "verification", Command: "go test ./..."}}},
			{ID: "edit", Status: "satisfied", Evidence: []completionEvidence{{Kind: "diff", Paths: []string{`C:\ws\main.go`}}}},
		},
	}
	final, reasons := adjudicateCompletion(report, collectReceipts(sess.Messages))
	if final.Status != "complete" || len(reasons) != 0 {
		t.Fatalf("有回执支撑不应降级: status=%s reasons=%v", final.Status, reasons)
	}
}

func TestAdjudicateDowngradesUnsupportedClaims(t *testing.T) {
	sess := childSessionWithReceipts()
	report := completionReport{
		Status: "complete", Summary: "done",
		Criteria: []completionCriterion{
			{ID: "tests", Status: "satisfied", Evidence: []completionEvidence{{Kind: "verification", Command: "go vet ./..."}}},
			{ID: "phantom", Status: "satisfied", Evidence: []completionEvidence{{Kind: "manual", Summary: "trust me"}}},
			{ID: "nofile", Status: "satisfied", Evidence: []completionEvidence{{Kind: "diff", Paths: []string{`C:\ws\other.go`}}}},
		},
	}
	final, reasons := adjudicateCompletion(report, collectReceipts(sess.Messages))
	if final.Status != "partial" {
		t.Fatalf("无回执支撑应降 partial: %s", final.Status)
	}
	if len(reasons) != 3 {
		t.Fatalf("三条无支撑判据应各给一条理由: %v", reasons)
	}
	for _, m := range report.Criteria { // 原报告不被就地改动语义上无断言必要，但降级副本必须已改
		_ = m
	}
	if final.Criteria[0].Status != "unsatisfied" {
		t.Fatal("降级应落在判据上")
	}
}

func TestAdjudicateOnlyTouchesComplete(t *testing.T) {
	report := completionReport{Status: "partial", Summary: "half", Criteria: []completionCriterion{
		{ID: "x", Status: "satisfied", Evidence: []completionEvidence{{Kind: "manual"}}},
	}}
	final, reasons := adjudicateCompletion(report, nil)
	if final.Status != "partial" || reasons != nil {
		t.Fatalf("partial/blocked/failed 不是完成主张，不裁决: %s %v", final.Status, reasons)
	}
}

func TestChildCompletionVerdictNoReport(t *testing.T) {
	sess := childSessionWithReceipts()
	if v := childCompletionVerdict(sess); v != "" {
		t.Fatalf("无报告不应附裁语: %q", v)
	}
	if v := childCompletionVerdict(nil); v != "" {
		t.Fatalf("nil 会话应返回空: %q", v)
	}
}

func TestChildCompletionVerdictDowngradeVisible(t *testing.T) {
	sess := childSessionWithReceipts()
	raw, _ := json.Marshal(completionReport{Status: "complete", Summary: "done",
		Criteria: []completionCriterion{{ID: "tests", Status: "satisfied",
			Evidence: []completionEvidence{{Kind: "verification", Command: "go vet ./..."}}}}})
	sess.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "t3", Name: completeSubtaskName, Arguments: string(raw)},
	}})
	sess.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "t3", Name: completeSubtaskName, Content: "accepted"})
	v := childCompletionVerdict(sess)
	if !strings.Contains(v, "partial") || !strings.Contains(v, "go vet") {
		t.Fatalf("降级裁语应含结论与缺失证据: %q", v)
	}
}

func TestCompleteSubtaskToolAlwaysSucceeds(t *testing.T) {
	tl := completeSubtaskTool{}
	out, err := tl.Execute(context.TODO(), []byte(`{"status":"complete","summary":"done"}`))
	if err != nil || !strings.Contains(out, "accepted") {
		t.Fatalf("提交应永远成功: %v %q", err, out)
	}
	if _, err := tl.Execute(context.TODO(), []byte(`{"status":"bogus","summary":"x"}`)); err == nil {
		t.Fatal("非法状态应报错")
	}
}
