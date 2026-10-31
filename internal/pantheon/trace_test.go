package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 2026-10-31 圆桌裁决第三刀验收：最小可追溯——能力逐席入档、材料快照哈希、
// 签注绑定小结版本并重渲染交付物、续议保留父会话。

func TestInputManifestPinsEvidenceAndAttachments(t *testing.T) {
	dir := t.TempDir()
	attachment := filepath.Join(dir, "diagram.png")
	if err := os.WriteFile(attachment, []byte("PNG-bytes-稳定内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	evidence := "已有多个竞品完成只读议事验证。"
	wantEvidence := sha256.Sum256([]byte(evidence))
	wantFile := sha256.Sum256([]byte("PNG-bytes-稳定内容"))

	got := buildInputManifest(StartRequest{
		Evidence:    []EvidenceItem{{Name: "调研", Content: evidence}},
		Attachments: []Attachment{{Name: "diagram.png", Path: attachment}},
	})
	if len(got) != 2 {
		t.Fatalf("manifest = %d entries, want 2", len(got))
	}
	if got[0].Kind != "evidence" || got[0].SHA256 != hex.EncodeToString(wantEvidence[:]) || got[0].Size != int64(len(evidence)) {
		t.Fatalf("evidence digest wrong: %+v", got[0])
	}
	if got[1].Kind != "attachment" || got[1].Path != attachment || got[1].SHA256 != hex.EncodeToString(wantFile[:]) {
		t.Fatalf("attachment digest wrong: %+v", got[1])
	}

	missing := buildInputManifest(StartRequest{
		Attachments: []Attachment{{Name: "gone.png", Path: filepath.Join(dir, "gone.png")}},
	})
	if len(missing) != 1 || missing[0].Error == "" {
		t.Fatalf("missing attachment must record an honest error: %+v", missing)
	}
}

func TestSignOffBindsVerdictVersionAndRerendersResult(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "签注版本绑定", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	signed, err := m.SignOff(session.ID, "accepted", "同意先试点")
	if err != nil {
		t.Fatal(err)
	}
	if signed.SignOff == nil {
		t.Fatal("signoff missing")
	}
	want := verdictFingerprint(done.Verdict)
	if want == "" {
		t.Fatal("verdict fingerprint must not be empty for a completed council")
	}
	if signed.SignOff.VerdictSHA256 != want {
		t.Fatalf("signoff verdict hash = %q, want %q (binds the exact version the user confirmed)",
			signed.SignOff.VerdictSHA256, want)
	}
	if signed.ResultPath == "" {
		t.Fatal("signoff must keep a result path")
	}
	data, err := os.ReadFile(signed.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, wantLine := range []string{
		"## 用户确认",
		"决定：确认小结",
		"同意先试点",
		"绑定小结版本：`sha256:" + want + "`",
	} {
		if !strings.Contains(doc, wantLine) {
			t.Fatalf("re-rendered deliverable misses %q:\n%s", wantLine, doc)
		}
	}
}

func TestRunModeRecordedPerCallAndExported(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "能力入档", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, contribution := range done.Contributions {
		if contribution.RunMode != "内存桩（只读）" {
			t.Fatalf("contribution %s runMode = %q, want per-call declared mechanism",
				contribution.ID, contribution.RunMode)
		}
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(export, "只读经 内存桩（只读）") {
		t.Fatalf("export participants must carry per-seat mechanism instead of a blanket read-only:\n%s", export)
	}
}

func TestParentSessionTracedInRecordAndExport(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	// 父会话真实跑完一场，续议继承其历史论点。
	parent, err := m.Start(StartRequest{
		Topic: "父场议题", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, parent.ID, StatusCompleted)

	// 父会话不存在 → fail-loud。
	if _, err := m.Start(StartRequest{
		Topic: "孤儿续议", AgentIDs: []string{"alpha"}, Rounds: 1,
		ParentSessionID: "pan_missing_parent",
	}); err == nil || !strings.Contains(err.Error(), "父会话不可用") {
		t.Fatalf("missing parent must fail loud, got err=%v", err)
	}

	session, err := m.Start(StartRequest{
		Topic: "续议谱系", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		ParentSessionID: parent.ID, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Request.ParentSessionID != parent.ID {
		t.Fatalf("parent session id = %q, want %q", done.Request.ParentSessionID, parent.ID)
	}
	// 历史论点立场卡进入续议简报：后位 prompt 含父会话观点。
	found := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "「BETA」") &&
			strings.Contains(call.Stdin, "历史论点 · 上一场") &&
			strings.Contains(call.Stdin, "先做只读验证") {
			found = true
		}
	}
	if !found {
		t.Fatal("parent stance card missing from continuation brief")
	}
	// 派生立场卡不入材料快照（快照只钉用户设题材料）。
	if len(done.InputManifest) != 0 {
		t.Fatalf("derived stance cards must not enter input manifest: %+v", done.InputManifest)
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"父会话：`" + parent.ID + "`", "不继承全文"} {
		if !strings.Contains(export, want) {
			t.Fatalf("export misses %q:\n%s", want, export)
		}
	}
}

// 2026-10-31 实弹反馈：中断保存与恢复——已有贡献保留，从中断处下一轮继续。
func TestResumeKeepsHistoryAndContinues(t *testing.T) {
	runner := defaultFakeRunner()
	// 第一幕：双席全失败 → 场记为 failed 且零贡献，Resume 指引重开整场。
	runner.outputs["alpha"] = func(commandSpec) (string, error) { return "", errors.New("模拟中断") }
	runner.outputs["beta"] = func(commandSpec) (string, error) { return "", errors.New("模拟中断") }
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	empty, err := m.Start(StartRequest{
		Topic: "全失败场", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, empty.ID, StatusFailed)
	if _, err := m.Resume(empty.ID); err == nil || !strings.Contains(err.Error(), "没有可续的历史发言") {
		t.Fatalf("empty session must refuse resume, got %v", err)
	}

	// 第二幕：合成一场「第 1 轮跑完即中断」的失败场，恢复后从第 2 轮继续。
	now := time.Now().UnixMilli()
	resumable := &Session{
		ID: "pan_resume1", Title: "恢复验证", Status: StatusFailed, Phase: PhaseDiscussion,
		CurrentRound: 1, CreatedAt: now, UpdatedAt: now,
		Request: StartRequest{
			Topic: "中断恢复", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
			Protocol: ProtocolCouncil, Rounds: 2,
		},
		Participants: []Participant{
			{AgentID: "alpha", Name: "ALPHA", Role: "participant", Status: "finished"},
			{AgentID: "beta", Name: "BETA", Role: "participant", Status: "failed"},
		},
		Contributions: []Contribution{{
			ID: "c1", Phase: PhaseOpening, Round: 1, AgentID: "alpha", AgentName: "ALPHA",
			Content: "既有历史观点：先做只读验证。", CreatedAt: now,
		}},
		Events: []Event{},
	}
	if err := m.store.save(resumable); err != nil {
		t.Fatal(err)
	}
	healthy := defaultFakeRunner()
	runner.outputs = healthy.outputs
	resumed, err := m.Resume("pan_resume1")
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != StatusRunning {
		t.Fatalf("resumed status = %q, want running", resumed.Status)
	}
	done := waitForStatus(t, m, "pan_resume1", StatusCompleted)
	// 贡献=既有 1 条 + 第 2 轮 2 席 + 汇总 1 条 = 4；既有历史原样保留。
	if len(done.Contributions) != 4 {
		t.Fatalf("contributions = %d, want 4", len(done.Contributions))
	}
	if done.Contributions[0].ID != "c1" || !strings.Contains(done.Contributions[0].Content, "既有历史观点") {
		t.Fatalf("existing history must be kept verbatim: %+v", done.Contributions[0])
	}
	if done.CurrentRound != 2 {
		t.Fatalf("current round = %d, want 2 (continued from interruption)", done.CurrentRound)
	}
	// 恢复事件入档 + 历史以立场卡进入第 2 轮简报。
	recovered := false
	for _, event := range done.Events {
		if strings.Contains(event.Text, "从中断处恢复") {
			recovered = true
		}
	}
	if !recovered {
		t.Fatal("resume event missing from timeline")
	}
	betaSawHistory := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "「BETA」") &&
			strings.Contains(call.Stdin, "既有历史观点：先做只读验证") {
			betaSawHistory = true
		}
	}
	if !betaSawHistory {
		t.Fatal("resumed round must see kept history as stance card")
	}
}

func TestInputManifestExported(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "材料快照入档", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Evidence: []EvidenceItem{{Name: "调研", Content: "竞品验证材料。"}}, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(done.InputManifest) != 1 || done.InputManifest[0].Name != "调研" {
		t.Fatalf("input manifest not persisted: %+v", done.InputManifest)
	}
	sum := sha256.Sum256([]byte("竞品验证材料。"))
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(export, "### 材料快照（开始时哈希）") ||
		!strings.Contains(export, "sha256:"+hex.EncodeToString(sum[:])) {
		t.Fatalf("export misses input snapshot:\n%s", export)
	}
}
