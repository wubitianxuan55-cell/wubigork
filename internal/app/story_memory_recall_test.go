package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/types"
)

func mkQueryNode() *types.OutlineNode {
	return &types.OutlineNode{
		Title:     "真相之夜",
		Summary:   "林晚在祠堂确认身世",
		Emotion:   "悲怆",
		Characters: []string{"char_a", "char_b", "char_c", "char_d", "char_e", "char_f", "char_g", "char_h", "char_i", "char_j"},
		KeyPoints: []string{"点1", "点2", "点3", "点4", "点5", "点6", "点7"},
	}
}

// TestBuildMemoryQuery 结构化 query（spec §12.4-1）：分段限额 + 总长 800 + 换行压空格。
func TestBuildMemoryQuery(t *testing.T) {
	q := buildMemoryQuery(mkQueryNode(), "写林晚与父亲对峙的高潮戏")

	if !strings.Contains(q, "人物：char_a、char_b、char_c、char_d、char_e、char_f、char_g、char_h") {
		t.Fatalf("人物应限额前 8: %s", q)
	}
	if strings.Contains(q, "char_i") {
		t.Fatalf("人物第 9 个起不应出现: %s", q)
	}
	if !strings.Contains(q, "关键事件：点1、点2、点3、点4、点5、点6") || strings.Contains(q, "点7") {
		t.Fatalf("关键事件应限额前 6: %s", q)
	}
	if !strings.Contains(q, "叙事目标：林晚在祠堂确认身世") || !strings.Contains(q, "情绪：悲怆") {
		t.Fatalf("叙事目标/情绪段缺失: %s", q)
	}
	if !strings.Contains(q, "本章要求：写林晚与父亲对峙的高潮戏") {
		t.Fatalf("本章要求段缺失: %s", q)
	}
	if strings.Contains(q, "\n") {
		t.Fatalf("换行应压成空格: %s", q)
	}
	if runeLen(q) > memoryQueryMaxLen {
		t.Fatalf("query 应 ≤%d rune，实际 %d", memoryQueryMaxLen, runeLen(q))
	}

	// 空 node / 超长 query
	if got := buildMemoryQuery(nil, "任意"); got != "" {
		t.Fatalf("nil node 应返回空: %q", got)
	}
	long := buildMemoryQuery(&types.OutlineNode{Summary: strings.Repeat("长", 2000)}, "")
	if runeLen(long) > memoryQueryMaxLen {
		t.Fatalf("超长叙事目标应被截到预算内: %d", runeLen(long))
	}
}

// TestSelectMemoryHits 四段流水线筛选段（spec §3.4）：阈值之上取 topK；
// 全低于阈值保留 fallback 条兜底。
func TestSelectMemoryHits(t *testing.T) {
	hit := func(id string, score float64) semantic.Hit {
		return semantic.Hit{ID: id, Score: score}
	}
	// 阈值之上取 topK（8），降序输入
	var hits []semantic.Hit
	for i := 0; i < 12; i++ {
		hits = append(hits, hit(string(rune('A'+i)), 0.9-float64(i)*0.01))
	}
	got := selectMemoryHits(hits)
	if len(got) != storyMemoryTopK || got[0].ID != "A" {
		t.Fatalf("阈值之上应取 topK=%d: %d", storyMemoryTopK, len(got))
	}

	// 全部低于阈值 → 兜底 3 条
	var low []semantic.Hit
	for i := 0; i < 6; i++ {
		low = append(low, hit(string(rune('a'+i)), 0.2-float64(i)*0.01))
	}
	got = selectMemoryHits(low)
	if len(got) != storyMemoryFallback || got[0].ID != "a" {
		t.Fatalf("无高分应兜底 %d 条: %d", storyMemoryFallback, len(got))
	}

	// 低于兜底数：原样返回
	two := []semantic.Hit{hit("x", 0.2), hit("y", 0.15)}
	if got := selectMemoryHits(two); len(got) != 2 {
		t.Fatalf("低于兜底数应原样: %d", len(got))
	}
}

// TestFormatMemoryLines 注入行格式：相关度两位小数 + 内容截断。
func TestFormatMemoryLines(t *testing.T) {
	hits := []semantic.Hit{
		{ID: "012-foreshadow-1", Score: 0.824, Text: strings.Repeat("忆", memoryLineMax+50)},
	}
	lines := formatMemoryLines(hits)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "- (相关度:0.82) ") {
		t.Fatalf("行格式不对: %q", lines[0])
	}
	if runeLen(lines[0]) > runeLen("- (相关度:0.82) ")+memoryLineMax {
		t.Fatalf("内容应截断: %d", runeLen(lines[0]))
	}
}

// TestRecallStoryMemories_FaultTolerant 召回全链容错：库不可用 / embedder
// 未配置（测试环境无 Herdsman）→ 返回 nil 绝不 panic。
func TestRecallStoryMemories_FaultTolerant(t *testing.T) {
	a := newFingerprintTestApp(t)
	pm := a.getPM()

	// 测试环境无 Herdsman 引擎 → localSearchEmbedder 返回 nil → 诚实降级
	if got := a.recallStoryMemories(pm, mkQueryNode(), "任意", 5); got != nil {
		t.Fatalf("无 embedder 应返回 nil，实际 %v", got)
	}
}
