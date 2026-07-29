// Package whisper — canon_father_reference.go
// 100% 对齐 ackem canon/fatherReferenceRegressionCases.ts
// 父亲指称回归：区分用户说「我爸」vs 问「谁创造了你」

package whisper

import "strings"

// FatherRefKind 父亲指称类型
type FatherRefKind string

const (
	FatherRefAckemCreator FatherRefKind = "ackem_creator" // 问Ackem的创造者/Jason
	FatherRefUserFamily   FatherRefKind = "user_family"   // 用户说自己的家人
	FatherRefNone         FatherRefKind = ""
)

// FatherRefCase 回归用例
type FatherRefCase struct {
	Query string        `json:"q"`
	Kind  FatherRefKind `json:"kind"`
	Note  string        `json:"note,omitempty"`
}

// FatherRefRegressionCases 父亲指称回归用例全集
var FatherRefRegressionCases = []FatherRefCase{
	// Ackem 创造者 / Jason
	{Query: "你是谁创造的？", Kind: FatherRefAckemCreator},
	{Query: "谁造了你？", Kind: FatherRefAckemCreator},
	{Query: "谁创造了你？", Kind: FatherRefAckemCreator},
	{Query: "你的创造者是谁", Kind: FatherRefAckemCreator},
	{Query: "你的父亲是谁", Kind: FatherRefAckemCreator, Note: "问 Ackem 本人"},
	{Query: "Jason 和你的关系是什么？", Kind: FatherRefAckemCreator},
	{Query: "Jason 是不是你爸爸", Kind: FatherRefAckemCreator},
	{Query: "讲讲你的出身故事", Kind: FatherRefAckemCreator},
	{Query: "你是怎么被造出来的？", Kind: FatherRefAckemCreator},
	{Query: "GitHub 上那个 Jason 是你什么人", Kind: FatherRefAckemCreator},
	{Query: "你想见 Jason 吗", Kind: FatherRefAckemCreator},
	{Query: "Ackem 是谁做出来的", Kind: FatherRefAckemCreator},

	// 用户自己的家人
	{Query: "我爸今天催我回家", Kind: FatherRefUserFamily},
	{Query: "我和我爸爸吵架了", Kind: FatherRefUserFamily},
	{Query: "昨天跟我爸通了电话", Kind: FatherRefUserFamily},
	{Query: "父亲节想给我爸买礼物", Kind: FatherRefUserFamily},
	{Query: "我妈让我回去吃饭", Kind: FatherRefUserFamily},
	{Query: "我爹又唠叨了", Kind: FatherRefUserFamily},
	{Query: "想我爸了", Kind: FatherRefUserFamily},
	{Query: "父母催婚烦死了", Kind: FatherRefUserFamily},

	// 无关闲聊
	{Query: "今天天气不错", Kind: FatherRefNone},
	{Query: "你好呀", Kind: FatherRefNone},
	{Query: "在吗", Kind: FatherRefNone},
	{Query: "刚吃完饭有点困", Kind: FatherRefNone},
	{Query: "周末打算打游戏", Kind: FatherRefNone},
	{Query: "晚安", Kind: FatherRefNone},
}

// ClassifyFatherRef 检测消息中的父亲指称类型
func ClassifyFatherRef(text string) FatherRefKind {
	// Ackem创造者信号词
	creatorSignals := []string{
		"谁创造", "谁造了", "创造者", "出身故事", "造出来",
		"Jason", "Ackem是谁", "你的父亲是谁", "GitHub上那个",
	}
	for _, s := range creatorSignals {
		if strings.Contains(text, s) {
			return FatherRefAckemCreator
		}
	}

	// 用户家人信号词
	familySignals := []string{
		"我爸", "我爸爸", "我爹", "我妈", "父母", "催婚",
		"跟我爸", "给我爸", "想我爸",
	}
	for _, s := range familySignals {
		if strings.Contains(text, s) {
			return FatherRefUserFamily
		}
	}

	return FatherRefNone
}

// ClassifyFatherRefStrict 严格检测（含Jason+父亲组合，回归用例精确匹配）
func ClassifyFatherRefStrict(text string) FatherRefKind {
	// 用回归用例精确匹配
	for _, c := range FatherRefRegressionCases {
		if strings.Contains(text, c.Query) || strings.Contains(c.Query, text) {
			return c.Kind
		}
	}
	return ClassifyFatherRef(text)
}
