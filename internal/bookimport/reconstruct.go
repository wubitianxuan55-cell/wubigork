package bookimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ── t2 P1 · AI 反推编排（规格 docs/distill/02-book-import.md §8.2 / §3.7 / §3.8）──
//
// 本文件是**纯函数 + 可注入调用缝**的编排层：不直接依赖任何 LLM 客户端，
// 由 app 层把「渲染模板 → 调模型」的函数传进来（CallJSON 的 call 参数），
// 因此归一化矩阵与重试语义可完全单测（规格 §8.2 的 Go 接口清单）。

// ProjectSuggestion 反推出的项目立项信息（对齐 schemas/book_import.py:22-29）。
type ProjectSuggestion struct {
	Title                string `json:"title"` // ★ 永不用 AI 结果（输入书名优先）
	Description          string `json:"description,omitempty"`
	Theme                string `json:"theme,omitempty"`
	Genre                string `json:"genre,omitempty"`
	NarrativePerspective string `json:"narrative_perspective"`
	TargetWords          int    `json:"target_words"`
}

// OutlineCharacter 大纲里的角色/组织引用（type 仅两值）。
type OutlineCharacter struct {
	Name string `json:"name"`
	Type string `json:"type"` // character | organization
}

// OutlineStructure 结构化章节大纲（对齐 §3.7 的 AI 输出契约）。
type OutlineStructure struct {
	ChapterNumber int                `json:"chapter_number"`
	Title         string             `json:"title"`
	Summary       string             `json:"summary"`
	Scenes        []string           `json:"scenes"`
	Characters    []OutlineCharacter `json:"characters"`
	KeyPoints     []string           `json:"key_points"`
	Emotion       string             `json:"emotion"`
	Goal          string             `json:"goal"`
}

// 叙事视角枚举（规格：中文三值 + 旧英文别名归一）。
const (
	PerspectiveFirst      = "第一人称"
	PerspectiveThird      = "第三人称"
	PerspectiveOmniscient = "全知视角"
)

// 字数与长度夹取阈值（规格 §8.2 Stage1 归一化）。
const (
	minTargetWords = 1000
	maxTargetWords = 3000000
	summaryMaxRunes = 2000
	emotionMaxRunes = 200
	goalMaxRunes    = 300
	scenesMax       = 6
	keyPointsMax    = 8
	charNameMaxRunes = 80
)

// 兜底文案（规格 §3.7 的规则兜底结构，逐字对齐）。
const (
	fallbackSummary    = "本章围绕主要人物与核心冲突推进剧情。"
	fallbackEmotion    = "紧张递进"
	fallbackGoal       = "承接前章并推动后续剧情发展"
	fallbackKeyPointA  = "推进主线冲突"
	fallbackKeyPointB  = "呈现角色动机与关系变化"
)

var fallbackScenes = []string{"主角在当前处境中做出关键选择", "冲突升级并形成新的悬念"}

// NormalizePerspective 视角归一：中文枚举直通；英文/旧别名（11 个）映射；
// 未知值返回空串由调用方决定回落（不猜）。
func NormalizePerspective(s string) string {
	v := strings.ToLower(strings.TrimSpace(s))
	v = strings.ReplaceAll(v, "-", "_")
	v = strings.ReplaceAll(v, " ", "_")
	switch v {
	case "第一人称", "first_person", "firstperson", "first", "i":
		return PerspectiveFirst
	case "第三人称", "third_person", "thirdperson", "third", "limited_third", "close_third":
		return PerspectiveThird
	case "全知视角", "全知", "omniscient", "god_view", "god", "all_knowing", "objective":
		return PerspectiveOmniscient
	}
	return ""
}

// FallbackProjectSuggestion 规则兜底立项信息（AI 不可用/解析失败时的诚实回落）。
func FallbackProjectSuggestion(title string, chapters []ParsedChapter) ProjectSuggestion {
	totalRunes := 0
	for _, ch := range chapters {
		totalRunes += utf8.RuneCountInString(ch.Content)
	}
	desc := "由导入正文自动生成：" + fmt.Sprintf("共 %d 章，约 %d 字。", len(chapters), totalRunes)
	return ProjectSuggestion{
		Title:                title,
		Description:          desc,
		Theme:                "（未推断）",
		Genre:                "未分类",
		NarrativePerspective: PerspectiveThird,
		TargetWords:          clampTargetWords(totalRunes * 2),
	}
}

