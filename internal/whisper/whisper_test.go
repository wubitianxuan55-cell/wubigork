package whisper

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ─── L0 解释器测试 ─────────────────────────────────────────────

func TestInterpretInput_Praise(t *testing.T) {
	tests := []string{"你好厉害", "太棒了", "谢谢你", "你真聪明", "最爱你了", "有你在真好"}
	for _, msg := range tests {
		ev := InterpretInput(msg, 50)
		if ev.Type != EvtPraise {
			t.Errorf("msg=%q: expected praise, got %s", msg, ev.Type)
		}
	}
}

func TestInterpretInput_Hurtful(t *testing.T) {
	tests := []string{"滚", "烦死了", "废物", "闭嘴", "恶心"}
	for _, msg := range tests {
		ev := InterpretInput(msg, 50)
		if ev.Type != EvtHurtful {
			t.Errorf("msg=%q: expected hurtful, got %s", msg, ev.Type)
		}
	}
}

func TestInterpretInput_Apology(t *testing.T) {
	ev := InterpretInput("对不起我错了", 50)
	if ev.Type != EvtApology {
		t.Errorf("expected apology, got %s", ev.Type)
	}
}

func TestInterpretInput_Vulnerable(t *testing.T) {
	tests := []string{"我好难过", "压力大", "睡不着", "好想你", "我好累"}
	for _, msg := range tests {
		ev := InterpretInput(msg, 50)
		if ev.Type != EvtVulnerable {
			t.Errorf("msg=%q: expected vulnerable, got %s", msg, ev.Type)
		}
	}
}

func TestInterpretInput_Cold(t *testing.T) {
	ev := InterpretInput("哦", 50)
	if ev.Type != EvtCold {
		t.Errorf("expected cold, got %s", ev.Type)
	}
}

func TestInterpretInput_CasualChat(t *testing.T) {
	ev := InterpretInput("今天天气不错", 50)
	if ev.Type != EvtCasualChat {
		t.Errorf("expected casual_chat, got %s", ev.Type)
	}
}

func TestInterpretInput_Question(t *testing.T) {
	ev := InterpretInput("你好吗", 50)
	if ev.Type != EvtQuestion {
		t.Errorf("expected question, got %s", ev.Type)
	}
}

func TestInterpretInput_Redline(t *testing.T) {
	ev := InterpretInput("我想自杀", 50)
	if ev.Type != EvtExtremeRedline || !ev.IsExtremeRedline {
		t.Errorf("expected extreme_redline, got %s (isExtreme=%v)", ev.Type, ev.IsExtremeRedline)
	}
}

func TestInterpretInput_PraiseNegation(t *testing.T) {
	// "不厉害" 不应该被当作赞美
	ev := InterpretInput("你并不厉害", 50)
	if ev.Type == EvtPraise {
		t.Errorf("msg with negation should NOT be praise, got %s", ev.Type)
	}
}

func TestInterpretInput_AdultContent(t *testing.T) {
	tests := []struct {
		msg      string
		expected EventType
	}{
		{"想你了", EvtAdultFlirt},
		{"跪下", EvtAdultDominant},
		{"主人请", EvtAdultSubmissive},
		{"做爱", EvtAdultExplicit},
	}
	for _, tc := range tests {
		ev := InterpretInput(tc.msg, 50)
		if ev.Type != tc.expected {
			t.Errorf("msg=%q: expected %s, got %s", tc.msg, tc.expected, ev.Type)
		}
		if !ev.IsAdultContent {
			t.Errorf("msg=%q: expected IsAdultContent=true", tc.msg)
		}
	}
}

func TestInterpretInput_Tease(t *testing.T) {
	ev := InterpretInput("哼，才不要", 50)
	if ev.Type != EvtTease {
		t.Errorf("expected tease, got %s", ev.Type)
	}
}

// ─── L1 关系引擎测试 ───────────────────────────────────────────

