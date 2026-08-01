// Package whisper — memory_retrieve.go
// 100% 对齐 ackem memory/retriever.ts
// 多路记忆检索器：触发词 + 相关性评分 + KG + 情节 + 关联扩散

package whisper

import (
	"math"
	"sort"
	"strings"
	"time"
)

// ── 类型别名 ──
// ─── MemoryRetriever ──────────────────────────────────────────

// sfPair 事实+得分对
type sfPair struct {
	f *Fact
	s float64
}

// MemoryRetriever 多路记忆检索器
type MemoryRetriever struct {
	FactStore  *FactStore
	KG         *KnowledgeGraph
	Episodes   *EpisodicStore
	AssocIndex *AssociationIndex
}

// NewRetriever 创建检索器
func NewRetriever(fs *FactStore, kg *KnowledgeGraph) *MemoryRetriever {
	return &MemoryRetriever{FactStore: fs, KG: kg}
}

// ─── Retrieve ─────────────────────────────────────────────────

// Retrieve 多路检索主入口
func (mr *MemoryRetriever) Retrieve(
	query string,
	hint RelevanceHint,
	budgetChars int,
	currentValence float64,
	currentAff float64,
	temporalCtx *TemporalContext,
	sessionID string,
	adultMode bool,
) RetrievalResult {
	result := RetrievalResult{}

	// 1. 过滤可见事实
	facts := mr.FactStore.PrivacyFilter(adultMode)

	// 2. 触发词匹配（boost 1.5x）
	triggered := mr.FactStore.SearchByTriggers(query)
	triggerIDs := make(map[string]bool)
	for _, f := range triggered {
		triggerIDs[f.ID] = true
	}

	// 3. 相关性评分排序
	var ranked []sfPair
	now := time.Now()
	for _, f := range facts {
		baseScore := ScoreRelevance(f, now, currentValence, currentAff)
		if triggerIDs[f.ID] {
			baseScore *= 1.5
		}
		if temporalCtx != nil {
			baseScore *= ComputeTemporalBoost(*temporalCtx)
		}
		ranked = append(ranked, sfPair{f, baseScore})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })

	// 4. Budget 管理：组装 tierBBlock
	var tierBParts []string
	chars := 0
	for _, sf := range ranked {
		block := len([]rune(sf.f.Subject)) + len([]rune(sf.f.Summary)) + 40
		if chars+block > budgetChars {
			break
		}
		summary := sf.f.Summary
		if len([]rune(summary)) > 120 {
			summary = string([]rune(summary)[:120])
		}
		tierBParts = append(tierBParts, "· "+sf.f.Subject+"："+summary)
		chars += block
		result.FactsUsed++
	}
	if len(tierBParts) > 0 {
		result.TierBBlock = "【你记得关于ta的事】\n" + strings.Join(tierBParts, "\n")
	}

	// 5. 知识图谱检索
	if mr.KG != nil {
		kgBlock := mr.KG.BuildContextBlock(query, KGCharBudget)
		if kgBlock != "" {
			if result.TierBBlock != "" {
				result.TierBBlock += "\n\n" + kgBlock
			} else {
				result.TierBBlock = kgBlock
			}
		}
	}

	// 6. 情节记忆检索
	if mr.Episodes != nil {
		eps := mr.Episodes.Search(query, 3)
		result.EpisodesUsed = len(eps)
		if len(eps) > 0 {
			var epLines []string
			epLines = append(epLines, "【相关记忆片段】")
			for _, ep := range eps {
				epLines = append(epLines, "· "+truncStr(ep.Summary, 100))
			}
			if result.TierBBlock != "" {
				result.TierBBlock += "\n\n" + strings.Join(epLines, "\n")
			}
		}
	}

	// 7. 关联扩散
	if mr.AssocIndex != nil {
		top := ranked
		if len(top) > 10 {
			top = top[:10]
		}
		for _, sf := range top {
			assocs := mr.AssocIndex.GetAssociations(sf.f.ID)
			for _, a := range assocs {
				result.ActivatedAssocIDs = append(result.ActivatedAssocIDs, a.ID)
				result.AssociationActivations++
			}
		}
	}

	// 8. 记忆回声
	result.MemoryEcho = ComputeMemoryEchoFacts(ranked, currentAff)

	result.SharedCount = mr.FactStore.Count()
	return result
}

// ─── ComputeMemoryEchoFacts ───────────────────────────────────

// ComputeMemoryEchoFacts 从检索结果计算记忆回声（对齐 ackem computeMemoryEcho）
func ComputeMemoryEchoFacts(ranked []sfPair, aff float64) MemoryEcho {
	if len(ranked) == 0 {
		return MemoryEcho{}
	}
	var sw, affSum, secSum, aroSum, domSum float64
	top := ranked
	if len(top) > 5 {
		top = top[:5]
	}
	for _, sf := range top {
		w := sf.f.Weight * sf.f.SelfRelevance
		if sf.f.EmotionalContext != nil {
			// 有情感上下文时使用真实值
			ec := sf.f.EmotionalContext
			intensity := ec.Intensity
			trust := ec.Trust
			decay := math.Exp(-0.003 * time.Since(sf.f.CreatedAt).Hours() / 24)
			w *= intensity * decay
			if w <= 0 {
				continue
			}
			sw += w
			affSum += ec.Valence * w * MemoryEchoAffWeight
			if trust > 50 {
				secSum += MemoryEchoSecPositive * w
			} else {
				secSum += MemoryEchoSecNegative * w
			}
			aroSum += intensity * w * MemoryEchoAroIntensityWeight
			// dom 基于 trust 信号
			domSignal := (trust - 50) / 50 * MemoryEchoDomTrustWeight
			domSum += domSignal * w
		} else {
			// 回退：无情感上下文时使用简单估算
			w2 := w * 0.3
			if w2 <= 0 {
				continue
			}
			sw += w2
			affSum += 1.0 * w2 * MemoryEchoAffWeight
			if sf.f.Confidence > 0.5 {
				secSum += MemoryEchoSecPositive * w2
			} else {
				secSum += MemoryEchoSecNegative * w2
			}
			aroSum += 1.0 * w2 * MemoryEchoAroIntensityWeight
			domSum += 0.1 * w2 * MemoryEchoDomTrustWeight
		}
	}
	if sw <= 0 {
		return MemoryEcho{}
	}
	return MemoryEcho{
		Aff: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, affSum/sw)),
		Sec: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, secSum/sw)),
		Aro: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, aroSum/sw)),
		Dom: math.Max(-MemoryEchoCap, math.Min(MemoryEchoCap, domSum/sw)),
	}
}
