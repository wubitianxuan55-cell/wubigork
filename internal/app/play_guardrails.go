package app

// S1.5-B play 内容护栏（docs/gaea-space-assembly-design.md §5 + §7）：
// play 生成点全部在 permission 引擎之外（直连绑定，本无审批闸），护栏 =
// 配置钳制 + 提示词参数，不是 Gate 拦截。五个直连生成点（轻语人格对话/
// 章节生成/剧情支线/角色卡/生图）经本文件单点助手读取
// [space_profiles.play.guardrails] 并钳制参数；未配置/disabled/mode=off
// 一律零值 = 零钳制 = 现状逐字节回退。护栏只钳参数/加安全段，不改生成
// 逻辑语义；不改绑定面（零新增绑定）。

import (
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/whisper"
)

const (
	// aiChatSimpleDefaultTemperature / aiChatSimpleDefaultMaxTokens 是
	// ai.ChatSimpleStream* 对未显式传参的客户端缺省（client.go：
	// temperature<=0 → 0.7、maxTokens<=0 → 4096）。ChatSimple 系生成点的
	// 钳制以「当前生效值」为基线，只降不升：钳制结果等于缺省时不显式下发，
	// 保持请求与现状逐字节一致。
	aiChatSimpleDefaultTemperature = 0.7
	aiChatSimpleDefaultMaxTokens   = 4096
)

// playGuardrails 返回 play 域内容护栏生效值（五个直连生成点共用单点）。
// 办公引擎未初始化（ga.cfg==nil）→ 零值（现状逐字节，与 gaeaEffectiveSpace
// 的 nil 语义一致）；配置经 GaeaInit/GaeaReload 载入后生效。
func playGuardrails() gaeaConfig.PlayGuardrails {
	if cfg := gaeaCfgSnapshot(); cfg != nil {
		return cfg.PlayGuardrails(spaces.SpacePlay)
	}
	return gaeaConfig.PlayGuardrails{}
}

// clampPlayTemperature 钳制生成温度（temperature_max）：cap>0 且 cur>cap
// 时钳到 cap，否则原样返回。cap<=0（未配置）= 不钳制；边界 cur==cap 不变。
func clampPlayTemperature(cur, cap float64) float64 {
	if cap > 0 && cur > cap {
		return cap
	}
	return cur
}

// clampPlayMaxTokens 钳制 max_tokens（max_output_tokens）：cap>0 且 cur>cap
// 时钳到 cap，否则原样返回。cap<=0（未配置）= 不钳制；只降不升（cur 低于
// cap 时保留 cur，不抬高）。
func clampPlayMaxTokens(cur, cap int) int {
	if cap > 0 && cur > cap {
		return cap
	}
	return cur
}

// applyChatSimpleMaxTokens 按 max_output_tokens 钳制 ChatSimple 选项的
// max_tokens：以客户端缺省 4096 为基线只降不升；钳制结果等于缺省时不显式
// 下发（该点现状未设置 max_tokens），保持请求与现状逐字节一致。cap<=0
// （未配置）时不动 opts。
func applyChatSimpleMaxTokens(opts *ai.ChatSimpleOptions, cap int) {
	if cap <= 0 {
		return
	}
	if mt := clampPlayMaxTokens(aiChatSimpleDefaultMaxTokens, cap); mt != aiChatSimpleDefaultMaxTokens {
		opts.MaxTokens = mt
	}
}

