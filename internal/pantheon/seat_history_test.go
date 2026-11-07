package pantheon

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestBuildSeatHistoryChronologicalReplay(t *testing.T) {
	session := &Session{ID: "pan_test"}
	// 时序（存档序=完成序）：alpha r1 → grok r1 → beta r1 → 用户插话 r1 →
	// grok 主席小结 r1 → beta r2 → grok r2。
	session.Contributions = []Contribution{
		{Round: 1, Phase: PhaseOpening, AgentID: "alpha", AgentName: "Alpha", Content: "alpha 第一轮开场"},
		{Round: 1, Phase: PhaseOpening, AgentID: "grok", AgentName: "Grok", Content: "grok 第一轮开场"},
		{Round: 1, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "beta 第一轮讨论"},
		{Round: 1, Phase: PhaseUserNote, AgentID: "user", Content: "用户插话内容"},
		{Round: 1, Phase: PhaseModeration, AgentID: "grok", AgentName: "Grok", Content: "grok 第一轮小结"},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "beta 第二轮讨论"},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "grok", AgentName: "Grok", Content: "grok 第二轮讨论"},
		{Round: 0, Phase: PhaseSynthesis, AgentID: "grok", AgentName: "Grok", Content: "汇总产物不进历史"},
	}
	history := buildSeatHistory(session, "grok", "【议题】测试议题", 0)
	if len(history) != 6 {
		t.Fatalf("history len = %d, want 6 (三对 user/assistant)", len(history))
	}
	for i := 0; i+1 < len(history); i += 2 {
		if history[i].Role != "user" || history[i+1].Role != "assistant" {
			t.Fatalf("第 %d 对角色 = %s/%s, want user/assistant", i/2, history[i].Role, history[i+1].Role)
		}
	}
	// 第一条 user：简报 + alpha r1（grok 发言前唯一可见者）。
	first := history[0].Content
	if !strings.Contains(first, "【议题】测试议题") || !strings.Contains(first, "alpha 第一轮开场") {
		t.Fatalf("首条 user 缺简报或 alpha r1：%q", first)
	}
	if strings.Contains(first, "beta 第一轮讨论") {
		t.Fatal("beta r1 在 grok r1 发言之后，不得提前进第一条 user")
	}
	// 时序配对：pair1=user[alpha r1]+asst[grok r1 讨论]；pair2=user[beta r1+
	// 用户插话]+asst[grok 主席小结]；pair3=user[beta r2]+asst[grok r2]。
	if got := history[1].Content; !strings.Contains(got, "grok 第一轮开场") {
		t.Fatalf("pair1 assistant 应为 grok r1 讨论：%q", got)
	}
	if got := history[2].Content; !strings.Contains(got, "用户插话内容") || !strings.Contains(got, "beta 第一轮讨论") {
		t.Fatalf("pair2 user 应含 beta r1 与用户插话：%q", got)
	}
	if got := history[3].Content; !strings.Contains(got, "主席小结") || !strings.Contains(got, "grok 第一轮小结") {
		t.Fatalf("pair2 assistant 应为 grok 主席小结：%q", got)
	}
	if got := history[4].Content; !strings.Contains(got, "beta 第二轮讨论") {
		t.Fatalf("pair3 user 应为 beta r2：%q", got)
	}
	if got := history[5].Content; !strings.Contains(got, "grok 第二轮讨论") {
		t.Fatalf("pair3 assistant 应为 grok r2：%q", got)
	}
	if strings.Contains(joinHistory(history), "汇总产物不进历史") {
		t.Fatal("synthesis 条目不得进历史")
	}
}

func TestBuildSeatHistoryRoundBound(t *testing.T) {
	session := &Session{ID: "pan_test", Contributions: []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: "第一轮内容"},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: "第二轮内容"},
	}}
	// 本席无任何发言：无配对，内容以纯 user 尾条保留（不丢记忆）。
	history := buildSeatHistory(session, "grok", "", 2)
	if len(history) != 1 || history[0].Role != "user" {
		t.Fatalf("bound=2 且本席未发言应得 1 条 user 尾，got %+v", history)
	}
	if !strings.Contains(history[0].Content, "第一轮内容") || strings.Contains(history[0].Content, "第二轮内容") {
		t.Fatalf("bound=2 尾条应只含第 1 轮（同轮可见性归提示词）：%q", history[0].Content)
	}
	full := buildSeatHistory(session, "grok", "", 0)
	if len(full) != 1 || !strings.Contains(full[0].Content, "第二轮内容") {
		t.Fatalf("bound=0（汇总）应含两轮全部他人内容：%q", full)
	}
}

