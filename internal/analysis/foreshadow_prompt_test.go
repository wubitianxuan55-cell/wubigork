package analysis

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// boolP 快捷指针。
func boolP(v bool) *bool { return &v }

// TestRenderForeshadowCandidates_Layers t1-P2.3：三层各就各位，L1 逐条带
// 紧邻回填指令（MuMu plot_analyzer.py:239——分层注入第 1 层独有的关键）。
func TestRenderForeshadowCandidates_Layers(t *testing.T) {
	items := []types.Foreshadow{
		{ID: "f_must", Title: "古钟之谜", Description: "村口古钟每晚自鸣，钟腹藏物",
			PlantedIn: "001.md", TargetResolveIn: "005.md", HintText: "钟声里混着金属回响"},
		{ID: "f_overdue", Title: "旧伤", Description: "左臂旧伤",
			PlantedIn: "001.md", TargetResolveIn: "002.md"},
		{ID: "f_far", Title: "远期伏笔", Description: "北方边境的密信",
			PlantedIn: "002.md", TargetResolveIn: "030.md"},
		{ID: "f_noplan", Title: "无计划", Description: "祠堂的铜匣",
			PlantedIn: "003.md"},
	}
	got := RenderForeshadowCandidates(items, 5)

	// L1：本章必须回收（第 5 章计划回收的 f_must），带埋入章/内容/暗示/回填指令
	if !strings.Contains(got, "【🎯 本章必须回收的伏笔】") {
		t.Fatalf("缺 L1 标题:\n%s", got)
	}
	if !strings.Contains(got, "【ID: f_must】古钟之谜") ||
		!strings.Contains(got, "埋入章节：第1章") ||
		!strings.Contains(got, "埋入暗示：钟声里混着金属回响") {
		t.Fatalf("L1 条目详情不完整:\n%s", got)
	}
	// 紧邻指令与 ID 成对出现（验收：L1 条目带 ⚠️ 回收时 … 填写: {id}）
	if !strings.Contains(got, "⚠️ 回收时 reference_stable_id 填写: f_must") {
		t.Fatalf("L1 缺紧邻回填指令:\n%s", got)
	}

	// L2：超期（计划 002 < 当前 005）
	if !strings.Contains(got, "【⚠️ 超期未回收伏笔") ||
		!strings.Contains(got, "【ID: f_overdue】旧伤（第1章埋入，原计划第2章回收）") {
		t.Fatalf("L2 超期条目缺失或格式不对:\n%s", got)
	}

	// L3：远期（>当前）与无计划都归「其他」
	if !strings.Contains(got, "【📋 其他已埋入伏笔") ||
		!strings.Contains(got, "【ID: f_far】远期伏笔（第2章埋入）") ||
		!strings.Contains(got, "【ID: f_noplan】无计划（第3章埋入）") {
		t.Fatalf("L3 其他条目缺失:\n%s", got)
	}

	// 操作指引收尾
	if !strings.Contains(got, "type='resolved'") || !strings.Contains(got, "reference_stable_id") {
		t.Fatalf("缺操作指引:\n%s", got)
	}
}

