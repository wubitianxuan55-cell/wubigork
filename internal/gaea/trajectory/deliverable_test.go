package trajectory

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// deliverableEntry 复用 fold_test.go 的 entry 形状，但允许指定 ts（登记表的
// updatedAt 倒序是核心语义，需要可控行时钟）。
func dEntry(seq, ts int64, kind string, payload any) session.LogEntry {
	b, _ := json.Marshal(payload)
	return session.LogEntry{Seq: seq, Ts: ts, Kind: kind, Payload: b}
}

func dDispatch(seq, ts int64, name, args string) session.LogEntry {
	return dEntry(seq, ts, "tool_dispatch", map[string]any{"id": "t" + itoa(int(seq)), "name": name, "args": args})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// TestFoldDeliverablesEmpty：空输入 → Available=true、Entries 序列化为 []
// 而非 null（nil 切片 JSON null 前端崩页的老坑）。
func TestFoldDeliverablesEmpty(t *testing.T) {
	reg := FoldDeliverables(nil)
	if !reg.Available || len(reg.Entries) != 0 || reg.Total != 0 {
		t.Fatalf("empty fold wrong: %+v", reg)
	}
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "null") {
		t.Fatalf("serialized JSON contains null: %s", b)
	}
}

// TestFoldDeliverablesGolden 黄金表：写类 + 生成/导出类工具的登记、去重、
// 最近保留、touches 累计、updatedAt 倒序、turn 归属、只计 destination。
func TestFoldDeliverablesGolden(t *testing.T) {
	cases := []struct {
		name    string
		entries []session.LogEntry
		want    []DeliverableEntry // 期望顺序 = updatedAt 倒序
		total   int
	}{
		{
			name: "write+edit 去重保留最近，touches 累计",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "write_file", `{"path":"docs/报告.docx","content":"v1"}`),
				dDispatch(3, 120, "edit_file", `{"path":"docs/报告.docx","old_string":"a","new_string":"b"}`),
				dDispatch(4, 130, "read_file", `{"path":"docs/报告.docx"}`), // 只读不登记
			},
			want: []DeliverableEntry{
				{Path: "docs/报告.docx", Tool: "edit_file", Turn: 1, UpdatedAt: 120, Touches: 2},
			},
			total: 1,
		},
		{
			name: "updatedAt 倒序 + turn 归属 + 多路径",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "write_file", `{"path":"docs/旧.md"}`),
				dEntry(3, 120, "turn_done", map[string]any{}),
				dEntry(4, 200, "turn_started", map[string]any{}),
				dDispatch(5, 210, "write_file", `{"path":"docs/新.md"}`),
			},
			want: []DeliverableEntry{
				{Path: "docs/新.md", Tool: "write_file", Turn: 2, UpdatedAt: 210, Touches: 1},
				{Path: "docs/旧.md", Tool: "write_file", Turn: 1, UpdatedAt: 110, Touches: 1},
			},
			total: 2,
		},
		{
			name: "move_file 只计 destination（源不是交付物）",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "move_file", `{"source":"docs/draft.md","destination":"docs/终稿.md"}`),
			},
			want: []DeliverableEntry{
				{Path: "docs/终稿.md", Tool: "move_file", Turn: 1, UpdatedAt: 110, Touches: 1},
			},
			total: 1,
		},
		{
			name: "生成/导出类工具（v4.24 新纳入，前端白名单此前漏登）",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "format_convert", `{"path":"docs/方案.docx","output":"docs/方案.md"}`),
				dDispatch(3, 120, "chart_gen", `{"output":"charts/成本.png"}`),
				dDispatch(4, 130, "diagram_gen", `{"output":"charts/拓扑.png"}`),
			},
			want: []DeliverableEntry{
				{Path: "charts/拓扑.png", Tool: "diagram_gen", Turn: 1, UpdatedAt: 130, Touches: 1},
				{Path: "charts/成本.png", Tool: "chart_gen", Turn: 1, UpdatedAt: 120, Touches: 1},
				{Path: "docs/方案.md", Tool: "format_convert", Turn: 1, UpdatedAt: 110, Touches: 1},
			},
			total: 3,
		},
		{
			name: "bash/screen_capture 不登记（路径无法从结构化参数权威提取）",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "bash", `{"command":"cp a.md b.md"}`),
				dDispatch(3, 120, "screen_capture", `{}`),
			},
			want:  []DeliverableEntry{},
			total: 0,
		},
		{
			name: "paths 数组 + edits 片段 + 覆盖键去重",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dDispatch(2, 110, "multi_edit", `{"edits":[{"path":"a.md"},{"file_path":"b.md"},{"path":"a.md"}],"paths":["c.md"]}`),
			},
			want: []DeliverableEntry{
				{Path: "a.md", Tool: "multi_edit", Turn: 1, UpdatedAt: 110, Touches: 1},
				{Path: "b.md", Tool: "multi_edit", Turn: 1, UpdatedAt: 110, Touches: 1},
				{Path: "c.md", Tool: "multi_edit", Turn: 1, UpdatedAt: 110, Touches: 1},
			},
			total: 3,
		},
		{
			name: "迁移/投影产物：assistant_message 内嵌工具调用与 tool_dispatch 同面",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dEntry(2, 110, "assistant_message", map[string]any{
					"text": "已写入。",
					"tool_calls": []map[string]any{
						{"id": "c1", "name": "write_file", "args": `{"path":"out/周报.docx"}`},
					},
				}),
			},
			want: []DeliverableEntry{
				{Path: "out/周报.docx", Tool: "write_file", Turn: 1, UpdatedAt: 110, Touches: 1},
			},
			total: 1,
		},
		{
			name: "partial 派发不登记（参数尚未完整）",
			entries: []session.LogEntry{
				dEntry(1, 100, "turn_started", map[string]any{}),
				dEntry(2, 110, "tool_dispatch", map[string]any{"id": "t1", "name": "write_file", "args": `{"path":"x.md"}`, "partial": true}),
			},
			want:  []DeliverableEntry{},
			total: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := FoldDeliverables(tc.entries)
			if !reg.Available {
				t.Fatal("fold 结果应 Available=true")
			}
			if reg.Total != tc.total {
				t.Fatalf("total = %d, want %d", reg.Total, tc.total)
			}
			if len(reg.Entries) != len(tc.want) {
				t.Fatalf("entries 数 = %d, want %d: %+v", len(reg.Entries), len(tc.want), reg.Entries)
			}
			for i, w := range tc.want {
				got := reg.Entries[i]
				if got != w {
					t.Fatalf("entries[%d] = %+v, want %+v", i, got, w)
				}
			}
			// 纯函数：同输入必同输出
			again := FoldDeliverables(tc.entries)
			if len(again.Entries) != len(reg.Entries) {
				t.Fatal("纯函数被破坏：同输入两次折叠结果不一致")
			}
		})
	}
}

