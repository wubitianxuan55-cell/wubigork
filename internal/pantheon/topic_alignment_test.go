package pantheon

import (
	"strings"
	"testing"
)

// 2026-11-16 三刀验收：议题对齐度（防轮间漂移）、成功标准覆盖面（防结论
// 缩小范围）、开场路径说明（防「前面怎么没人发言」的困惑追问——说明按
// 议事路径区分，不写死一段）。

func TestOpeningPromptExplainsOpeningMode(t *testing.T) {
	// 独立开场（默认协议）第 1 轮全员并行：空讨论区必须被解释成正常现象，
	// 且说明必须点名「独立开场」这一路径。
	prompt := openingPrompt(StartRequest{Topic: "并行", Protocol: ProtocolIndependent},
		AgentStatus{ID: "alpha", Name: "ALPHA"}, nil, 2, 3, nil)
	for _, want := range []string{"【独立开场说明】", "「独立开场」", "只是名册序号", "不是发言丢失或加载失败", "不要在正文里追问或汇报", "目前还没有已经发表的讨论记录"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("independent opening note misses %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "你是本轮第一位发言者") {
		t.Fatalf("parallel opening must not claim sole-first-speaker:\n%s", prompt)
	}
	// 互盲驳论第 1 轮同样并行，但说明必须是互盲口径。
	blind := openingPrompt(StartRequest{Topic: "互盲", Protocol: ProtocolBlindRebuttal},
		AgentStatus{ID: "alpha", Name: "ALPHA"}, nil, 3, 3, nil)
	for _, want := range []string{"【互盲驳论说明】", "「互盲驳论」", "全部轮次并行作答", "不是发言丢失或加载失败"} {
		if !strings.Contains(blind, want) {
			t.Fatalf("blind rebuttal opening note misses %q:\n%s", want, blind)
		}
	}
	if strings.Contains(blind, "【独立开场说明】") {
		t.Fatalf("blind rebuttal must not borrow the independent label:\n%s", blind)
	}
	// 圆桌接力第 1 轮是真串行：首位就是第一位发言者，原口径保留、无并行说明。
	relayFirst := openingPrompt(StartRequest{Topic: "接力", Protocol: ProtocolCouncil},
		AgentStatus{ID: "alpha", Name: "ALPHA"}, nil, 1, 3, nil)
	if !strings.Contains(relayFirst, "你是本轮第一位发言者") {
		t.Fatalf("council first seat keeps serial semantics:\n%s", relayFirst)
	}
	if strings.Contains(relayFirst, "【独立开场说明】") || strings.Contains(relayFirst, "【互盲驳论说明】") {
		t.Fatalf("council must not inject parallel notes:\n%s", relayFirst)
	}
	// 接力第 1 轮后位看得到前序发言：不注入任何说明。
	relayLater := openingPrompt(StartRequest{Topic: "接力", Protocol: ProtocolCouncil},
		AgentStatus{ID: "beta", Name: "BETA"},
		[]Contribution{{ID: "c1", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: "立场"}}, 2, 3, nil)
	if strings.Contains(relayLater, "【独立开场说明】") || strings.Contains(relayLater, "【互盲驳论说明】") {
		t.Fatalf("council later seat needs no opening note:\n%s", relayLater)
	}
	// 单席会话（total=1）没有并行可言，保留原口径。
	solo := openingPrompt(StartRequest{Topic: "单席", Protocol: ProtocolIndependent},
		AgentStatus{ID: "alpha", Name: "ALPHA"}, nil, 1, 1, nil)
	if strings.Contains(solo, "【独立开场说明】") || !strings.Contains(solo, "你是本轮第一位发言者") {
		t.Fatalf("single-seat session keeps first-speaker wording:\n%s", solo)
	}
}

func TestModerationPromptRequiresAlignment(t *testing.T) {
	prompt := moderationPrompt(StartRequest{Topic: "对齐"}, AgentStatus{ID: "alpha", Name: "ALPHA"},
		[]Contribution{{ID: "c1", Round: 1, AgentID: "beta", AgentName: "BETA", Content: "发言"}}, 1, 2, nil, nil)
	for _, want := range []string{`"alignment"`, "on-track", "drift", "driftNote", "锚定议题核心目标"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("moderation prompt misses alignment contract %q:\n%s", want, prompt)
		}
	}
}

