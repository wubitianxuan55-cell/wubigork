package pantheon

import (
	"context"
	"strings"
	"testing"
)

func TestParseChairEnvelopeLastCompleteWins(t *testing.T) {
	text := "1. 真分歧：A 与 B。\n" +
		"示例（勿模仿）：<pantheon-chair>{\"assignments\":[{\"agentId\":\"alpha\",\"instruction\":\"旧草稿\"}]}</pantheon-chair>\n" +
		"2. 裁决建议：交用户。\n" +
		"<pantheon-chair>{\"assignments\":[{\"agentId\":\"codex\",\"instruction\":\"先复核台账 issue-1\"}]}</pantheon-chair>"
	assignments, visible, malformed := parseChairEnvelope(text)
	if malformed {
		t.Fatal("合法信封不应判 malformed")
	}
	if len(assignments) != 1 || assignments[0].AgentID != "codex" || !strings.Contains(assignments[0].Instruction, "issue-1") {
		t.Fatalf("应取最后一个完整信封: %+v", assignments)
	}
	if strings.Contains(visible, "pantheon-chair") || strings.Contains(visible, "旧草稿") {
		t.Fatalf("可见文本不得残留信封原文：%q", visible)
	}
	if !strings.Contains(visible, "真分歧") || !strings.Contains(visible, "裁决建议") {
		t.Fatalf("可见文本应保留人类可读小结：%q", visible)
	}
}

func TestParseChairEnvelopeMalformedDegrades(t *testing.T) {
	text := "正常小结。\n<pantheon-chair>{\"assignments\":[{\"agentId\": broken"
	assignments, visible, malformed := parseChairEnvelope(text)
	if !malformed {
		t.Fatal("损坏信封应判 malformed")
	}
	if assignments != nil {
		t.Fatalf("malformed 时不得产出派工: %+v", assignments)
	}
	if strings.Contains(visible, "pantheon-chair") {
		t.Fatalf("残片不得上屏：%q", visible)
	}
	if !strings.Contains(visible, "正常小结") {
		t.Fatalf("有效正文必须保留：%q", visible)
	}
}

func TestParseChairEnvelopeAbsent(t *testing.T) {
	assignments, visible, malformed := parseChairEnvelope("只有三段小结，没有信封。")
	if assignments != nil || malformed {
		t.Fatalf("无信封应零值: %+v %v", assignments, malformed)
	}
	if visible != "只有三段小结，没有信封。" {
		t.Fatalf("无信封时可见文本应原样: %q", visible)
	}
}

func TestChairInstructionForRoundLookup(t *testing.T) {
	session := &Session{ID: "pan_t", Contributions: []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "codex", Content: "讨论"},
		{Round: 1, Phase: PhaseModeration, AgentID: "grok", ChairAssignments: []ChairAssignment{
			{AgentID: "codex", Instruction: "先复核 issue-1"},
			{AgentID: "ghost", Instruction: "不存在的席位"},
		}},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "codex", Content: "第二轮"},
	}}
	if got := chairInstructionFor(session, "codex", 2); got != "先复核 issue-1" {
		t.Fatalf("定向派工查表 = %q", got)
	}
	if got := chairInstructionFor(session, "grok", 2); got != "" {
		t.Fatalf("无派工席位应得空: %q", got)
	}
	if got := chairInstructionFor(session, "codex", 1); got != "" {
		t.Fatalf("第 1 轮无上一轮小结: %q", got)
	}
}

func TestDuplicatedAssignmentWarningTripAtThree(t *testing.T) {
	session := &Session{ID: "pan_t"}
	same := ChairAssignment{AgentID: "codex", Instruction: " 继续复核   issue-1 "}
	for _, round := range []int{1, 2} {
		session.Contributions = append(session.Contributions, Contribution{
			Round: round, Phase: PhaseModeration, AgentID: "grok",
			ChairAssignments: []ChairAssignment{same},
		})
	}
	if warning := duplicatedAssignmentWarning(session); warning != "" {
		t.Fatalf("两次不应触发: %q", warning)
	}
	session.Contributions = append(session.Contributions, Contribution{
		Round: 3, Phase: PhaseModeration, AgentID: "grok",
		ChairAssignments: []ChairAssignment{
			{AgentID: "codex", Instruction: "继续复核 issue-1"}, // 空白差异归一化后同键
		},
	})
	warning := duplicatedAssignmentWarning(session)
	if warning == "" || !strings.Contains(warning, "3 次") || !strings.Contains(warning, "codex") {
		t.Fatalf("三次应触发: %q", warning)
	}
	// 同一警告不重复触发（addWarning 不去重，靠本函数扫描 Warnings）。
	session.Warnings = append(session.Warnings, warning)
	if again := duplicatedAssignmentWarning(session); again != "" {
		t.Fatalf("已告警的键不得重复触发: %q", again)
	}
}

