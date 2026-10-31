package pantheon

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// 2026-10-31 圆桌裁决第一刀验收基线：
//   - 第 2 席 3000 字发言后，第 3 席 prompt 含全文（而非旧 1200 字静默截断）；
//   - 截断必须带原长与贡献 ID，且与作者自己的省略号区分；
//   - 导出同时出现共识信号与停止信号。
func longStatement(prefix string) string {
	segments := make([]string, 0, 150)
	for i := 0; i < 150; i++ {
		segments = append(segments, fmt.Sprintf("%s第%03d段论证：证据链%d完整可复核。", prefix, i, i))
	}
	return strings.Join(segments, "")
}

func TestDistributionDeliversSameRoundStatementInFull(t *testing.T) {
	runner := defaultFakeRunner()
	statement := longStatement("Alpha") + "\n协作请求：请帮助核实末段边界。Alpha末段独有标记ZXQW。"
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		return statement, nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "分发保真", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, // 同轮全文分发是圆桌接力语义；默认已是独立开场
		Rounds:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil {
		t.Fatalf("missing verdict: %+v", done)
	}
	betaCalls := 0
	for _, call := range runner.calls {
		if !strings.Contains(call.Stdin, "「BETA」") {
			continue
		}
		betaCalls++
		switch {
		case strings.Contains(call.Stdin, "第 1 轮开场陈述"):
			// 同轮分发：全文在位（Rounds:1 会被既有规约抬到 2 轮，此为第 1 轮）。
			for _, want := range []string{
				"Alpha第000段论证", // 头部在位 → 全文而非仅尾部
				"Alpha第075段论证", // 中段在位 → 旧 1200 字截断已根除
				"Alpha末段独有标记ZXQW",
			} {
				if !strings.Contains(call.Stdin, want) {
					t.Fatalf("beta prompt lost same-round statement segment %q:\n%s", want, call.Stdin)
				}
			}
			if strings.Contains(call.Stdin, "〔gaea 分发截断") {
				t.Fatalf("same-round statement within budget must not be truncated:\n%s", call.Stdin)
			}
		case strings.Contains(call.Stdin, "第 2 轮补充论证"):
			// 往轮立场卡：带原长与贡献 ID，不携带全文。
			if !strings.Contains(call.Stdin, "〔gaea 分发截断：仅保留节选，原长 ") {
				t.Fatalf("previous-round statement must degrade to annotated stance card:\n%s", call.Stdin)
			}
			if strings.Contains(call.Stdin, "Alpha末段独有标记ZXQW") {
				t.Fatalf("stance card must not carry full text:\n%s", call.Stdin)
			}
		}
	}
	if betaCalls == 0 {
		t.Fatal("no beta prompts captured")
	}
}

func TestFormatContributionKeepsCompleteTailBeyondBudget(t *testing.T) {
	head := longStatement("头部前缀")
	body := head + "中间填充" + strings.Repeat("中", 6000) + "尾部独有标记QNZW。"
	item := Contribution{ID: "pan_s-r1-7", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: body}

	got := formatContributionBody(item, 1)
	if !strings.Contains(got, "〔gaea 分发截断：仅保留完整尾部，原长") {
		t.Fatalf("missing tail-keep marker:\n%s", got)
	}
	if !strings.Contains(got, "贡献 ID pan_s-r1-7") {
		t.Fatalf("marker must carry contribution ID:\n%s", got)
	}
	if !strings.Contains(got, "尾部独有标记QNZW") {
		t.Fatal("complete tail must survive")
	}
	if strings.Contains(got, "头部前缀第000段论证") {
		t.Fatal("head must be dropped for tail-keep, not silently mixed")
	}
}

func TestStanceCardAnnotatesPreviousRounds(t *testing.T) {
	body := longStatement("往轮") + "往轮独有尾部标记PLMK。"
	item := Contribution{ID: "pan_s-r1-9", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: body}

	got := formatContributionBody(item, 2)
	if !strings.Contains(got, "〔gaea 分发截断：仅保留节选，原长") {
		t.Fatalf("missing stance-card marker:\n%s", got)
	}
	if !strings.Contains(got, "全文见贡献 ID pan_s-r1-9") {
		t.Fatalf("card must point at full contribution ID:\n%s", got)
	}
	if !strings.HasPrefix(got, "往轮第000段论证") {
		t.Fatalf("card must keep mechanical excerpt head, no model re-summary:\n%s", got)
	}
	if strings.Contains(got, "往轮独有尾部标记PLMK") {
		t.Fatal("card must not silently carry the full text")
	}
}