func TestDiscussionPromptSeededOmitsGradient(t *testing.T) {
	long := strings.Repeat("往轮论证正文。", 300) // 1800 字 > 800 立场卡线
	history := []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: long},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "当轮 beta 发言"},
	}
	req := StartRequest{Topic: "种子形态"}
	agent := AgentStatus{ID: "grok", Name: "Grok"}
	plain := discussionPrompt(req, agent, history, 2, 1, 2, nil, nil)
	if !strings.Contains(plain, "仅保留节选") {
		t.Fatal("非 seeded 形态回归：应保留立场卡梯度")
	}
	seeded := discussionPromptMode(req, agent, history, 2, 1, 2, nil, nil, historySeeded, "")
	if strings.Contains(seeded, "仅保留节选") || strings.Contains(seeded, "归档发言") {
		t.Fatalf("seeded 形态不得重复往轮梯度：%q", seeded)
	}
	if !strings.Contains(seeded, "已逐字保留在本会话上下文中") {
		t.Fatal("seeded 形态缺会话上下文指针行")
	}
	if !strings.Contains(seeded, "当轮 beta 发言") {
		t.Fatal("seeded 形态应保留当轮他人发言全文")
	}
	if strings.Contains(seeded, long) {
		t.Fatal("seeded 形态不得内联往轮长文（已在会话历史里）")
	}
}

func TestSynthesisPromptSeededOmitsRecord(t *testing.T) {
	history := []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: strings.Repeat("甲。", 900)},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: strings.Repeat("乙。", 900)},
	}
	req := StartRequest{Topic: "汇总"}
	fac := AgentStatus{ID: "grok", Name: "Grok"}
	plain := synthesisPrompt(req, fac, history, nil, nil)
	if !strings.Contains(plain, "立场卡节选") {
		t.Fatal("非 seeded 汇总回归：应保留分层说明")
	}
	seeded := synthesisPromptMode(req, fac, history, nil, nil, historySeeded, nil)
	if strings.Contains(seeded, "立场卡节选") || strings.Contains(seeded, "归档发言") || strings.Contains(seeded, "甲。") {
		t.Fatalf("seeded 汇总不得重复讨论记录梯度：%q", seeded)
	}
	if !strings.Contains(seeded, "已逐字保留在本会话上下文中") || !strings.Contains(seeded, "整理要求") {
		t.Fatal("seeded 汇总缺指针行或整理要求段")
	}
}

type sessionedGrokBackend struct {
	fakeGrokBackend
	muHist    sync.Mutex
	histories [][]SeatMessage
	promptsS  []string
}

func (b *sessionedGrokBackend) StreamGrokSession(
	_ context.Context,
	history []SeatMessage,
	prompt string,
	_ string,
	_ []string,
	emit func(string),
	_ func(ToolTrace),
) (string, error) {
	b.muHist.Lock()
	b.histories = append(b.histories, append([]SeatMessage(nil), history...))
	b.promptsS = append(b.promptsS, prompt)
	b.muHist.Unlock()
	if emit != nil {
		emit("")
	}
	return b.CallGrok(context.Background(), prompt)
}