func TestParseChairAlignment(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		status string
		note   string
		bad    bool
	}{
		{"在轨", "<pantheon-chair>{\"decision\":\"continue\",\"alignment\":\"on-track\"}</pantheon-chair>", chairAlignmentOnTrack, "", false},
		{"漂移带注", "<pantheon-chair>{\"alignment\":\"drift\",\"driftNote\":\"全员陷入实现细节\"}</pantheon-chair>", chairAlignmentDrift, "全员陷入实现细节", false},
		{"下划线归一", "<pantheon-chair>{\"alignment\":\"on_track\"}</pantheon-chair>", chairAlignmentOnTrack, "", false},
		{"缺省不算错", "<pantheon-chair>{\"decision\":\"continue\"}</pantheon-chair>", "", "", false},
		{"无信封", "普通小结文本", "", "", false},
		{"未知值降级", "<pantheon-chair>{\"alignment\":\"sideways\"}</pantheon-chair>", "", "", true},
		{"坏 JSON 降级", "<pantheon-chair>{\"alignment\":</pantheon-chair>", "", "", true},
	}
	for _, tc := range cases {
		got, malformed := parseChairAlignment(tc.text)
		if got.Status != tc.status || got.Note != tc.note || malformed != tc.bad {
			t.Fatalf("%s: got %+v malformed=%v, want status=%q note=%q bad=%v", tc.name, got, malformed, tc.status, tc.note, tc.bad)
		}
	}
}

func TestConsecutiveDriftRounds(t *testing.T) {
	moderation := func(round int, alignment string) Contribution {
		return Contribution{ID: string(rune('a' + round)), Phase: PhaseModeration, Round: round, ChairAlignment: alignment}
	}
	none := []Contribution{
		{ID: "c1", Phase: PhaseOpening, Round: 1, Content: "发言"},
		moderation(1, chairAlignmentOnTrack),
		{ID: "c2", Phase: PhaseDiscussion, Round: 2, Content: "发言"},
	}
	if got := consecutiveDriftRounds(none); got != 0 {
		t.Fatalf("streak on-track = %d, want 0", got)
	}
	one := append(append([]Contribution{}, none...), moderation(1, chairAlignmentDrift))
	if got := consecutiveDriftRounds(one); got != 1 {
		t.Fatalf("streak single drift = %d, want 1", got)
	}
	two := append(append([]Contribution{}, one...), moderation(2, chairAlignmentDrift))
	if got := consecutiveDriftRounds(two); got != 2 {
		t.Fatalf("streak double drift = %d, want 2", got)
	}
	// 判定缺失（主席未报）断开连击：不把「没说」当「漂了」。
	broken := append(append([]Contribution{}, two...), moderation(3, ""))
	if got := consecutiveDriftRounds(broken); got != 0 {
		t.Fatalf("missing verdict must break streak, got %d", got)
	}
	// 中间夹杂的非小结贡献不影响回溯。
	mixed := append(append([]Contribution{}, one...), Contribution{ID: "c3", Phase: PhaseDiscussion, Round: 2, Content: "发言"})
	if got := consecutiveDriftRounds(mixed); got != 1 {
		t.Fatalf("streak with interleaved discussion = %d, want 1", got)
	}
}

func TestChairPullBackNotice(t *testing.T) {
	notice := chairPullBackNotice(2, 2, "全员陷入实现细节")
	for _, want := range []string{"【宿主拉回（第 2 轮）】", "连续 2 轮", "全员陷入实现细节", "议题核心目标", "遗留台账"} {
		if !strings.Contains(notice, want) {
			t.Fatalf("pull-back notice misses %q:\n%s", want, notice)
		}
	}
	noNote := chairPullBackNotice(3, 2, "")
	if !strings.Contains(noNote, "漂移」。") {
		t.Fatalf("notice without note must still read cleanly:\n%s", noNote)
	}
}