// TestFoldDeliverablesCap200：上限 200 条防爆炸，Total 仍返回完整登记数。
func TestFoldDeliverablesCap200(t *testing.T) {
	entries := []session.LogEntry{dEntry(1, 100, "turn_started", map[string]any{})}
	seq := int64(2)
	for i := 0; i < 260; i++ {
		entries = append(entries, dDispatch(seq, 100+int64(i), "write_file", `{"path":"p/`+itoa(i)+`.md"}`))
		seq++
	}
	reg := FoldDeliverables(entries)
	if reg.Total != 260 {
		t.Fatalf("total = %d, want 260（截断前完整数）", reg.Total)
	}
	if len(reg.Entries) != maxDeliverables {
		t.Fatalf("entries = %d, want %d", len(reg.Entries), maxDeliverables)
	}
	// 倒序：最旧的 p/0.md 应被截掉，最旧的保留条目是 p/60.md（ts=160）
	if reg.Entries[maxDeliverables-1].Path != "p/60.md" {
		t.Fatalf("最旧保留条目 = %s, want p/60.md", reg.Entries[maxDeliverables-1].Path)
	}
	if reg.Entries[0].Path != "p/259.md" {
		t.Fatalf("最新条目 = %s, want p/259.md", reg.Entries[0].Path)
	}
}