// TestRenderForeshadowCandidates_Exclusions 排除口径：已回收/废弃/pending 不出现，
// include_in_context=false 不进 prompt 面；层内按埋入章升序确定性排序。
func TestRenderForeshadowCandidates_Exclusions(t *testing.T) {
	items := []types.Foreshadow{
		{ID: "f_done", Title: "已回收", Description: "x", PlantedIn: "001.md",
			TargetResolveIn: "005.md", Status: types.ForeshadowRevealed, RevealedIn: "004.md"},
		{ID: "f_drop", Title: "已废弃", Description: "x", PlantedIn: "001.md",
			TargetResolveIn: "005.md", Status: types.ForeshadowAbandoned},
		{ID: "f_pend", Title: "未埋入", Description: "x", PlantedIn: "005.md",
			Status: types.ForeshadowPending},
		{ID: "f_muted", Title: "作者排除", Description: "x", PlantedIn: "001.md",
			TargetResolveIn: "005.md", IncludeInContext: boolP(false)},
		{ID: "f_keep", Title: "保留", Description: "x", PlantedIn: "001.md",
			TargetResolveIn: "005.md"},
	}
	got := RenderForeshadowCandidates(items, 5)
	for _, gone := range []string{"f_done", "f_drop", "f_pend", "f_muted"} {
		if strings.Contains(got, gone) {
			t.Fatalf("%s 不应出现在候选清单:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "f_keep") {
		t.Fatalf("f_keep 应保留:\n%s", got)
	}
	// auto_remind=false 只抑制生成侧 L3 提醒（D8），分析侧候选不消隐
	items = append(items, types.Foreshadow{ID: "f_quiet", Title: "静默提醒", Description: "x",
		PlantedIn: "002.md", TargetResolveIn: "030.md", AutoRemind: boolP(false)})
	if !strings.Contains(RenderForeshadowCandidates(items, 5), "f_quiet") {
		t.Fatalf("auto_remind=false 不应在分析侧消隐（D8 只管生成侧 L3）")
	}
}

// TestRenderForeshadowCandidates_Caps 渲染折叠上限：超期 ≤5、其他 ≤10，
// 溢出如实注记不静默截断。
func TestRenderForeshadowCandidates_Caps(t *testing.T) {
	var items []types.Foreshadow
	for i := 1; i <= 7; i++ { // 7 条超期（计划章 002 < 当前 005）
		items = append(items, types.Foreshadow{
			ID: idOf("od", i), Title: idOf("超期", i), Description: "x",
			PlantedIn: "001.md", TargetResolveIn: "002.md",
		})
	}
	for i := 1; i <= 13; i++ { // 13 条无计划 → 其他
		items = append(items, types.Foreshadow{
			ID: idOf("np", i), Title: idOf("其他", i), Description: "x",
			PlantedIn: "003.md",
		})
	}
	got := RenderForeshadowCandidates(items, 5)

	if got := strings.Count(got, "od"); got != candidateOverdueMax { // 仅 ID 各含一次
		t.Fatalf("超期应折叠到 %d 条，出现 %d 个 id 片段", candidateOverdueMax, got)
	}
	if !strings.Contains(got, "还有2个超期伏笔未列出") {
		t.Fatalf("超期溢出注记缺失:\n%s", got)
	}
	if !strings.Contains(got, "还有3个伏笔未列出") {
		t.Fatalf("其他溢出注记缺失:\n%s", got)
	}
	if strings.Contains(got, "np13") {
		t.Fatalf("其他层应折叠到 10 条:\n%s", got)
	}
}

// TestRenderForeshadowCandidates_EmptyAndFallback 空态诚实说明；无标题条目回退
// 描述截断；截断按长度判断（D14：未超长不加省略号）。
func TestRenderForeshadowCandidates_EmptyAndFallback(t *testing.T) {
	if got := RenderForeshadowCandidates(nil, 1); !strings.Contains(got, "没有已埋入待回收的伏笔") {
		t.Fatalf("空态话术缺失: %q", got)
	}

	short := types.Foreshadow{ID: "f_s", Description: "短描述", PlantedIn: "001.md", TargetResolveIn: "001.md"}
	got := RenderForeshadowCandidates([]types.Foreshadow{short}, 1)
	if strings.Contains(got, "短描述...") {
		t.Fatalf("未超长不得加省略号:\n%s", got)
	}
	long := types.Foreshadow{ID: "f_l", Description: strings.Repeat("长", 300), PlantedIn: "001.md", TargetResolveIn: "001.md"}
	got = RenderForeshadowCandidates([]types.Foreshadow{long}, 1)
	if !strings.Contains(got, strings.Repeat("长", candidateContentMax)+"...") {
		t.Fatalf("超长应按 rune 截断 200 加省略号:\n%s", got)
	}
}

// idOf 生成 "前缀N" 形式 ID。
func idOf(prefix string, n int) string {
	return prefix + string(rune('0'+n%10)) + string(rune('a'+n%26))
}