func TestParseChairDecisionVectors(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		wantStatus string
		wantDetail string
		malformed  bool
	}{
		{
			name:       "no envelope is continue",
			text:       "普通小结",
			wantStatus: chairDecisionContinue,
		},
		{
			name:       "empty object is continue",
			text:       `<pantheon-chair>{}</pantheon-chair>`,
			wantStatus: chairDecisionContinue,
		},
		{
			name:       "explicit continue",
			text:       `<pantheon-chair>{"decision":"continue","assignments":[{"agentId":"alpha","instruction":"复核"}]}</pantheon-chair>`,
			wantStatus: chairDecisionContinue,
		},
		{
			name:       "completed with detail",
			text:       `<pantheon-chair>{"decision":"completed","detail":"可收口"}</pantheon-chair>`,
			wantStatus: chairDecisionCompleted,
			wantDetail: "可收口",
		},
		{
			name:       "needs input underscore normalized",
			text:       `<pantheon-chair>{"decision":"needs_input","detail":"需要你选 A/B"}</pantheon-chair>`,
			wantStatus: chairDecisionNeedsInput,
			wantDetail: "需要你选 A/B",
		},
		{
			name:       "blocked without detail is malformed",
			text:       `<pantheon-chair>{"decision":"blocked"}</pantheon-chair>`,
			wantStatus: chairDecisionContinue,
			malformed:  true,
		},
		{
			name:       "unknown decision is malformed",
			text:       `<pantheon-chair>{"decision":"later","detail":"x"}</pantheon-chair>`,
			wantStatus: chairDecisionContinue,
			malformed:  true,
		},
		{
			name:       "last complete envelope wins",
			text:       `<pantheon-chair>{"decision":"completed","detail":"旧"}</pantheon-chair>` + "\n" + `<pantheon-chair>{"decision":"blocked","detail":"新"}</pantheon-chair>`,
			wantStatus: chairDecisionBlocked,
			wantDetail: "新",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, malformed := parseChairDecision(tc.text)
			if got.Status != tc.wantStatus || got.Detail != tc.wantDetail || malformed != tc.malformed {
				t.Fatalf("got %+v malformed=%v, want status=%q detail=%q malformed=%v", got, malformed, tc.wantStatus, tc.wantDetail, tc.malformed)
			}
		})
	}
}

func TestModerationPromptCarriesReceipts(t *testing.T) {
	req := StartRequest{Topic: "回执", Rounds: 2, Preset: PresetRoundtable}
	fac := AgentStatus{ID: "alpha", Name: "ALPHA"}
	prompt := moderationPrompt(req, fac, []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "BETA", Content: "beta 发言"},
	}, 1, 1, nil, []ChairAssignmentReceipt{
		{Round: 1, AgentID: "beta", AgentName: "BETA", Status: ReceiptSpoke, ContributionID: "c-1"},
	})
	for _, want := range []string{"上一轮派工回执", "已发言（未证明完成）", "贡献 ID c-1", "不要把「已发言」写成「已完成」"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestModerationPromptCarriesLedgerRosterAndEnvelope(t *testing.T) {
	req := StartRequest{Topic: "信封"}
	fac := AgentStatus{ID: "grok", Name: "Grok"}
	roundContribs := []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "codex", AgentName: "CODEX", Content: "codex 发言"},
		{Round: 1, Phase: PhaseDiscussion, AgentID: "user", Content: "用户插话"},
	}
	prompt := moderationPrompt(req, fac, roundContribs, 1, 2, []Issue{
		{ID: "pan_t-issue-1", Round: 1, AgentName: "CODEX", Summary: "缓存账目口径未对齐", Status: "open"},
	}, nil)
	if !strings.Contains(prompt, "pan_t-issue-1") || !strings.Contains(prompt, "当前分歧台账") {
		t.Fatal("moderationPrompt 必须携带 open 台账（注入不对称点销项）")
	}
	if !strings.Contains(prompt, "codex（CODEX）") {
		t.Fatalf("moderationPrompt 必须带席位 ID 名册（信封 agentId 不可猜）：%q", prompt)
	}
	if !strings.Contains(prompt, chairEnvelopeOpen) || !strings.Contains(prompt, "没有定向指令就不要输出信封") {
		t.Fatal("moderationPrompt 缺信封协议段")
	}
	if !strings.Contains(prompt, "补充论证") == false && strings.Contains(prompt, "下轮必答") == false {
		t.Fatal("moderationPrompt 结构不完整")
	}
}

