package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// line 是测试用的一行日志。
type line struct {
	Seq     int64          `json:"seq"`
	Ts      int64          `json:"ts"`
	Kind    string         `json:"kind"`
	Payload map[string]any `json:"payload"`
}

// writeLog 把给定行写成合法的 <id>.gaea-log.jsonl，返回路径。
func writeLog(t *testing.T, name string, lines []line) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	var buf []byte
	for i, l := range lines {
		l.Seq = int64(i + 1)
		l.Ts = 1_760_000_000 + int64(i)
		b, err := json.Marshal(l)
		if err != nil {
			t.Fatalf("marshal line %d: %v", i, err)
		}
		buf = append(buf, b...)
		buf = append(buf, '\n')
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	return path
}

// TestScanLogPairsAttention 锁定核心口径：一次打断一张卡、同 id 成对、
// 旧日志缺 source 记 legacy、未解决卡被列出来。
func TestScanLogPairsAttention(t *testing.T) {
	path := writeLog(t, "s1.gaea-log.jsonl", []line{
		{Kind: kindTurnStarted, Payload: map[string]any{"trigger": "user"}},
		{Kind: kindApprovalRequest, Payload: map[string]any{"id": "a1", "tool": "bash", "subject": "rm"}},
		{Kind: kindApprovalResolve, Payload: map[string]any{
			"id": "a1", "tool": "bash", "subject": "rm", "source": "gate",
			"decision": "allow_once", "outcome": "allowed", "latency_ms": 1500,
		}},
		{Kind: kindAskRequest, Payload: map[string]any{
			"id": "q1",
			"questions": []any{
				map[string]any{"id": "q1", "prompt": "继续?"},
				map[string]any{"id": "q2", "prompt": "哪个方案?"},
			},
		}},
		{Kind: kindAskResolve, Payload: map[string]any{"id": "q1", "outcome": "answered", "answered": 1, "latency_ms": 800}},
		{Kind: kindTurnDone, Payload: map[string]any{"err": ""}},
		{Kind: kindTurnStarted, Payload: map[string]any{"trigger": "wake"}},
		{Kind: kindApprovalRequest, Payload: map[string]any{
			"id": "a2", "tool": "cost_save", "subject": "写入成本库", "source": "hard_ask",
		}},
		{Kind: kindTurnDone, Payload: map[string]any{"err": ""}},
	})

	st, err := scanLog(path)
	if err != nil {
		t.Fatalf("scanLog: %v", err)
	}
	if st.Session != "s1" {
		t.Errorf("Session = %q, want s1", st.Session)
	}
	if st.Turns != 2 {
		t.Errorf("Turns = %d, want 2", st.Turns)
	}
	// 发起者拆分：人轮是打断率的分母，唤醒轮不该混进去。
	if st.TurnsUser != 1 || st.TurnsWake != 1 || st.TurnsSubagent != 0 {
		t.Errorf("turns user/wake/subagent = %d/%d/%d, want 1/1/0",
			st.TurnsUser, st.TurnsWake, st.TurnsSubagent)
	}
	if st.Approvals != 2 || st.Asks != 1 {
		t.Errorf("approvals/asks = %d/%d, want 2/1", st.Approvals, st.Asks)
	}
	if st.Cards != 3 {
		t.Errorf("Cards = %d, want 3（ask 按卡不按问题数）", st.Cards)
	}
	if st.Questions != 2 {
		t.Errorf("Questions = %d, want 2（仅诊断计数）", st.Questions)
	}
	if st.Unresolved != 1 || len(st.UnresolvedIDs) != 1 || st.UnresolvedIDs[0] != "a2" {
		t.Errorf("Unresolved = %v (ids %v), want [a2]", st.Unresolved, st.UnresolvedIDs)
	}
	if st.Outcomes["allowed"] != 1 || st.Outcomes["answered"] != 1 {
		t.Errorf("Outcomes = %v, want allowed:1 answered:1", st.Outcomes)
	}
	// a1 请求行缺 source（legacy），但结果侧带 gate → 精确归位为 gate；
	// a2 为 hard_ask；q1 提问不参与来源统计。
	if st.Sources["gate"] != 1 || st.Sources["hard_ask"] != 1 || len(st.Sources) != 2 {
		t.Errorf("Sources = %v, want gate:1 hard_ask:1", st.Sources)
	}
	if st.LatencyN != 2 || st.LatencyMS != 1150 {
		t.Errorf("latency = %d/%d, want 1150 over 2 samples", st.LatencyMS, st.LatencyN)
	}
}

// TestScanLogIgnoresDuplicateResolve 锁定重复结果事件不重复计数。
func TestScanLogIgnoresDuplicateResolve(t *testing.T) {
	path := writeLog(t, "s2.gaea-log.jsonl", []line{
		{Kind: kindApprovalRequest, Payload: map[string]any{"id": "a1", "tool": "bash", "subject": "x"}},
		{Kind: kindApprovalResolve, Payload: map[string]any{"id": "a1", "outcome": "denied", "latency_ms": 100}},
		{Kind: kindApprovalResolve, Payload: map[string]any{"id": "a1", "outcome": "denied", "latency_ms": 900}},
	})
	st, err := scanLog(path)
	if err != nil {
		t.Fatalf("scanLog: %v", err)
	}
	if st.Outcomes["denied"] != 1 || st.LatencyN != 1 || st.LatencyMS != 100 {
		t.Errorf("outcomes=%v latency=%d/%d, want denied:1 and 100/1", st.Outcomes, st.LatencyMS, st.LatencyN)
	}
}

// TestCollectLogsExpandsDirs 锁定目录展开与其去重（work/play 分区同扫时
// 平铺目录不会重复计入）。
func TestCollectLogsExpandsDirs(t *testing.T) {
	dir := t.TempDir()
	writeLogAt(t, filepath.Join(dir, "a.gaea-log.jsonl"))
	writeLogAt(t, filepath.Join(dir, "b.gaea-log.jsonl"))
	if err := os.WriteFile(filepath.Join(dir, "ignored.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write ignored: %v", err)
	}
	got, err := collectLogs([]string{dir, filepath.Join(dir, "a.gaea-log.jsonl"), filepath.Join(dir, "missing")})
	if err != nil {
		t.Fatalf("collectLogs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("collectLogs = %v, want 2 个日志（目录内两个 + 显式文件去重）", got)
	}
}

// writeLogAt 写一份最小合法日志到指定路径。
func writeLogAt(t *testing.T, path string) {
	t.Helper()
	l := line{Seq: 1, Ts: 1_760_000_000, Kind: kindTurnStarted, Payload: map[string]any{}}
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
