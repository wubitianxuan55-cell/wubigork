// Package novelgate 章节「写前契约 + 写后质量」的**确定性**检查（零 LLM、零网络）。
//
// 蒸馏来源：oh-story-claudecode（MIT）的 deslop-gates.md / quality-rubric.md 中
// **可机械判定**的维度（机制重推导，非代码搬运）；语义维度（卖点/动机/伏笔回收）
// 仍由 LLM 评审负责，见 .gaea/skills/novel-review/。
package novelgate

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/types"
)

// Issue 一条确定性发现（severity 对齐评审 rubric 的 S1-S4）。
type Issue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // S1 | S2 | S3 | S4
	Message  string `json:"message"`
	Evidence string `json:"evidence,omitempty"`
}

// 阈值（口径=runes，不是字节）。
const (
	shortSentenceRunes = 5    // 「短句」上界
	shortRatioLimit    = 0.40 // 短句占比告警线（且句数 ≥ minSentences）
	minSentences       = 20
	avgSentenceMin     = 10 // 平均句长下限
	paragraphMaxRunes  = 400
	ellipsisRunLimit   = 2 // 连续省略号出现次数上限
)

var (
	sentenceSplitRe = regexp.MustCompile(`[。！？!?…]+`)
	ellipsisRe      = regexp.MustCompile(`(?:…{2,}|\.{3,})`)
	// RE2 不支持反向引用，用「同类标点连续 3 个以上」表达堆砌。
	repeatPunctRe = regexp.MustCompile(`[！？!?]{3,}`)
)

// OutlineContractIssues 写前契约（确定性）：本章计划齐备性。
// 缺项不阻断创作，但生成前应让用户知道「模型没有任何抓手」。
func OutlineContractIssues(node types.OutlineNode) []Issue {
	var out []Issue
	if strings.TrimSpace(node.Title) == "" {
		out = append(out, Issue{Code: "outline_title_empty", Severity: "S2",
			Message: "本章标题为空：建议先写标题（生成时会作为章节名落库）"})
	}
	if strings.TrimSpace(node.Summary) == "" {
		out = append(out, Issue{Code: "outline_summary_empty", Severity: "S2",
			Message: "本章计划（summary）为空：先补「谁·在哪·发生什么」，否则模型只能自由发挥"})
	}
	if len(node.KeyPoints) == 0 {
		out = append(out, Issue{Code: "outline_keypoints_empty", Severity: "S3",
			Message: "关键要点为空：建议列 2-6 条本章必须落地的点"})
	}
	if strings.TrimSpace(node.Emotion) == "" {
		out = append(out, Issue{Code: "outline_emotion_empty", Severity: "S4",
			Message: "情感基调为空：建议一句话（如「紧张递进」），便于节奏控制"})
	}
	return out
}

// ChapterQualityIssues 写后质量体检（确定性）：
// 段落堆叠 / 电报体短句 / 标点堆砌 / 省略号滥用 / 通篇句号化 / 空正文。
func ChapterQualityIssues(text string) []Issue {
	body := strings.TrimSpace(text)
	if body == "" {
		return []Issue{{Code: "chapter_empty", Severity: "S1", Message: "正文为空"}}
	}
	var out []Issue

	// ① 段落堆叠
	longest, longestNo := 0, 0
	for i, p := range strings.Split(body, "\n") {
		if n := utf8.RuneCountInString(strings.TrimSpace(p)); n > longest {
			longest, longestNo = n, i+1
		}
	}
	if longest > paragraphMaxRunes {
		out = append(out, Issue{Code: "paragraph_too_long", Severity: "S3",
			Message:  "存在超长段落（" + itoa(longest) + " 字），建议按戏剧单元/镜头断开",
			Evidence: "第 " + itoa(longestNo) + " 行"})
	}

	// ② 句长节奏（电报体）
	sentences := splitSentences(body)
	if len(sentences) >= minSentences {
		short, total := 0, 0
		for _, s := range sentences {
			n := utf8.RuneCountInString(s)
			total += n
			if n <= shortSentenceRunes {
				short++
			}
		}
		avg := total / len(sentences)
		ratio := float64(short) / float64(len(sentences))
		switch {
		case ratio > shortRatioLimit && avg < avgSentenceMin:
			out = append(out, Issue{Code: "telegraph_style", Severity: "S2",
				Message:  "疑似电报体：短句占比 " + pct(ratio) + "、平均句长 " + itoa(avg) + " 字——把长句拆碎同样是 AI 味",
				Evidence: firstShort(sentences)})
		case avg < avgSentenceMin:
			out = append(out, Issue{Code: "sentences_too_short", Severity: "S3",
				Message: "平均句长偏短（" + itoa(avg) + " 字）：叙述默认应是逗号长句，短句只作孤立重拍"})
		}
	}

	// ③ 标点堆砌与省略号
	if m := repeatPunctRe.FindString(body); m != "" {
		out = append(out, Issue{Code: "punctuation_pileup", Severity: "S3",
			Message: "连续堆叠问号/感叹号：情绪用画面与动作承担", Evidence: m})
	}
	if runs := len(ellipsisRe.FindAllString(body, -1)); runs > ellipsisRunLimit {
		out = append(out, Issue{Code: "ellipsis_overuse", Severity: "S3",
			Message: "省略号使用 " + itoa(runs) + " 处：靠省略号造停顿是典型 AI 腔"})
	}

	// ④ 通篇句号化（缺逗号节奏）
	periods := strings.Count(body, "。")
	commas := strings.Count(body, "，")
	if periods >= 30 && commas*4 < periods {
		out = append(out, Issue{Code: "period_heavy", Severity: "S3",
			Message: "标点以句号为主（句号 " + itoa(periods) + " / 逗号 " + itoa(commas) + "）：中文叙述更依赖逗号长句节奏"})
	}
	return out
}

// splitSentences 按句末标点切句并去空白。
func splitSentences(text string) []string {
	parts := sentenceSplitRe.Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstShort(sentences []string) string {
	for _, s := range sentences {
		if utf8.RuneCountInString(s) <= shortSentenceRunes {
			return s
		}
	}
	return ""
}

func pct(v float64) string {
	return itoa(int(v*100)) + "%"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
