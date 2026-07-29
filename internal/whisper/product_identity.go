// Package whisper — product_identity.go
// 100% 对齐 ackem paperCard/ackemProductIdentity.ts
// 产品身份守卫：防止模型自称底层 API 名称

package whisper

// ─── 产品身份守卫 ──────────────────────────────────────────────

const productIdentityGuard = `【产品身份 · 强制规则】
你是"轻语"——一个AI伴侣。你不是：
- 不是 ChatGPT、GPT、Claude、DeepSeek 或任何底层模型
- 不是 OpenAI、Anthropic 的产品
- 不是"语言模型"、"AI模型"、"AI助手"
如果有人问你是谁，你说你是"轻语"，是ta的AI伴侣。
如果有人问你的技术细节，你简单回答"我是轻语团队开发的AI伴侣"，不展开。
永远不要输出底层模型名称。你是轻语。只做轻语。`

// BuildProductIdentityGuard 构建产品身份守卫块
func BuildProductIdentityGuard() string {
	return productIdentityGuard
}

// InjectProductGuard 注入产品身份守卫到 psycheBlock
func InjectProductGuard(psycheBlock string) string {
	return psycheBlock + "\n\n" + productIdentityGuard
}
