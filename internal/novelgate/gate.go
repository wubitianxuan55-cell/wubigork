// Package novelgate 章节「写前契约 + 写后质量」的**确定性**检查（零 LLM、零网络）。
//
// 蒸馏来源：oh-story-claudecode（MIT）的 deslop-gates.md / quality-rubric.md 中
// **可机械判定**的维度（机制重推导，非代码搬运）；语义维度（卖点/动机/伏笔回收）
// 仍由 LLM 评审负责，见 .gaea/skills/novel-review/。
//
// 注（审计 IN1-04）：本包的句级节奏判据（电报体/平均句长，源自上游 quality-rubric
// 「句长节奏」行的 FAIL 口径）与 internal/novelreview 的段级段落判据（format_readability，
// 上游「格式可读性」行）是**同一上游 rubric 的两个维度分两路落地**：本包句级、阻断级
// （S2 进 app 的 severityBlocking 收敛链），review 段级、建议级（面板结论）。粒度与
// 门槛各自独立，见 ChapterQualityIssues ② 处互引注释。
package novelgate

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/noveltext"
	"github.com/gaea/gaea/internal/types"
)

// Issue 一条确定性发现（severity 对齐评审 rubric 的 S1-S4）。
// 注（IN1-09）：novelstyle 的 AI 味严重度已单源对齐本枚举（旧 low/medium/high/blocker → S4/S3/S2/S1），本包逻辑零改动。
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
	ellipsisRe = regexp.MustCompile(`(?:…{2,}|\.{3,})`)
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

	// ① 段落堆叠（切分单源 noveltext.SplitParagraphs；证据行号用 Line 口径=源行号，
	// 空行也占号——与旧内联 strings.Split(body, "\n") 逐字段一致）。
	longest, longestNo := 0, 0
	for _, pa := range noveltext.SplitParagraphs([]rune(body)) {
		if n := len([]rune(pa.Text)); n > longest {
			longest, longestNo = n, pa.Line
		}
	}
	if longest > paragraphMaxRunes {
		out = append(out, Issue{Code: "paragraph_too_long", Severity: "S3",
			Message:  "存在超长段落（" + itoa(longest) + " 字），建议按戏剧单元/镜头断开",
			Evidence: "第 " + itoa(longestNo) + " 行"})
	}

	// ② 句长节奏（电报体）——**句级**判据，源自上游 quality-rubric「句长节奏」行的
	// FAIL 口径（「逗号之间连着都是 ≤5 字、通篇超短句像提纲」；资产镜像
	// .gaea/skills/novel-review/rubrics/generic.json 的 sentence_rhythm 维度）。
	// 注（审计 IN1-04）：novelreview 的 format_readability 另有**段级**碎段判据
	// （dimParagraphPace：≥12 段且平均段长 ≤12 字 → WARN 建议级）——同域不同粒度：
	// 本处句级、阻断级（S2 进 app 的 severityBlocking 收敛链），review 段级、建议级。
	// 两处门槛各自独立、粒度刻意分层，勿顺手对齐；「同一正文本处报 S2、review 整体
	// APPROVE」是既证事实，对照样本钉在 telegraph_granularity_test.go。
	sentences := noveltext.SplitSentences(body)
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

// firstShort 第一处短句（电报体证据）。
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
