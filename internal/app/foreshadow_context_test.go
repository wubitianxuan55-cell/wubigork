package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newForeshadowTestProject 创建伏笔分层注入测试项目。
func newForeshadowTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "伏笔分层测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return pm
}

// writeForeshadows 测试助手：写入伏笔文件。
func writeForeshadows(t *testing.T, pm *project.Manager, items ...types.Foreshadow) {
	t.Helper()
	if err := pm.WriteForeshadows(&types.ForeshadowFile{SchemaVersion: 2, Items: items}); err != nil {
		t.Fatalf("写入伏笔: %v", err)
	}
}

// countEntryLines 统计区段中以 "- " 开头的条目行数。
func countSectionEntries(section string) int {
	n := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "- ") {
			n++
		}
	}
	return n
}

// entriesUnderHeader 取某层标题之后的条目数（"- " 开头的行为条目首行，
// 续行缩进两空格不算；遇到下一层标题「【」或区段头停止）。
func entriesUnderHeader(section, header string) int {
	i := strings.Index(section, header)
	if i < 0 {
		return -1
	}
	n := 0
	for _, line := range strings.Split(section[i+len(header):], "\n") {
		if strings.HasPrefix(line, "【") || strings.HasPrefix(line, "## ") {
			break
		}
		if strings.HasPrefix(line, "- ") {
			n++
		}
	}
	return n
}

// TestForeshadowSectionFourLayers 四层各就各位：本章必须回收 / 超期 /
// 近期参考 / 本章计划埋入 / 无计划兜底，空层标题不出现。
func TestForeshadowSectionFourLayers(t *testing.T) {
	pm := newForeshadowTestProject(t)
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "must1", Title: "古剑胎记", Description: "主角背上有一道古剑胎记", PlantedIn: "001.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted, ResolutionNotes: "识破血脉时揭晓"},
		types.Foreshadow{ID: "od1", Title: "商队谜团", Description: "酒馆老板总在打听商队路线", PlantedIn: "002.md", TargetResolveIn: "003.md", Status: types.ForeshadowPlanted},
		types.Foreshadow{ID: "near1", Title: "北境传闻", Description: "北境冰湖下封印的旧神", PlantedIn: "001.md", TargetResolveIn: "009.md", Status: types.ForeshadowPlanted},
		types.Foreshadow{ID: "plan1", Title: "入门试炼", Description: "宗门大比将开，主角需-hidden-通过试炼", PlantedIn: "005.md", Status: types.ForeshadowPending, HintText: "以暗线提及报名帖"},
		types.Foreshadow{ID: "np1", Description: "无计划的旧伏笔", PlantedIn: "001.md", Status: types.ForeshadowPlanted},
	)
	got := buildForeshadowSection(pm, 5)
	if got == "" {
		t.Fatal("有伏笔时区段不应为空")
	}
	if !strings.Contains(got, "## 伏笔调度（分层约束）") || !strings.Contains(got, "调度规则") {
		t.Fatalf("区段头与调度规则缺失:\n%s", got)
	}
	for _, want := range []string{
		"【🎯 本章必须回收的伏笔 - 请务必在本章完成回收】",
		"- ID:must1 | 古剑胎记",
		"伏笔内容：主角背上有一道古剑胎记",
		"回收提示：识破血脉时揭晓",
		"【⚠️ 超期待回收伏笔 - 请尽快回收】",
		"- ID:od1 | 商队谜团 [已超期2章]",
		"原计划第3章回收",
		"【📋 近期待回收伏笔（仅供参考，请勿在本章回收）】",
		"- 北境传闻（计划第9章回收，还有4章）",
		"本章请勿提前回收",
		"【✨ 本章计划埋入伏笔】",
		"- 入门试炼",
		"埋入提示：以暗线提及报名帖",
		"【📌 已埋入未规划回收章的伏笔（背景约束）】",
		"- ID:np1 | 无计划的旧伏笔（埋入第1章）",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("缺少 %q，实际为:\n%s", want, got)
		}
	}
}

// TestForeshadowSectionLayerLimits L2 ≤3 条、L3 ≤5 条（spec §4.1），
// 条目总量封顶 ctxForeshadowMaxItems。
func TestForeshadowSectionLayerLimits(t *testing.T) {
	pm := newForeshadowTestProject(t)
	items := make([]types.Foreshadow, 0, 20)
	for i := 0; i < 7; i++ { // 7 条超期 → 只出 3 条
		items = append(items, types.Foreshadow{
			ID: "od", Title: "超期伏笔", Description: "超期内容",
			PlantedIn: "001.md", TargetResolveIn: "003.md", Status: types.ForeshadowPlanted,
		})
	}
	for i := 0; i < 8; i++ { // 8 条近期 → 只出 5 条
		items = append(items, types.Foreshadow{
			ID: "near", Title: "近期伏笔", Description: "近期内容",
			PlantedIn: "001.md", TargetResolveIn: "008.md", Status: types.ForeshadowPlanted,
		})
	}
	writeForeshadows(t, pm, items...)
	got := buildForeshadowSection(pm, 5)
	if n := entriesUnderHeader(got, "【⚠️ 超期待回收伏笔 - 请尽快回收】"); n != fsLayerOverdueMax {
		t.Errorf("超期层应封顶 %d 条, got %d:\n%s", fsLayerOverdueMax, n, got)
	}
	if n := entriesUnderHeader(got, "【📋 近期待回收伏笔（仅供参考，请勿在本章回收）】"); n != fsLayerNearMax {
		t.Errorf("近期层应封顶 %d 条, got %d:\n%s", fsLayerNearMax, n, got)
	}
	if n := countSectionEntries(got); n > ctxForeshadowMaxItems {
		t.Errorf("条目总量应封顶 %d, got %d", ctxForeshadowMaxItems, n)
	}
	if n := len([]rune(got)); n > ctxForeshadowBudget {
		t.Errorf("整区（含标题）应不超过 %d rune, got %d", ctxForeshadowBudget, n)
	}
}

