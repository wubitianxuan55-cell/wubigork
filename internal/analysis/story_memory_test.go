package analysis

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// mkV2 构造带各类信号的 V2 载荷。
func mkV2() *types.AnalysisResultV2 {
	v2 := &types.AnalysisResultV2{Summary: "本章主角确认身世"}
	v2.Scores = types.AnalysisScores{Pacing: 8, Engagement: 8, Coherence: 8}
	v2.Hooks = []types.Hook{
		{Type: "悬念", Content: "章尾铜匣自开", Strength: 7, Position: "结尾"},
		{Type: "情感", Content: "弱钩不入库", Strength: 5},
	}
	v2.Foreshadows = []types.ForeshadowHit{
		{Type: "planted", Title: "星门钥匙", Content: "月圆之夜星门开启", Strength: 8},
		{Type: "resolved", Title: "旧伤", Content: "左臂旧伤迸裂", Strength: 0}, // 缺省 5
	}
	v2.PlotPoints = []types.PlotPoint{
		{Content: "主线推进", Importance: 0.8},
		{Content: "过场", Importance: 0.4}, // 低于门槛
	}
	v2.Conflict = types.Conflict{Level: 8, Description: "父女对峙"}
	v2.CharacterStates = []types.CharacterStateChangeV2{
		{Name: "林晚", OldState: "怀疑", NewState: "确认", PsychologicalChange: "释然", KeyEvent: "读到遗书"},
	}
	return v2
}

// TestExtractStoryMemories_RuleTable 六类规则表逐条验证（spec §2.3）。
func TestExtractStoryMemories_RuleTable(t *testing.T) {
	v2 := mkV2()
	mems := ExtractStoryMemories(12, "012.md", v2, "正文")

	byType := map[string][]types.StoryMemory{}
	for _, m := range mems {
		byType[m.Type] = append(byType[m.Type], m)
	}

	// ① chapter_summary：必有、固定 0.6、取 summary
	if got := byType[types.MemoryTypeChapterSummary]; len(got) != 1 ||
		got[0].Content != "本章主角确认身世" || got[0].Importance != MemorySummaryImportance {
		t.Fatalf("chapter_summary 不对: %+v", got)
	}
	// ② hook：门槛 6 —— 7 入库、5 不入；importance=min(7/10,1)=0.7
	if got := byType[types.MemoryTypeHook]; len(got) != 1 || got[0].Content != "章尾铜匣自开" ||
		got[0].Importance != 0.7 || got[0].Title != "悬念" {
		t.Fatalf("hook 门槛/重要性不对: %+v", got)
	}
	// ③ foreshadow：全部入库；planted=1/resolved=2；strength 0→缺省 5→0.5
	fs := byType[types.MemoryTypeForeshadow]
	if len(fs) != 2 {
		t.Fatalf("伏笔应全部入库: %+v", fs)
	}
	if fs[0].IsForeshadow != types.ForeshadowFlagPlanted || fs[0].Importance != 0.8 {
		t.Fatalf("planted 标记/重要性不对: %+v", fs[0])
	}
	if fs[1].IsForeshadow != types.ForeshadowFlagResolved || fs[1].Importance != 0.5 {
		t.Fatalf("resolved 标记/缺省重要性不对: %+v", fs[1])
	}
	// ④ plot_point：门槛 0.6 —— 0.8 入、0.4 不入，importance 原样
	pp := byType[types.MemoryTypePlotPoint]
	if len(pp) != 2 || pp[0].Content != "主线推进" || pp[0].Importance != 0.8 {
		t.Fatalf("plot_point 门槛不对: %+v", pp)
	}
	// 末条应是 conflict→plot_point（level 8 → 0.8，title=冲突）
	if pp[1].Title != "冲突" || pp[1].Importance != 0.8 || pp[1].Content != "父女对峙" {
		t.Fatalf("冲突应记为 plot_point: %+v", pp[1])
	}
	// ⑤ character_event：固定 0.7、related_characters=[角色名]、内容含差分
	ce := byType[types.MemoryTypeCharacterEvent]
	if len(ce) != 1 || ce[0].Importance != MemoryCharEventImportance ||
		ce[0].RelatedCharacters[0] != "林晚" || !strings.Contains(ce[0].Content, "怀疑 → 确认") {
		t.Fatalf("character_event 不对: %+v", ce)
	}
}

// TestExtractStoryMemories_DeterministicIDs 确定性 ID（spec §12.2：
// <MMM>-<type>-<ordinal>）：同类型按序编号，重提取同 ID（幂等前提）。
func TestExtractStoryMemories_DeterministicIDs(t *testing.T) {
	v2 := mkV2()
	first := ExtractStoryMemories(12, "012.md", v2, "正文")
	second := ExtractStoryMemories(12, "012.md", v2, "正文")
	if len(first) != len(second) {
		t.Fatalf("两次提取条数应一致")
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("确定性 ID 漂移: %q vs %q", first[i].ID, second[i].ID)
		}
	}
	// ID 形状：章号零变形 + 类型 + 类型内序号（伏笔两条 → -1/-2）
	fsIDs := []string{}
	for _, m := range first {
		if m.Type == types.MemoryTypeForeshadow {
			fsIDs = append(fsIDs, m.ID)
		}
	}
	if len(fsIDs) != 2 || fsIDs[0] != "012-foreshadow-1" || fsIDs[1] != "012-foreshadow-2" {
		t.Fatalf("伏笔 ID 序号编排不对: %v", fsIDs)
	}
}

// TestExtractStoryMemories_SummaryFallbackChain chapter_summary 三级回退链
// （summary → 前 3 条推进点 → 正文前 300 字）。
func TestExtractStoryMemories_SummaryFallbackChain(t *testing.T) {
	content := strings.Repeat("正", 400)

	// 回退 2：无 summary → 前 3 条推进点拼接
	v2 := &types.AnalysisResultV2{}
	v2.Scores = types.AnalysisScores{Pacing: 8, Engagement: 8, Coherence: 8}
	for _, c := range []string{"点一", "点二", "点三", "点四·应被裁"} {
		v2.PlotPoints = append(v2.PlotPoints, types.PlotPoint{Content: c, Importance: 0.9})
	}
	mems := ExtractStoryMemories(1, "001.md", v2, content)
	var sum types.StoryMemory
	for _, m := range mems {
		if m.Type == types.MemoryTypeChapterSummary {
			sum = m
		}
	}
	if !strings.Contains(sum.Content, "点三") || strings.Contains(sum.Content, "点四") {
		t.Fatalf("回退链应取前 3 条推进点: %q", sum.Content)
	}

	// 回退 3：推进点也空 → 正文前 300 字
	v2.PlotPoints = nil
	mems = ExtractStoryMemories(1, "001.md", v2, content)
	for _, m := range mems {
		if m.Type == types.MemoryTypeChapterSummary && m.TextLen != memorySummaryFallbackLen {
			t.Fatalf("正文兜底应截 300 rune，实际 %d", m.TextLen)
		}
	}

	// 三级全空：无 summary 记忆（不造数）；nil 载荷返回 nil
	for _, m := range ExtractStoryMemories(1, "001.md", &types.AnalysisResultV2{}, "") {
		if m.Type == types.MemoryTypeChapterSummary {
			t.Fatalf("三级全空不应有 chapter_summary: %+v", m)
		}
	}
	if got := ExtractStoryMemories(1, "001.md", nil, content); got != nil {
		t.Fatalf("nil 载荷应返回 nil")
	}
}
