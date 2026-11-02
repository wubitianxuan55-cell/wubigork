package session

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// attentionPayload 构造一条日志 payload（测试内联 map，避免依赖 log.go 私有类型）。
func attentionPayload(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}

// TestUnresolvedAttention 锁定配对口径：重复请求只算一张、结果事件只关闭
// 对应 id、孤立结果（没有请求行）不产生悬空卡。
func TestUnresolvedAttention(t *testing.T) {
	entries := []LogEntry{
		{Kind: KindApprovalRequest, Payload: attentionPayload(t, map[string]any{
			"id": "a1", "tool": "bash", "subject": "rm -rf build", "source": event.ApprovalSourceGate,
		})},
		// 孤立结果：没有对应请求行，不应凭空造卡，也不应影响别的 id
		{Kind: KindApprovalResolved, Payload: attentionPayload(t, map[string]any{"id": "ghost", "outcome": event.OutcomeAllowed})},
		{Kind: KindAskRequest, Payload: attentionPayload(t, map[string]any{
			"id": "q1", "questions": []any{map[string]any{"id": "q1", "prompt": "继续?"}},
		})},
		// 同一 id 的重复请求（日志修复/重放）：只算第一张
		{Kind: KindApprovalRequest, Payload: attentionPayload(t, map[string]any{
			"id": "a1", "tool": "bash", "subject": "rm -rf build", "source": event.ApprovalSourceGate,
		})},
		// 提问 q1 已被回答：不再是悬空卡
		{Kind: KindAskResolved, Payload: attentionPayload(t, map[string]any{"id": "q1", "outcome": event.OutcomeAnswered})},
	}

	orphans := UnresolvedAttention(entries)
	if len(orphans) != 1 {
		t.Fatalf("orphans = %d (%+v), want 1", len(orphans), orphans)
	}
	got := orphans[0]
	if got.Kind != KindApprovalRequest || got.ID != "a1" || got.Tool != "bash" || got.Source != event.ApprovalSourceGate {
		t.Fatalf("orphan = %+v", got)
	}
}

// TestCloseOrphanAttentionWritesAbandoned 锁定进程先死时的收尾：悬空卡补写
// outcome=abandoned 的结果事件（latency 0 = 不知道挂了多久），且幂等。
func TestCloseOrphanAttentionWritesAbandoned(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")

	w, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindUserMessage, map[string]string{"content": "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindApprovalRequest, map[string]any{
		"id": "a1", "tool": "cost_save", "subject": "写入成本库：钢材",
		"source": event.ApprovalSourceHardAsk, "reason": "跨库写入", "request": false,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindAskRequest, map[string]any{
		"id": "q1", "questions": []any{map[string]any{"id": "q1", "prompt": "选哪个?"}},
	}); err != nil {
		t.Fatal(err)
	}
	w.Close()

	n, err := CloseOrphanAttention(logPath, "")
	if err != nil {
		t.Fatalf("CloseOrphanAttention: %v", err)
	}
	if n != 2 {
		t.Fatalf("closed = %d, want 2", n)
	}

	entries, err := ReadLogRepaired(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if orphans := UnresolvedAttention(entries); len(orphans) != 0 {
		t.Fatalf("补写后仍有悬空卡: %+v", orphans)
	}

	var approval, ask bool
	for _, e := range entries {
		switch e.Kind {
		case KindApprovalResolved:
			var p struct {
				ID        string `json:"id"`
				Source    string `json:"source"`
				Decision  string `json:"decision"`
				Outcome   string `json:"outcome"`
				LatencyMS int64  `json:"latency_ms"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				t.Fatalf("unmarshal approval_resolved: %v", err)
			}
			if p.ID != "a1" {
				t.Fatalf("approval_resolved id = %q", p.ID)
			}
			approval = true
			if p.Outcome != event.OutcomeAbandoned || p.Decision != "" || p.LatencyMS != 0 {
				t.Fatalf("approval_resolved = %+v, want abandoned/空决策/0ms", p)
			}
			// 请求侧的判别字段必须原样带回来，不能被收口写成 gate
			if p.Source != event.ApprovalSourceHardAsk {
				t.Fatalf("source = %q, want %q", p.Source, event.ApprovalSourceHardAsk)
			}
		case KindAskResolved:
			var p struct {
				ID       string `json:"id"`
				Outcome  string `json:"outcome"`
				Answered int    `json:"answered"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				t.Fatalf("unmarshal ask_resolved: %v", err)
			}
			if p.ID != "q1" || p.Outcome != event.OutcomeAbandoned || p.Answered != 0 {
				t.Fatalf("ask_resolved = %+v", p)
			}
			ask = true
		}
	}
	if !approval || !ask {
		t.Fatalf("补写不全：approval=%v ask=%v", approval, ask)
	}

	// 幂等：第二次收口不再补写
	again, err := CloseOrphanAttention(logPath, "")
	if err != nil {
		t.Fatalf("second CloseOrphanAttention: %v", err)
	}
	if again != 0 {
		t.Fatalf("second close = %d, want 0（幂等）", again)
	}
}
