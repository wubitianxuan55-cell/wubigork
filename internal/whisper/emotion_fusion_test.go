package whisper

import (
	"strings"
	"testing"
)

// ─── T6-5.1: emotion_fusion.go 全函数覆盖 ─────────────────────────

func TestEmotionFusion_ToDisplay(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{100, 100}, {-100, 0}, {0, 50}, {80, 90}, {-20, 40},
		{50.3, 75}, {49.6, 75}, {99, 100}, {-99, 1},
	}
	for _, c := range cases {
		if got := toDisplay(c.in); got != c.want {
			t.Errorf("toDisplay(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestEmotionFusion_GetIntensityLevelAllLevels(t *testing.T) {
	cases := []struct {
		aff  int
		want string
	}{
		{0, "低"}, {49, "低"}, {50, "中"}, {69, "中"},
		{70, "高"}, {89, "高"}, {90, "极高"}, {100, "极高"},
	}
	for _, c := range cases {
		if got := getIntensityLevel(c.aff); got != c.want {
			t.Errorf("getIntensityLevel(%d) = %q, want %q", c.aff, got, c.want)
		}
	}
}

func TestEmotionFusion_DescribeInnerFeeling(t *testing.T) {
	if got := describeInnerFeeling("SWEET_ATTACHMENT"); got == "" || !strings.Contains(got, "靠近") {
		t.Errorf("SWEET_ATTACHMENT 应有靠近描述, got %q", got)
	}
	if got := describeInnerFeeling("COLD_DETACHED"); got == "" {
		t.Error("COLD_DETACHED 应有描述")
	}
	if got := describeInnerFeeling("NO_SUCH_LABEL"); got != "正常状态" {
		t.Errorf("未知标签应回退「正常状态」, got %q", got)
	}
}

func TestEmotionFusion_GetEmotionTendency(t *testing.T) {
	if got := getEmotionTendency("TSUNDERE"); !strings.Contains(got, "嘴硬") {
		t.Errorf("TSUNDERE 倾向应含「嘴硬」, got %q", got)
	}
	if got := getEmotionTendency("ANGRY_ATTACK"); !strings.Contains(got, "攻击性") {
		t.Errorf("ANGRY_ATTACK 倾向应含「攻击性」, got %q", got)
	}
	if got := getEmotionTendency("UNKNOWN_LABEL"); got != "平稳、正常" {
		t.Errorf("未知标签倾向应回退「平稳、正常」, got %q", got)
	}
}

func TestEmotionFusion_GetEmotionMaxLengthKnown(t *testing.T) {
	cases := map[string]int{
		"SWEET_ATTACHMENT": 60, "SHY_HEARTBEAT": 30, "TSUNDERE": 30,
		"HURT_GRIEVANCE": 40, "ANGRY_ATTACK": 30, "COLD_DETACHED": 15,
		"FEARFUL_OBEDIENT": 30, "QUIET_FOND": 30, "CALM_RATIONAL": 60,
	}
	for label, want := range cases {
		if got := getEmotionMaxLength(label); got != want {
			t.Errorf("getEmotionMaxLength(%s) = %d, want %d", label, got, want)
		}
	}
	if got := getEmotionMaxLength("NOPE"); got != 60 {
		t.Errorf("未知标签默认长度应为 60, got %d", got)
	}
}

func TestEmotionFusion_GenerateFusionStrategy(t *testing.T) {
	p := PersonalityTemplate{
		Label:             "傲娇",
		CoreContradiction: "表面冷淡内心在意",
		SpeakingStyle:     "嘴硬但藏不住关心",
	}
	got := generateFusionStrategy(p, "TSUNDERE")
	for _, want := range []string{"傲娇", "【傲娇】", "表面冷淡内心在意", "嘴硬但藏不住关心", "嘴硬、否定、但藏不住关心"} {
		if !strings.Contains(got, want) {
			t.Errorf("融合策略缺少 %q: %q", want, got)
		}
	}
	got2 := generateFusionStrategy(p, "WEIRD_LABEL")
	if !strings.Contains(got2, "WEIRD_LABEL") {
		t.Errorf("未知标签应原样出现: %q", got2)
	}
}

func TestEmotionFusion_BuildReactionOpenerInstruction(t *testing.T) {
	got := buildReactionOpenerInstruction("CALM_RATIONAL")
	if !strings.Contains(got, "开头短反应") || !strings.Contains(got, "推荐「") {
		t.Errorf("指令应含推荐词结构, got %q", got)
	}
	if got := buildReactionOpenerInstruction("NO_POOL_LABEL"); got != "" {
		t.Errorf("无词库标签应返回空, got %q", got)
	}
}

func TestEmotionFusion_GetImperfectionHint(t *testing.T) {
	got := getImperfectionHint("SHY_HEARTBEAT")
	if !strings.Contains(got, "15%") || !strings.Contains(got, "省略号") {
		t.Errorf("SHY_HEARTBEAT 应提示 15%% 省略号, got %q", got)
	}
	if got := getImperfectionHint("COLD_DETACHED"); got != "" {
		t.Errorf("0 概率标签应返回空, got %q", got)
	}
	if got := getImperfectionHint("UNKNOWN"); got != "" {
		t.Errorf("未知标签应返回空, got %q", got)
	}
}

func TestEmotionFusion_GetEmotionProhibitions(t *testing.T) {
	got := getEmotionProhibitions("SWEET_ATTACHMENT")
	if len(got) == 0 {
		t.Fatal("SWEET_ATTACHMENT 应有禁止清单")
	}
	if got := getEmotionProhibitions("NOPE"); got != nil {
		t.Errorf("未知标签应返回 nil, got %v", got)
	}
}

func TestEmotionFusion_MergeProhibitions(t *testing.T) {
	got := mergeProhibitions([]string{"a", "b"}, []string{"b", "c"}, false)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("合并去重结果错误: %v", got)
	}
	got = mergeProhibitions([]string{"道歉", "示弱", "别哭", "正常项"}, []string{"x"}, true)
	for _, item := range got {
		if strings.Contains(item, "道歉") || strings.Contains(item, "示弱") || strings.Contains(item, "哭") {
			t.Errorf("道歉场景不应保留禁止项 %q: %v", item, got)
		}
	}
	if len(got) != 2 || got[0] != "正常项" || got[1] != "x" {
		t.Errorf("道歉过滤后应剩 [正常项 x], got %v", got)
	}
	ten := make([]string, 10)
	for i := range ten {
		ten[i] = string(rune('a' + i))
	}
	got = mergeProhibitions(ten, nil, false)
	if len(got) != 8 {
		t.Errorf("合并后应截断到 8 条, got %d", len(got))
	}
}

func TestEmotionFusion_SelectExamples(t *testing.T) {
	p := PersonalityTemplate{
		ExamplesHigh:   []string{"h1", "h2"},
		ExamplesMedium: []string{"m1", "m2"},
		ExamplesLow:    []string{"l1", "l2"},
	}
	if got := selectExamples(p, 100, 5); len(got) != 2 || got[0] != "h1" {
		t.Errorf("高亲密度应选 ExamplesHigh: %v", got)
	}
	if got := selectExamples(p, 0, 5); len(got) != 2 || got[0] != "m1" {
		t.Errorf("中亲密度应选 ExamplesMedium: %v", got)
	}
	if got := selectExamples(p, -100, 5); len(got) != 2 || got[0] != "l1" {
		t.Errorf("低亲密度应选 ExamplesLow: %v", got)
	}
	emptyHigh := PersonalityTemplate{ExamplesMedium: []string{"m1"}}
	if got := selectExamples(emptyHigh, 100, 5); len(got) != 1 || got[0] != "m1" {
		t.Errorf("High 为空应回退 Medium: %v", got)
	}
	if got := selectExamples(p, 100, 1); len(got) != 1 {
		t.Errorf("maxExamples 应截断, got %v", got)
	}
}

func TestEmotionFusion_BuildPrioritySectionFusion(t *testing.T) {
	got := buildPrioritySectionFusion()
	for _, want := range []string{"行为优先级", "人格核心设定", "禁止清单", "安全覆写"} {
		if !strings.Contains(got, want) {
			t.Errorf("优先级区块缺少 %q", want)
		}
	}
}

func TestEmotionFusion_BuildPersonalitySectionFusion(t *testing.T) {
	p := PersonalityTemplate{
		Label: "傲娇", CoreContradiction: "口是心非",
		SpeechPatterns: []string{"哼", "才不是"}, SpeakingStyle: "简短",
	}
	got := buildPersonalitySectionFusion(p)
	if !strings.Contains(got, "傲娇") || !strings.Contains(got, "口是心非") || !strings.Contains(got, "简短") {
		t.Errorf("人格区块缺少关键内容: %q", got)
	}
	if !strings.Contains(got, "哼") || !strings.Contains(got, "才不是") {
		t.Errorf("人格区块应含语癖: %q", got)
	}
}

func TestEmotionFusion_BuildEmotionSectionFusion(t *testing.T) {
	got := buildEmotionSectionFusion("SWEET_ATTACHMENT", 100, 50, -50, 0, "极高", "想靠近")
	for _, want := range []string{"甜蜜依恋", "极高", "亲密感 100/100", "安全感 75/100", "唤醒度 25/100", "支配度 50/100", "想靠近"} {
		if !strings.Contains(got, want) {
			t.Errorf("情绪区块缺少 %q: %q", want, got)
		}
	}
}

func TestEmotionFusion_BuildFusionSectionFusion(t *testing.T) {
	got := buildFusionSectionFusion("策略文本")
	if !strings.Contains(got, "策略文本") || !strings.Contains(got, "融合执行策略") {
		t.Errorf("融合策略区块结构错误: %q", got)
	}
}

func TestEmotionFusion_BuildProhibitionSectionFusion(t *testing.T) {
	got := buildProhibitionSectionFusion([]string{"直球表白", "大段话"})
	if !strings.Contains(got, "绝对禁止清单") || !strings.Contains(got, "× 直球表白") || !strings.Contains(got, "× 大段话") {
		t.Errorf("禁止清单区块错误: %q", got)
	}
	empty := buildProhibitionSectionFusion(nil)
	if !strings.Contains(empty, "绝对禁止清单") {
		t.Errorf("空清单也应输出区块头: %q", empty)
	}
}

func TestEmotionFusion_BuildExampleSectionFusion(t *testing.T) {
	got := buildExampleSectionFusion([]string{"例一", "例二"})
	if !strings.Contains(got, "参考示例") || !strings.Contains(got, "· 例一") || !strings.Contains(got, "· 例二") {
		t.Errorf("示例区块错误: %q", got)
	}
}

func TestEmotionFusion_BuildCharacterStateBlock_Full(t *testing.T) {
	p := PersonalityTemplate{
		Label: "傲娇", CoreContradiction: "口是心非",
		SpeechPatterns: []string{"哼"}, SpeakingStyle: "简短",
		Prohibitions: []string{"人格禁止A"},
		ExamplesHigh: []string{"ex1"}, ExamplesMedium: []string{"ex2"}, ExamplesLow: []string{"ex3"},
	}
	emotion := EmotionStateFusion{Aff: 90, Sec: 60, Aro: 30, Dom: 0, PrimaryLabel: "TSUNDERE"}
	got := BuildCharacterStateBlock(p, emotion, false, "normal")
	for _, want := range []string{
		"行为优先级", "人格基底", "傲娇", "动态情绪", "【傲娇】", "情绪强度：极高",
		"融合执行策略", "绝对禁止清单", "人格禁止A", "参考示例", "开头短反应",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("完整角色块缺少 %q", want)
		}
	}
	if !strings.Contains(got, "× 直球甜腻") {
		t.Errorf("应包含情绪禁止项: %q", got)
	}
}

