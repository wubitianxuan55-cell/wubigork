package bookimport

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNormalizePerspective_Aliases(t *testing.T) {
	cases := []struct{ in, want string }{
		{"第一人称", PerspectiveFirst},
		{"First-Person", PerspectiveFirst},
		{"firstperson", PerspectiveFirst},
		{"第三人称", PerspectiveThird},
		{"third_person", PerspectiveThird},
		{"limited third", PerspectiveThird},
		{"全知视角", PerspectiveOmniscient},
		{"omniscient", PerspectiveOmniscient},
		{"God-View", PerspectiveOmniscient},
		{"all_knowing", PerspectiveOmniscient},
		{"unknown-value", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizePerspective(c.in); got != c.want {
			t.Errorf("NormalizePerspective(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeProjectSuggestion_FieldFallbacksAndClamps(t *testing.T) {
	fb := FallbackProjectSuggestion("输入书名", []ParsedChapter{{Title: "第一章", Content: strings.Repeat("文", 500)}})

	// 全空对象 ⇒ 全回落；title 永用输入书名的兜底值
	got := NormalizeProjectSuggestion(map[string]any{}, fb)
	if got.Title != "输入书名" || got.NarrativePerspective != fb.NarrativePerspective || got.TargetWords != fb.TargetWords {
		t.Fatalf("空对象应整体回落: %+v", got)
	}

	// AI 给了书名也不采纳；字段类型错（数字/数组）一律回落
	got = NormalizeProjectSuggestion(map[string]any{
		"title":                 "AI 编的书名",
		"description":           123,
		"theme":                 []string{"主题"},
		"genre":                 "都市、悬疑",
		"narrative_perspective": "first_person",
		"target_words":          float64(120000),
	}, fb)
	if got.Title != "输入书名" {
		t.Fatalf("title 必须强制用输入值: %q", got.Title)
	}
	if got.Description != fb.Description || got.Theme != fb.Theme {
		t.Fatalf("类型错的字段应回落: %+v", got)
	}
	if got.Genre != "都市、悬疑" || got.NarrativePerspective != PerspectiveFirst || got.TargetWords != 120000 {
		t.Fatalf("合法字段应采纳: %+v", got)
	}

	// target_words 夹取：<1000 回落，>3e6 夹到 3e6
	got = NormalizeProjectSuggestion(map[string]any{"target_words": float64(12)}, fb)
	if got.TargetWords != fb.TargetWords {
		t.Fatalf("target_words <1000 应回落: %d", got.TargetWords)
	}
	got = NormalizeProjectSuggestion(map[string]any{"target_words": float64(9e6)}, fb)
	if got.TargetWords != maxTargetWords {
		t.Fatalf("target_words 应夹到 %d: %d", maxTargetWords, got.TargetWords)
	}
}

func TestNormalizeOutlineBatch_PositionalAlignmentAndForcedTitles(t *testing.T) {
	batch := []ParsedChapter{
		{Title: "第一章 甲", Content: "甲的内容。"},
		{Title: "第二章 乙", Content: "乙的内容。"},
		{Title: "第三章 丙", Content: "丙的内容。"},
	}
	// AI 乱序返回 + 少返一项 + 返回错位字段：一律按**输入位置**对齐
	raw := []any{
		map[string]any{"chapter_number": 99, "title": "AI 乱写的标题", "summary": "第一章的概要"},
		nil, // 非对象槽位 ⇒ fallback
	}
	got := NormalizeOutlineBatch(raw, batch, 1)
	if len(got) != 3 {
		t.Fatalf("应逐位补齐 3 条: %d", len(got))
	}
	if got[0].ChapterNumber != 1 || got[0].Title != "第一章 甲" || got[0].Summary != "第一章的概要" {
		t.Fatalf("第 1 槽位应对齐输入并强制标题/章号: %+v", got[0])
	}
	if got[1].ChapterNumber != 2 || got[1].Title != "第二章 乙" || got[1].Summary == "" {
		t.Fatalf("第 2 槽位（AI 少返）应回退规则结构: %+v", got[1])
	}
	if got[2].ChapterNumber != 3 || got[2].Title != "第三章 丙" {
		t.Fatalf("第 3 槽位应回退规则结构: %+v", got[2])
	}
}

func TestNormalizeOutlineBatch_FieldLevelFallbacks(t *testing.T) {
	batch := []ParsedChapter{{Title: "第一章", Content: "他推开门，雨还在下。"}}
	raw := []any{map[string]any{
		"summary":  strings.Repeat("概", summaryMaxRunes+50),
		"scenes":   []any{"场景一", "", 42, "场景二", "场景三", "场景四", "场景五", "场景六", "场景七"},
		"characters": []any{
			map[string]any{"name": "林晚", "type": "organization"},
			map[string]any{"name": "沈砚", "type": "Person"},
			map[string]any{"type": "character"}, // 无 name ⇒ 丢
			"裸字符串",                             // 非对象 ⇒ 丢
		},
		"key_points": []any{"要点1", "要点2", "要点3", "要点4", "要点5", "要点6", "要点7", "要点8", "要点9"},
		"emotion":    strings.Repeat("情", emotionMaxRunes+10),
		"goal":       strings.Repeat("目", goalMaxRunes+10),
	}}
	got := NormalizeOutlineBatch(raw, batch, 1)[0]
	if n := len([]rune(got.Summary)); n != summaryMaxRunes {
		t.Fatalf("summary 应截断到 %d rune: %d", summaryMaxRunes, n)
	}
	if len(got.Scenes) != scenesMax {
		t.Fatalf("scenes 应截断到 %d 条: %d", scenesMax, len(got.Scenes))
	}
	if len(got.Characters) != 2 || got.Characters[0].Type != "organization" || got.Characters[1].Type != "character" {
		t.Fatalf("characters 类型归一化异常: %+v", got.Characters)
	}
	if len(got.KeyPoints) != keyPointsMax {
		t.Fatalf("key_points 应截断到 %d 条: %d", keyPointsMax, len(got.KeyPoints))
	}
	if len([]rune(got.Emotion)) != emotionMaxRunes || len([]rune(got.Goal)) != goalMaxRunes {
		t.Fatalf("emotion/goal 未截断: %d %d", len([]rune(got.Emotion)), len([]rune(got.Goal)))
	}
}

func TestFallbackStructure_SummaryFromFirstSentence(t *testing.T) {
	got := FallbackStructure(ParsedChapter{Title: "第一章", Content: "他推开门，雨还在下。后面还有很多字。"}, 7)
	if got.ChapterNumber != 7 || got.Title != "第一章" || !strings.HasPrefix(got.Summary, "他推开门，雨还在下。") {
		t.Fatalf("兜底结构异常: %+v", got)
	}
	if len(got.Scenes) != 2 || len(got.KeyPoints) != 2 || got.Emotion != fallbackEmotion || got.Goal != fallbackGoal {
		t.Fatalf("兜底常量不符规格: %+v", got)
	}
	empty := FallbackStructure(ParsedChapter{Title: "空章"}, 1)
	if empty.Summary != fallbackSummary {
		t.Fatalf("空正文应取规格默认摘要: %q", empty.Summary)
	}
}

func TestCallJSON_RetryCarriesFailureAndExpectedType(t *testing.T) {
	var prompts []string
	call := func(_ context.Context, p string) (string, error) {
		prompts = append(prompts, p)
		switch len(prompts) {
		case 1:
			return "抱歉，我无法解析", nil
		case 2:
			return "```json\n{\"not\":\"array\"}\n```", nil
		default:
			return "[{\"chapter_number\":1}]", nil
		}
	}
	v, err := CallJSON(context.Background(), "基础提示", ExpectArray, 3, call)
	if err != nil {
		t.Fatalf("第 3 次应成功: %v", err)
	}
	if items, ok := v.([]any); !ok || len(items) != 1 {
		t.Fatalf("解析结果异常: %#v", v)
	}
	if len(prompts) != 3 {
		t.Fatalf("应调用 3 次: %d", len(prompts))
	}
	if prompts[0] != "基础提示" {
		t.Fatalf("第 1 次应为原始提示: %q", prompts[0])
	}
	for i, p := range prompts[1:] {
		if !strings.Contains(p, "第"+itoa(i+2)+"次重试") || !strings.Contains(p, "期望类型: array") ||
			!strings.Contains(p, "上次错误: ") {
			t.Fatalf("第 %d 次重试提示缺少负反馈要素: %q", i+2, p)
		}
	}
	// 失败原文被截断注入（规格：截 200 字）
	if strings.Contains(prompts[1], strings.Repeat("抱歉，我无法解析", 30)) {
		t.Fatalf("失败原文未截断")
	}
}

func TestCallJSON_ExpectedObjectMismatchFails(t *testing.T) {
	call := func(_ context.Context, _ string) (string, error) { return "[1,2,3]", nil }
	_, err := CallJSON(context.Background(), "p", ExpectObject, 2, call)
	if err == nil || !strings.Contains(err.Error(), "期望对象") {
		t.Fatalf("类型不符应报错且提示期望类型: %v", err)
	}
}

func TestCallJSON_TransportErrorNotRetried_CtxCancelHonored(t *testing.T) {
	calls := 0
	call := func(_ context.Context, _ string) (string, error) { calls++; return "", errors.New("网络失败") }
	if _, err := CallJSON(context.Background(), "p", ExpectObject, 3, call); err == nil || calls != 1 {
		t.Fatalf("传输错误应直接返回且不重试: calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	if _, err := CallJSON(ctx, "p", ExpectObject, 3, call); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("已取消的 ctx 不应发起调用: calls=%d err=%v", calls, err)
	}
}

func TestExtractJSON_TolerantInput(t *testing.T) {
	for _, in := range []string{
		"```json\n[{\"a\":1}]\n```",
		"好的，结果如下：\n[{\"a\":1}]\n以上。",
		"[{\"a\":1}]",
	} {
		v, err := ExtractJSON(in)
		if err != nil {
			t.Fatalf("应容忍包裹文本 %q: %v", in, err)
		}
		if arr, ok := v.([]any); !ok || len(arr) != 1 {
			t.Fatalf("解析结果异常: %#v", v)
		}
	}
	if _, err := ExtractJSON("没有 JSON"); err == nil {
		t.Fatal("无 JSON 应报错")
	}
}

func TestSampledTextAndBatchText(t *testing.T) {
	chapters := []ParsedChapter{
		{Title: "第一章", Content: strings.Repeat("甲", 50)},
		{Title: "第二章", Content: strings.Repeat("乙", 50)},
		{Title: "第三章", Content: strings.Repeat("丙", 50)},
		{Title: "第四章", Content: strings.Repeat("丁", 50)},
	}
	sample := SampledText(chapters, 3, 20)
	if !strings.Contains(sample, "第一章") || !strings.Contains(sample, "第三章") || strings.Contains(sample, "第四章") {
		t.Fatalf("采样章数不符（应前 3 章）: %q", sample)
	}
	if strings.Contains(sample, strings.Repeat("甲", 21)) {
		t.Fatalf("单章应按 rune 截断到 20 字")
	}
	batch := BatchText(chapters[:2])
	if !strings.Contains(batch, "### 第一章") || !strings.Contains(batch, "### 第二章") {
		t.Fatalf("批次正文应含章标题行: %q", batch)
	}
}
