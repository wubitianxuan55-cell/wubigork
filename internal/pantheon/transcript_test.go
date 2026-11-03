package pantheon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 2026-11-03 论证文件化验收基线：
//   - 更早轮次发言归档为文件，提示词只带索引行+预览+全文路径；
//   - 大块材料超阈值归档，简报不再全文内联；
//   - 归档文件幂等、可自愈（session.json 是权威全文，缺失即从内存补写）。

func TestTranscriptEnsureContributionFileIdempotentAndSelfHealing(t *testing.T) {
	root := t.TempDir()
	tr, err := newTranscript(root, "pan_x")
	if err != nil {
		t.Fatal(err)
	}
	item := Contribution{ID: "pan_x-alpha-r1-1000", Content: "论证全文归档自愈标记SELFHEAL。"}
	path, err := tr.ensureContributionFile(item)
	if err != nil {
		t.Fatal(err)
	}
	again, err := tr.ensureContributionFile(item)
	if err != nil {
		t.Fatal(err)
	}
	if again != path {
		t.Fatalf("ensure must be idempotent: %q vs %q", path, again)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != item.Content {
		t.Fatalf("archived content mismatch:\n%q", string(data))
	}
	// 自愈：文件丢失后从内存贡献补写（旧会话 Resume 无缝）。
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	healed, err := tr.ensureContributionFile(item)
	if err != nil {
		t.Fatal(err)
	}
	if healed != path {
		t.Fatalf("healed path mismatch: %q vs %q", path, healed)
	}
	data, err = os.ReadFile(healed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "SELFHEAL") {
		t.Fatalf("healed file lost content:\n%q", string(data))
	}
	// 路径穿越与空正文拒绝。
	if _, err := tr.ensureContributionFile(Contribution{ID: "../evil", Content: "x"}); err == nil {
		t.Fatal("path traversal ID must be rejected")
	}
	if _, err := tr.ensureContributionFile(Contribution{ID: "pan_x-alpha-r2-2", Content: "  "}); err == nil {
		t.Fatal("empty content must be rejected")
	}
}

func TestEvidenceBeyondThresholdArchivesToFile(t *testing.T) {
	root := t.TempDir()
	tr, err := newTranscript(root, "pan_x")
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("证据材料正文。", 400) + "尾部独有证据标记EVTAIL。" // 2815 字符 > evidenceInlineRunes
	req := StartRequest{
		Topic:    "材料归档",
		Evidence: []EvidenceItem{{Name: "长材料/报告:终版", Content: content}},
	}
	brief := buildBrief(req, &tr)
	if !strings.Contains(brief, "原长 2815 字符") {
		t.Fatalf("archive note must carry original length:\n%s", brief)
	}
	if !strings.Contains(brief, "全文见文件 ") {
		t.Fatalf("brief must point at archived evidence file:\n%s", brief)
	}
	if strings.Contains(brief, "EVTAIL") {
		t.Fatal("brief must not inline the evidence tail")
	}
	path := filepath.Join(root, "sessions", "pan_x", "evidence", "01-长材料-报告-终版.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("evidence file missing: %v", err)
	}
	if string(data) != content {
		t.Fatal("archived evidence content must equal the original material")
	}
}

func TestSmallEvidenceStaysInline(t *testing.T) {
	tr, err := newTranscript(t.TempDir(), "pan_x")
	if err != nil {
		t.Fatal(err)
	}
	req := StartRequest{
		Topic:    "小材料内联",
		Evidence: []EvidenceItem{{Name: "短材料", Content: "短材料正文。"}},
	}
	brief := buildBrief(req, &tr)
	if !strings.Contains(brief, "【材料 · 短材料】\n短材料正文。") {
		t.Fatalf("small evidence must stay inline:\n%s", brief)
	}
}

func TestOlderRoundArchivesToIndexedFile(t *testing.T) {
	root := t.TempDir()
	tr, err := newTranscript(root, "pan_x")
	if err != nil {
		t.Fatal(err)
	}
	body := longStatement("古早轮") + "古早轮独有尾部标记ARCH9。"
	item := Contribution{ID: "pan_x-alpha-r1-7", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: body}

	got := formatContributionBody(item, 3, &tr)
	if !strings.Contains(got, "归档发言 · 原长") {
		t.Fatalf("older round must degrade to archive index line:\n%s", got)
	}
	if !strings.Contains(got, "全文见文件 ") {
		t.Fatalf("index line must carry full-text path:\n%s", got)
	}
	if !strings.Contains(got, "贡献 ID pan_x-alpha-r1-7") {
		t.Fatalf("index line must carry contribution ID:\n%s", got)
	}
	if strings.Contains(got, "古早轮独有尾部标记ARCH9") {
		t.Fatalf("archive preview must not carry the tail:\n%s", got)
	}
	if !strings.Contains(got, "归档预览截断") {
		t.Fatalf("preview must be annotated:\n%s", got)
	}
	// 归档文件真实落盘且全文在位。
	paths := tr.archiveContributions([]Contribution{item})
	if len(paths) != 1 {
		t.Fatalf("archiveContributions = %v", paths)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ARCH9") {
		t.Fatal("archived file must hold the full statement")
	}
}

func TestArchiveFailureFallsBackToStanceCard(t *testing.T) {
	// contrib 落盘点被同名文件顶住：归档失败必须降级回立场卡，不挡分发。
	tmp := t.TempDir()
	tr, err := newTranscript(tmp, "pan_x")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tr.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tr.dir, "contrib"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := longStatement("降级") + "降级独有尾部标记FALLBK。"
	item := Contribution{ID: "pan_x-alpha-r1-1", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: body}
	got := formatContributionBody(item, 3, &tr)
	if !strings.Contains(got, "〔gaea 分发截断：仅保留节选，原长") {
		t.Fatalf("archive failure must fall back to stance card:\n%s", got)
	}
	if strings.Contains(got, "全文见文件") {
		t.Fatalf("failed archive must not claim a file:\n%s", got)
	}
}

// 端到端：三轮会议后，末轮 prompt 中第 1 轮长发言只以归档索引出现，
// 全文不内联——实弹「22 次超限截断」的形态根修验收。
func TestEndToEndOlderRoundsArchiveInPrompts(t *testing.T) {
	runner := defaultFakeRunner()
	firstStatement := longStatement("首轮阿尔法") + "首轮独有尾部标记E2EARC。"
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "第 1 轮开场陈述") {
			return firstStatement, nil
		}
		return "短发言，无新增分歧。", nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "归档端到端", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCompleted)
	checked := false
	for _, call := range runner.calls {
		if !strings.Contains(call.Stdin, "「BETA」") || !strings.Contains(call.Stdin, "第 3 轮补充论证") {
			continue
		}
		checked = true
		if !strings.Contains(call.Stdin, "全文见文件 ") {
			t.Fatalf("round-3 beta prompt must carry archive index lines:\n%s", call.Stdin)
		}
		if strings.Contains(call.Stdin, "首轮独有尾部标记E2EARC") {
			t.Fatalf("round-1 full text must not be inlined into round-3 prompt:\n%s", call.Stdin)
		}
		if !strings.Contains(call.Stdin, "首轮阿尔法第000段论证") {
			t.Fatalf("archive preview head must stay recognizable:\n%s", call.Stdin)
		}
	}
	if !checked {
		t.Fatal("no round-3 beta prompt captured")
	}
}