func TestSynthesisCoverageSection(t *testing.T) {
	withCriteria := synthesisPrompt(StartRequest{Topic: "覆盖", SuccessCriteria: "成本可复核\n结论带证据强度"}, AgentStatus{Name: "Grok"}, nil, nil, nil)
	for _, want := range []string{"【成功标准逐条核对】", "成本可复核", "结论带证据强度", `"coverage"`, "addressed", "unaddressed", "不得静默漏项"} {
		if !strings.Contains(withCriteria, want) {
			t.Fatalf("synthesis coverage misses %q:\n%s", want, withCriteria)
		}
	}
	without := synthesisPrompt(StartRequest{Topic: "无标准"}, AgentStatus{Name: "Grok"}, nil, nil, nil)
	if strings.Contains(without, "coverage") || strings.Contains(without, "成功标准逐条核对") {
		t.Fatalf("no criteria must not fabricate coverage contract:\n%s", without)
	}
}

func TestParseVerdictCoverage(t *testing.T) {
	raw := `{"summary":"结论","recommendation":"建议","coverage":[
		{"criterion":"成本可复核","status":"addressed"},
		{"criterion":"结论带证据强度","status":"partial"},
		{"criterion":"","status":"addressed"},
		{"criterion":"时序预算","status":"sideways"}
	]}`
	verdict := parseVerdict(raw, "alpha", 0)
	if len(verdict.Coverage) != 3 {
		t.Fatalf("coverage = %+v, want 3 entries (empty criterion dropped)", verdict.Coverage)
	}
	if verdict.Coverage[0].Status != "addressed" {
		t.Fatalf("first entry status = %q", verdict.Coverage[0].Status)
	}
	if verdict.Coverage[1].Status != "partial" || verdict.Coverage[1].Note == "" {
		t.Fatalf("partial entry must carry a note, got %+v", verdict.Coverage[1])
	}
	// 无法识别的状态按 unaddressed 保守入账：宁可报未回应也不静默丢失。
	if verdict.Coverage[2].Status != "unaddressed" || verdict.Coverage[2].Note == "" {
		t.Fatalf("unknown status must degrade to unaddressed with note, got %+v", verdict.Coverage[2])
	}
	// 无 JSON 的 partial 降级路径不带 coverage。
	fallback := parseVerdict("纯文本小结", "alpha", 0)
	if len(fallback.Coverage) != 0 || !fallback.Partial {
		t.Fatalf("fallback verdict must be partial without coverage: %+v", fallback)
	}
}

// 端到端：主席连续两轮判定漂移 → 第二次小结追加宿主拉回 + 告警；下一轮
// 席位提示词带拉回指令。
func TestConsecutiveDriftTriggersPullBack(t *testing.T) {
	runner := defaultFakeRunner()
	moderationCount := 0
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		switch {
		case strings.Contains(spec.Stdin, "轮间小结"):
			moderationCount++
			text := "分歧集中在验证方式；建议下轮各自补一条核验路径。\n"
			// 两轮都报 drift；第二轮宿主应触发拉回。
			text += "<pantheon-chair>{\"decision\":\"continue\",\"alignment\":\"drift\",\"driftNote\":\"全员陷入实现细节\"}</pantheon-chair>"
			return text, nil
		default:
			return fakeOutputFor(idFromPrompt(spec.Stdin, "alpha"), spec.Stdin), nil
		}
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "漂移拉回", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if moderationCount < 2 {
		t.Fatalf("moderations = %d, want >=2", moderationCount)
	}
	var driftModerations []Contribution
	for _, c := range done.Contributions {
		if c.Phase == PhaseModeration {
			if c.ChairAlignment != chairAlignmentDrift {
				t.Fatalf("moderation alignment = %q, want drift", c.ChairAlignment)
			}
			driftModerations = append(driftModerations, c)
		}
	}
	if len(driftModerations) != 2 {
		t.Fatalf("drift moderations = %d, want 2", len(driftModerations))
	}
	if strings.Contains(driftModerations[0].Content, "宿主拉回") {
		t.Fatalf("first drift must not trigger pull-back:\n%s", driftModerations[0].Content)
	}
	if !strings.Contains(driftModerations[1].Content, "【宿主拉回（第 2 轮）】") ||
		!strings.Contains(driftModerations[1].Content, "全员陷入实现细节") {
		t.Fatalf("second drift must carry pull-back notice:\n%s", driftModerations[1].Content)
	}
	joined := strings.Join(done.Warnings, "\n")
	if !strings.Contains(joined, "连续 2 轮判定议题漂移") {
		t.Fatalf("warnings missing drift alert:\n%s", joined)
	}
	// 小结全文是下一轮议程：第 3 轮席位提示词必须携带拉回指令。
	sawPullBack := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "补充论证") && strings.Contains(call.Stdin, "宿主拉回（第 2 轮）") {
			sawPullBack = true
			break
		}
	}
	if !sawPullBack {
		t.Fatal("round-3 prompts must carry the host pull-back via moderation agenda")
	}
}

