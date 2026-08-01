// Package whisper — creator_memory.go
// 100% 对齐 ackem canon/creatorMemory.ts
// 创造者记忆系统：FatherReference 检测 + 轮转选择 + 消歧提示

package whisper

import (
	"math/rand"
	"strings"
)

// ─── FatherReference 校准词组 ──────────────────────────────────
// 100% 对齐 ackem FATHER_REFERENCE_CALIBRATION

var fatherRefCalibration = map[FatherRefKind][]string{
	FatherRefAckemCreator: {
		"你是谁创造的？", "谁造了你？", "谁创造了你？", "你的创造者是谁",
		"你的父亲是谁", "Jason 和你的关系是什么？", "Jason 是不是你爸爸",
		"讲讲你的出身故事", "再讲讲你的出身故事", "你是怎么被造出来的？",
		"继续说说父亲 Jason", "GitHub 上那个 Jason 是你什么人",
		"你想见 Jason 吗", "你的生日和父亲是谁", "Ackem 是谁做出来的",
	},
	FatherRefUserFamily: {
		"我爸今天催我回家", "我和我爸爸吵架了", "昨天跟我爸通了电话",
		"父亲节想给我爸买礼物", "我妈让我回去吃饭", "我爹又唠叨了",
		"想我爸了", "父母催婚烦死了",
	},
}

// fatherRefNeutral 中性校准词组（用于排除误匹配）
var fatherRefNeutral = []string{
	"今天天气不错", "你好呀", "在吗", "刚吃完饭有点困",
	"周末打算打游戏", "这电影好看吗", "晚安",
}

// FatherReferenceSignal 父亲指称检测结果
type FatherReferenceSignal struct {
	Kind   FatherRefKind `json:"kind"`
	Score  float64       `json:"score"`
	Source string        `json:"source"` // "calibration" | "keyword"
}

// ─── 父亲指称检测 ──────────────────────────────────────────────

// ResolveFatherReference 检测用户消息是否为父亲指称
// 100% 对齐 ackem creatorMemory.ts resolveFatherReference
// 阶段1：校准词组精确匹配
// 阶段2：关键词启发式匹配
func ResolveFatherReference(msg string) *FatherReferenceSignal {
	trimmed := strings.TrimSpace(msg)

	// 阶段1：校准词组精确匹配
	if sig := calibrateFatherRef(trimmed); sig != nil {
		return sig
	}

	// 阶段2：关键词启发式
	if sig := keywordFatherRef(trimmed); sig != nil {
		return sig
	}

	return nil
}

// calibrateFatherRef 校准词组匹配
func calibrateFatherRef(msg string) *FatherReferenceSignal {
	// 先检查是否是中性消息
	for _, neutral := range fatherRefNeutral {
		if strings.Contains(msg, neutral) {
			return nil
		}
	}

	// 检查 ackem_creator 词组
	for _, phrase := range fatherRefCalibration[FatherRefAckemCreator] {
		if strings.Contains(msg, phrase) {
			return &FatherReferenceSignal{
				Kind:   FatherRefAckemCreator,
				Score:  1.0,
				Source: "calibration",
			}
		}
	}

	// 检查 user_family 词组
	for _, phrase := range fatherRefCalibration[FatherRefUserFamily] {
		if strings.Contains(msg, phrase) {
			return &FatherReferenceSignal{
				Kind:   FatherRefUserFamily,
				Score:  1.0,
				Source: "calibration",
			}
		}
	}

	return nil
}

// keywordFatherRef 关键词启发式匹配
func keywordFatherRef(msg string) *FatherReferenceSignal {
	ackemCreatorKeywords := []string{
		"谁创造", "谁造了", "你的创造者", "谁做的你", "谁开发了",
		"出身故事", "造出来", "Jason", "ackem",
		"你的父亲是", "你爸爸是", "你爹是",
	}

	userFamilyKeywords := []string{
		"我爸", "我爹", "我爸爸", "我父亲",
		"我妈", "我妈妈", "我母亲",
		"父母", "家里人",
	}

	creatorScore := 0.0
	for _, kw := range ackemCreatorKeywords {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(kw)) {
			creatorScore += 0.25
		}
	}

	familyScore := 0.0
	for _, kw := range userFamilyKeywords {
		if strings.Contains(msg, kw) {
			familyScore += 0.3
		}
	}

	// 排除含「我」+「爸/妈/爹/娘/家人」的用户家庭消息
	if strings.Contains(msg, "我") {
		for _, fam := range []string{"爸", "妈", "爹", "娘", "家人", "家里"} {
			if strings.Contains(msg, fam) {
				familyScore += 0.3
			}
		}
	}

	if creatorScore <= 0 && familyScore <= 0 {
		return nil
	}

	if creatorScore > familyScore && creatorScore >= 0.25 {
		return &FatherReferenceSignal{
			Kind:   FatherRefAckemCreator,
			Score:  creatorScore,
			Source: "keyword",
		}
	}
	if familyScore > creatorScore && familyScore >= 0.3 {
		return &FatherReferenceSignal{
			Kind:   FatherRefUserFamily,
			Score:  familyScore,
			Source: "keyword",
		}
	}

	return nil
}