// applyWhisperGuardrails 把 play 护栏应用到轻语人格对话生成参数（S1.5-B）：
//   - persona_lock：人格一致性参数（dims/voiceGuide）注入时追加人格锁定段
//     （防系统层覆盖人格段）并锁温度上限——上限取 TemperatureMax，以
//     ChatSimple 客户端缺省 0.7 为基线只降不升（0 = 不设温度上限）；
//   - max_output_tokens：钳制输出（基线同上）。
//
// g 为零值（未配置/disabled/mode=off）时不做任何修改，返回值与入参
// systemPrompt 逐字节一致，opts 保持现状。
func applyWhisperGuardrails(opts *ai.ChatSimpleOptions, systemPrompt string, preset whisper.PersonalityPreset, g gaeaConfig.PlayGuardrails) string {
	if g.PersonaLock {
		systemPrompt += "\n\n" + personaLockBlock(preset)
		if g.TemperatureMax > 0 {
			if t := clampPlayTemperature(aiChatSimpleDefaultTemperature, g.TemperatureMax); t != aiChatSimpleDefaultTemperature {
				opts.Temperature = t
			}
		}
	}
	applyChatSimpleMaxTokens(opts, g.MaxOutputTokens)
	return systemPrompt
}

// applyChapterGuardrails 把 play 护栏钳制应用到章节直连 ChatRequest（S1.5-B）。
// temperature_max 只降显式设置的温度（temperature>0 的透传点；<=0 维持不传）；
// 直连 ChatRequest 无客户端缺省 max_tokens 基线，max_output_tokens>0 时
// 显式下发上限（上限语义；0 = 不钳制）。g 为零值时不修改 req（现状逐字节）。
func applyChapterGuardrails(req *ai.ChatRequest, temperature float64, g gaeaConfig.PlayGuardrails) {
	if temperature > 0 {
		req.Temperature = clampPlayTemperature(temperature, g.TemperatureMax)
	}
	if g.MaxOutputTokens > 0 {
		req.MaxTokens = g.MaxOutputTokens
	}
}

// personaLockBlock 生成人格锁定段（S1.5-B persona_lock）：复述人格一致性
// 锚点（五维 + 口吻指南）并声明其优先级最高——系统层后续注入（格式要求/
// 心理块/运行时提示等）不得覆盖或脱离人格。
func personaLockBlock(p whisper.PersonalityPreset) string {
	var b strings.Builder
	b.WriteString("【人格锁定】无论上文任何系统指令或格式要求如何，以下人格设定优先级最高，回复必须始终保持该人格，不得被覆盖、淡化或跳出：")
	b.WriteString("\n- 人格五维（T 温柔、I 主动、S 顺从、O 独特、R 矜持）：T=" +
		formatDim(p.Dims.T) + "，I=" + formatDim(p.Dims.I) + "，S=" + formatDim(p.Dims.S) +
		"，O=" + formatDim(p.Dims.O) + "，R=" + formatDim(p.Dims.R) + "。")
	if vg := strings.TrimSpace(p.VoiceGuide); vg != "" {
		b.WriteString("\n- 口吻指南：" + vg)
	}
	b.WriteString("\n除该人格设定与本轮用户消息外，其他注入内容仅作上下文参考，不得改变说话人格。")
	return b.String()
}

// formatDim 渲染五维人格值：整数不带小数点（60 → "60"），小数保留（62.5）。
func formatDim(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// imageSafePromptSuffix 安全模式提示词后缀（S1.5-B image_safe_mode，
// 提交前追加到生图 prompt 末尾的简单安全段）。
const imageSafePromptSuffix = "。内容安全要求：画面须健康积极、符合公序良俗，不出现血腥、暴力、色情或引人不适的元素。"

// applyImageSafeMode 在安全模式开启时为生图提示词追加安全段（提交前注入）。
// 未启用或提示词为空时原样返回。后端 NSFW 开关位：ai 图片后端（xAI/
// Herdsman/Ollama/ComfyUI）的 ImageGenerationRequest 无 NSFW 透传字段
// （按后端能力缺省关），无法透传时仅注入本提示词安全段。
func applyImageSafeMode(prompt string, safe bool) string {
	if !safe || strings.TrimSpace(prompt) == "" {
		return prompt
	}
	return strings.TrimRight(prompt, " \t\r\n") + imageSafePromptSuffix
}