// TestForeshadowSectionNoFarFutureAndDup 远期（> cur+lookahead）不注入；
// 同一条目只属于一层（target==cur 只在 L1，target<cur 只在 L2）。
func TestForeshadowSectionNoFarFutureAndDup(t *testing.T) {
	pm := newForeshadowTestProject(t)
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "far1", Title: "远期伏笔", Description: "很远之后才回收", PlantedIn: "001.md", TargetResolveIn: "030.md", Status: types.ForeshadowPlanted},
		types.Foreshadow{ID: "edge1", Title: "本章到期", Description: "本章必须回收的", PlantedIn: "001.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted},
	)
	got := buildForeshadowSection(pm, 5)
	if strings.Contains(got, "远期伏笔") {
		t.Errorf("远期伏笔不应注入:\n%s", got)
	}
	if strings.Count(got, "edge1") != 1 {
		t.Errorf("同一条目不得跨层重复出现:\n%s", got)
	}
	if !strings.Contains(got, "本章必须回收的伏笔") {
		t.Fatalf("target==cur 应在必须回收层:\n%s", got)
	}
}

// TestForeshadowSectionIncludeAndRemindSwitches include_in_context=false 全排除；
// auto_remind=false 只抑制近期参考层（D8），不抑制必须回收/超期。
func TestForeshadowSectionIncludeAndRemindSwitches(t *testing.T) {
	pm := newForeshadowTestProject(t)
	off := false
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "hidden", Title: "被排除的伏笔", Description: "作者显式排除", PlantedIn: "001.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted, IncludeInContext: &off},
		types.Foreshadow{ID: "quietnear", Title: "静音近期伏笔", Description: "不提醒但内容存在", PlantedIn: "001.md", TargetResolveIn: "008.md", Status: types.ForeshadowPlanted, AutoRemind: &off},
	)
	got := buildForeshadowSection(pm, 5)
	if strings.Contains(got, "被排除的伏笔") {
		t.Errorf("include_in_context=false 应全层排除:\n%s", got)
	}
	if strings.Contains(got, "静音近期伏笔") {
		t.Errorf("auto_remind=false 应抑制近期参考层:\n%s", got)
	}
	if got != "" {
		t.Fatalf("全部被抑制时区段应为空:\n%s", got)
	}

	// auto_remind=false 但本章到期 → 必须回收层照出（硬约束不消隐）
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "quietmust", Title: "静音但到期", Description: "本章必须回收", PlantedIn: "001.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted, AutoRemind: &off},
	)
	got = buildForeshadowSection(pm, 5)
	if !strings.Contains(got, "静音但到期") || !strings.Contains(got, "本章必须回收的伏笔") {
		t.Errorf("auto_remind=false 不得抑制必须回收层:\n%s", got)
	}
}

// TestForeshadowSectionEmptyLayersOmitted 只有无计划条目时不输出其它层标题，
// 且无数据/读失败静默返回 ""。
func TestForeshadowSectionEmptyLayersOmitted(t *testing.T) {
	pm := newForeshadowTestProject(t)
	if got := buildForeshadowSection(pm, 5); got != "" {
		t.Fatalf("无伏笔应返回空:\n%s", got)
	}
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "np1", Description: "唯一无计划条目", PlantedIn: "001.md", Status: types.ForeshadowPlanted},
	)
	got := buildForeshadowSection(pm, 5)
	for _, absent := range []string{
		"本章必须回收的伏笔", "超期待回收伏笔", "近期待回收伏笔", "本章计划埋入伏笔",
	} {
		if strings.Contains(got, absent) {
			t.Errorf("空层标题 %q 不应输出:\n%s", absent, got)
		}
	}
	if !strings.Contains(got, "已埋入未规划回收章的伏笔") {
		t.Fatalf("无计划层应输出:\n%s", got)
	}
}

// TestResolveTargetChapterNum 章号解析三路：显式指定 / 分支父节点 / 顺延新章。
func TestResolveTargetChapterNum(t *testing.T) {
	of := &types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 3},
		{ID: "n2", OrderIndex: 7},
	}}
	if got := resolveTargetChapterNum(of, 12, ""); got != 12 {
		t.Errorf("显式章号应直通: got %d", got)
	}
	if got := resolveTargetChapterNum(of, 0, "n2"); got != 7 {
		t.Errorf("分支续写应取父节点章号: got %d", got)
	}
	if got := resolveTargetChapterNum(of, 0, "missing"); got != 0 {
		t.Errorf("父节点缺失应返回 0: got %d", got)
	}
	if got := resolveTargetChapterNum(of, 0, ""); got != 3 {
		t.Errorf("无指定应顺延为 len+1: got %d", got)
	}
}

// TestForeshadowSectionImportanceOrder L1 层内按重要性降序（缺省 0.5 沉底）。
func TestForeshadowSectionImportanceOrder(t *testing.T) {
	pm := newForeshadowTestProject(t)
	writeForeshadows(t, pm,
		types.Foreshadow{ID: "low", Title: "低重要性", Description: "低", PlantedIn: "001.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted},
		types.Foreshadow{ID: "high", Title: "高重要性", Description: "高", PlantedIn: "002.md", TargetResolveIn: "005.md", Status: types.ForeshadowPlanted, Importance: 0.9},
	)
	got := buildForeshadowSection(pm, 5)
	if strings.Index(got, "high") > strings.Index(got, "low") {
		t.Errorf("重要性降序应 high 在前:\n%s", got)
	}
}
