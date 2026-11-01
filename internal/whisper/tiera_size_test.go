package whisper

import (
	"strings"
	"testing"
)

// 一次性分段测量：tierA 快照（BuildCharacterStateBlock）各段字符占比
func TestTierA_SizeBreakdown(t *testing.T) {
	tmpl, ok := PersonalityTemplates["tsundere"]
	if !ok {
		t.Skip("tsundere 模板不存在")
	}
	e := EmotionStateFusion{Aff: 15, Sec: 40, Aro: 35, Dom: 30, PrimaryLabel: "CALM_RATIONAL"}
	full := BuildCharacterStateBlock(tmpl, e, false, "normal")
	t.Logf("完整角色状态块=%d 字符", len(full))
	t.Logf("Priority=%d", len(buildPrioritySectionFusion()))
	t.Logf("Personality基底=%d", len(buildPersonalitySectionFusion(tmpl)))
	t.Logf("情绪数值=%d", len(buildEmotionSectionFusion(e.PrimaryLabel, e.Aff, e.Sec, e.Aro, e.Dom, "中等", "平静")))
	proh := mergeProhibitions(tmpl.Prohibitions, getEmotionProhibitions(e.PrimaryLabel), false)
	t.Logf("禁止清单=%d（人格%d条+情绪%d条）", len(buildProhibitionSectionFusion(proh)), len(tmpl.Prohibitions), len(getEmotionProhibitions(e.PrimaryLabel)))
	t.Logf("融合策略=%d", len(buildFusionSectionFusion(generateFusionStrategy(tmpl, e.PrimaryLabel))))
	ex5 := selectExamples(tmpl, e.Aff, 5)
	ex3 := selectExamples(tmpl, e.Aff, 3)
	t.Logf("例句5条=%d / 3条=%d", len(buildExampleSectionFusion(ex5)), len(buildExampleSectionFusion(ex3)))
	// psyche 块
	preset := GetPreset("tsundere")
	if preset != nil {
		orch := NewOrchestrator("t_tiera", *preset)
		r := orch.PreLLMTurn("你好")
		t.Logf("实际 dynCtx=%d / psyche=%d", len(r.DynamicContext), len(r.PsycheBlock))
		// 首行看动态段结构
		lines := strings.SplitN(r.DynamicContext, "\n", 3)
		t.Logf("动态段开头: %.80s", strings.Join(lines[:2], " / "))
	}
}