func TestSynthesisReceivesFinalRoundInFull(t *testing.T) {
	earlier := Contribution{
		ID: "pan_s-r1-1", Round: 1, AgentID: "alpha", AgentName: "ALPHA",
		Content: longStatement("首轮") + "首轮独有尾部标记OKNZ。",
	}
	final := Contribution{
		ID: "pan_s-r3-4", Round: 3, AgentID: "beta", AgentName: "BETA",
		Content: longStatement("末轮") + "末轮独有尾部标记WQVN。",
	}
	prompt := synthesisPrompt(StartRequest{Topic: "汇总保真"}, AgentStatus{Name: "Grok"}, []Contribution{earlier, final})
	if !strings.Contains(prompt, "末轮独有尾部标记WQVN") {
		t.Fatal("summarizer must receive final-round statement in full")
	}
	if strings.Contains(prompt, "末轮独有尾部标记WQVN〔") || !strings.Contains(prompt, "原长") {
		t.Fatal("final-round full text must not be cut or unannotated")
	}
	if !strings.Contains(prompt, "全文见贡献 ID pan_s-r1-1") {
		t.Fatal("earlier rounds must degrade to stance cards with contribution ID")
	}
}

func TestHistoryRewritesStrippedConsensusSignal(t *testing.T) {
	item := Contribution{
		ID: "pan_s-r2-2", Round: 2, AgentID: "alpha", AgentName: "ALPHA",
		Content: "当前判断保持。", Consensus: "no",
	}
	got := formatContributionBody(item, 3)
	if !strings.Contains(got, "（协议行回写 · 该席第 2 轮共识状态：仍有分歧）") {
		t.Fatalf("stripped consensus line must be rewritten into history:\n%s", got)
	}
	// 回写行以全角括号开头，不得被 consensusMarker 再解析为当前轮声明。
	if consensusMarker.MatchString(got) {
		t.Fatal("rewritten line must not be re-parseable as a live consensus declaration")
	}
}

func TestExportShowsStopSignals(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "停止信号导出", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## 停止信号",
		"达到安全上限 2 轮，未达成全员一致",
		"alpha=仍有分歧",
		"beta=仍有分歧",
	} {
		if !strings.Contains(export, want) {
			t.Fatalf("export stop signals miss %q:\n%s", want, export)
		}
	}
	if done.ResultPath == "" {
		t.Fatal("completed session missing resultPath")
	}
	data, err := os.ReadFile(done.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if !strings.Contains(doc, "> 共识状态：仍有分歧（协议行提取）") {
		t.Fatalf("final document must carry per-seat consensus signal:\n%s", doc)
	}

	// 共识达成腿：结果行应报告提前停止轮次。
	agreeRunner := consensusRunner("一致")
	agree := newTestManager(t, agreeRunner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer agree.Close()
	agreeSession, err := agree.Start(StartRequest{
		Topic: "停止信号导出", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, agree, agreeSession.ID, StatusCompleted)
	agreeExport, err := agree.Export(agreeSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agreeExport, "第 2 轮达成全员一致，提前停止") {
		t.Fatalf("consensus-reached outcome line missing:\n%s", agreeExport)
	}
}

func TestStopWarningSeparatesAbsenceFromDisagreement(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	runner.outputs["beta"] = func(commandSpec) (string, error) {
		return "", errors.New("beta 离线")
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "缺席与分歧分账", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var warning string
	for _, item := range done.Warnings {
		if strings.Contains(item, "未达成全员一致") {
			warning = item
		}
	}
	if warning == "" {
		t.Fatalf("missing shortfall warning: %v", done.Warnings)
	}
	for _, want := range []string{
		"实质分歧 1 席（alpha）",
		"末轮缺席 1 席（beta）——缺席是运行状态，不构成实质分歧",
	} {
		if !strings.Contains(warning, want) {
			t.Fatalf("warning misses %q: %s", want, warning)
		}
	}
}

func TestStopWarningReportsClippedInputs(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:    "超限截断入账",
		Context:  strings.Repeat("约束材料。", 5200),
		AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var warning string
	for _, item := range done.Warnings {
		if strings.Contains(item, "未达成全员一致") {
			warning = item
		}
	}
	if warning == "" {
		t.Fatalf("missing shortfall warning: %v", done.Warnings)
	}
	if !strings.Contains(warning, "提示词超限截断 4 次") {
		t.Fatalf("clipped inputs must be accounted: %s", warning)
	}
}
