package pantheon

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestEvaluateSessionDerivesComparisonMetricsWithoutMutation(t *testing.T) {
	session := &Session{
		ID:           "pan_eval",
		Title:        "协议对照样例",
		Status:       StatusCompleted,
		CurrentRound: 2,
		CreatedAt:    1_000,
		UpdatedAt:    20_000,
		Request: StartRequest{
			Topic:     "是否采用定向质询",
			AgentIDs:  []string{"alpha", "beta"},
			Protocol:  ProtocolIndependent,
			Preset:    PresetTargetedChallenge,
			Rounds:    2,
			Workspace: "C:/work",
		},
		Participants: []Participant{
			{AgentID: "alpha", Name: "Alpha", Version: "1.2.3"},
			{AgentID: "beta", Name: "Beta", Version: "4.5.6"},
		},
		Contributions: []Contribution{
			{ID: "c1", AgentID: "alpha", Phase: PhaseOpening, EvidenceHandle: "README.md:1", RunMode: "read-only"},
			{ID: "c2", AgentID: "beta", Phase: PhaseOpening, EvidenceHandle: "未核验：缺样本", RunMode: "plan"},
			{ID: "c3", AgentID: "alpha", Phase: PhaseDiscussion, Content: "该判断未核验", RunMode: "read-only"},
			{ID: "c4", AgentID: "user", Phase: PhaseUserNote, Content: "补充一个限制"},
			{ID: "c5", AgentID: "alpha", Phase: PhaseSynthesis, RunMode: "read-only"},
		},
		Events: []Event{
			{Kind: EventAgentStart, At: 1_100},
			{Kind: EventAgentStart, At: 1_200},
			{Kind: EventAgentRetry, At: 2_000},
			{Kind: EventAgentError, At: 2_100},
			{Kind: EventAgentStart, At: 2_200},
			{Kind: EventToolStart, At: 2_300},
			{Kind: EventAgentOutput, At: 4_000, DurationMs: 2_000},
			{Kind: EventAgentOutput, At: 7_500, DurationMs: 3_500},
			{Kind: EventComplete, At: 11_000},
		},
		Issues: []Issue{
			{Status: "resolved"},
			{Status: "dissent"},
			{Status: "open"},
		},
		Attestations: []Attestation{
			{Position: AttestationSupport},
			{Position: AttestationUnknown},
		},
		Verdict: &Verdict{Partial: true, RevisedFromSHA256: "old"},
		SignOff: &SignOff{Decision: "accepted"},
	}
	before, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}

	run := EvaluateSession(session)
	if run.CaseKey == "" || run.Variant != EvaluationVariantTargetedChallenge {
		t.Fatalf("wrong case projection: %+v", run)
	}
	if run.CallCount != 3 || run.RetryCount != 1 || run.ErrorCount != 1 || run.ToolCallCount != 1 {
		t.Fatalf("event counters wrong: %+v", run)
	}
	if run.WallDurationMs != 10_000 || run.AgentDurationMs != 5_500 {
		t.Fatalf("duration metrics wrong: wall=%d agent=%d", run.WallDurationMs, run.AgentDurationMs)
	}
	if run.DiscussionCount != 3 || run.EvidenceHandledCount != 2 ||
		run.UnverifiedMentions != 1 || run.UserNoteCount != 1 {
		t.Fatalf("evidence counters wrong: %+v", run)
	}
	if run.IssueCount != 3 || run.ResolvedIssueCount != 1 || run.DissentIssueCount != 1 {
		t.Fatalf("issue counters wrong: %+v", run)
	}
	if run.AttestationCount != 2 || run.UnknownAttestations != 1 ||
		!run.VerdictPartial || !run.VerdictRevised || run.SignOffDecision != "accepted" {
		t.Fatalf("verdict attestation projection wrong: %+v", run)
	}
	if run.RuntimeVersions["alpha"] != "1.2.3" || len(run.RunModes) != 2 {
		t.Fatalf("runtime evidence wrong: %+v", run)
	}
	if run.ParticipantCount != 2 || run.VersionedParticipants != 2 {
		t.Fatalf("participant version coverage wrong: %+v", run)
	}
	exported := ExportMarkdown(session)
	if !strings.Contains(exported, "## 运行度量（可复算）") ||
		!strings.Contains(exported, run.CaseKey) ||
		!strings.Contains(exported, "Token/费用：未采集") {
		t.Fatal("audit export omits recomputable evaluation metrics")
	}

	after, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("EvaluateSession mutated its input")
	}
}

