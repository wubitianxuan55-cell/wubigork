package agent

// D1 输出预算（dsh-ops 蒸馏 2026-11-10 第二波）：信封序列化**前**对 payload
// 按「转义后字节」预算截断——旧口径序列化后按原文字节截断整个 JSON，超帽时
// 必劈坏 JSON（切在字符串/转义中间），模型看到残缺信封；且 JSON 转义放大
// （引号/换行 ×2、控制符 ×6）使实际进上下文的字节可超帽。截断后
// truncateToolOutput(env) 退化为永不触发的 backstop（劈坏 JSON 根修）。

import (
	"github.com/gaea/gaea/internal/gaea/agent/textutils"
)

// firstNonEmpty 取第一个非空串（截断 notice 合并用）。
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// envelopeReserve 是信封骨架+伴随字段的最坏预留：键名（ok/success/code/
// data/tool/result 或 detail）+tool 名转义+errMsg 首行+B2 三字段+截断标记
// 冗余。宁可多留 640B，不让 backstop 在正常路径触发。
const envelopeReserve = 640

// budgetPayloadForEnvelope 把 payload 截到「信封序列化后总长 ≤ 48KiB」。
// 返回 (payload, 截断 notice)——notice 非空时调用方须把 truncated 事实接回
// toolOutcome（信封不再触发 backstop，标志不能丢）。成功路径的 data.result
// 与失败路径的 data.detail 共用。
func budgetPayloadForEnvelope(payload string) (string, string) {
	cap := textutils.MaxToolOutputBytes - envelopeReserve
	if textutils.EscapeJSONLen(payload) <= cap {
		return payload, ""
	}
	return textutils.TruncateToEscapedBudget(payload, textutils.MaxToolResultLines, cap)
}
