package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/ai"
)

// AI 伴读截断预算（防 prompt 爆炸）。
const (
	// readingMaxChapterRunes 摘要路径的正文头部截断上限（维持旧行为）。
	readingMaxChapterRunes = 9000
	// readingMaxContextRunes 划线提问时正文窗口的总 rune 上限。
	readingMaxContextRunes = 12000
	// readingSelWindowRunes 划线前后各保留的正文 rune 数。
	readingSelWindowRunes = 3000
	// readingMaxHistoryTurns 问书历史最多保留的轮数（保留最近）。
	readingMaxHistoryTurns = 6
	// readingMaxHistoryARunes 历史单轮回答的 rune 截断上限。
	readingMaxHistoryARunes = 500
)

// readingTurn 问书历史一轮：q=用户问题，a=助手回答（historyJSON 的数组元素）。
type readingTurn struct {
	Q string `json:"q"`
	A string `json:"a"`
}

// truncateChapterBody 摘要路径的正文头部截断（维持旧行为与旧提示语）。
func truncateChapterBody(body string) string {
	r := []rune(body)
	if len(r) > readingMaxChapterRunes {
		return string(r[:readingMaxChapterRunes]) + "……（正文过长已截断）"
	}
	return body
}

// parseReadingHistory 解析问书历史 JSON（[{"q":"...","a":"..."}]）。
// 空串、解析失败、非数组一律返回 nil——退回单轮行为，兼容旧签名调用方。
func parseReadingHistory(historyJSON string) []readingTurn {
	if strings.TrimSpace(historyJSON) == "" {
		return nil
	}
	var turns []readingTurn
	if err := json.Unmarshal([]byte(historyJSON), &turns); err != nil {
		return nil
	}
	out := make([]readingTurn, 0, len(turns))
	for _, t := range turns {
		t.Q = strings.TrimSpace(t.Q)
		t.A = strings.TrimSpace(t.A)
		if t.Q == "" {
			continue // 没有问题的轮次无上下文价值，跳过
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// trimReadingHistory 防止 prompt 爆炸：只保留最近 readingMaxHistoryTurns 轮，
// 每轮回答用 truncateRunes（gaea_subagents.go）截到 readingMaxHistoryARunes rune。
func trimReadingHistory(turns []readingTurn) []readingTurn {
	if len(turns) > readingMaxHistoryTurns {
		turns = turns[len(turns)-readingMaxHistoryTurns:]
	}
	for i := range turns {
		turns[i].A = truncateRunes(turns[i].A, readingMaxHistoryARunes)
	}
	return turns
}

// readingHistoryPrompt 把历史轮渲染为 prompt 片段（用户/助手对白）。
func readingHistoryPrompt(turns []readingTurn) string {
	var b strings.Builder
	for i, t := range turns {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "用户：%s\n助手：%s", t.Q, t.A)
	}
	return b.String()
}

// indexOfRunes 子序列首次出现的 rune 偏移，找不到返回 -1。
func indexOfRunes(hay, needle []rune) int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return -1
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// locateSelection 在正文中定位 selection 的 rune 偏移。先精确匹配；失败后按
// 「连续空白折叠为单个空格」归一化再匹配（前端划词会把跨行选区折叠成空格），
// 并把归一化偏移映射回原文偏移。找不到返回 -1。
func locateSelection(text, selection string) int {
	if selection == "" {
		return -1
	}
	if i := strings.Index(text, selection); i >= 0 {
		return len([]rune(text[:i]))
	}
	tr := []rune(text)
	sr := []rune(selection)
	norm := make([]rune, 0, len(tr))
	back := make([]int, 0, len(tr)) // norm[i] → tr 中的原始 rune 下标
	prevSpace := true               // 开头空白直接丢弃
	for i, c := range tr {
		if unicode.IsSpace(c) {
			if prevSpace {
				continue
			}
			norm = append(norm, ' ')
			back = append(back, i)
			prevSpace = true
			continue
		}
		norm = append(norm, c)
		back = append(back, i)
		prevSpace = false
	}
	j := indexOfRunes(norm, sr)
	if j < 0 {
		return -1
	}
	return back[j]
}

// readingSelectionWindow 划线提问的正文窗口：全文在窗口内则原样返回；否则
// 优先保证 selection 完整并保留其前后各 readingSelWindowRunes rune（总上限
// readingMaxContextRunes，极端长划线时均匀压缩前后窗）。selection 定位不到时
// 返回空串，由调用方退回「仅摘选原文」的旧行为。
func readingSelectionWindow(chapterText, selection string) string {
	r := []rune(chapterText)
	if len(r) <= readingMaxContextRunes {
		return chapterText
	}
	idx := locateSelection(chapterText, selection)
	if idx < 0 {
		return ""
	}
	sel := []rune(selection)
	selStart, selEnd := idx, idx+len(sel)
	before, after := readingSelWindowRunes, readingSelWindowRunes
	// 划线本身极长时保完整划线，前后窗压缩到剩余预算
	if span := selEnd - selStart; span+before+after > readingMaxContextRunes {
		room := readingMaxContextRunes - span
		if room < 0 {
			room = 0
		}
		before = room / 2
		after = room - before
	}
	start := idx - before
	if start < 0 {
		start = 0
	}
	end := selEnd + after
	if end > len(r) {
		end = len(r)
	}
	var b strings.Builder
	if start > 0 {
		b.WriteString("……（前文略）\n")
	}
	b.WriteString(string(r[start:end]))
	if end < len(r) {
		b.WriteString("\n……（后文略）")
	}
	return b.String()
}

// readingAskUserPrompt 组装 ask 的 user prompt。无历史时与旧单轮格式逐字一致。
func readingAskUserPrompt(title, context, question string, history []readingTurn) string {
	if len(history) == 0 {
		return fmt.Sprintf("章节：%s\n\n【摘选原文】\n%s\n\n【问题】%s", title, context, question)
	}
	return fmt.Sprintf("章节：%s\n\n【此前对话】\n%s\n\n【摘选原文】\n%s\n\n【问题】%s",
		title, readingHistoryPrompt(history), context, question)
}

// NovelReadingAsk AI 阅读伴读：summary = 章节摘要，ask = 划线提问。
// 只使用当前章节本地文本，不写任何文件；模型走 novel 功能路由。
// historyJSON 为问书历史 [{"q":"...","a":"..."}] 数组 JSON（仅 ask 生效），
// 空串/解析失败按单轮处理，与旧签名行为完全兼容。
func (a *writingState) NovelReadingAsk(kind, title, chapterText, selection, question, historyJSON string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	kind = strings.TrimSpace(kind)
	if title == "" {
		title = "未命名章节"
	}

	var system, user string
	maxTokens := 1024
	switch kind {
	case "summary":
		body := strings.TrimSpace(chapterText)
		if body == "" {
			return "", fmt.Errorf("本章暂无内容")
		}
		system = "你是一位专业的小说阅读伴读。用简洁、有信息量的中文概括本章内容，不复述原文，不用套话开头。"
		user = fmt.Sprintf("章节：%s\n\n【正文】\n%s\n\n请给出本章摘要：3-5 条要点，每条不超过 60 字，使用“- ”列表。",
			title, truncateChapterBody(body))
	case "ask":
		sel := strings.TrimSpace(selection)
		q := strings.TrimSpace(question)
		if sel == "" {
			return "", fmt.Errorf("请先摘选原文")
		}
		if q == "" {
			return "", fmt.Errorf("请输入问题")
		}
		history := trimReadingHistory(parseReadingHistory(historyJSON))
		system = "你是一位小说阅读伴读助手。基于用户摘选的原文片段回答问题，语言精炼、贴合上下文；若原文信息不足，明确说明而非编造。"
		if len(history) > 0 {
			system += "这是连续追问中的一轮：请结合【此前对话】理解代词与省略指代（如「那他后来呢」），与之前的回答保持一致；只围绕当前问题补充新信息，不重复此前内容。"
		}
		// 上下文加厚：正文窗口优先包住划线（前后各 ~3000，总上限 12000）；
		// 划线定位不到时退回旧行为——只送摘选原文（头部截断上限同步放宽到 12000）。
		context := readingSelectionWindow(chapterText, sel)
		if context == "" {
			context = truncateRunes(sel, readingMaxContextRunes)
		}
		user = readingAskUserPrompt(title, context, q, history)
		maxTokens = 2048
	default:
		return "", fmt.Errorf("未知的伴读类型：%s", kind)
	}

	eng, model, _ := a.routeModel("novel")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{
		EngineID:    eng,
		Temperature: 0.6,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("AI 伴读失败: %w", err)
	}
	return strings.TrimSpace(reply), nil
}