// 单轮 drift 不触发拉回（1 轮可能是误判）。
func TestSingleDriftDoesNotPullBack(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "轮间小结") {
			return "小结正文。\n<pantheon-chair>{\"alignment\":\"drift\",\"driftNote\":\"略有跑题\"}</pantheon-chair>", nil
		}
		return fakeOutputFor(idFromPrompt(spec.Stdin, "alpha"), spec.Stdin), nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "单轮漂移", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, c := range done.Contributions {
		if c.Phase == PhaseModeration && strings.Contains(c.Content, "宿主拉回") {
			t.Fatalf("single drift must not inject pull-back:\n%s", c.Content)
		}
	}
	for _, w := range done.Warnings {
		if strings.Contains(w, "议题漂移") {
			t.Fatalf("single drift must not warn: %s", w)
		}
	}
}

// 在轨判定阻断连击：drift → on-track → drift 不触发拉回。
func TestOnTrackBreaksDriftStreak(t *testing.T) {
	runner := defaultFakeRunner()
	alignmentByRound := map[int]string{1: chairAlignmentDrift, 2: chairAlignmentOnTrack}
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "轮间小结") {
			round := 1
			if strings.Contains(spec.Stdin, "第 2 轮的主持人") {
				round = 2
			}
			text := "小结正文。\n<pantheon-chair>{\"alignment\":\"" + alignmentByRound[round] + "\"}</pantheon-chair>"
			return text, nil
		}
		return fakeOutputFor(idFromPrompt(spec.Stdin, "alpha"), spec.Stdin), nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "在轨断连击", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	for _, c := range done.Contributions {
		if c.Phase == PhaseModeration && strings.Contains(c.Content, "宿主拉回") {
			t.Fatalf("on-track must break the streak:\n%s", c.Content)
		}
	}
}

// 汇总带 coverage 的端到端：议题声明成功标准 → 汇总 JSON 携带 coverage 入档。
func TestSynthesisCarriesCoverageEndToEnd(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		if strings.Contains(spec.Stdin, "你的职责是协助整理") {
			return `{"summary":"结论回应了两条标准","recommendation":"先补第二条标准的证据","coverage":[
				{"criterion":"成本可复核","status":"addressed"},
				{"criterion":"结论带证据强度","status":"unaddressed","note":"全场没有证据强度对照"}
			]}`, nil
		}
		return fakeOutputFor(idFromPrompt(spec.Stdin, "alpha"), spec.Stdin), nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "覆盖面入档", SuccessCriteria: "成本可复核\n结论带证据强度",
		AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil || len(done.Verdict.Coverage) != 2 {
		t.Fatalf("verdict coverage = %+v, want 2 entries", done.Verdict)
	}
	if done.Verdict.Coverage[1].Status != "unaddressed" {
		t.Fatalf("second criterion status = %q, want unaddressed", done.Verdict.Coverage[1].Status)
	}
}
