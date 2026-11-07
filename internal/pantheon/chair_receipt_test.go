package pantheon

import (
	"strings"
	"testing"
)

func TestChairAssignmentReceiptsDeriveFromArchive(t *testing.T) {
	session := &Session{
		ID:           "pan_receipt",
		Status:       StatusRunning,
		CurrentRound: 2,
		Participants: []Participant{
			{AgentID: "alpha", Name: "ALPHA", Status: SeatFinished},
			{AgentID: "beta", Name: "BETA", Status: SeatSkipped},
			{AgentID: "gamma", Name: "GAMMA", Status: SeatFailed},
			{AgentID: "delta", Name: "DELTA", Status: SeatUnavailable},
		},
		Contributions: []Contribution{
			{
				ID: "mod-1", Phase: PhaseModeration, Round: 1, AgentID: "alpha", AgentName: "ALPHA（主持）",
				CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"}, CreatedAt: 100,
				ChairAssignments: []ChairAssignment{
					{AgentID: "alpha", Instruction: "复核缓存口径"},
					{AgentID: "beta", Instruction: "补一条反例"},
					{AgentID: "gamma", Instruction: "核验版本号"},
					{AgentID: "delta", Instruction: "补一个外部来源"},
				},
			},
			{ID: "alpha-r2", Phase: PhaseDiscussion, Round: 2, AgentID: "alpha", AgentName: "ALPHA", Content: "已复核"},
		},
		Events: []Event{
			{Seq: 10, Kind: EventAgentOutput, AgentID: "alpha", CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"}, At: 100},
			{Seq: 11, Kind: EventAgentSkipped, AgentID: "beta", At: 110, Text: "跳过"},
			{Seq: 12, Kind: EventAgentError, AgentID: "gamma", At: 120, Text: "版本核验超时"},
		},
	}
	receipts := chairAssignmentReceipts(session)
	if len(receipts) != 4 {
		t.Fatalf("receipts = %d, want 4: %+v", len(receipts), receipts)
	}
	byAgent := map[string]ChairAssignmentReceipt{}
	for _, receipt := range receipts {
		byAgent[receipt.AgentID] = receipt
	}
	if got := byAgent["alpha"]; got.Status != ReceiptSpoke || got.ContributionID != "alpha-r2" {
		t.Fatalf("alpha receipt = %+v, want spoke/alpha-r2", got)
	}
	if got := byAgent["beta"]; got.Status != ReceiptSkipped {
		t.Fatalf("beta receipt = %+v, want skipped", got)
	}
	if got := byAgent["gamma"]; got.Status != ReceiptFailed || !strings.Contains(got.Detail, "超时") {
		t.Fatalf("gamma receipt = %+v, want failed with detail", got)
	}
	if got := byAgent["delta"]; got.Status != ReceiptUnavailable {
		t.Fatalf("delta receipt = %+v, want unavailable", got)
	}
}

func TestChairAssignmentReceiptsPendingWhenNoNextRound(t *testing.T) {
	session := &Session{
		ID: "pan_pending", Status: StatusCompleted, CurrentRound: 1,
		Participants: []Participant{{AgentID: "alpha", Name: "ALPHA", Status: SeatFinished}},
		Contributions: []Contribution{{
			ID: "mod-1", Phase: PhaseModeration, Round: 1, AgentID: "alpha", AgentName: "ALPHA（主持）",
			CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"}, CreatedAt: 100,
			ChairAssignments: []ChairAssignment{{AgentID: "beta", Instruction: "补核验"}},
		}},
		Events: []Event{{Seq: 10, Kind: EventAgentOutput, AgentID: "alpha", CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"}, At: 100}},
	}
	receipts := chairAssignmentReceipts(session)
	if len(receipts) != 1 || receipts[0].Status != ReceiptPending {
		t.Fatalf("receipts = %+v, want one pending", receipts)
	}
	if !strings.Contains(receipts[0].Detail, "下一轮前结束") {
		t.Fatalf("pending detail = %q", receipts[0].Detail)
	}
}

func TestRenderFinalDocumentCarriesEvidenceAndReceipts(t *testing.T) {
	session := &Session{
		Title: "交付物", Status: StatusCompleted, CurrentRound: 2,
		Request:      StartRequest{Topic: "证据与回执", Preset: PresetRoundtable, Protocol: ProtocolIndependent},
		Participants: []Participant{{AgentID: "alpha", Name: "ALPHA", Status: SeatFinished}},
		Contributions: []Contribution{
			{
				ID: "mod-1", Phase: PhaseModeration, Round: 1, AgentID: "alpha", AgentName: "ALPHA（主持）",
				Content: "小结", CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"},
				ChairAssignments: []ChairAssignment{{AgentID: "beta", Instruction: "补一条反例"}},
			},
			{
				ID: "alpha-r2", Phase: PhaseDiscussion, Round: 2, AgentID: "alpha", AgentName: "ALPHA",
				Content: "已复核。", ToolEvidence: &ToolEvidence{Calls: 2, OK: 1, Failed: 1},
			},
		},
		Events: []Event{
			{Seq: 10, Kind: EventAgentOutput, AgentID: "alpha", CallRef: CallRef{CallID: "call-1", AttemptID: "attempt-1"}},
			{Seq: 11, Kind: EventAgentSkipped, AgentID: "beta"},
		},
	}
	doc := RenderFinalDocument(session)
	for _, want := range []string{"宿主观察：2 次工具调用", "主席派工与终局", "补一条反例", "已跳过"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document missing %q:\n%s", want, doc)
		}
	}
}

func TestFormatChairReceiptsHonestWording(t *testing.T) {
	text := formatChairReceipts([]ChairAssignmentReceipt{
		{Round: 1, AgentID: "alpha", AgentName: "ALPHA", Status: ReceiptSpoke, ContributionID: "c-1"},
		{Round: 1, AgentID: "beta", AgentName: "BETA", Status: ReceiptFailed, Detail: "超时"},
	})
	for _, want := range []string{"只证明该席是否在下一轮发言", "已发言（未证明完成）", "贡献 ID c-1", "失败"} {
		if !strings.Contains(text, want) {
			t.Fatalf("receipt text missing %q:\n%s", want, text)
		}
	}
}
