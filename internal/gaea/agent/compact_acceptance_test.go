package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A3 验收闸与配对完整性 ────────────────────────────────────────────────

// TestCompactPreservesToolPairIntegrity 是 closedPrefixEnd 的 gaea 回归形态：
// 压缩（含保留拼接）后，会话历史必须仍是合法形态——每个 tool 结果的调用
// 必须出现在它之前（上游「折叠区必须在闭合边界结束」不变式）。
func TestCompactPreservesToolPairIntegrity(t *testing.T) {
	s := NewSession("system prompt here")
	for i := 0; i < 30; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "do task " + strings.Repeat("u", 20)})
		s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{ID: string(rune('a'+i%26)) + string(rune('0'+i/26)), Name: "write_file", Arguments: `{"path":"f` + string(rune('a'+i%26)) + `.txt"}`},
		}})
		s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: string(rune('a'+i%26)) + string(rune('0'+i/26)), Name: "write_file", Content: strings.Repeat("o", 300)})
	}
	a := testAgentC(4000, 0.8, 5)
	a.session = s
	a.maybeCompact(context.Background(), &provider.Usage{PromptTokens: 3700})

	callIDs := map[string]bool{}
	for _, m := range a.session.Messages {
		for _, tc := range m.ToolCalls {
			callIDs[tc.ID] = true
		}
		if m.Role == provider.RoleTool {
			if !callIDs[m.ToolCallID] {
				t.Fatalf("tool 结果 %q 的调用不在其之前——折叠区拆散了配对", m.ToolCallID)
			}
		}
	}
}

// TestCompactSkipsWhenFixedPrefixDominates：固定前缀（digest 累计）占大头的
// 会话——tail 预算下限吃掉折叠区，候选下限仍 ≥ trigger → 验收闸跳过并上屏
// （上游同型场景=NoopCandidateAboveTrigger；例外路径的 ≥25% 节省在 gaea 的
// tail 下限常数下由 trigger 闸覆盖，见 compact.go 闸注释）。
func TestCompactSkipsWhenFixedPrefixDominates(t *testing.T) {
	sink := &collectorSink{}
	s := NewSession("system prompt here")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "start"})
	// 假 digest（summaryTagOpen 形态进 pinnedPrefixLen 固定前缀）：9000 字符
	s.Add(provider.Message{Role: provider.RoleUser, Content: summaryTagOpen + "\n" + strings.Repeat("d", 9000) + "\n" + summaryTagClose})
	// 足量后续内容让折叠区非空（> tail 预算 8000 字符）
	for i := 0; i < 30; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "more " + strings.Repeat("u", 20)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "ok " + strings.Repeat("w", 380)})
	}
	a := testAgentC(4000, 0.8, 5)
	a.sink = sink
	a.session = s
	before := len(s.Messages)
	a.maybeCompact(context.Background(), &provider.Usage{PromptTokens: 5400})

	if len(s.Messages) != before {
		t.Fatal("候选下限 ≥ trigger 应跳过（会话不缩）")
	}
	found := false
	for _, e := range sink.events {
		if e.Kind == event.Notice && strings.Contains(e.Text, "compaction skipped") {
			found = true
		}
	}
	if !found {
		t.Fatal("跳过应上屏说明口径")
	}
}

// ── A5 折叠区召回（recall_archived）──────────────────────────────────────

func writeArchive(t *testing.T, dir, name string, msgs []provider.Message) {
	t.Helper()
	var b strings.Builder
	for _, m := range msgs {
		line, _ := json.Marshal(m)
		b.WriteString(string(line) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRecallArchivedRanksAndBounds(t *testing.T) {
	dir := t.TempDir()
	writeArchive(t, dir, "20261024-100000.000.jsonl", []provider.Message{
		{Role: provider.RoleUser, Content: "请把配置切换到 postgres 数据库"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{ID: "c1", Name: "edit_file", Arguments: `{"path":"config.yaml"}`},
		}},
		{Role: provider.RoleTool, ToolCallID: "c1", Name: "edit_file", Content: "已把 driver: sqlite 改为 driver: postgres，端口 5432"},
	})
	writeArchive(t, dir, "20261024-110000.000.jsonl", []provider.Message{
		{Role: provider.RoleUser, Content: "写一首关于秋天的诗"},
		{Role: provider.RoleAssistant, Content: "秋风起兮白云飞……"},
	})
	tl := recallArchivedTool{}
	out, err := tl.Execute(WithArchiveDir(context.Background(), dir),
		[]byte(`{"query":"postgres 端口"}`))
	if err != nil {
		t.Fatalf("检索不应报错: %v", err)
	}
	if !strings.Contains(out, "postgres") || !strings.Contains(out, "5432") {
		t.Fatalf("命中片段应含关键内容: %q", out)
	}
	if !strings.Contains(out, "edit_file") {
		t.Fatalf("tool_output 片段应带所属调用（有来路）: %q", out)
	}
	if strings.Contains(out, "秋天") {
		t.Fatalf("无关命中（低分尾噪声）不应返回: %q", out)
	}
}

func TestRecallArchivedHonestEmptyAndMissing(t *testing.T) {
	tl := recallArchivedTool{}
	// 未配置 archive 目录
	if _, err := tl.Execute(context.Background(), []byte(`{"query":"x"}`)); err == nil {
		t.Fatal("未配置应如实报错")
	}
	// 目录存在但无命中
	dir := t.TempDir()
	writeArchive(t, dir, "a.jsonl", []provider.Message{
		{Role: provider.RoleUser, Content: "hello world"},
	})
	out, err := tl.Execute(WithArchiveDir(context.Background(), dir), []byte(`{"query":"数据库"}`))
	if err != nil {
		t.Fatalf("无命中是正常态不是错误: %v", err)
	}
	if !strings.Contains(out, "no matches") {
		t.Fatalf("无命中应诚实说明: %q", out)
	}
}
