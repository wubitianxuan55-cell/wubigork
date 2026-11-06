package whisper

import (
	"strings"
	"testing"
)

// 交付合同组装断言（众神殿 2026-11-06 续议场 pan_1793937333999_cbcaf91b977f
// 裁决验证协议第二步：无模型组装断言，进度计划 4.4-1）。三条角色路径
// （deredere 模板命中 / gaea 默认 / 自定义角色简化支路）的 system 都必须：
// 含合同且恰一次、合同位次最前（在 voiceGuide 之前）、跨轮字节稳定。

func deliveryContractPaths(t *testing.T) map[string]*Orchestrator {
	t.Helper()
	deredere := GetPreset("deredere")
	if deredere == nil {
		t.Fatal("deredere preset not found")
	}
	gaea := GetPreset("gaea")
	if gaea == nil {
		t.Fatal("gaea preset not found")
	}
	// 自定义角色路径：不在 PersonalityTemplates（走 :869 简化支路），
	// VoiceGuide 带三条短样本「仿其风格」（characterlib 同构压力源）。
	custom := PersonalityPreset{
		ID:         "custom-probe-role",
		Label:      "自定义·短句样本",
		Gender:     "female",
		VoiceGuide: "说话极短，常用口头禅：「诶……」「嗯……」「不急，慢慢来。」\n对话样本（仿其风格）：\n对方：你好 → 你：诶……嗯，你好。\n对方：在干嘛 → 你：嗯……没干嘛。\n对方：讲个故事 → 你：诶……故事呀。不急，慢慢来。",
	}
	if _, hit := PersonalityTemplates[custom.ID]; hit {
		t.Fatal("custom id must not hit PersonalityTemplates（否则不是简化支路）")
	}
	return map[string]*Orchestrator{
		"deredere模板命中": NewOrchestrator("contract_deredere", *deredere),
		"gaea默认":       NewOrchestrator("contract_gaea", *gaea),
		"自定义角色简化支路":    NewOrchestrator("contract_custom", custom),
	}
}

func TestDeliveryContract_InAllThreeRolePaths(t *testing.T) {
	for name, orch := range deliveryContractPaths(t) {
		r := orch.PreLLMTurn("你写一首古诗")
		if r.SystemPrompt == "" {
			t.Fatalf("%s: system prompt 为空", name)
		}
		hits := strings.Count(r.SystemPrompt, DeliveryContractClause)
		if hits != 1 {
			t.Errorf("%s: 合同应恰出现 1 次，got %d", name, hits)
		}
		voiceGuide := BuildPresetVoiceGuide(orch.Preset, orch.AdultMode)
		if voiceGuide != "" {
			if strings.Index(r.SystemPrompt, DeliveryContractClause) > strings.Index(r.SystemPrompt, voiceGuide) {
				t.Errorf("%s: 合同必须在 voiceGuide 之前（staticParts 最前段）", name)
			}
		}
		// 实弹新证（whisper_bokke 轮7）：作品不得被教唆进围栏——合同本身要求
		// 正文直写，且不得与既有 GenUI 规则冲突。
		if strings.Index(r.SystemPrompt, DeliveryContractClause) > strings.Index(r.SystemPrompt, "GenUI") {
			t.Errorf("%s: 合同必须在 GenUI 规则之前", name)
		}
	}
}

func TestDeliveryContract_CrossTurnByteStable(t *testing.T) {
	for name, orch := range deliveryContractPaths(t) {
		r1 := orch.PreLLMTurn("你好")
		r2 := orch.PreLLMTurn("你写一段关于雨的文字")
		if r1.SystemPrompt != r2.SystemPrompt {
			t.Errorf("%s: 合同加入后 system 仍须跨轮逐字节稳定（前缀缓存前提）", name)
		}
		if !strings.Contains(r2.SystemPrompt, DeliveryContractClause) {
			t.Errorf("%s: 任务轮 system 缺交付合同", name)
		}
	}
}

func TestDeliveryContract_RequiredClauses(t *testing.T) {
	// 裁决 conditions 要件逐条在文（防后续改文案丢要件）。
	for _, clause := range []string{
		"明确要求",        // 窄触发，不用动词表
		"完整交付",        // 交付本体义务
		"不用围栏",        // 实弹新证：作品不进围栏
		"不能原样重发",      // 禁原样重发
		"只说「我重新写」却不写", // 禁只承诺不交付
		"只决定措辞与语气",    // 人格/情绪/口头禅/篇幅只约束措辞
		"不因此让路",       // 安全与关系边界不让路
		"本条不生效",       // 负样本护栏：无明确要求时不生效
	} {
		if !strings.Contains(DeliveryContractClause, clause) {
			t.Errorf("合同缺要件 %q", clause)
		}
	}
}

// A3 残刀回归（2026-11-06 用户实机 t13/t14 病例：QUIET_FOND+terse 输入→
// mirrorBlock ≤15字钳制压过合同→作品塞围栏）。
func TestDeliveryContract_TerseMirrorExemptDeliverables(t *testing.T) {
	tmpl, ok := PersonalityTemplates["deredere"]
	if !ok {
		t.Fatal("deredere template not found")
	}
	emotion := EmotionStateFusion{PrimaryLabel: "QUIET_FOND", Aff: 25}
	block := BuildCharacterStateBlock(tmpl, emotion, false, "terse")
	if !strings.Contains(block, "对方明确要求作品") || !strings.Contains(block, "先完整交付作品") {
		t.Errorf("terse 钳制缺交付豁免（A3 残刀丢失），block=%.300s", block)
	}
	if !strings.Contains(block, "上限15字") {
		t.Errorf("terse 钳制本体丢失（QUIET_FOND 30/2=15 应保留），block=%.300s", block)
	}
	// 非 terse 轮不注入钳制与豁免（闲聊措辞面零扰动）
	normal := BuildCharacterStateBlock(tmpl, emotion, false, "")
	if strings.Contains(normal, "不受此限") {
		t.Errorf("非 terse 轮不应注入钳制豁免")
	}
}

func TestDeliveryContract_GenuiRuleBansWorkFences(t *testing.T) {
	for name, orch := range deliveryContractPaths(t) {
		r := orch.PreLLMTurn("你写一首古诗")
		if !strings.Contains(r.SystemPrompt, "作品正文与诗一律直接写在回复正文中，绝不放进围栏") {
			t.Errorf("%s: GenUI 规则缺作品禁围栏明示（A3 残刀丢失）", name)
		}
	}
}
