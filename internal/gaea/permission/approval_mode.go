package permission

import "strings"

// 审批模式归一（codex 蒸馏 T4，Never 语义）：审批模式为 never（绝不弹窗）
// 时，命中「需批准」的策略不能静默执行也不能静默跳过——硬拒绝并把理由回喂
// 模型，让无人值守的回合能换法子收敛，而不是卡死或误放行。

// NormalizeAsk 按审批模式把 Ask 归一到最终决策。
// mode=never → Deny 并返回理由；mode="" / "on-request" / "untrusted" /
// 其他 → 保持 Ask（现状语义，逐字节不变）。理由文案可区分来源（含 never 字样），
// 与用户主动拒绝在工具结果文本上可分辨。
func NormalizeAsk(mode string, tool string) (Decision, string) {
	if strings.ToLower(strings.TrimSpace(mode)) == "never" {
		return Deny, "approval required by policy (tool " + tool +
			"), but approval mode is never — do not retry this call; choose another approach or finish your turn and explain"
	}
	return Ask, ""
}
