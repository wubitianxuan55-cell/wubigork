package agent

// v4.221 子代理证据落账回归：子代理回合收尾把写盘证据卡落 Journal，
// SessionID=run.Ref（sa_…），Journal 按会话分文件——DAG 节点产物按会话
// 精确归因（docs/gaea-office-dag-63-design-2026-09.md §3 定案）由此供给。
// 用例：①带 journalDir 的 TaskTool 跑完，卡落在 ref 名下；②同 TaskTool
// 二次运行产生独立会话文件（互不串账）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// changeTool 模拟写盘工具：经 evidence.RecordChange 上报一张证据卡
// （真实 write_file/edit_file 的上报路径同款，见 tool/builtin/writefile.go）。
type changeTool struct{ name, target string }

func (c changeTool) Name() string            { return c.name }
func (c changeTool) Description() string     { return "records a change card" }
func (c changeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (c changeTool) ReadOnly() bool          { return false }

func (c changeTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	evidence.RecordChange(ctx, evidence.ChangeRecord{Tool: c.name, Target: c.target})
	return "ok", nil
}

func TestSubagentRunJournalsWritesByRef(t *testing.T) {
	tmp := t.TempDir()
	journalDir := filepath.Join(tmp, "journal")
	store := NewSubagentStore(t.TempDir())

	reg := tool.NewRegistry()
	reg.Add(changeTool{name: "note", target: "docs/n.txt"})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "note", `{}`)},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	task := NewTaskTool(prov, nil, reg, 20, 0, 0.0, "", "sys", nil)
	task.WithTranscripts(store)
	task.SetSubagentJournalDir(journalDir)
	parentReg := tool.NewRegistry()
	parentReg.Add(task)

	out, err := task.Execute(context.Background(), []byte(`{"prompt":"make a note"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("final answer missing: %q", out)
	}

	// 恰好一个会话文件（=本次 run 的 ref），卡落在 ref 名下、Target 原样保留。
	st, err := evidence.OpenJournal(journalDir)
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	entries, err := os.ReadDir(journalDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var stems []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			stems = append(stems, strings.TrimSuffix(e.Name(), ".jsonl"))
		}
	}
	if len(stems) != 1 || !strings.HasPrefix(stems[0], "sa_") {
		t.Fatalf("journal sessions = %v, want exactly one sa_ session file", stems)
	}
	recs, err := st.List(stems[0])
	if err != nil || len(recs) != 1 {
		t.Fatalf("List %s: err=%v recs=%d", stems[0], err, len(recs))
	}
	if recs[0].SessionID != stems[0] || recs[0].Target != "docs/n.txt" {
		t.Fatalf("record not attributed to subagent session: %+v", recs[0])
	}

	// 二次运行：全新 ref → 独立会话文件，互不串账。（重置脚本计数器让第二跑
	// 再走一次 note 工具。）
	prov.call = 0
	if _, err := task.Execute(context.Background(), []byte(`{"prompt":"make a note"}`)); err != nil {
		t.Fatalf("Execute #2: %v", err)
	}
	entries, _ = os.ReadDir(journalDir)
	if len(entries) != 2 {
		t.Fatalf("second run should create its own session file, got %d entries", len(entries))
	}
}
