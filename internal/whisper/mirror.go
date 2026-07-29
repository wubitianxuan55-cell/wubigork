// Package whisper — mirror.go
// 100% 对齐 ackem engine/mirror.ts
// 镜中自省：从自我描述文本中检测矛盾断言

package whisper

import "strings"

// ─── MirrorAssertion ──────────────────────────────────────────

// MirrorAssertion 镜中断言
type MirrorAssertion struct {
	Text    string  `json:"text"`
	Subject string  `json:"subject"` // 我/ta/我们
	Valence float64 `json:"valence"` // -1(负面) ~ 1(正面)
	Topic   string  `json:"topic"`
}

// Contradiction 矛盾检测结果
type Contradiction struct {
	Old         MirrorAssertion `json:"old"`
	New         MirrorAssertion `json:"new"`
	Topic       string          `json:"topic"`
	Description string          `json:"description"`
}

// ─── 断言抽取 ──────────────────────────────────────────────────

var mirrorPositiveWords = []string{"喜欢", "开心", "重要", "珍惜", "温柔", "幸运", "幸福", "美好", "懂", "理解", "陪伴", "爱"}
var mirrorNegativeWords = []string{"讨厌", "难过", "不好", "失败", "没用", "不配", "害怕", "担心", "孤独", "离开", "失去"}
var mirrorTopicWords = []string{"陪伴", "聊天", "理解", "帮助", "性格", "感情", "工作", "生活", "自己", "关系", "沉默", "回应", "关心", "未来"}

// ExtractAssertions 从文本中抽取自我断言
func ExtractAssertions(text string) []MirrorAssertion {
	var out []MirrorAssertion
	lines := strings.FieldsFunc(text, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n'
	})
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if len([]rune(t)) < 4 {
			continue
		}
		if !strings.Contains(t, "我") {
			continue
		}
		v := estimateMirrorValence(t)
		topic := extractMirrorTopic(t)
		subject := "我"
		if strings.Contains(t, "我们") {
			subject = "我们"
		} else if strings.Contains(t, "ta") || strings.Contains(t, "他") || strings.Contains(t, "她") {
			subject = "ta"
		}
		out = append(out, MirrorAssertion{Text: t, Subject: subject, Valence: v, Topic: topic})
	}
	return out
}

func estimateMirrorValence(text string) float64 {
	var s float64
	for _, w := range mirrorPositiveWords {
		if strings.Contains(text, w) {
			s += 0.4
		}
	}
	for _, w := range mirrorNegativeWords {
		if strings.Contains(text, w) {
			s -= 0.5
		}
	}
	return clampF(s, -1, 1)
}

func extractMirrorTopic(text string) string {
	for _, t := range mirrorTopicWords {
		if strings.Contains(text, t) {
			return t
		}
	}
	return "自我"
}

// ─── 矛盾检测 ──────────────────────────────────────────────────

// DetectContradictions 检测新旧断言间的矛盾
func DetectContradictions(oldAsserts, newAsserts []MirrorAssertion) []Contradiction {
	var out []Contradiction

	// 精确话题匹配（快速路径）
	for _, na := range newAsserts {
		for _, oa := range oldAsserts {
			if oa.Topic != na.Topic {
				continue
			}
			if mathAbs(oa.Valence-na.Valence) >= 0.6 {
				oldPreview := oa.Text
				if len([]rune(oldPreview)) > 30 {
					oldPreview = string([]rune(oldPreview)[:30]) + "…"
				}
				newPreview := na.Text
				if len([]rune(newPreview)) > 30 {
					newPreview = string([]rune(newPreview)[:30]) + "…"
				}
				out = append(out, Contradiction{
					Old: oa, New: na, Topic: na.Topic,
					Description: "关于「" + na.Topic + "」，之前觉得「" + oldPreview + "」但现在认为「" + newPreview + "」",
				})
			}
		}
	}
	return out
}