func TestManagerModelSeatReceivesSeededHistory(t *testing.T) {
	backend := &sessionedGrokBackend{}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(backend))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic:           "会话历史接线",
		AgentIDs:        []string{"alpha", "grok"},
		ChairmanAgentID: "alpha",
		Rounds:          2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil {
		t.Fatalf("session 未形成小结: %+v", done.Error)
	}
	backend.muHist.Lock()
	histories := append([][]SeatMessage(nil), backend.histories...)
	prompts := append([]string(nil), backend.promptsS...)
	backend.muHist.Unlock()
	// grok 调用 = 第 1 轮开场 + 第 2 轮讨论（主持 alpha，无 grok 小结；汇总走
	// alpha CLI 席）。开场历史为空（独立开场互不可见），讨论历史非空。
	if len(histories) != 2 {
		t.Fatalf("grok 会话后端调用 = %d, want 2", len(histories))
	}
	if len(histories[0]) != 0 {
		t.Fatalf("第 1 轮开场历史应为空（独立开场纪律），got %d 条", len(histories[0]))
	}
	joined := joinHistory(histories[1])
	if !strings.Contains(joined, "先做只读验证") {
		t.Fatalf("第 2 轮讨论历史应含 alpha 第 1 轮发言原文：%q", joined)
	}
	if len(histories[1]) < 2 || histories[1][0].Role != "user" || histories[1][1].Role != "assistant" {
		t.Fatalf("历史消息形态应为 user/assistant 配对：%+v", histories[1])
	}
	// seeded 提示词不再重复往轮梯度：第 2 轮 prompt 不含立场卡/归档行。
	if strings.Contains(prompts[1], "仅保留节选") || strings.Contains(prompts[1], "归档发言") {
		t.Fatalf("seeded 讨论提示词不得带往轮梯度：%q", prompts[1])
	}
}

func TestGrokAdapterSessionWiring(t *testing.T) {
	withSession := newGrokAdapter(&sessionedGrokBackend{})
	if withSession.streamWithToolsSession == nil {
		t.Fatal("实现 GrokHistoryBackend 的后端应接线 streamWithToolsSession")
	}
	plain := newGrokAdapter(&fakeGrokBackend{})
	if plain.streamWithToolsSession != nil {
		t.Fatal("未实现 GrokHistoryBackend 的后端不得接线（自动退回单发提示词）")
	}
}

func TestDiscussionPromptDeltaCarriesNewcomersFullText(t *testing.T) {
	history := []Contribution{
		{Round: 1, Phase: PhaseOpening, AgentID: "grok", AgentName: "Grok", Content: "grok 开场"},
		{Round: 1, Phase: PhaseOpening, AgentID: "alpha", AgentName: "Alpha", Content: "alpha 开场全文"},
		{Round: 1, Phase: PhaseUserNote, AgentID: "user", Content: "插话要求补核验"},
		{Round: 1, Phase: PhaseModeration, AgentID: "alpha", AgentName: "Alpha", Content: "alpha 主席小结"},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "beta 第二轮发言全文"},
	}
	req := StartRequest{Topic: "增量投递"}
	agent := AgentStatus{ID: "grok", Name: "Grok"}
	delta := discussionPromptMode(req, agent, history, 2, 1, 2, nil, nil, historyDelta, "")
	if strings.Contains(delta, "仅保留节选") || strings.Contains(delta, "归档发言") || strings.Contains(delta, "立场卡节选") {
		t.Fatalf("delta 形态不得重复梯度：%q", delta)
	}
	// 自 grok 开场以来的新增全部全文在场（含跨轮），且带说话者/插话标注。
	for _, want := range []string{"alpha 开场全文", "插话要求补核验", "alpha 主席小结", "beta 第二轮发言全文", "【用户插话】"} {
		if !strings.Contains(delta, want) {
			t.Fatalf("delta 形态缺新增载荷 %q", want)
		}
	}
	if strings.Contains(delta, "grok 开场") {
		t.Fatal("本席自己的旧发言不得重复进提示词（已在引擎侧会话）")
	}
	if !strings.Contains(delta, "上方共享观点里标注「用户插话」的条目") {
		t.Fatal("delta 含插话时【用户插话】指引段应触发")
	}
	// 无新增（插话/小结均在 grok 上次发言之前）→ 空载荷文案。
	older := []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: "alpha 先发言"},
		{Round: 1, Phase: PhaseDiscussion, AgentID: "grok", AgentName: "Grok", Content: "grok 后发言"},
	}
	empty := discussionPromptMode(req, agent, older, 2, 1, 2, nil, nil, historyDelta, "")
	if !strings.Contains(empty, "自你上次发言以来没有新增发言或插话") {
		t.Fatalf("无新增时应给空载荷文案：%q", empty)
	}
}