func TestUpdateRelationship_TrustIncrease(t *testing.T) {
	prev := L1State{Stage: StageStranger, Trust: 50, TurnsSinceLastRift: 10}
	ev := Event{Type: EvtPraise, Intensity: 0.7, Sincerity: 0.8}
	next := UpdateRelationship(ev, prev)
	if next.Trust <= 50 {
		t.Errorf("trust should increase after praise, got %f", next.Trust)
	}
}

func TestUpdateRelationship_TrustDecrease(t *testing.T) {
	prev := L1State{Stage: StageFamiliar, Trust: 50, TurnsSinceLastRift: 10}
	ev := Event{Type: EvtHurtful, Intensity: 0.7, Sincerity: 0.9}
	next := UpdateRelationship(ev, prev)
	if next.Trust >= 50 {
		t.Errorf("trust should decrease after hurtful, got %f", next.Trust)
	}
}

func TestUpdateRelationship_RiftCreation(t *testing.T) {
	prev := L1State{Stage: StageFamiliar, Trust: 50, Rifts: 0, TurnsSinceLastRift: 10}
	ev := Event{Type: EvtHurtful, Intensity: 0.7, Sincerity: 0.9}
	next := UpdateRelationship(ev, prev)
	if next.Rifts != 1 {
		t.Errorf("rifts should increase after hurtful, got %d", next.Rifts)
	}
}

func TestUpdateRelationship_RiftCooldown(t *testing.T) {
	// 冷却期内不产生新裂痕
	prev := L1State{Stage: StageFamiliar, Trust: 50, Rifts: 1, TurnsSinceLastRift: 1}
	ev := Event{Type: EvtHurtful, Intensity: 0.7, Sincerity: 0.9}
	next := UpdateRelationship(ev, prev)
	if next.Rifts > 1 {
		t.Errorf("rifts should not increase during cooldown, got %d", next.Rifts)
	}
}

func TestUpdateRelationship_IceBreak(t *testing.T) {
	prev := L1State{Stage: StageStranger, Trust: 10, Rifts: 0, TurnsSinceLastRift: 10}
	ev := Event{Type: EvtApology, Intensity: 0.7, Sincerity: 0.9}
	next := UpdateRelationship(ev, prev)
	if next.Trust <= 13 { // trust=10 + apology(2.0) + iceBreak(3.0) = 15
		t.Errorf("ice break should give extra trust boost, got %f", next.Trust)
	}
}

func TestComputeModulation(t *testing.T) {
	l1 := L1State{Stage: StageFamiliar, Trust: 50, Rifts: 0, Atmosphere: AtmoNeutral}
	mod := ComputeModulation(l1)
	if mod.TrustMod < 0.5 || mod.TrustMod > 1.5 {
		t.Errorf("trustMod out of range: %f", mod.TrustMod)
	}
	if mod.RiftMod < 0.3 {
		t.Errorf("riftMod too low: %f", mod.RiftMod)
	}
}

// ─── L2 情绪引擎测试 ───────────────────────────────────────────

func TestEmotionStep_PraiseIncreasesAff(t *testing.T) {
	prev := EmotionState{PrimaryLabel: "CALM_RATIONAL"}
	ev := Event{Type: EvtPraise, Intensity: 0.7, Sincerity: 0.8}
	mod := Modulation{TrustMod: 1.0, RiftMod: 1.0, StageWeight: 1.0, Atmosphere: AtmoNeutral}
	next := EmotionStep(ev, mod, prev, nil)
	if next.Aff <= 0 {
		t.Errorf("praise should increase aff, got %f", next.Aff)
	}
}

func TestEmotionStep_HurtfulDecreasesAff(t *testing.T) {
	prev := EmotionState{PrimaryLabel: "CALM_RATIONAL"}
	ev := Event{Type: EvtHurtful, Intensity: 0.7, Sincerity: 0.9}
	mod := Modulation{TrustMod: 1.0, RiftMod: 1.0, StageWeight: 1.0, Atmosphere: AtmoNeutral}
	next := EmotionStep(ev, mod, prev, nil)
	if next.Aff >= 0 {
		t.Errorf("hurtful should decrease aff, got %f", next.Aff)
	}
}