// FallbackStructure 规则兜底的大纲结构（规格 §3.7 末段逐字对齐；summary 取正文首句）。
func FallbackStructure(ch ParsedChapter, chapterNumber int) OutlineStructure {
	summary := firstSentence(ch.Content, 120)
	if summary == "" {
		summary = fallbackSummary
	}
	return OutlineStructure{
		ChapterNumber: chapterNumber,
		Title:         strings.TrimSpace(ch.Title),
		Summary:       summary,
		Scenes:        append([]string{}, fallbackScenes...),
		Characters:    []OutlineCharacter{},
		KeyPoints:     []string{fallbackKeyPointA, fallbackKeyPointB},
		Emotion:       fallbackEmotion,
		Goal:          fallbackGoal,
	}
}

// NormalizeProjectSuggestion 逐字段归一化（**绝不信任 AI**：空/类型错即回落）。
// title 永不用 AI 结果（规格 §3.7 字段表末行）。
func NormalizeProjectSuggestion(raw map[string]any, fallback ProjectSuggestion) ProjectSuggestion {
	out := fallback
	out.Title = fallback.Title // 强制输入书名
	if v := strField(raw, "description"); v != "" {
		out.Description = truncateRunes(v, 1000)
	}
	if v := strField(raw, "theme"); v != "" {
		out.Theme = truncateRunes(v, 100)
	}
	if v := strField(raw, "genre"); v != "" {
		out.Genre = truncateRunes(v, 60)
	}
	if v := NormalizePerspective(strField(raw, "narrative_perspective")); v != "" {
		out.NarrativePerspective = v
	}
	if n, ok := intField(raw, "target_words"); ok {
		if n >= minTargetWords {
			out.TargetWords = clampTargetWords(n)
		}
	}
	return out
}

// NormalizeOutlineBatch 批量归一化：**按输入章节顺序逐位取**（不按 AI 返回的
// chapter_number 对齐），非对象槽位直接以 fallback 填充——AI 少返/乱序都不会错章；
// 最后仍做「数量不符 ⇒ 整批规则结构」的断言式防线（宁可全丢，不用错位结构）。
func NormalizeOutlineBatch(raw any, batch []ParsedChapter, startNumber int) []OutlineStructure {
	items, _ := raw.([]any)
	out := make([]OutlineStructure, 0, len(batch))
	for i, ch := range batch {
		num := startNumber + i
		fb := FallbackStructure(ch, num)
		if i >= len(items) {
			out = append(out, fb)
			continue
		}
		obj, ok := items[i].(map[string]any)
		if !ok {
			out = append(out, fb)
			continue
		}
		out = append(out, normalizeOneStructure(obj, fb))
	}
	if len(out) != len(batch) {
		// 断言式防线：结构性错位时整批回退规则结构。
		out = out[:0]
		for i, ch := range batch {
			out = append(out, FallbackStructure(ch, startNumber+i))
		}
	}
	return out
}

// normalizeOneStructure 单章归一化（规格 §3.7 字段表逐条）。
func normalizeOneStructure(obj map[string]any, fb OutlineStructure) OutlineStructure {
	out := fb // chapter_number / title 强制用输入值
	if v := strField(obj, "summary"); v != "" {
		out.Summary = truncateRunes(v, summaryMaxRunes)
	} else if v := strField(obj, "content"); v != "" {
		out.Summary = truncateRunes(v, summaryMaxRunes)
	}
	if scenes := stringListField(obj, "scenes", scenesMax); len(scenes) > 0 {
		out.Scenes = scenes
	}
	if chars := characterListField(obj, "characters"); len(chars) > 0 {
		out.Characters = chars
	}
	if keys := stringListField(obj, "key_points", keyPointsMax); len(keys) > 0 {
		out.KeyPoints = keys
	}
	if v := strField(obj, "emotion"); v != "" {
		out.Emotion = truncateRunes(v, emotionMaxRunes)
	}
	if v := strField(obj, "goal"); v != "" {
		out.Goal = truncateRunes(v, goalMaxRunes)
	}
	return out
}

// ── JSON 解析与负反馈重试（规格 §3.8）──────────────────────────

// JSONHint 重试提示（对齐 ai_service.py:689-690）：把上次失败原文**截 200 字**注入，
// 让模型自我纠偏，并显式声明期望类型（object / array）。
func JSONHint(prompt string, attempt int, expected string, failed string) string {
	return fmt.Sprintf("%s\n\n⚠️ 第%d次重试，请返回纯JSON，不要markdown包裹。期望类型: %s。上次错误: %.200s...",
		prompt, attempt, expected, failed)
}

// ExtractJSON 从模型回复中取出 JSON（容忍 markdown 代码块包裹与前后解释文字）。
func ExtractJSON(resp string) (any, error) {
	s := strings.TrimSpace(resp)
	s = stripCodeFence(s)
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		return nil, fmt.Errorf("回复中没有 JSON 结构")
	}
	open := s[start]
	closeCh := byte('}')
	if open == '[' {
		closeCh = ']'
	}
	end := strings.LastIndexByte(s, closeCh)
	if end <= start {
		return nil, fmt.Errorf("JSON 结构不完整")
	}
	var v any
	if err := json.Unmarshal([]byte(s[start:end+1]), &v); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	return v, nil
}

