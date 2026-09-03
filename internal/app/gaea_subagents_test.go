package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// writeSubagentFixture 构造一个带 subagents 目录的会话：两个子代理
// （一个 completed 带 transcript、一个 running 无 transcript）。
func writeSubagentFixture(t *testing.T) (sessionPath, sessionDir string) {
	t.Helper()
	root := t.TempDir()
	// 会话目录形态：<root>/.gaea/sessions/<file>.jsonl（sessionDirForPath 校验要求）
	sessionDir = filepath.Join(root, ".gaea", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath = filepath.Join(sessionDir, "s1.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(sessionDir, "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 子代理 A：completed，带 transcript（任务 + 工具调用 + 回答）
	refA := "sa_20260817_100000_0000000001_a1a1a1a1"
	metaA := map[string]interface{}{
		"ref": refA, "status": "completed",
		"createdAt": time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339Nano),
		"updatedAt": time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339Nano),
		"model":     "deepseek-v4-flash", "toolScope": []string{"read_file", "grep"},
	}
	writeJSON(t, filepath.Join(subDir, refA+".meta.json"), metaA)
	transcriptA := []provider.Message{
		{Role: provider.RoleUser, Content: "找出项目里所有调用 format_convert 的地方并总结模式"},
		{Role: provider.RoleTool, Name: "grep"},
		{Role: provider.RoleAssistant, Content: "共发现 5 处调用，集中在 format_convert.go 与工具注册表。"},
	}
	writeMessages(t, filepath.Join(subDir, refA+".jsonl"), transcriptA)

	// 子代理 B：running，无 transcript（只有 meta）
	refB := "sa_20260817_110000_0000000002_b2b2b2b2"
	metaB := map[string]interface{}{
		"ref": refB, "status": "running",
		"createdAt": time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339Nano),
		"updatedAt": time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339Nano),
	}
	writeJSON(t, filepath.Join(subDir, refB+".meta.json"), metaB)
	return sessionPath, sessionDir
}

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMessages(t *testing.T, path string, msgs []provider.Message) {
	t.Helper()
	// transcript 是 JSONL（每条消息一行），与 session.Save 的落盘格式一致
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, m := range msgs {
		if err := enc.Encode(m); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGaeaSubagentRuns(t *testing.T) {
	sessionPath, _ := writeSubagentFixture(t)
	a := &App{core: &core{}}
	v := a.GaeaSubagentRuns(sessionPath)

	if !v.Available {
		t.Fatal("应有 subagents 目录 → Available=true")
	}
	if v.Total != 2 || v.Running != 1 {
		t.Fatalf("total=%d running=%d，期望 2/1", v.Total, v.Running)
	}
	// 按创建时间倒序：B（新）在前，A（旧）在后
	if len(v.Runs) != 2 || !strings.HasPrefix(v.Runs[0].Ref, "sa_") {
		t.Fatalf("runs 数量或排序异常：%+v", v.Runs)
	}
	if v.Runs[0].Status != "running" || v.Runs[1].Status != "completed" {
		t.Fatalf("状态顺序异常：%s / %s", v.Runs[0].Status, v.Runs[1].Status)
	}
	// A 的任务摘要 / 回答 / 工具计数来自 transcript
	completed := v.Runs[1]
	if !strings.Contains(completed.Task, "format_convert") {
		t.Fatalf("任务摘要异常：%q", completed.Task)
	}
	if !strings.Contains(completed.Answer, "5 处调用") {
		t.Fatalf("回答摘要异常：%q", completed.Answer)
	}
	if completed.ToolCalls != 1 {
		t.Fatalf("工具调用计数 = %d，期望 1", completed.ToolCalls)
	}
	if completed.Model != "deepseek-v4-flash" || len(completed.ToolScope) != 2 {
		t.Fatalf("meta 字段异常：model=%q scope=%v", completed.Model, completed.ToolScope)
	}
}

func TestGaeaSubagentRuns_NoSubagents(t *testing.T) {
	t.Chdir(t.TempDir())
	// 构造合法会话目录但无 subagents/ 子目录
	sessionDir := filepath.Join(".gaea", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(sessionDir, "s1.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{}}
	if v := a.GaeaSubagentRuns(sessionPath); v.Available {
		t.Fatal("无 subagents 目录应 Available=false")
	}
}

// TestGaeaSubagentRuns_ModelTool：mt_（本地模型工具）记录与 sa_ 同列表可见，
// kind=model_tool + tool 名 + meta.Title 优先作任务摘要；GaeaSubagentTranscript
// 也接受 mt_ ref（完整输入输出可回看）。
func TestGaeaSubagentRuns_ModelTool(t *testing.T) {
	sessionPath, sessionDir := writeSubagentFixture(t)
	subDir := filepath.Join(sessionDir, "subagents")
	ref := "mt_20260903_120000_0000000001_c3c3c3c3"
	meta := map[string]interface{}{
		"ref": ref, "status": "completed", "kind": "model_tool", "tool": "vision",
		"title":     "vision · 识别图片 C:\\x.png",
		"createdAt": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
		"updatedAt": time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano),
	}
	writeJSON(t, filepath.Join(subDir, ref+".meta.json"), meta)
	writeMessages(t, filepath.Join(subDir, ref+".jsonl"), []provider.Message{
		{Role: provider.RoleUser, Content: "vision · 识别图片 C:\\x.png"},
		{Role: provider.RoleAssistant, Content: "图中标题：2026 年 9 月产值表；表格 3 行。"},
	})

	a := &App{core: &core{}}
	v := a.GaeaSubagentRuns(sessionPath)
	var mt *SubagentRunView
	for i := range v.Runs {
		if v.Runs[i].Ref == ref {
			mt = &v.Runs[i]
			break
		}
	}
	if mt == nil {
		t.Fatalf("mt_ 记录未出现在 runs：%+v", v.Runs)
	}
	if mt.Kind != "model_tool" || mt.Tool != "vision" {
		t.Fatalf("kind/tool 异常：kind=%q tool=%q", mt.Kind, mt.Tool)
	}
	if !strings.Contains(mt.Task, "识别图片") || !strings.Contains(mt.Answer, "产值表") {
		t.Fatalf("task/answer 异常：%q / %q", mt.Task, mt.Answer)
	}

	tv, err := a.GaeaSubagentTranscript(sessionPath, ref)
	if err != nil {
		t.Fatalf("GaeaSubagentTranscript(mt_): %v", err)
	}
	if len(tv.Messages) != 2 || tv.Messages[0].Role != "user" || tv.Messages[1].Role != "assistant" {
		t.Fatalf("transcript 结构异常：%+v", tv.Messages)
	}
	if !strings.Contains(tv.Task, "识别图片") {
		t.Fatalf("transcript task 未取 meta.Title：%q", tv.Task)
	}
	// 旧 sa_ meta 无 kind → 读端补 subagent（不回归）
	for _, r := range v.Runs {
		if strings.HasPrefix(r.Ref, "sa_") && r.Kind != "subagent" {
			t.Fatalf("旧 sa_ 记录 kind 应为 subagent，实际 %q", r.Kind)
		}
	}
}

func TestGaeaSubagentRuns_Validation(t *testing.T) {
	a := &App{core: &core{}}
	// 空路径 / 非法路径（sessionDirForPath 拒绝）
	if v := a.GaeaSubagentRuns(""); v.Available {
		t.Fatal("空路径应不可用")
	}
	if v := a.GaeaSubagentRuns(filepath.Join(t.TempDir(), "outside.jsonl")); v.Available {
		t.Fatal("会话目录外的路径应被拒绝")
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("短文本", 10); got != "短文本" {
		t.Fatalf("短文本不应截断：%q", got)
	}
	long := strings.Repeat("汉", 200)
	got := truncateRunes(long, 120)
	if len([]rune(got)) != 120 || !strings.HasSuffix(got, "…") {
		t.Fatalf("截断结果异常：len=%d suffix=%q", len([]rune(got)), got[len(got)-1:])
	}
}

func TestSummarizeSubagentTranscript_Missing(t *testing.T) {
	task, answer, calls, lastText, lastTool := summarizeSubagentTranscript(filepath.Join(t.TempDir(), "nope.jsonl"))
	if task != "" || answer != "" || calls != 0 || lastText != "" || lastTool != "" {
		t.Fatalf("缺失 transcript 应返回空：%q/%q/%d/%q/%q", task, answer, calls, lastText, lastTool)
	}
}

// TestSummarizeSubagentTranscript_Activity：C2 活动行——lastText（最后 assistant 文本）
// 与 lastTool（最后一次工具调用 name+结果头）从 transcript 尾部派生。
func TestSummarizeSubagentTranscript_Activity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sa_x.jsonl")
	lines := []string{
		`{"role":"user","content":"帮我查一下机械挖土方综合单价"}`,
		`{"role":"assistant","content":"我来检索成本库。"}`,
		`{"role":"tool","name":"cost_search","content":"找到 3 条\nHP300 台班 3200\n…"}`,
		`{"role":"assistant","content":"结论：机械挖土方综合单价约 12.5 元/m³"}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("写 transcript: %v", err)
	}
	task, answer, calls, lastText, lastTool := summarizeSubagentTranscript(path)
	if task != "帮我查一下机械挖土方综合单价" {
		t.Fatalf("任务摘要异常：%q", task)
	}
	if !strings.Contains(answer, "结论：机械挖土方") {
		t.Fatalf("最后回答异常：%q", answer)
	}
	if calls != 1 {
		t.Fatalf("工具调用计数异常：%d", calls)
	}
	if lastText != "结论：机械挖土方综合单价约 12.5 元/m³" {
		t.Fatalf("lastText 异常：%q", lastText)
	}
	if !strings.HasPrefix(lastTool, "cost_search: 找到 3 条") {
		t.Fatalf("lastTool 异常：%q", lastTool)
	}
	if strings.Contains(lastTool, "\n") {
		t.Fatalf("lastTool 应为单行摘要，实际含换行：%q", lastTool)
	}
}