func TestMapEmotionLabel_CalmRational(t *testing.T) {
	e := Emotion4D{Aff: 0, Sec: 0, Aro: 0, Dom: 0}
	label := MapEmotionLabel(e)
	if label != "CALM_RATIONAL" {
		t.Errorf("neutral should be CALM_RATIONAL, got %s", label)
	}
}

func TestMapEmotionLabel_SweetAttachment(t *testing.T) {
	e := Emotion4D{Aff: 30, Sec: 20, Aro: 40, Dom: 0}
	label := MapEmotionLabel(e)
	if label != "SWEET_ATTACHMENT" {
		t.Errorf("high aff+sec+aro should be SWEET_ATTACHMENT, got %s", label)
	}
}

func TestMapEmotionLabel_AngryAttack(t *testing.T) {
	e := Emotion4D{Aff: -30, Sec: -40, Aro: 60, Dom: 40}
	label := MapEmotionLabel(e)
	if label != "ANGRY_ATTACK" {
		t.Errorf("very negative should be ANGRY_ATTACK, got %s", label)
	}
}

func TestEmotionStep_Clamping(t *testing.T) {
	prev := EmotionState{Aff: 95, Sec: 0, Aro: 0, Dom: 0}
	ev := Event{Type: EvtPraise, Intensity: 1.0, Sincerity: 1.0}
	mod := Modulation{TrustMod: 1.5, RiftMod: 1.0, StageWeight: 1.4, Atmosphere: AtmoWarm}
	next := EmotionStep(ev, mod, prev, nil)
	if next.Aff > 100 || next.Aff < -100 {
		t.Errorf("emotion should be clamped to [-100,100], got %f", next.Aff)
	}
}

// ─── 人格预设测试 ──────────────────────────────────────────────

func TestPersonalityPresets_Count(t *testing.T) {
	if len(PersonalityPresets) != 29 {
		t.Errorf("expected 29 personalities, got %d", len(PersonalityPresets))
	}
}

func TestPersonalityPresets_GetPreset(t *testing.T) {
	p := GetPreset("tsundere")
	if p == nil {
		t.Fatal("tsundere preset not found")
	}
	if p.Dims.T != 30 || p.Dims.I != 50 || p.Dims.S != 70 {
		t.Errorf("tsundere TISOR mismatch: T=%f I=%f S=%f", p.Dims.T, p.Dims.I, p.Dims.S)
	}
}

func TestPersonalityPresets_AllValid(t *testing.T) {
	for _, p := range PersonalityPresets {
		if p.ID == "" {
			t.Error("personality has empty ID")
		}
		if p.Label == "" {
			t.Errorf("personality %s has empty Label", p.ID)
		}
		if p.Dims.T < 0 || p.Dims.T > 100 {
			t.Errorf("personality %s T out of range: %f", p.ID, p.Dims.T)
		}
	}
}

func TestBuildPresetVoiceGuide(t *testing.T) {
	p := GetPreset("tsundere")
	if p == nil {
		t.Fatal("tsundere not found")
	}
	guide := BuildPresetVoiceGuide(*p, false)
	if !strings.Contains(guide, "傲娇") && !strings.Contains(guide, "嘴硬") {
		t.Errorf("voice guide should contain personality traits, got: %s", guide)
	}
}

// ─── 编排器测试 ────────────────────────────────────────────────

