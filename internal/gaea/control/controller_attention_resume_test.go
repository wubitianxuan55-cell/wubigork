package control

// 悬空卡闭合（2026-11-02 第二刀）：进程在卡还开着时被杀，恢复会话必须给
// 每张悬空卡补一个 outcome=abandoned 的结局，否则统计器只能把它读成
// 「用户一直没回答」，「放弃」与「未解决」分不开。

import (
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

func TestResumeFromDiskClosesOrphanAttention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	logPath := session.LogPathFor(path)

	w, err := session.OpenLog(logPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(session.KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	// 模拟崩溃现场：请求行写了，进程还没来得及写解决事件就死了。
	if _, err := w.Append(session.KindApprovalRequest, map[string]any{
		"id": "a1", "tool": "cost_save", "subject": "写入成本库：钢材",
		"source": event.ApprovalSourceHardAsk, "reason": "跨库写入", "request": false,
	}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	exec := agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard)
	c := New(Options{Runner: &fakeStateRunner{}, Executor: exec, Sink: event.Discard, SessionDir: dir, LogFormat: "event"})
	if _, err := c.ResumeFromDisk(path); err != nil {
		t.Fatalf("ResumeFromDisk: %v", err)
	}

	entries, err := session.ReadLogRepaired(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if orphans := session.UnresolvedAttention(entries); len(orphans) != 0 {
		t.Fatalf("恢复后仍有悬空卡: %+v", orphans)
	}

	tr := trajectory.FoldTrajectory(entries)
	var found bool
	records := append([]trajectory.Record{}, tr.BetweenTurns...)
	for _, turn := range tr.Turns {
		records = append(records, turn.Records...)
	}
	for _, r := range records {
		if r.Approval == nil || r.Approval.ID != "a1" {
			continue
		}
		found = true
		if r.Approval.Outcome != event.OutcomeAbandoned {
			t.Fatalf("轨迹 outcome = %q, want %q", r.Approval.Outcome, event.OutcomeAbandoned)
		}
		if r.Approval.Source != event.ApprovalSourceHardAsk {
			t.Fatalf("轨迹 source = %q, want %q", r.Approval.Source, event.ApprovalSourceHardAsk)
		}
		if r.Approval.LatencyMS != 0 {
			t.Fatalf("abandoned 的 latency = %d, want 0（不知道挂了多久）", r.Approval.LatencyMS)
		}
	}
	if !found {
		t.Fatal("轨迹里没有 a1 的审批记录（请求未被折叠）")
	}
}