// ExpectedKind 期望类型（对齐 MuMu 的 expected_type，避免「AI 返对象但按数组解包」）。
type ExpectedKind string

const (
	ExpectObject ExpectedKind = "object"
	ExpectArray  ExpectedKind = "array"
)

// CheckExpected 校验解析结果类型。
func CheckExpected(v any, kind ExpectedKind) error {
	switch kind {
	case ExpectObject:
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("期望对象（object），实际为 %T", v)
		}
	case ExpectArray:
		if _, ok := v.([]any); !ok {
			return fmt.Errorf("期望数组（array），实际为 %T", v)
		}
	}
	return nil
}

// CallJSON 带负反馈重试的 JSON 调用（maxAttempts ≤0 视为 3）。
// call 由调用方注入（渲染模板 + 调模型）；**传输错误直接返回**（换个提示无济于事），
// 只有解析/类型错误才带失败原文重试。
func CallJSON(ctx context.Context, prompt string, kind ExpectedKind, maxAttempts int, call func(ctx context.Context, prompt string) (string, error)) (any, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	last := ""
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p := prompt
		if attempt > 1 {
			p = JSONHint(prompt, attempt, string(kind), last)
		}
		resp, err := call(ctx, p)
		if err != nil {
			return nil, err
		}
		v, err := ExtractJSON(resp)
		if err == nil {
			err = CheckExpected(v, kind)
		}
		if err == nil {
			return v, nil
		}
		last = resp
		lastErr = err
	}
	return nil, fmt.Errorf("JSON 解析失败（已重试 %d 次）: %w", maxAttempts, lastErr)
}

// SampledText 采样前 N 章（每章截 maxRunes 字）供 Stage1 立项反推。
func SampledText(chapters []ParsedChapter, maxChapters, maxRunes int) string {
	if maxChapters <= 0 {
		maxChapters = 3
	}
	if maxRunes <= 0 {
		maxRunes = 2000
	}
	var sb strings.Builder
	for i, ch := range chapters {
		if i >= maxChapters {
			break
		}
		sb.WriteString("### " + strings.TrimSpace(ch.Title) + "\n")
		sb.WriteString(truncateRunes(strings.TrimSpace(ch.Content), maxRunes))
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// BatchText 组装一批章节正文（供大纲反推模板的 chapters_text 输入）。
func BatchText(batch []ParsedChapter) string {
	var sb strings.Builder
	for _, ch := range batch {
		sb.WriteString("### " + strings.TrimSpace(ch.Title) + "\n")
		sb.WriteString(strings.TrimSpace(ch.Content))
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// ── 内部工具 ──────────────────────────────────────────────────

func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	lines = lines[1:]
	if n := len(lines); n > 0 && strings.HasPrefix(strings.TrimSpace(lines[n-1]), "```") {
		lines = lines[:n-1]
	}
	return strings.Join(lines, "\n")
}

func strField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func intField(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}

func stringListField(m map[string]any, key string, max int) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		s, ok := it.(string)
		if !ok {
			continue
		}
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= max {
			break
		}
	}
	return out
}

func characterListField(m map[string]any, key string) []OutlineCharacter {
	v, ok := m[key]
	if !ok {
		return nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]OutlineCharacter, 0, len(items))
	for _, it := range items {
		obj, ok := it.(map[string]any)
		if !ok {
			continue // 非对象项直接丢（spec：必须 dict 且有 name）
		}
		name := strings.TrimSpace(strField(obj, "name"))
		if name == "" {
			continue
		}
		t := strField(obj, "type")
		if t != "organization" {
			t = "character" // 其余一律 character（含空/未知）
		}
		out = append(out, OutlineCharacter{Name: truncateRunes(name, charNameMaxRunes), Type: t})
	}
	return out
}

func clampTargetWords(n int) int {
	if n < minTargetWords {
		return minTargetWords
	}
	if n > maxTargetWords {
		return maxTargetWords
	}
	return n
}

// firstSentence 取正文首句（用于兜底 summary）：到首个句读或 max 字为止。
func firstSentence(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	limit := len(runes)
	if limit > max {
		limit = max
	}
	for i := 0; i < limit; i++ {
		if isSentenceBoundary(runes[i]) {
			return strings.TrimSpace(string(runes[:i+1]))
		}
	}
	return strings.TrimSpace(string(runes[:limit]))
}
