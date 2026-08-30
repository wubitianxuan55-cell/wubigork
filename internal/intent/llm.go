// llm.go — intent 内核 LLM 兜底分类器的受控输出解析（v4.8）。
//
// 规则引擎（intent.go）未命中时，App 层可用一个轻量 LLM 调用做兜底分类；
// 本文件只放纯类型与校验（宁漏勿误的最后一道闸），LLM 调用与装配在
// internal/app/intent_llm.go。包保持零外部依赖（仅标准库）。
package intent

import (
	"encoding/json"
	"strings"
)

// FallbackMinConfidence LLM 兜底的置信门：低于此值按未命中处理（走聊天管道）。
const FallbackMinConfidence = 0.75

// fallbackAllowedActions v1 放行面：低风险且可校验的动作。
// generate_image 不放行（误触发 = 废图 + GPU 成本，规则引擎 reImage 已是
// 高精度锚定）；reminder 不放行（需与时间解析联动，reReminder 已是宽匹配位）。
var fallbackAllowedActions = map[Action]bool{
	ActionNavigate:   true,
	ActionStatus:     true,
	ActionReadScreen: true,
}

// ParseFallback 解码并校验 LLM 分类输出；任何不合规（坏 JSON / 白名单外动作 /
// 置信不足 / navigate 缺 target）返回 nil——调用方按未命中走聊天管道。
// 输入容忍 ```json 围栏与前后噪音（取首个 '{' 到最后一个 '}'）。
func ParseFallback(raw string) *Intent {
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end <= start {
		return nil
	}
	var r struct {
		Action     string  `json:"action"`
		Target     string  `json:"target"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &r); err != nil {
		return nil
	}
	act := Action(strings.TrimSpace(r.Action))
	if !fallbackAllowedActions[act] {
		return nil
	}
	if r.Confidence < FallbackMinConfidence {
		return nil
	}
	target := strings.TrimSpace(r.Target)
	if act == ActionNavigate && target == "" {
		return nil
	}
	return &Intent{Action: act, Target: target}
}