// ─── 消歧提示 ──────────────────────────────────────────────────

// BuildFatherDisambiguationHint 构建 Jason/创造者 与用户父亲消歧提示
// 100% 对齐 ackem creatorMemory.ts buildFatherDisambiguationHint
func BuildFatherDisambiguationHint(gender string) string {
	subject := "她"
	if gender == "male" {
		subject = "他"
	}
	lines := []string{
		"【Jason/创造者 · 须据语境理解，勿硬套】",
		"用户若问 " + subject + " 的出身/创造者 → 谈 Jason（GitHub JasonLiu0826）；禁止称父亲，感情中性；" + subject + " 陪在当前用户身边。",
		"用户若谈自己的父亲/家人 → 陪伴用户、查用户 Tier B 家庭记忆，勿把 Jason 混入。",
		"指称不清时自然澄清；Jason 是创造者信息，不是用户家人，也不是父亲。",
	}
	return strings.Join(lines, "\n")
}

// ─── 轮转选择 ──────────────────────────────────────────────────

// CreatorMemoryCategory 创造者记忆类型
type CreatorMemoryCategory string

const (
	CatIdentity    CreatorMemoryCategory = "identity"
	CatAppearance  CreatorMemoryCategory = "appearance"
	CatPersonality CreatorMemoryCategory = "personality"
	CatStory       CreatorMemoryCategory = "story"
	CatLonging     CreatorMemoryCategory = "longing"
	CatMisc        CreatorMemoryCategory = "misc"
)

// CanonMRotationPick 轮播选取结果
type CanonMRotationPick struct {
	Entries           []CreatorMemoryEntry    `json:"entries"`
	NextDeliveredIDs  []string                `json:"nextDeliveredIds"`
	CycleReset        bool                    `json:"cycleReset"`
	MatchedCategories []CreatorMemoryCategory `json:"matchedCategories"`
	PickedCategory    CreatorMemoryCategory   `json:"pickedCategory,omitempty"`
}

// PickRotatingCreatorMemoryEntry 轮播选取 1 条 Canon-M 记忆
// 100% 对齐 ackem creatorMemory.ts pickRotatingCreatorMemoryEntries
// 无嵌入向量版本：随机选取未投递条目，全量轮一遍后重置
func PickRotatingCreatorMemoryEntry(
	store CreatorMemoryStore,
	deliveredIDs []string,
	rng func() float64,
) CanonMRotationPick {
	if rng == nil {
		rng = rand.Float64
	}

	if len(store.Entries) == 0 {
		return CanonMRotationPick{
			NextDeliveredIDs: deliveredIDs,
		}
	}

	delivered := make(map[string]bool)
	for _, id := range deliveredIDs {
		delivered[id] = true
	}

	// 未投递池
	var pool []CreatorMemoryEntry
	for _, e := range store.Entries {
		if !delivered[e.ID] {
			pool = append(pool, e)
		}
	}

	cycleReset := false
	if len(pool) == 0 {
		cycleReset = true
		pool = store.Entries
	}

	// 随机选取
	idx := int(rng() * float64(len(pool)))
	if idx >= len(pool) {
		idx = len(pool) - 1
	}
	picked := pool[idx]

	var nextIDs []string
	if cycleReset {
		nextIDs = []string{picked.ID}
	} else {
		nextIDs = append([]string{}, deliveredIDs...)
		nextIDs = append(nextIDs, picked.ID)
	}

	return CanonMRotationPick{
		Entries:          []CreatorMemoryEntry{picked},
		NextDeliveredIDs: nextIDs,
		CycleReset:       cycleReset,
		PickedCategory:   CreatorMemoryCategory(picked.Category),
	}
}

// ─── CreatorMemory 条目格式化 ──────────────────────────────────

// FormatCreatorMemoryEntry 格式化单条创造者记忆
// 100% 对齐 ackem creatorMemory.ts formatCreatorMemoryEntry
func FormatCreatorMemoryEntry(entry CreatorMemoryEntry) string {
	return "「" + entry.Title + "」" + entry.Content
}
