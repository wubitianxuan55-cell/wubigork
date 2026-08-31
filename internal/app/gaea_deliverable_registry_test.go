package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDeliverableFixture 构造带事件日志的会话：turn1 写文件 + 编辑同一文件，
// turn2 move_file 改名 + chart_gen 出图（v4.24 生成类纳入）。
// 直接落 JSONL（受控 ts，验证 updatedAt 倒序）。
func writeDeliverableFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sessionDir := filepath.Join(root, ".gaea", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(sessionDir, "s1.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := []string{
		logLine(1, 100, "turn_started", map[string]any{}),
		logLine(2, 110, "tool_dispatch", map[string]any{"id": "t1", "name": "write_file", "args": `{"path":"docs/报告.docx","content":"v1"}`}),
		logLine(3, 120, "tool_dispatch", map[string]any{"id": "t2", "name": "edit_file", "args": `{"path":"docs/报告.docx","old_string":"a"}`}),
		logLine(4, 130, "turn_done", map[string]any{}),
		logLine(5, 200, "turn_started", map[string]any{}),
		logLine(6, 210, "tool_dispatch", map[string]any{"id": "t3", "name": "move_file", "args": `{"source":"docs/draft.md","destination":"docs/报告.docx"}`}),
		logLine(7, 220, "tool_dispatch", map[string]any{"id": "t4", "name": "chart_gen", "args": `{"output":"charts/成本.png"}`}),
		logLine(8, 230, "turn_done", map[string]any{}),
	}
	logPath := filepath.Join(sessionDir, "s1.gaea-log.jsonl")
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return sessionPath
}

// logLine 拼一行事件日志（与 session formatLogLine 形状一致：seq/ts/kind/payload）。
func logLine(seq, ts int64, kind string, payload any) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	raw, err := json.Marshal(map[string]any{"seq": seq, "ts": ts, "kind": kind, "payload": json.RawMessage(b)})
	if err != nil {
		return ""
	}
	return string(raw)
}

// TestGaeaDeliverableRegistry：折叠口径端到端（写+编辑同路径去重保留最近、
// move 只计 destination、生成类纳入、updatedAt 倒序、touches 累计）。
func TestGaeaDeliverableRegistry(t *testing.T) {
	sessionPath := writeDeliverableFixture(t)
	a := &App{core: &core{}}
	v := a.GaeaDeliverableRegistry(sessionPath)

	if !v.Available {
		t.Fatal("有事件日志 → Available=true")
	}
	if v.Total != 2 {
		t.Fatalf("total = %d, want 2（去重后：报告.docx + 成本.png）", v.Total)
	}
	if len(v.Entries) != 2 {
		t.Fatalf("entries = %d, want 2: %+v", len(v.Entries), v.Entries)
	}
	// updatedAt 倒序：charts/成本.png（220）在前
	first, second := v.Entries[0], v.Entries[1]
	if first.Path != "charts/成本.png" || first.Tool != "chart_gen" || first.Turn != 2 || first.UpdatedAt != 220 || first.Touches != 1 {
		t.Fatalf("entries[0] 异常：%+v", first)
	}
	// 报告.docx：写→编辑→移动 三次触碰，保留最近一次（move_file，turn 2）
	if second.Path != "docs/报告.docx" || second.Tool != "move_file" || second.Turn != 2 || second.UpdatedAt != 210 || second.Touches != 3 {
		t.Fatalf("entries[1] 异常：%+v", second)
	}
}

// TestGaeaDeliverableRegistry_NoLog：无事件日志返回 Available=false（不报错）。
func TestGaeaDeliverableRegistry_NoLog(t *testing.T) {
	t.Chdir(t.TempDir())
	sessionDir := filepath.Join(".gaea", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(sessionDir, "s1.jsonl")
	if err := os.WriteFile(sessionPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{}}
	if v := a.GaeaDeliverableRegistry(sessionPath); v.Available || len(v.Entries) != 0 {
		t.Fatalf("无日志应 Available=false： %+v", v)
	}
}

// TestGaeaDeliverableRegistry_Validation：空路径 / 会话目录外路径拒绝（防穿越）。
func TestGaeaDeliverableRegistry_Validation(t *testing.T) {
	a := &App{core: &core{}}
	if v := a.GaeaDeliverableRegistry(""); v.Available {
		t.Fatal("空路径应不可用")
	}
	if v := a.GaeaDeliverableRegistry(filepath.Join(t.TempDir(), "outside.jsonl")); v.Available {
		t.Fatal("会话目录外的路径应被拒绝")
	}
}

// TestGaeaDeliverableRegistry_ViewJSON：视图 JSON 形状（字段名逐个钉住，
// 前端按 available/entries/total 消费，entries 元素为 path/tool/turn/updatedAt/touches）。
func TestGaeaDeliverableRegistry_ViewJSON(t *testing.T) {
	sessionPath := writeDeliverableFixture(t)
	a := &App{core: &core{}}
	b, err := json.Marshal(a.GaeaDeliverableRegistry(sessionPath))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, field := range []string{`"available":`, `"entries":`, `"total":`, `"path":`, `"tool":`, `"turn":`, `"updatedAt":`, `"touches":`} {
		if !strings.Contains(s, field) {
			t.Fatalf("JSON 缺字段 %s：%s", field, s)
		}
	}
}