func TestDiscussionPromptFullWindowCarriesAllRoundsInline(t *testing.T) {
	history := []Contribution{
		{Round: 1, Phase: PhaseOpening, AgentID: "grok", AgentName: "Grok", Content: "grok 第一轮全文"},
		{Round: 1, Phase: PhaseOpening, AgentID: "alpha", AgentName: "Alpha", Content: "alpha 第一轮全文"},
		{Round: 1, Phase: PhaseUserNote, AgentID: "user", Content: "用户插话全文"},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "beta 第二轮全文"},
	}
	req := StartRequest{Topic: "全量窗口"}
	agent := AgentStatus{ID: "grok", Name: "Grok"}
	prompt := discussionPromptMode(req, agent, history, 2, 1, 2, nil, nil, historyFullWindow, "")
	// 全部往轮全文在场（含本席自己与他人、插话），无立场卡/归档压缩。
	for _, want := range []string{"grok 第一轮全文", "alpha 第一轮全文", "用户插话全文", "beta 第二轮全文", "（你此前的发言"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("全量窗口缺载荷 %q", want)
		}
	}
	if strings.Contains(prompt, "仅保留节选") || strings.Contains(prompt, "归档发言 · 原长") {
		t.Fatalf("全量窗口不得带梯度压缩：%q", prompt)
	}
	if !strings.Contains(prompt, "无需重新探索工作空间") {
		t.Fatal("全量窗口缺「别重新探索」纪律（Seed 证伪的根因防线）")
	}
}

func TestFullRecordBlockBudgetDropsOldest(t *testing.T) {
	items := make([]Contribution, 0, 6)
	for i := 1; i <= 6; i++ {
		items = append(items, Contribution{
			Round: i, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha",
			Content: strings.Repeat("长", 100),
		})
	}
	block := fullRecordBlock(items, "grok", 300) // 预算只够 ~3 条
	if strings.Contains(block, "（更早的") {
		// 有省略标注：最旧的被舍弃
		if strings.Contains(block, "第 1 轮") {
			t.Fatal("预算内应舍弃最旧发言")
		}
		if !strings.Contains(block, "第 6 轮") {
			t.Fatal("最新发言必须保真")
		}
	} else {
		t.Fatal("超预算应出现省略标注")
	}
	_ = items
}

func TestSynthesisPromptFullWindowInlinesRecord(t *testing.T) {
	history := []Contribution{
		{Round: 1, Phase: PhaseDiscussion, AgentID: "alpha", AgentName: "Alpha", Content: strings.Repeat("甲。", 900)},
		{Round: 2, Phase: PhaseDiscussion, AgentID: "beta", AgentName: "Beta", Content: "beta 第二轮全文"},
	}
	fac := AgentStatus{ID: "grok", Name: "Grok"}
	prompt := synthesisPromptMode(StartRequest{Topic: "汇总"}, fac, history, nil, nil, historyFullWindow, nil)
	if !strings.Contains(prompt, "beta 第二轮全文") || !strings.Contains(prompt, "甲。") {
		t.Fatalf("全量窗口汇总应内联全部发言全文：%q", prompt[:400])
	}
	if strings.Contains(prompt, "立场卡节选") {
		t.Fatal("全量窗口汇总不得带梯度压缩")
	}
}

func TestSeatDeltaCapableRequiresActiveEngineSession(t *testing.T) {
	backend := &sessionedGrokBackend{}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(backend), newDSHAdapter("", "key"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "增量资格", AgentIDs: []string{"alpha", "dsh"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCompleted)
	// DSH 适配器有 buildResume，但假 runner 不产出 session 事件 → 无活跃
	// 引擎会话 → 不得进入 delta 形态。
	if m.seatDeltaCapable(session, "dsh") {
		t.Fatal("无活跃 EngineSessionID 时不得判定 delta capable")
	}
	if m.seatDeltaCapable(session, "alpha") {
		t.Fatal("无 buildResume 的 CLI 席不得判定 delta capable")
	}
	if !m.seatHistorySeeded("grok") {
		t.Fatal("grok（GrokHistoryBackend）应判定 seeded")
	}
}

func joinHistory(history []SeatMessage) string {
	var b strings.Builder
	for _, m := range history {
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}