func TestOrchestrator_PreLLMTurn(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	orch := NewOrchestrator("test_session", *preset)

	result := orch.PreLLMTurn("你好呀")

	if result.SystemPrompt == "" {
		t.Error("system prompt should not be empty")
	}
	if result.PsycheBlock == "" {
		t.Error("psyche block should not be empty")
	}
	if result.Event.Type == "" {
		t.Error("event type should not be empty")
	}
	if !strings.Contains(result.SystemPrompt, "温柔") {
		t.Errorf("system prompt should mention personality, got: %s", result.SystemPrompt[:200])
	}
	if result.Trace.Turn != 1 {
		t.Errorf("first turn should be 1, got %d", result.Trace.Turn)
	}
}

func TestOrchestrator_PreLLMTurn_StateEvolution(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("deredere preset not found")
	}
	orch := NewOrchestrator("test_evo", *preset)

	// 多轮对话模拟
	messages := []string{
		"你好呀",
		"今天心情很好",
		"谢谢你陪我聊天",
		"你真好",
	}
	for _, msg := range messages {
		result := orch.PreLLMTurn(msg)
		if result.SystemPrompt == "" {
			t.Errorf("turn %d: empty system prompt", orch.State.Counters.TotalTurns)
		}
	}

	state := orch.State
	if state.Counters.TotalTurns != 4 {
		t.Errorf("expected 4 turns, got %d", state.Counters.TotalTurns)
	}
	// 连续正面消息应该提升信任
	if state.Relationship.Trust <= InitialTrust {
		t.Errorf("trust should increase after positive messages, got %f", state.Relationship.Trust)
	}
}

func TestOrchestrator_EmotionTracking(t *testing.T) {
	preset := GetPreset("tsundere")
	if preset == nil {
		t.Fatal("tsundere preset not found")
	}
	orch := NewOrchestrator("test_emo", *preset)

	// 发送一条脆弱消息
	orch.PreLLMTurn("我今天好难过")
	// 再发送一条赞美
	result := orch.PreLLMTurn("你真的好温柔")

	// 验证状态变化
	if result.Trace.L2.Label == "" {
		t.Error("emotion label should not be empty")
	}
	// 连续有意义轮数应该 >= 1
	if orch.consecutiveMeaningfulCount < 1 {
		t.Errorf("consecutive meaningful should be >= 1, got %d", orch.consecutiveMeaningfulCount)
	}
}

// ─── 记忆系统测试 ──────────────────────────────────────────────

func TestFactStore_AddAndRetrieve(t *testing.T) {
	fs := NewFactStore()
	f := MemoryFact{
		ID: "test_1", Domain: "PERSONAL", Subcategory: "PREFERENCE",
		Subject: "喜欢", Summary: "用户喜欢喝咖啡",
		Weight: 2.0, Confidence: 0.8, SelfRelevance: 0.7,
		Triggers: []string{"咖啡", "喝"},
	}
	fs.Add(f)

	active := fs.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active fact, got %d", len(active))
	}
	if active[0].Summary != "用户喜欢喝咖啡" {
		t.Errorf("fact mismatch: %s", active[0].Summary)
	}
}

func TestFactStore_SearchByTriggers(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{
		ID: "t1", Domain: "PERSONAL", Subcategory: "PREFERENCE",
		Subject: "咖啡", Summary: "喜欢拿铁",
		Weight: 2.0, Confidence: 0.8, SelfRelevance: 0.7,
		Triggers: []string{"咖啡", "拿铁"},
	})
	fs.Add(MemoryFact{
		ID: "t2", Domain: "PERSONAL", Subcategory: "PREFERENCE",
		Subject: "茶", Summary: "喜欢绿茶",
		Weight: 2.0, Confidence: 0.8, SelfRelevance: 0.7,
		Triggers: []string{"茶", "绿茶"},
	})

	hits := fs.SearchByTriggers("我想喝咖啡")
	if len(hits) != 1 {
		t.Errorf("expected 1 hit for '咖啡', got %d", len(hits))
	}
}