func TestEvaluationCaseKeyKeepsProtocolVariablesOutAndPinsMaterials(t *testing.T) {
	base := &Session{
		Title: "A",
		Request: StartRequest{
			Topic: "同一议题", Context: "同一背景", SuccessCriteria: "同一验收",
			Workspace: "C:/work", AgentIDs: []string{"a", "b"},
			Protocol: ProtocolIndependent, Preset: PresetRoundtable, Rounds: 2,
		},
		InputManifest: []InputDigest{{Kind: "evidence", Name: "memo", SHA256: "aaa", Size: 12}},
	}
	changed := *base
	changed.Title = "B"
	changed.Request = base.Request
	changed.Request.Preset = PresetRedTeam
	changed.Request.Rounds = 3
	if EvaluationCaseKey(base) != EvaluationCaseKey(&changed) {
		t.Fatal("protocol variables must not split a comparison case")
	}

	differentCriteria := *base
	differentCriteria.Request = base.Request
	differentCriteria.Request.SuccessCriteria = "不同验收"
	if EvaluationCaseKey(base) == EvaluationCaseKey(&differentCriteria) {
		t.Fatal("different success criteria must split a comparison case")
	}

	differentMaterial := *base
	differentMaterial.InputManifest = []InputDigest{{Kind: "evidence", Name: "memo", SHA256: "bbb", Size: 12}}
	if EvaluationCaseKey(base) == EvaluationCaseKey(&differentMaterial) {
		t.Fatal("different material digest must split a comparison case")
	}

	derivedVisualSummary := *base
	derivedVisualSummary.InputManifest = append(
		append([]InputDigest(nil), base.InputManifest...),
		InputDigest{Kind: "evidence", Name: "共享识图 · chart.png", SHA256: "changes-every-run"},
	)
	if EvaluationCaseKey(base) != EvaluationCaseKey(&derivedVisualSummary) {
		t.Fatal("derived image summary must not split a comparison case")
	}
}

func TestEvaluationVariantClassification(t *testing.T) {
	tests := []struct {
		name     string
		request  StartRequest
		expected string
	}{
		{
			name: "single",
			request: StartRequest{
				AgentIDs: []string{"alpha"}, Protocol: ProtocolIndependent,
				Preset: PresetRoundtable, Rounds: 1,
			},
			expected: EvaluationVariantSingle,
		},
		{
			name: "independent multi",
			request: StartRequest{
				AgentIDs: []string{"alpha", "beta"}, Protocol: ProtocolIndependent,
				Preset: PresetRoundtable, Rounds: 1,
			},
			expected: EvaluationVariantIndependent,
		},
		{
			name: "roundtable",
			request: StartRequest{
				AgentIDs: []string{"alpha", "beta"}, Protocol: ProtocolIndependent,
				Preset: PresetRoundtable, Rounds: 2,
			},
			expected: EvaluationVariantRoundtable,
		},
		{
			name: "targeted",
			request: StartRequest{
				AgentIDs: []string{"alpha", "beta"}, Protocol: ProtocolIndependent,
				Preset: PresetTargetedChallenge, Rounds: 2,
			},
			expected: EvaluationVariantTargetedChallenge,
		},
		{
			name: "red team",
			request: StartRequest{
				AgentIDs: []string{"alpha", "beta"}, Protocol: ProtocolIndependent,
				Preset: PresetRedTeam, Rounds: 2,
			},
			expected: EvaluationVariantRedTeam,
		},
		{
			name: "legacy council",
			request: StartRequest{
				AgentIDs: []string{"alpha", "beta"}, Protocol: ProtocolCouncil,
				Preset: PresetRoundtable, Rounds: 2,
			},
			expected: EvaluationVariantCouncilRelay,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &Session{Request: test.request}
			if got := EvaluationVariantOf(session); got != test.expected {
				t.Fatalf("variant = %q, want %q", got, test.expected)
			}
		})
	}
}

