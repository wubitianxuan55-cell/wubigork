package contextview

// 耗时统计（ContextTiming）折叠测试：口径边界——全指标、无工具、无增量、
// 多轮与未闭合末轮、配对边界（partial/未配对/迁移日志）、排行截断 20。
// 日志 ts 为秒级，所有期望值都是秒差 ×1000。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// tsEntry 构造指定 ts（秒）的日志条目（fold_test.go 的 entry 固定 ts=base+seq，
// 不适合时长断言）。
func tsEntry(seq, ts int64, kind string, payload any) session.LogEntry {
	b, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return session.LogEntry{Seq: seq, Ts: ts, Kind: kind, Payload: b}
}

func dispatchPayload(id, name string) any {
	return map[string]any{"id": id, "name": name, "args": `{}`, "partial": false}
}

func TestTimingFullTurn(t *testing.T) {
	// 单轮两步：header→reasoning/text→message→usage→并行工具→header→text→
	// message→usage→turn_done。
	const base = int64(1700000000)
	entries := []session.LogEntry{
		tsEntry(1, base, "turn_started", map[string]any{}),
		tsEntry(2, base, "request_header", headerPayload("sys", "read_file", "bash")),
		tsEntry(3, base+2, "reasoning", map[string]any{"text": "思考增量"}),
		tsEntry(4, base+3, "text", map[string]any{"text": "文本增量"}),
		tsEntry(5, base+5, "message", map[string]any{"text": "第一步回复"}),
		tsEntry(6, base+5, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": 1}),
		tsEntry(7, base+5, "tool_dispatch", dispatchPayload("t1", "read_file")),
		tsEntry(8, base+5, "tool_dispatch", dispatchPayload("t2", "bash")),
		tsEntry(9, base+8, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "body"}),
		tsEntry(10, base+12, "tool_result", map[string]any{"id": "t2", "name": "bash", "output": "ok"}),
		tsEntry(11, base+12, "request_header", headerPayload("sys", "read_file", "bash")),
		tsEntry(12, base+13, "text", map[string]any{"text": "第二步增量"}),
		tsEntry(13, base+15, "message", map[string]any{"text": "第二步回复"}),
		tsEntry(14, base+15, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": 1}),
		tsEntry(15, base+20, "turn_done", map[string]any{"err": ""}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	tm := tl.Timing
	if tm == nil {
		t.Fatal("timing should be present")
	}
	if tm.WallMs != 20_000 {
		t.Errorf("wallMs = %d, want 20000（turn_started→turn_done）", tm.WallMs)
	}
	// step1: header@0→首增量 reasoning@2；step2: header@12→首增量 text@13。
	if tm.TTFTMs != 3_000 {
		t.Errorf("ttftMs = %d, want 3000（2000+1000）", tm.TTFTMs)
	}
	// step1: 首 token@2→message@5；step2: 首 token@13→message@15。
	if tm.GenMs != 5_000 {
		t.Errorf("genMs = %d, want 5000（3000+2000）", tm.GenMs)
	}
	if tm.Calls != 2 {
		t.Errorf("calls = %d, want 2（每步一条 message）", tm.Calls)
	}
	// 工具并行派发、分别收结果：重复计（与 dsh 同口径）。
	if tm.ToolsMs != 10_000 || tm.ToolCalls != 2 {
		t.Errorf("toolsMs/toolCalls = %d/%d, want 10000/2", tm.ToolsMs, tm.ToolCalls)
	}
	if len(tm.Tools) != 2 {
		t.Fatalf("tools = %d entries, want 2: %+v", len(tm.Tools), tm.Tools)
	}
	// 按 ms 降序：bash(7000) 在 read_file(3000) 前。
	if tm.Tools[0].Name != "bash" || tm.Tools[0].Calls != 1 || tm.Tools[0].Ms != 7_000 {
		t.Errorf("tools[0] = %+v, want bash 1×7000", tm.Tools[0])
	}
	if tm.Tools[1].Name != "read_file" || tm.Tools[1].Ms != 3_000 {
		t.Errorf("tools[1] = %+v, want read_file 1×3000", tm.Tools[1])
	}
}

func TestTimingNoToolsNoDeltas(t *testing.T) {
	// 纯问答轮：无工具（tools* 三项应缺省），无增量落盘（纯工具步口径：
	// message 即首个产物，等待整体记 ttft，gen 留 0）。
	const base = int64(1700000000)
	entries := []session.LogEntry{
		tsEntry(1, base, "turn_started", map[string]any{}),
		tsEntry(2, base, "request_header", headerPayload("sys")),
		tsEntry(3, base+4, "message", map[string]any{"text": "直接回答"}),
		tsEntry(4, base+4, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": 1}),
		tsEntry(5, base+6, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	tm := tl.Timing
	if tm == nil {
		t.Fatal("timing should be present")
	}
	if tm.WallMs != 6_000 {
		t.Errorf("wallMs = %d, want 6000", tm.WallMs)
	}
	if tm.TTFTMs != 4_000 {
		t.Errorf("ttftMs = %d, want 4000（header→message 充当首 token）", tm.TTFTMs)
	}
	if tm.GenMs != 0 {
		t.Errorf("genMs = %d, want 0（无增量可测，不伪造）", tm.GenMs)
	}
	if tm.Calls != 1 {
		t.Errorf("calls = %d, want 1", tm.Calls)
	}
	if tm.ToolsMs != 0 || tm.ToolCalls != 0 || len(tm.Tools) != 0 {
		t.Errorf("tools metrics should be absent: %+v", tm)
	}
	// omitempty：序列化后不应出现工具三项键。
	b, err := json.Marshal(tm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"toolsMs"`, `"toolCalls"`, `"tools"`} {
		if strings.Contains(string(b), key) {
			t.Errorf("timing JSON should omit %s: %s", key, b)
		}
	}
}

func TestTimingMultiTurnOpenTail(t *testing.T) {
	// 两轮：第一轮正常闭合；末轮 turn_done 缺失（中断）→ 以最后一条日志收尾。
	const base = int64(1700000000)
	entries := []session.LogEntry{
		tsEntry(1, base, "turn_started", map[string]any{}),
		tsEntry(2, base, "request_header", headerPayload("sys")),
		tsEntry(3, base+10, "message", map[string]any{"text": "第一轮回复"}),
		tsEntry(4, base+15, "turn_done", map[string]any{}),
		tsEntry(5, base+20, "turn_started", map[string]any{}),
		tsEntry(6, base+20, "request_header", headerPayload("sys")),
		tsEntry(7, base+25, "message", map[string]any{"text": "第二轮回复"}),
		tsEntry(8, base+25, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": 2}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	tm := tl.Timing
	if tm == nil {
		t.Fatal("timing should be present")
	}
	if tm.WallMs != 20_000 {
		t.Errorf("wallMs = %d, want 20000（15000 + 末轮 5000 按最后一条收尾）", tm.WallMs)
	}
	if tm.Calls != 2 {
		t.Errorf("calls = %d, want 2", tm.Calls)
	}
	// 两步都无增量：ttft 各自 header→message（10s + 5s）。
	if tm.TTFTMs != 15_000 {
		t.Errorf("ttftMs = %d, want 15000", tm.TTFTMs)
	}
}

func TestTimingPairingEdges(t *testing.T) {
	// 配对边界：partial 派发（流式预发射）、无结果的派发（中断）、无派发的
	// 结果（迁移日志）都不计数；同工具并行两次重复计。
	const base = int64(1700000000)
	entries := []session.LogEntry{
		tsEntry(1, base, "turn_started", map[string]any{}),
		tsEntry(2, base, "request_header", headerPayload("sys", "grep")),
		tsEntry(3, base, "tool_dispatch", map[string]any{"id": "p1", "name": "grep", "args": `{}`, "partial": true}),
		tsEntry(4, base+9, "tool_result", map[string]any{"id": "p1", "name": "grep", "output": "x"}),
		tsEntry(5, base+10, "tool_dispatch", dispatchPayload("d1", "bash")),
		tsEntry(6, base+10, "tool_dispatch", dispatchPayload("a1", "grep")),
		tsEntry(7, base+10, "tool_dispatch", dispatchPayload("a2", "grep")),
		tsEntry(8, base+14, "tool_result", map[string]any{"id": "a1", "name": "grep", "output": "x"}),
		tsEntry(9, base+16, "tool_result", map[string]any{"id": "a2", "name": "grep", "output": "x"}),
		tsEntry(10, base+18, "tool_result", map[string]any{"id": "m1", "name": "read_file", "output": "迁移日志无派发"}),
		tsEntry(11, base+20, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	tm := tl.Timing
	if tm == nil {
		t.Fatal("timing should be present")
	}
	// 只有 a1(4s)+a2(6s) 配对成功：grep 2×10000。
	if tm.ToolsMs != 10_000 || tm.ToolCalls != 2 {
		t.Errorf("toolsMs/toolCalls = %d/%d, want 10000/2（partial/未配对/迁移结果均不计）", tm.ToolsMs, tm.ToolCalls)
	}
	if len(tm.Tools) != 1 || tm.Tools[0].Name != "grep" || tm.Tools[0].Calls != 2 || tm.Tools[0].Ms != 10_000 {
		t.Errorf("tools = %+v, want grep 2×10000", tm.Tools)
	}
}

func TestTimingToolsCap20(t *testing.T) {
	// 25 个工具各配对一次，时长递减 → 排行截断 20，toolsMs/toolCalls 仍全量。
	const base = int64(1700000000)
	var entries []session.LogEntry
	seq := int64(1)
	entries = append(entries, tsEntry(seq, base, "turn_started", map[string]any{}))
	entries = append(entries, tsEntry(seq, base, "request_header", headerPayload("sys")))
	totalMs := int64(0)
	for i := 1; i <= 25; i++ {
		dur := int64(25 - i) // t01=24s … t25=0s
		totalMs += dur * 1000
		id := "d" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		entries = append(entries, tsEntry(seq, base, "tool_dispatch", dispatchPayload(id, "tool_"+twoDigits(i))))
		entries = append(entries, tsEntry(seq, base+dur, "tool_result", map[string]any{"id": id, "name": "tool_" + twoDigits(i), "output": "x"}))
		seq += 2
	}
	entries = append(entries, tsEntry(seq, base, "turn_done", map[string]any{}))
	tl := FoldTimeline(entries, 1_000_000, 0)
	tm := tl.Timing
	if tm == nil {
		t.Fatal("timing should be present")
	}
	if len(tm.Tools) != 20 {
		t.Fatalf("tools = %d entries, want 20（上限截断）", len(tm.Tools))
	}
	if tm.Tools[0].Name != "tool_01" || tm.Tools[0].Ms != 24_000 {
		t.Errorf("tools[0] = %+v, want tool_01 24000（ms 降序榜首）", tm.Tools[0])
	}
	if tm.Tools[19].Name != "tool_20" {
		t.Errorf("tools[19] = %s, want tool_20", tm.Tools[19].Name)
	}
	if tm.ToolsMs != totalMs || tm.ToolCalls != 25 {
		t.Errorf("toolsMs/toolCalls = %d/%d, want %d/25（合计不截断）", tm.ToolsMs, tm.ToolCalls, totalMs)
	}
}

// twoDigits 把 1..99 格式化为两位十进制（测试工具名定序用）。
func twoDigits(i int) string {
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func TestTimingOmittedWhenUnmeasurable(t *testing.T) {
	// 迁移日志形态：全部条目同一秒（ToLogEntries 用同一 now）→ 所有差值为零
	// ……但 calls 仍真实可数。完全无可测数据（零事件、全零指标）时 Timing
	// 整体省略，不伪造「0ms 卡」。
	t.Run("empty session", func(t *testing.T) {
		tl := FoldTimeline(nil, 1_000_000, 0)
		if tl.Timing != nil {
			t.Fatalf("empty fold should omit timing, got %+v", tl.Timing)
		}
		b, _ := json.Marshal(tl)
		if strings.Contains(string(b), `"timing"`) {
			t.Fatalf("empty timeline JSON should have no timing key: %s", b)
		}
	})
	t.Run("same-second migrated log", func(t *testing.T) {
		// 同秒迁移日志：时长全零但 calls=1 可数 → Timing 存在且只有 calls。
		const now = int64(1700000000)
		entries := []session.LogEntry{
			tsEntry(1, now, "turn_started", map[string]any{}),
			tsEntry(2, now, "request_header", headerPayload("sys")),
			tsEntry(3, now, "assistant_message", map[string]any{"text": "旧会话回复"}),
			tsEntry(4, now, "turn_done", map[string]any{}),
		}
		tl := FoldTimeline(entries, 1_000_000, 0)
		tm := tl.Timing
		if tm == nil {
			t.Fatal("timing should carry the countable calls")
		}
		if tm.Calls != 1 {
			t.Errorf("calls = %d, want 1", tm.Calls)
		}
		if tm.WallMs != 0 || tm.TTFTMs != 0 || tm.GenMs != 0 || tm.ToolsMs != 0 {
			t.Errorf("same-second log must not invent durations: %+v", tm)
		}
	})
}