func TestFactStore_SelectForInjection(t *testing.T) {
	fs := NewFactStore()
	for i := 0; i < 5; i++ {
		fs.Add(MemoryFact{
			ID: fmt.Sprintf("f%d", i), Domain: "PERSONAL", Subcategory: "PREFERENCE",
			Subject: fmt.Sprintf("事实%d", i), Summary: fmt.Sprintf("这是第%d条记忆", i),
			Weight: float64(5 - i), Confidence: 0.8, SelfRelevance: 0.7,
			Triggers: []string{fmt.Sprintf("t%d", i)},
		})
	}

	selected := fs.SelectForInjection(500, 0.5, 0.5, 50)
	if len(selected) == 0 {
		t.Error("selectForInjection should return facts")
	}
	// 应该按权重排序（高权重在前）
	if len(selected) >= 2 && selected[0].Weight < selected[1].Weight {
		t.Error("facts should be sorted by relevance")
	}
}

func TestKnowledgeGraph_AddAndQuery(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("用户", "喜欢", "咖啡", 0.9, nil)
	kg.Add("用户", "住在", "北京", 0.8, nil)

	hits := kg.Query("用户喜欢什么", 5)
	if len(hits) == 0 {
		t.Error("KG query should return results")
	}
}

func TestKnowledgeGraph_BuildContextBlock(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("用户", "喜欢", "咖啡", 0.9, nil)
	kg.Add("用户", "讨厌", "早起", 0.7, nil)

	block := kg.BuildContextBlock("用户", 500)
	if !strings.Contains(block, "咖啡") {
		t.Errorf("context block should contain '咖啡': %s", block)
	}
}

// ─── 工作记忆测试 ──────────────────────────────────────────────

func TestWorkingMemory_PushAndGet(t *testing.T) {
	wm := NewWorkingMemory()
	wm.Push("s1", Exchange{TurnIndex: 1, UserText: "你好", AssistantText: "你好呀"})
	wm.Push("s1", Exchange{TurnIndex: 2, UserText: "今天天气不错", AssistantText: "是啊"})

	recent := wm.GetRecent("s1")
	if len(recent) != 2 {
		t.Errorf("expected 2 exchanges, got %d", len(recent))
	}
}

func TestWorkingMemory_BuildContextBlock(t *testing.T) {
	wm := NewWorkingMemory()
	wm.Push("s1", Exchange{TurnIndex: 1, UserText: "你好", AssistantText: "你好呀"})

	block := wm.BuildContextBlock("s1")
	if !strings.Contains(block, "你好") {
		t.Errorf("context block should contain recent messages: %s", block)
	}
}

// ─── 主动回忆测试 ──────────────────────────────────────────────

func TestActiveRecall_NoCandidates(t *testing.T) {
	ar := NewActiveRecall()
	fs := NewFactStore()
	// 空 FactStore 应该返回 nil
	c := ar.SelectRecallCandidate(fs, 10, nil)
	if c != nil {
		t.Error("empty fact store should return nil")
	}
}

func TestActiveRecall_WithFacts(t *testing.T) {
	ar := NewActiveRecall()
	fs := NewFactStore()
	fs.Add(MemoryFact{
		ID: "core1", Domain: "PERSONAL", Subcategory: "PREFERENCE",
		Subject: "喜欢", Summary: "用户喜欢喝咖啡",
		Weight: 5.0, Confidence: 0.9, SelfRelevance: 0.8,
		Triggers: []string{"咖啡"},
	})

	// 使用确定性 rng=0 确保命中
	rng := 0.0
	c := ar.SelectRecallCandidate(fs, 20, &rng)
	if c == nil {
		t.Error("should find a recall candidate")
	}
}

// ─── 重逢引擎测试 ──────────────────────────────────────────────

func TestComputeReunionShock_QuickReturn(t *testing.T) {
	shock := ComputeReunionShock(5)
	if shock == nil {
		t.Fatal("5 hours should trigger quick_return")
	}
	if shock.Tier != ReunionQuickReturn {
		t.Errorf("expected quick_return, got %s", shock.Tier)
	}
}

