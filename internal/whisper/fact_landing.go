// Package whisper — fact_landing.go
// 100% 对齐 ackem memory/factLanding.ts
// 统一事实落地：写入+时间锚点+KG三元组+名字降级+事实去重

package whisper

// WriteFactRowsInput 事实落地输入
type WriteFactRowsInput struct {
	SessionID         string
	TurnIndex         int
	UserMsg           string
	Rows              []ExtractedFact
	L1                L1State
	L2                EmotionState
	FS                *FactStore
	KG                *KnowledgeGraph
	AdultPrivacyLevel string
}

// WriteFactRowsResult 事实落地结果
type WriteFactRowsResult struct {
	NewFacts   []*Fact
	NewFactIDs []string
}

// WriteFactRows 统一事实落地管线
func WriteFactRows(input WriteFactRowsInput) WriteFactRowsResult {
	rows := input.Rows
	if len(rows) == 0 {
		return WriteFactRowsResult{}
	}

	emo := CaptureEmotionalContext(input.L1, input.L2)
	var newFacts []*Fact
	var newFactIDs []string

	for _, row := range rows {
		// 自动触发词提取
		autoTriggers := ExtractTriggers(row.Subject, row.Summary)
		mergedTriggers := mergeUnique(row.Triggers, autoTriggers)

		// 名字事实降级：新姓名/昵称写入时，旧姓名降级
		if row.Subcategory == "BASIC_PROFILE" && (row.Subject == "用户姓名" || row.Subject == "用户昵称") {
			downgradeNameFacts(input.FS, row.Subject)
		}

		// 权重和置信度归一化
		weight := row.Weight
		if weight <= 0 {
			if meta, ok := CategoryMetaMap[row.Subcategory]; ok {
				weight = meta.DefaultWeight
			} else {
				weight = 1
			}
		}
		confidence := row.Confidence
		if confidence <= 0 {
			if meta, ok := CategoryMetaMap[row.Subcategory]; ok {
				confidence = meta.DefaultConfidence
			} else {
				confidence = 0.7
			}
		}

		// 写入事实
		fact := input.FS.Add(MemoryFact{
			Domain:           row.Domain,
			Subcategory:      row.Subcategory,
			Subject:          row.Subject,
			Summary:          row.Summary,
			Weight:           weight,
			Confidence:       confidence,
			SelfRelevance:    row.SelfRelevance,
			Triggers:         mergedTriggers,
			SourceSessionID:  input.SessionID,
			SourceTurnIndex:  input.TurnIndex,
			EmotionalContext: &emo,
		})

		if fact != nil && fact.ID != "" {
			// KG 三元组提取
			if input.KG != nil {
				triples := ExtractTriples(row.Subject, row.Summary, fact.ID, row.Subcategory, "")
				for _, t := range triples {
					input.KG.AddTriple(attachEmotion(Triple{
						Subject: t.Subject, Predicate: t.Predicate, Object: t.Object,
						Confidence: t.Confidence, SourceFactIDs: t.SourceFactIDs,
					}, &emo))
				}
				// v4.9 因果维度：导入事实同样提取因果三元组（与 ingest 一致）
				for _, t := range extractCausalTriples(fact) {
					input.KG.AddTriple(t)
				}
			}

			newFacts = append(newFacts, fact)
			newFactIDs = append(newFactIDs, fact.ID)
		}
	}

	return WriteFactRowsResult{NewFacts: newFacts, NewFactIDs: newFactIDs}
}

func downgradeNameFacts(fs *FactStore, newSubject string) {
	for _, f := range fs.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户姓名" || f.Subject == "用户昵称") &&
			f.Subject != newSubject {
			// 降级权重，让新名字优先
			f.Weight = clampF(f.Weight*0.5, 0, 3)
		}
	}
}

func mergeUnique(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
