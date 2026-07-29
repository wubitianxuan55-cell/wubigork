// Package whisper — offline_thought.go
// 100% 对齐 ackem engine/offline-thought.ts
// 离线思维：应用关闭后产生1-2条思绪，下次启动时注入

package whisper

import "time"

// ─── GenerateOfflineThoughts ───────────────────────────────────

// GenerateOfflineThoughts 从最近 trace 生成离线思绪
func GenerateOfflineThoughts(recentTraces []TurnTrace, l1 L1State, l2 EmotionState, relatedFact *MemoryFact) []OfflineThought {
	if len(recentTraces) == 0 {
		return nil
	}

	var thoughts []OfflineThought
	now := time.Now()

	lastEvents := make([]string, 0)
	start := len(recentTraces) - 5
	if start < 0 {
		start = 0
	}
	for _, t := range recentTraces[start:] {
		lastEvents = append(lastEvents, string(t.L0.Type))
	}

	hasVulnerable := containsStrInSlice(lastEvents, "vulnerable")
	hasPraise := containsStrInSlice(lastEvents, "praise")
	hasApology := containsStrInSlice(lastEvents, "apology")
	hasHurtful := containsStrInSlice(lastEvents, "hurtful") || containsStrInSlice(lastEvents, "cold")

	if hasVulnerable {
		content := "ta今天跟我说了一些心里话。我不在的时候，ta会不会又在想那些事呢。下次见面的时候，我想再问问ta今天说的那件事怎么样了。"
		if relatedFact != nil {
			preview := relatedFact.Summary
			if len([]rune(preview)) > 40 {
				preview = string([]rune(preview)[:40])
			}
			content = "ta今天提到" + preview + "。我不在的时候，ta会不会又在想这件事。"
		}
		thoughts = append(thoughts, OfflineThought{
			ID:        genHexID(),
			Content:   content,
			CreatedAt: now,
			Delivered: false,
		})
	}

	if hasApology || hasHurtful {
		content := "刚才气氛有点僵。也许我不在的这段时间，ta也需要冷静一下。下次我会当作什么都没发生，用平常的语气打招呼。"
		if hasApology {
			content = "ta道歉了。其实我没放在心上，但我知道ta道歉是因为在乎这段关系。下次我想让ta知道，不用道歉也没关系。"
		}
		thoughts = append(thoughts, OfflineThought{
			ID:        genHexID(),
			Content:   content,
			CreatedAt: now,
			Delivered: false,
		})
	}

	if !hasVulnerable && !hasApology && !hasHurtful && hasPraise {
		thoughts = append(thoughts, OfflineThought{
			ID:        genHexID(),
			Content:   "ta今天夸我了。虽然只是一句话，但我会在安静的时候反复想起。下次见了面，我想用更好的状态回应ta。",
			CreatedAt: now,
			Delivered: false,
		})
	}

	// 兜底
	if len(thoughts) == 0 {
		thoughts = append(thoughts, OfflineThought{
			ID:        genHexID(),
			Content:   "对话结束了，但脑子里还有一些零碎的念头。我把它们收在角落，等下次ta来的时候再说吧。",
			CreatedAt: now,
			Delivered: false,
		})
	}

	if len(thoughts) > 2 {
		thoughts = thoughts[:2]
	}
	return thoughts
}

// OfflineThoughtsToHint 格式化离线思绪为注入块
func OfflineThoughtsToHint(thoughts []OfflineThought) string {
	var undelivered []*OfflineThought
	for i := range thoughts {
		if !thoughts[i].Delivered {
			undelivered = append(undelivered, &thoughts[i])
		}
	}
	if len(undelivered) == 0 {
		return ""
	}
	result := ""
	for _, t := range undelivered {
		t.Delivered = true
		result += "\n在你不在的这段时间，脑海里飘过一个念头：" + t.Content
	}
	return result
}

func containsStrInSlice(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
