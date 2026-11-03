package pantheon

import (
	"strings"
	"testing"
)

func TestParseVerdictCarriesReadOnlyActions(t *testing.T) {
	raw := `{
  "summary":"保留只读边界",
  "recommendation":"先做小范围验证",
  "proposedActions":[{
    "action":"完成 8 组同任务成本对照",
    "rationale":"验证辩论增益",
    "dependsOn":["冻结模型版本"],
    "acceptanceEvidence":"8 组原始记录与汇总表",
    "suggestedOwner":"用户",
    "risk":"样本不足会误导结论"
  }]
}`
	verdict := parseVerdict(raw, "Chair", 0)
	if len(verdict.ProposedActions) != 1 {
		t.Fatalf("actions = %+v", verdict.ProposedActions)
	}
	action := verdict.ProposedActions[0]
	if action.ID != "action-1" ||
		action.Action == "" ||
		action.AcceptanceEvidence == "" ||
		len(action.DependsOn) != 1 {
		t.Fatalf("action fields = %+v", action)
	}
}

func TestProposedActionsExportAsReadOnly(t *testing.T) {
	session := &Session{
		Title:   "行动提案",
		Status:  StatusCompleted,
		Phase:   PhaseCompleted,
		Request: StartRequest{Topic: "行动提案", Preset: PresetRoundtable, Protocol: ProtocolIndependent},
		Verdict: &Verdict{
			Summary:        "结论",
			Recommendation: "先验证",
			ProposedActions: []ProposedAction{{
				ID:                 "action-1",
				Action:             "执行只读对照",
				AcceptanceEvidence: "保留原始结果",
			}},
		},
	}
	out := RenderFinalDocument(session)
	if !strings.Contains(out, "### 行动提案（只读）") ||
		!strings.Contains(out, "行动提案未执行") {
		t.Fatalf("action plan export missing:\n%s", out)
	}
}
