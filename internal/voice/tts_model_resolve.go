// Package voice — Herdsman TTS 默认模型动态解析（H0-3）
//
// 背景（实测）：本机 herdsman 已装模型里，qwen3-tts-customvoice /
// qwen3-tts-voicedesign / qwen3-tts-voiceclone 三个目录均为 0MB（未安装，
// 调用会返回 "model ... is not installed"）；edge-tts 服务端返回空音频；
// 本地唯一可用 TTS 是 voxcpm2。而代码里默认值硬编码为 qwen3-tts-customvoice，
// 导致默认语音路径必然先失败再回退。这里提供一个纯函数，在配置值不可用时
// 按优先级从「已安装模型」列表中动态选出实际可用的模型。
package voice

import "strings"

// defaultHerdsmanTTSModel 默认 Herdsman TTS 模型（与 voice_config.go 中
// DefaultVoiceConfig / Validate 的默认值保持一致）。
const defaultHerdsmanTTSModel = "qwen3-tts-customvoice"

// herdsmanTTSPriority Herdsman TTS 模型回退优先级（voxcpm2 必须第一：
// 本机实测唯一可用本地 TTS；其余按可用性从高到低排列）。
var herdsmanTTSPriority = []string{
	"voxcpm2",
	"qwen3-tts-customvoice",
	"qwen3-tts-voicedesign",
	"qwen3-tts-voiceclone",
	"edge-tts",
	"cosyvoice",
}

// ResolveHerdsmanTTSModel 动态解析实际可用的 Herdsman TTS 模型。
//
// 规则：
//  1. configured 非空且 installed 包含它（大小写不敏感、双向包含匹配）→ 返回 configured；
//  2. 否则按 herdsmanTTSPriority 优先级选 installed 中第一个命中项（同样大小写不敏感、包含匹配）；
//  3. 未命中任何已装模型 → 回退 configured（configured 为空时回退默认 qwen3-tts-customvoice）。
//
// installed 为空/nil（拿不到已装列表）时不做解析，等价于原逻辑：直接返回 configured
// （空则默认值），且不标记回退。
//
// 返回值：
//   - model: 解析出的模型 ID
//   - usedFallback: 是否走了回退（结果与 configured 不一致，或 configured 为空走了默认值）
//   - resolvedFromInstalled: 结果是否来自已装列表的命中项（而非原样返回 configured）
func ResolveHerdsmanTTSModel(configured string, installed []string) (model string, usedFallback bool, resolvedFromInstalled bool) {
	raw := strings.TrimSpace(configured)
	cfg := raw
	if cfg == "" {
		cfg = defaultHerdsmanTTSModel
	}

	// 拿不到已装列表：无法解析，行为等价于原逻辑
	if len(installed) == 0 {
		return cfg, false, false
	}

	// 1. 配置值本身已装 → 直接用配置值
	if containsModelMatch(installed, cfg) {
		return cfg, false, false
	}

	// 2. 按优先级选已装列表中第一个命中项（返回已装列表中的真实 ID）
	for _, candidate := range herdsmanTTSPriority {
		if id := findInstalledMatch(installed, candidate); id != "" {
			return id, true, true
		}
	}

	// 3. 全部未命中 → 回退 configured（空则默认值）
	return cfg, true, false
}

// containsModelMatch 判断 installed 中是否有条目与 target 匹配
// （大小写不敏感、双向包含匹配，覆盖大小写/前后缀差异）。
func containsModelMatch(installed []string, target string) bool {
	for _, id := range installed {
		if modelIDMatch(id, target) {
			return true
		}
	}
	return false
}

// findInstalledMatch 返回 installed 中第一个与 target 匹配的条目（原样返回，含空白修剪）。
func findInstalledMatch(installed []string, target string) string {
	for _, id := range installed {
		if modelIDMatch(id, target) {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

// modelIDMatch 两个模型 ID 是否匹配：大小写不敏感，且一方包含另一方即视为匹配。
func modelIDMatch(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.Contains(a, b) || strings.Contains(b, a)
}