func TestDiscussionPromptInjectsChairInstruction(t *testing.T) {
	agent := AgentStatus{ID: "codex", Name: "CODEX"}
	prompt := discussionPromptMode(StartRequest{Topic: "定向"}, agent, nil, 2, 1, 2, nil, nil, historyGradient, "先复核 issue-1 的缓存口径")
	if !strings.Contains(prompt, "【主席定向指令（第 1 轮小结）】") || !strings.Contains(prompt, "先复核 issue-1 的缓存口径") {
		t.Fatalf("定向指令段未注入：%q", prompt)
	}
	plain := discussionPrompt(StartRequest{Topic: "无定向"}, agent, nil, 2, 1, 2, nil, nil)
	if strings.Contains(plain, "主席定向指令") {
		t.Fatal("无派工时不得出现定向指令段")
	}
}

func TestDetectInteractionRingVectors(t *testing.T) {
	modID := "pan_t-grok-mod-r1"
	contribs := func(ids ...[2]string) []Contribution {
		out := make([]Contribution, 0, len(ids))
		for i, id := range ids {
			out = append(out, Contribution{
				ID: id[0], AgentID: id[1], AgentName: id[1],
				DiscussionAct: DiscussionActChallenge, TargetContributionID: modID,
			})
			_ = i
		}
		return out
	}
	// V5 星形免疫：多席全部指向同一 hub（主席小结）→ 不是环。
	star := []Contribution{
		{ID: modID, AgentID: "grok", Phase: PhaseModeration, Content: "小结"},
	}
	star = append(star, contribs([2]string{"a1", "alpha"}, [2]string{"b1", "beta"}, [2]string{"c1", "codex"})...)
	if ring, ok := DetectInteractionRing(star); ok {
		t.Fatalf("星形不得判环: %v", ring)
	}
	// A→B→C→A 三席互指 → 环。
	ringCase := []Contribution{
		{ID: "x1", AgentID: "alpha", DiscussionAct: DiscussionActChallenge, TargetContributionID: "y1"},
		{ID: "y1", AgentID: "beta", DiscussionAct: DiscussionActRespond, TargetContributionID: "z1"},
		{ID: "z1", AgentID: "codex", DiscussionAct: DiscussionActChallenge, TargetContributionID: "x1"},
	}
	ring, ok := DetectInteractionRing(ringCase)
	if !ok || len(ring) != 3 {
		t.Fatalf("三席环应检出: %v %v", ring, ok)
	}
	// 二元往返与链形不判环。
	two := []Contribution{
		{ID: "p1", AgentID: "alpha", DiscussionAct: DiscussionActChallenge, TargetContributionID: "p2"},
		{ID: "p2", AgentID: "beta", DiscussionAct: DiscussionActRespond, TargetContributionID: "p1"},
	}
	if _, ok := DetectInteractionRing(two); ok {
		t.Fatal("二元往返不判环")
	}
	chain := []Contribution{
		{ID: "q1", AgentID: "alpha", DiscussionAct: DiscussionActChallenge, TargetContributionID: "q2"},
		{ID: "q2", AgentID: "beta", DiscussionAct: DiscussionActRespond, TargetContributionID: "q3"},
		{ID: "q3", AgentID: "codex", Content: "无动作收尾"},
	}
	if _, ok := DetectInteractionRing(chain); ok {
		t.Fatal("开链不判环")
	}
	// 未定位目标/自指 = 无边（insufficient 纪律）。
	dangling := []Contribution{
		{ID: "d1", AgentID: "alpha", DiscussionAct: DiscussionActChallenge, TargetContributionID: "不存在的贡献"},
		{ID: "d2", AgentID: "beta", DiscussionAct: DiscussionActChallenge, TargetContributionID: "d2"},
	}
	if _, ok := DetectInteractionRing(dangling); ok {
		t.Fatal("未定位与自指不得成边")
	}
}