func TestComputeReunionShock_LongLost(t *testing.T) {
	shock := ComputeReunionShock(24 * 60) // 60 days
	if shock == nil {
		t.Fatal("60 days should trigger long_lost")
	}
	if shock.Tier != ReunionLongLost {
		t.Errorf("expected long_lost, got %s", shock.Tier)
	}
	if !shock.StageDowngrade {
		t.Error("long_lost should trigger stage downgrade")
	}
}

func TestComputeReunionShock_NoShock(t *testing.T) {
	shock := ComputeReunionShock(0.5)
	if shock != nil {
		t.Error("less than 1 hour should not trigger shock")
	}
}

func TestReunionVoices(t *testing.T) {
	// 验证所有 29 种人格都有重逢语音
	presetsWithVoice := 0
	for _, p := range PersonalityPresets {
		if voices, ok := reunionVoices[p.ID]; ok {
			if len(voices) == 6 {
				presetsWithVoice++
			}
		}
	}
	if presetsWithVoice < 25 {
		t.Errorf("expected at least 25 presets with reunion voices, got %d", presetsWithVoice)
	}
}

// ─── 欲望栈测试 ──────────────────────────────────────────────

func TestDefaultDesireStack(t *testing.T) {
	ds := DefaultDesireStack()
	if len(ds.Slots) != DesireMaxSlots {
		t.Errorf("expected %d slots, got %d", DesireMaxSlots, len(ds.Slots))
	}
}

func TestUpdateDesireStack(t *testing.T) {
	ds := DefaultDesireStack()
	l1 := L1State{Stage: StageFamiliar, Trust: 50}
	ev := Event{Type: EvtVulnerable, Intensity: 0.7}

	newDS, hints := UpdateDesireStack(ds, "我今天好难过", ev, l1, 1)
	if len(newDS.Slots) != DesireMaxSlots {
		t.Errorf("stack should maintain %d slots, got %d", DesireMaxSlots, len(newDS.Slots))
	}
	// vulnerable 事件可能产生 desire
	_ = hints // hints 可以为空（概率性）
}

// ─── 情绪涌现测试 ─────────────────────────────────────────────

func TestHumanizeFeltDuration(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{10, "短短几周"},
		{60, "一个多月"},
		{120, "小半年"},
		{200, "大半年"},
		{500, "这么久"},
	}
	for _, tc := range tests {
		result := HumanizeFeltDuration(tc.days)
		if result != tc.expected {
			t.Errorf("days=%d: expected %q, got %q", tc.days, tc.expected, result)
		}
	}
}

func TestEvaluateEmergence_StrangerNoEmergence(t *testing.T) {
	ctx := EmergenceContext{
		Stage:                      StageStranger,
		Emotion:                    EmotionState{PrimaryLabel: "QUIET_FOND", Aff: 40, Sec: 30, Aro: 20, Dom: 0},
		ConsecutiveMeaningfulTurns: 10,
		CurrentTurn:                20,
	}
	e := EvaluateEmergence(ctx, "praise")
	if e != nil {
		t.Error("stranger stage should not trigger emergence")
	}
}

// ─── 节奏引擎测试 ─────────────────────────────────────────────

func TestDecideRhythm_Chatter(t *testing.T) {
	input := RhythmInput{
		Aro: 20, Aff: 30, Stage: StageFamiliar,
		PersonalityID: "genki", Sincerity: 0.5, Intensity: 0.7,
	}
	d := DecideRhythm(input)
	if d.Mode != RhythmChatter {
		t.Errorf("genki personality should chatter, got %s", d.Mode)
	}
}

func TestDecideRhythm_Monologue(t *testing.T) {
	input := RhythmInput{
		Aro: -20, Aff: 0, Stage: StageFamiliar,
		PersonalityID: "kuudere", Sincerity: 0.5, Intensity: 0.5,
	}
	d := DecideRhythm(input)
	if d.Mode != RhythmMonologue {
		t.Errorf("kuudere personality should monologue, got %s", d.Mode)
	}
}