func TestBuildEvaluationReportGroupsCasesAndOpensGate(t *testing.T) {
	var runs []EvaluationRun
	for _, caseKey := range []string{"case_a", "case_b"} {
		for index, variant := range evaluationCoreVariants {
			runs = append(runs, EvaluationRun{
				SessionID:        caseKey + "_" + variant,
				CaseKey:          caseKey,
				Title:            caseKey,
				Variant:          variant,
				Status:           StatusCompleted,
				ActualRounds:     index + 1,
				ExpectedRounds:   2,
				CallCount:        4 + index,
				WallDurationMs:   int64(10_000 + index*1_000),
				RuntimeVersions:  map[string]string{"alpha": "1.0", "beta": "2.0"},
				ParticipantCount: 2, VersionedParticipants: 2,
				SignOffDecision: "accepted",
			})
		}
	}
	report := BuildEvaluationReport(runs, 2)
	if !report.GateReady || report.QualifiedCases != 2 || report.CaseCount != 2 {
		t.Fatalf("report should be ready: %+v", report)
	}
	if len(report.Variants) != 4 {
		t.Fatalf("variant summaries = %d, want 4", len(report.Variants))
	}
	for _, summary := range report.Variants {
		if summary.CompletedSessions != 2 || summary.AcceptedSessions != 2 {
			t.Fatalf("summary not grouped: %+v", summary)
		}
	}
}

func TestBuildEvaluationReportFailsClosedOnMissingVariantsAndVersions(t *testing.T) {
	report := BuildEvaluationReport([]EvaluationRun{{
		SessionID: "pan_only_roundtable",
		CaseKey:   "case_partial",
		Title:     "不完整对照",
		Variant:   EvaluationVariantRoundtable,
		Status:    StatusCompleted,
	}}, 20)
	if report.GateReady || report.QualifiedCases != 0 {
		t.Fatalf("incomplete evidence must not open the gate: %+v", report)
	}
	joined := strings.Join(report.Warnings, "\n")
	if !strings.Contains(joined, "0/20") ||
		!strings.Contains(joined, "存在缺失的 Harness/模型版本快照") ||
		!strings.Contains(joined, "固定投票") {
		t.Fatalf("warnings miss missing evidence:\n%s", joined)
	}
	markdown := FormatEvaluationMarkdown(report, true)
	if !strings.Contains(markdown, "门禁：未就绪") ||
		!strings.Contains(markdown, "Token 与费用未入档") {
		t.Fatalf("markdown does not fail closed:\n%s", markdown)
	}
}

func TestStartSnapshotsParticipantRuntimeVersion(t *testing.T) {
	alpha := fakeAdapter("alpha")
	alpha.probe = func(ctx context.Context) AgentStatus {
		status := AgentStatus{
			ID: "alpha", Name: "Alpha", Kind: "fake", Available: true,
			Executable: "alpha", Version: "alpha-1.2.3",
		}
		return status
	}
	beta := fakeAdapter("beta")
	mgr := newTestManager(t, defaultFakeRunner(), alpha, beta)

	session, err := mgr.Start(StartRequest{
		Topic: "版本快照", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Participants[0].Version != "alpha-1.2.3" {
		t.Fatalf("runtime version not snapshotted: %+v", session.Participants)
	}
	_ = mgr.Cancel(session.ID)
	mgr.Close()
}

func TestManagerEvaluateProtocolsReadsStoredLedger(t *testing.T) {
	mgr := newTestManager(t, defaultFakeRunner())
	baseRequest := StartRequest{
		Topic: "同一议题", Context: "同一背景", SuccessCriteria: "同一验收",
		Workspace: "C:/work", AgentIDs: []string{"alpha"}, Protocol: ProtocolIndependent,
		Preset: PresetRoundtable, Rounds: 1,
	}
	roundtableRequest := baseRequest
	roundtableRequest.AgentIDs = []string{"alpha", "beta"}
	roundtableRequest.Rounds = 2
	for _, session := range []*Session{
		{
			ID: "pan_eval_single", Title: "单席", Status: StatusCompleted,
			CurrentRound: 1, CreatedAt: 1_000, UpdatedAt: 2_000, Request: baseRequest,
			Participants: []Participant{{AgentID: "alpha", Name: "Alpha", Version: "1.0"}},
		},
		{
			ID: "pan_eval_roundtable", Title: "圆桌", Status: StatusCompleted,
			CurrentRound: 2, CreatedAt: 3_000, UpdatedAt: 5_000, Request: roundtableRequest,
			Participants: []Participant{
				{AgentID: "alpha", Name: "Alpha", Version: "1.0"},
				{AgentID: "beta", Name: "Beta", Version: "2.0"},
			},
		},
	} {
		if err := mgr.store.save(session); err != nil {
			t.Fatal(err)
		}
	}

	report, err := mgr.EvaluateProtocols(1)
	if err != nil {
		t.Fatal(err)
	}
	if report.CaseCount != 1 || len(report.Runs) != 2 || report.GateReady {
		t.Fatalf("stored ledger projection wrong: %+v", report)
	}
	if len(report.Variants) != 2 {
		t.Fatalf("variant summaries = %d, want 2", len(report.Variants))
	}
}