// TestManagerChairEnvelopeFlowsToNextRound 端到端：grok（模型席、主持）第 1 轮
// 小结带私有信封 → 剥除入档 → 第 2 轮 alpha 的 CLI 提示词收到定向指令、且
// 存档 Contribution 不含信封原文。
func TestManagerChairEnvelopeFlowsToNextRound(t *testing.T) {
	envelope := "\n<pantheon-chair>{\"assignments\":[{\"agentId\":\"alpha\",\"instruction\":\"先复核超时判据 issue-1\"},{\"agentId\":\"ghost\",\"instruction\":\"应被丢弃\"}]}</pantheon-chair>"
	runner := &fakeRunner{outputs: map[string]func(commandSpec) (string, error){
		"alpha": func(spec commandSpec) (string, error) {
			return fakeOutputFor("alpha", spec.Stdin), nil
		},
		"beta": func(spec commandSpec) (string, error) {
			return fakeOutputFor("beta", spec.Stdin), nil
		},
	}}
	backend := &envelopeGrokBackend{envelope: envelope}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"), newGrokAdapter(backend))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "信封流转",
		AgentIDs:        []string{"alpha", "beta", "grok"},
		ChairmanAgentID: "grok",
		Rounds:          2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil {
		t.Fatalf("未形成小结: %+v", done.Error)
	}
	// 存档：moderation 贡献带派工、可见文本无信封原文；ghost 被过滤。
	var moderation *Contribution
	for i := range done.Contributions {
		c := &done.Contributions[i]
		if c.Phase == PhaseModeration && len(c.ChairAssignments) > 0 {
			moderation = c
			break
		}
	}
	if moderation == nil {
		t.Fatal("moderation 贡献缺 ChairAssignments")
	}
	if len(moderation.ChairAssignments) != 1 || moderation.ChairAssignments[0].AgentID != "alpha" {
		t.Fatalf("非参会席位派工应被过滤: %+v", moderation.ChairAssignments)
	}
	if strings.Contains(moderation.Content, "pantheon-chair") {
		t.Fatalf("存档正文不得含信封原文: %q", moderation.Content)
	}
	// 事件流：ghost 丢弃留痕。
	ghostWarned := false
	for _, w := range done.Warnings {
		if strings.Contains(w, "非参会席位") {
			ghostWarned = true
		}
	}
	if !ghostWarned {
		t.Fatalf("ghost 派工丢弃应留痕: %v", done.Warnings)
	}
	// 第 2 轮 alpha 提示词收到定向指令。
	runner.mu.Lock()
	found := false
	for _, spec := range runner.calls {
		if strings.Contains(spec.Stdin, "「ALPHA」") && strings.Contains(spec.Stdin, "【主席定向指令（第 1 轮小结）】") &&
			strings.Contains(spec.Stdin, "先复核超时判据 issue-1") {
			found = true
		}
	}
	runner.mu.Unlock()
	if !found {
		t.Fatal("第 2 轮 alpha 提示词未收到主席定向指令")
	}
}

// envelopeGrokBackend 主持桩：小结固定带信封尾巴。
type envelopeGrokBackend struct {
	fakeGrokBackend
	envelope string
}

func (b *envelopeGrokBackend) CallGrok(_ context.Context, prompt string) (string, error) {
	out := fakeOutputFor("grok", prompt)
	if strings.Contains(prompt, "轮间小结") {
		return out + b.envelope, nil
	}
	b.mu.Lock()
	b.prompts = append(b.prompts, prompt)
	b.mu.Unlock()
	return out, nil
}