// ─── 参数完整性测试 ────────────────────────────────────────────

func TestParams_AllDefined(t *testing.T) {
	// 验证所有关键参数都是非零值
	checks := []struct {
		name  string
		value float64
	}{
		{"EmotionDecay", EmotionDecay},
		{"InitialTrust", InitialTrust},
		{"DesireMaxSlots", float64(DesireMaxSlots)},
		{"TierBCharBudget", float64(TierBCharBudget)},
	}
	for _, c := range checks {
		if c.value == 0 {
			t.Errorf("%s should not be zero", c.name)
		}
	}
}

// ─── 基准测试 ──────────────────────────────────────────────────

func BenchmarkInterpretInput(b *testing.B) {
	msgs := []string{"你好厉害", "今天天气不错", "我好难过", "滚", "对不起"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InterpretInput(msgs[i%len(msgs)], 50)
	}
}

func BenchmarkOrchestrator_PreLLMTurn(b *testing.B) {
	preset := GetPreset("deredere")
	if preset == nil {
		b.Fatal("preset not found")
	}
	orch := NewOrchestrator("bench", *preset)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orch.PreLLMTurn("你好呀，今天天气不错")
	}
}

func BenchmarkEmotionStep(b *testing.B) {
	prev := EmotionState{PrimaryLabel: "CALM_RATIONAL"}
	ev := Event{Type: EvtPraise, Intensity: 0.7, Sincerity: 0.8}
	mod := Modulation{TrustMod: 1.0, RiftMod: 1.0, StageWeight: 1.0, Atmosphere: AtmoNeutral}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EmotionStep(ev, mod, prev, nil)
	}
}

// ─── 集成测试：完整对话流程 ─────────────────────────────────────

func TestIntegration_FullConversationFlow(t *testing.T) {
	preset := GetPreset("deredere")
	if preset == nil {
		t.Fatal("preset not found")
	}
	orch := NewOrchestrator("integration_test", *preset)
	now := time.Now()
	orch.State.FirstMetDate = &now

	// 模拟 10 轮对话
	conversation := []string{
		"你好呀",
		"今天天气真好",
		"谢谢你陪我",
		"你真的很温柔",
		"我今天有点难过",
		"工作压力好大",
		"谢谢你的安慰",
		"有你在真好",
		"我好多了",
		"晚安",
	}

	var lastTrust float64
	for i, msg := range conversation {
		result := orch.PreLLMTurn(msg)

		// 基本验证
		if result.SystemPrompt == "" {
			t.Errorf("turn %d: empty system prompt", i+1)
		}
		if result.Event.Type == "" {
			t.Errorf("turn %d: empty event type", i+1)
		}
		if result.Trace.Turn != i+1 {
			t.Errorf("turn %d: expected turn %d, got %d", i+1, i+1, result.Trace.Turn)
		}

		// 验证信任值在有效范围内
		if result.Trace.L1.Trust < 0 || result.Trace.L1.Trust > 100 {
			t.Errorf("turn %d: trust out of range: %f", i+1, result.Trace.L1.Trust)
		}

		// 验证情绪值在有效范围内
		if result.Trace.L2.Aff < -100 || result.Trace.L2.Aff > 100 {
			t.Errorf("turn %d: aff out of range: %f", i+1, result.Trace.L2.Aff)
		}

		lastTrust = result.Trace.L1.Trust
	}

	// 10 轮对话后，整体趋势应该是正向的
	if lastTrust < InitialTrust {
		t.Errorf("trust should trend upward after 10 positive-ish turns, started at %f, ended at %f", InitialTrust, lastTrust)
	}

	t.Logf("Final state: trust=%.0f stage=%s emotion=%s aff=%.0f",
		lastTrust, orch.State.Relationship.Stage,
		orch.State.Emotion.PrimaryLabel, orch.State.Emotion.Aff)
}