func TestEmotionFusion_BuildCharacterStateBlock_TerseMirror(t *testing.T) {
	p := PersonalityTemplate{
		Label: "平静理性", CoreContradiction: "无", SpeakingStyle: "平稳",
		Prohibitions: []string{"禁止项"},
		ExamplesMedium: []string{"m1"},
	}
	emotion := EmotionStateFusion{Aff: 0, Sec: 0, Aro: 0, Dom: 0, PrimaryLabel: "CALM_RATIONAL"}
	got := BuildCharacterStateBlock(p, emotion, false, "terse")
	if !strings.Contains(got, "回复上限30字") {
		t.Errorf("terse 模式应含回复上限提示: %q", got)
	}
}

func TestEmotionFusion_BuildCharacterStateBlock_ApologyFilters(t *testing.T) {
	p := PersonalityTemplate{
		Label: "傲娇", CoreContradiction: "口是心非", SpeakingStyle: "简短",
		Prohibitions: []string{"禁止道歉", "示弱", "常规禁止"},
		ExamplesMedium: []string{"m1"},
	}
	emotion := EmotionStateFusion{Aff: 50, Sec: 50, Aro: 50, Dom: 50, PrimaryLabel: "ANGRY_ATTACK"}
	got := BuildCharacterStateBlock(p, emotion, true, "normal")
	if strings.Contains(got, "× 禁止道歉") || strings.Contains(got, "× 示弱") || strings.Contains(got, "× 委婉道歉") {
		t.Errorf("道歉场景不应输出道歉/示弱禁止: %q", got)
	}
	if !strings.Contains(got, "× 常规禁止") {
		t.Errorf("非道歉/示弱类禁止应保留: %q", got)
	}
}

func TestEmotionFusion_OpenerPoolExhaustedFallback(t *testing.T) {
	// 最近使用词覆盖整个词库 → fresh 为空 → 回退整个 pool
	globalOpenerState.mu.Lock()
	globalOpenerState.recent = []string{"好的", "是的", "对", "嗯", "行", "可以"}
	globalOpenerState.mu.Unlock()
	got := buildReactionOpenerInstruction("CALM_RATIONAL")
	if !strings.Contains(got, "推荐「") {
		t.Errorf("词库耗尽应回退整个词库: %q", got)
	}
	// 清理共享状态
	globalOpenerState.mu.Lock()
	globalOpenerState.recent = nil
	globalOpenerState.mu.Unlock()
}

func TestEmotionFusion_BuildEmotionSectionFusionUnknownLabel(t *testing.T) {
	got := buildEmotionSectionFusion("WEIRD_LABEL", 0, 0, 0, 0, "低", "感受")
	if !strings.Contains(got, "WEIRD_LABEL") {
		t.Errorf("未知标签应原样显示: %q", got)
	}
}
