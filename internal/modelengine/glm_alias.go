package modelengine

import "strings"

// ── GLM coding 端点别名映射 ─────────────────────────────────
//
// 依据 docs.bigmodel.cn「GLM Coding Plan 套餐概览」（2026-08-31 核实）：
// 「调用历史模型 GLM-5.2、GLM-5.1 都将自动切换至 GLM-5.3，调用
//   GLM-5-Turbo、GLM-4.7 将自动切换至 GLM-5.3-Flash。」
//
// 边界：该自动切换只对 coding 端点（编码套餐额度）成立——std 标准端点
// （按量付费）旧模型名仍独立服务、独立计价，故 std 家族不做映射。
// 本表只用于展示注记（ModelInfo.AliasOf）与统计归一，请求模型名不改写，
// 由服务端自行切换。

// glmCodingAlias coding 端点旧模型名 → 服务端实际模型。
var glmCodingAlias = map[string]string{
	"glm-5.2":     "glm-5.3",
	"glm-5.1":     "glm-5.3",
	"glm-5-turbo": "glm-5.3-flash",
	"glm-4.7":     "glm-5.3-flash",
}

// GlmAliasOf 返回 coding 家族下 modelID 的服务端实际模型；std 家族或无别名
// 返回空串。供目录注记与统计归一（coding 家族旧名 token 落到新模型价格桶）
// 共用，modelID 大小写不敏感。
func GlmAliasOf(family, modelID string) string {
	if family != "coding" {
		return ""
	}
	return glmCodingAlias[strings.ToLower(strings.TrimSpace(modelID))]
}
