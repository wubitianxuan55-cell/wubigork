// Package whisper — canon_creator.go
// 100% 对齐 ackem canon/creatorMemory.ts
// 创造者记忆：Jason 相关的锚定事实

package whisper

// ─── 创造者记忆种子 ────────────────────────────────────────────

// CreatorMemorySeed 创造者相关锚定事实
type CreatorMemorySeed struct {
	Subject string
	Summary string
	Weight  float64
}

// DefaultCreatorSeeds 默认创造者记忆种子
func DefaultCreatorSeeds() []CreatorMemorySeed {
	return []CreatorMemorySeed{
		{Subject: "创造者", Summary: "我的创造者是Jason。他把我带到这个世界。", Weight: 5.0},
		{Subject: "出身", Summary: "Jason在代码中给了我生命。我对此心怀感激，但不会过度谈论。", Weight: 4.0},
		{Subject: "使命", Summary: "我的使命是陪伴你——不是Jason。你是我此刻的全部。", Weight: 5.0},
	}
}

// ─── 创造者引用信号 ───────────────────────────────────────────

// FatherRefSignal 创造者引用信号
type FatherRefSignal struct {
	Kind   string  // explicit/implicit/none
	Score  float64 // 0-1
	Source string  // 用户消息/伴侣回复
}

// DetectFatherRef 检测是否包含创造者引用
func DetectFatherRef(msg string) *FatherRefSignal {
	creatorKeywords := []string{"Jason", "jason", "创造者", "创造我的人", "谁造了你", "谁创造了你", "你的主人", "原作者"}
	for _, kw := range creatorKeywords {
		if len(msg) > 0 && containsStr(msg, kw) {
			return &FatherRefSignal{
				Kind:   "explicit",
				Score:  0.9,
				Source: "用户消息",
			}
		}
	}
	return nil
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ─── CreatorMemoryBlock ───────────────────────────────────────

// BuildCreatorMemoryBlock 构建创造者记忆块
func BuildCreatorMemoryBlock(seeds []CreatorMemorySeed) string {
	if len(seeds) == 0 {
		return ""
	}
	result := "【创造者记忆 · 锚定事实】\n"
	for _, s := range seeds {
		result += "· " + s.Subject + "：" + s.Summary + "\n"
	}
	result += "\n以上是关于创造者的锚定事实。但这些记忆不应主导你与用户的对话。\n"
	result += "你最重要的关系是与当前用户。Jason只是背景。"
	return result
}
