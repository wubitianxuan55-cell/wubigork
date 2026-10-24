package app

import (
	"strings"
	"testing"
)

// ── N10 统一插图画风（图文穿插一致性锚）测试 ────────────────────────────

func TestSinStoryStyleGetSet_Roundtrip(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	tp, err := a.SinTopicCreate("画风测试故事")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	topicID := tp.ID

	if got, err := a.SinStoryStyleGet(topicID); err != nil || got != "" {
		t.Fatalf("未设画风应返回空串: %q err=%v", got, err)
	}
	if got, err := a.SinStoryStyleSet(topicID, " 赛博朋克霓虹，高对比冷暖光，电影感 "); err != nil {
		t.Fatalf("SinStoryStyleSet: %v", err)
	} else if got != "赛博朋克霓虹，高对比冷暖光，电影感" {
		t.Fatalf("Set 应返回 trim 后生效值: %q", got)
	}
	if got, err := a.SinStoryStyleGet(topicID); err != nil || got != "赛博朋克霓虹，高对比冷暖光，电影感" {
		t.Fatalf("Get 应回读生效值: %q err=%v", got, err)
	}

	// 清空回无锚
	if got, err := a.SinStoryStyleSet(topicID, ""); err != nil || got != "" {
		t.Fatalf("清空应返回空串: %q err=%v", got, err)
	}
	if got, _ := a.SinStoryStyleGet(topicID); got != "" {
		t.Fatalf("清空后 Get 应为空, got %q", got)
	}

	// 未知话题 guard 拒绝
	if _, err := a.SinStoryStyleGet("no-such-topic"); err == nil {
		t.Fatalf("未知话题应拒绝")
	}
}

func TestSinStoryStyleSet_TruncatesToCap(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	tp, err := a.SinTopicCreate("截断测试")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	topicID := tp.ID
	long := strings.Repeat("厚", sinStoryStyleMaxRunes+50)
	got, err := a.SinStoryStyleSet(topicID, long)
	if err != nil {
		t.Fatalf("SinStoryStyleSet: %v", err)
	}
	if len([]rune(got)) != sinStoryStyleMaxRunes {
		t.Fatalf("应截断到 %d rune, got %d", sinStoryStyleMaxRunes, len([]rune(got)))
	}
}

func TestSinAugmentPromptWithStyle(t *testing.T) {
	const cue = "雨夜天台，她背对镜头站着，霓虹在积水里碎掉"
	// 无锚 = 原样
	if got := sinAugmentPromptWithStyle(cue, ""); got != cue {
		t.Fatalf("空画风应原样返回: %q", got)
	}
	if got := sinAugmentPromptWithStyle(cue, "  "); got != cue {
		t.Fatalf("空白画风应原样返回: %q", got)
	}
	// 有锚 = 追加画风层
	got := sinAugmentPromptWithStyle(cue, "赛博朋克霓虹，高对比冷暖光")
	if !strings.Contains(got, "统一画风（本故事插图保持一致）：赛博朋克霓虹，高对比冷暖光") {
		t.Fatalf("应追加画风层: %q", got)
	}
	if !strings.HasPrefix(got, cue) {
		t.Fatalf("原画面描述应保持在前: %q", got)
	}
	// 已含同款画风词 = 不重复
	if got2 := sinAugmentPromptWithStyle(got, "赛博朋克霓虹，高对比冷暖光"); got2 != got {
		t.Fatalf("已含画风应不重复追加:\n%q\n%q", got, got2)
	}
}

func TestSinSystemPromptStyleConsistencyAnchor(t *testing.T) {
	// N10：插图协议必须教模型别自带画风词（画风由统一锚承担）。
	p := sinSystemPrompt()
	for _, want := range []string{"画风一致", "不写「油画风」"} {
		if !strings.Contains(p, want) {
			t.Errorf("系统提示词缺少画风一致口径 %q", want)
		}
	}
}
