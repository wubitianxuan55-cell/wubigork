// Package whisper — strategy_topic.go
// 100% 对齐 ackem engine/strategy/topicSelector.ts
// 话题仲裁：涌现/欲望/特殊日期 之间的优先级选择

package whisper

// TopicCandidate 话题候选
type TopicCandidate struct {
	Source    string // emergence/desire/special_date/casual
	Topic     string
	Priority  float64 // 越高越优先
	Injection string  // 注入文本
}

// ResolveTopicSelection 话题仲裁：选择最高优先级的话题
func ResolveTopicSelection(
	emergence *EmergenceState,
	desireHints []string,
	specialDateLabel string,
	stage RelationshipStage,
) *TopicCandidate {
	var candidates []TopicCandidate

	// 特殊日期最高优先级
	if specialDateLabel != "" {
		candidates = append(candidates, TopicCandidate{
			Source:    "special_date",
			Topic:     specialDateLabel,
			Priority:  0.9,
			Injection: "\n\n【特别的日子】今天是" + specialDateLabel + "。自然地提到这一点。",
		})
	}

	// 情绪涌现次优先
	if emergence != nil && !emergence.HasExpressed {
		candidates = append(candidates, TopicCandidate{
			Source:    "emergence",
			Topic:     "心里涌起的感觉",
			Priority:  0.7,
			Injection: "\n\n【情绪涌动】你心里涌起一种感觉，想和ta分享。自然地表达出来。",
		})
	}

	// 欲望提示
	for _, h := range desireHints {
		candidates = append(candidates, TopicCandidate{
			Source:    "desire",
			Topic:     h,
			Priority:  0.5,
			Injection: "\n\n【想做的事】" + h + "（自然地融入对话）",
		})
	}

	// 陌生人阶段不仲裁
	if stage == StageStranger && len(candidates) == 0 {
		return nil
	}

	if len(candidates) == 0 {
		return nil
	}

	// 选最高优先级
	best := &candidates[0]
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Priority > best.Priority {
			best = &candidates[i]
		}
	}

	if best.Source == "casual" {
		return nil
	}
	return best
}

// ShouldArbitrateTopic 是否需要话题仲裁
func ShouldArbitrateTopic(stage RelationshipStage, emergence *EmergenceState, desireHints []string) bool {
	if emergence != nil && !emergence.HasExpressed {
		return true
	}
	if len(desireHints) > 0 && stage != StageStranger {
		return true
	}
	return false
}

// FormatSelectedTopicInjection 格式化选中话题的注入文本
func FormatSelectedTopicInjection(candidate *TopicCandidate) string {
	if candidate == nil {
		return ""
	}
	return candidate.Injection
}
