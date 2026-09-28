package project

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── 章节计划表 chapters/plans.json（规格 §7.1-2）───────────────

func planFixture12() types.ChapterPlan {
	return types.ChapterPlan{
		SubIndex:       12,
		Title:          "第12章·月下的交易",
		PlotSummary:    "主角在废墟与旧神使徒达成交易，代价是遗忘母亲的脸。",
		KeyEvents:      []string{"旧神使徒现身废墟", "交易以记忆为代价成交"},
		CharacterFocus: []string{"主角", "旧神使徒"},
		EmotionalTone:  "压抑中转希望",
		NarrativeGoal:  "让主角第一次主动触碰旧神体系",
		ConflictType:   "人vs超自然",
		EndingType:     string(types.EndingSuspense),
		EstimatedWords: 3000,
	}
}

func TestChapterPlans_RoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	// 章号键必须是十进制字符串（"12"，不是 "012"）。
	if want := filepath.Join(dir, "chapters", "plans.json"); pm.ChapterPlansPath() != want {
		t.Fatalf("路径不符: got %s want %s", pm.ChapterPlansPath(), want)
	}

	p12 := planFixture12()
	p13 := types.ChapterPlan{
		SubIndex:       13,
		Title:          "第13章·空碗",
		KeyEvents:      []string{"记忆反噬", "同伴怀疑主角"},
		CharacterFocus: []string{"主角"},
		EndingType:     string(types.EndingTwist),
		NarrativeGoal:  "把代价具象化",
		ConflictType:   "人vs自我",
	}
	// Version 故意留 0 → 写入时自动补 1。
	if err := pm.WriteChapterPlans(&types.ChapterPlanFile{
		Plans: map[string]types.ChapterPlan{"12": p12, "13": p13},
	}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	got, err := pm.ReadChapterPlans()
	if err != nil {
		t.Fatalf("读回失败: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("Version 应自动补 1: got %d", got.Version)
	}
	if len(got.Plans) != 2 {
		t.Fatalf("计划条数不符: %+v", got.Plans)
	}
	if !reflect.DeepEqual(got.Plans["12"], p12) {
		t.Fatalf("第12章计划往返不一致:\n got %+v\nwant %+v", got.Plans["12"], p12)
	}
	if !reflect.DeepEqual(got.Plans["13"], p13) {
		t.Fatalf("第13章计划往返不一致:\n got %+v\nwant %+v", got.Plans["13"], p13)
	}

	// 窄方法：存在 → (plan, true, nil)
	one, ok, err := pm.ReadChapterPlan(12)
	if err != nil || !ok {
		t.Fatalf("ReadChapterPlan(12): ok=%v err=%v", ok, err)
	}
	if !reflect.DeepEqual(*one, p12) {
		t.Fatalf("窄方法读回不一致:\n got %+v\nwant %+v", *one, p12)
	}
	// 未登记章 → (nil, false, nil)，不靠 error 表达
	if p, ok, err := pm.ReadChapterPlan(99); err != nil || ok || p != nil {
		t.Fatalf("未登记章应为 (nil,false,nil): p=%+v ok=%v err=%v", p, ok, err)
	}

	assertNoTempLeftovers(t, filepath.Join(dir, "chapters"))
}

func TestChapterPlans_MissingIsEmptyNormalState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	f, err := pm.ReadChapterPlans()
	if err != nil {
		t.Fatalf("文件缺失应为正常态，却报错: %v", err)
	}
	if f == nil || f.Version != 1 || f.Plans == nil || len(f.Plans) != 0 {
		t.Fatalf("文件缺失应返回空结构: %+v", f)
	}
	// 空表可安全写回（nil map 陷阱守卫）
	if err := pm.WriteChapterPlans(f); err != nil {
		t.Fatalf("空表写回失败: %v", err)
	}
	if p, ok, err := pm.ReadChapterPlan(12); err != nil || ok || p != nil {
		t.Fatalf("空表应无第12章: p=%+v ok=%v err=%v", p, ok, err)
	}
}

func TestChapterPlans_CorruptFileNotOverwritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	path := pm.ChapterPlansPath()
	corrupt := []byte(`{"version":1,"plans":{"12":{"title":"半写的档`)
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := pm.ReadChapterPlans(); err == nil {
		t.Fatal("损坏文件必须报错")
	}
	if _, _, err := pm.ReadChapterPlan(12); err == nil {
		t.Fatal("损坏文件下 ReadChapterPlan 必须报错（不得静默当作不存在）")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("原文件应仍在: %v", err)
	}
	if string(after) != string(corrupt) {
		t.Fatalf("损坏文件被改动（绝不允许覆盖）:\n got %s\nwant %s", after, corrupt)
	}
}

func TestChapterPlans_WriteNilIsEmptyTable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm := &Manager{Dir: dir} // 未经 Create：chapters 目录不存在，写入应自动创建

	if err := pm.WriteChapterPlans(nil); err != nil {
		t.Fatalf("nil 应视为空表写入: %v", err)
	}
	f, err := pm.ReadChapterPlans()
	if err != nil {
		t.Fatalf("读回失败: %v", err)
	}
	if f.Version != 1 || len(f.Plans) != 0 {
		t.Fatalf("空表内容不符: %+v", f)
	}
	raw, err := os.ReadFile(pm.ChapterPlansPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"plans"`) {
		t.Fatalf("空表也必须写出 plans 键（不能是 null）: %s", raw)
	}
	if strings.Contains(string(raw), `"plans": null`) {
		t.Fatalf("plans 不得写成 null: %s", raw)
	}
	assertNoTempLeftovers(t, filepath.Join(dir, "chapters"))
}

// TestChapterPlans_KeyIsDecimalNotPadded 锁定键口径：十进制章节号字符串。
// 只想登记的章号能找到，补零键（"012"）不是同一章。
func TestChapterPlans_KeyIsDecimalNotPadded(t *testing.T) {
	dir := t.TempDir()
	pm := &Manager{Dir: dir}
	if err := pm.WriteChapterPlans(&types.ChapterPlanFile{
		Plans: map[string]types.ChapterPlan{"012": {Title: "补零键（非契约口径）"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := pm.ReadChapterPlan(12); err != nil || ok {
		t.Fatalf("补零键不应被当作第12章: ok=%v err=%v", ok, err)
	}

	if err := pm.WriteChapterPlans(&types.ChapterPlanFile{
		Plans: map[string]types.ChapterPlan{"12": {Title: "第12章"}},
	}); err != nil {
		t.Fatal(err)
	}
	p, ok, err := pm.ReadChapterPlan(12)
	if err != nil || !ok || p.Title != "第12章" {
		t.Fatalf("十进制键 12 应命中第12章: p=%+v ok=%v err=%v", p, ok, err)
	}
}

// ── 计划偏差 analysis/plan-deviation/<MMM>.json（规格 §7.4）────

func deviationFixture12() types.PlanDeviation {
	return types.PlanDeviation{
		ChapterNum:      12,
		HasPlan:         true,
		Analyzed:        true,
		MissingEvents:   []string{"交易以记忆为代价成交"},
		EndingMismatch:  true,
		PlannedEnding:   "悬念",
		ActualEnding:    "自然过渡",
		EmotionDrift:    false,
		PlannedEmotion:  "压抑中转希望",
		ActualEmotion:   "平稳",
		DuplicateEvents: []string{"月下独白"},
		Summary:         "关键交易被淡化，结尾未留钩子",
		NextSuggestion:  "第13章补一笔记忆反噬的代价",
	}
}

func TestPlanDeviation_RoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	if want := filepath.Join(dir, "analysis", "plan-deviation", "012.json"); pm.PlanDeviationPath(12) != want {
		t.Fatalf("路径不符: got %s want %s", pm.PlanDeviationPath(12), want)
	}

	pd := deviationFixture12()
	if err := pm.WritePlanDeviation(&pd); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	got, ok, err := pm.ReadPlanDeviation(12)
	if err != nil || !ok {
		t.Fatalf("读回: ok=%v err=%v", ok, err)
	}
	if !reflect.DeepEqual(*got, pd) {
		t.Fatalf("偏差往返不一致:\n got %+v\nwant %+v", *got, pd)
	}
	// 章号三位补零，与章号两位数区分
	if pd1 := pm.PlanDeviationPath(1); filepath.Base(pd1) != "001.json" {
		t.Fatalf("章号应三位补零: %s", pd1)
	}
	assertNoTempLeftovers(t, filepath.Join(dir, "analysis", "plan-deviation"))
}

func TestPlanDeviation_MissingIsNormalState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	pd, ok, err := pm.ReadPlanDeviation(12)
	if err != nil {
		t.Fatalf("文件缺失应为正常态，却报错: %v", err)
	}
	if ok || pd != nil {
		t.Fatalf("缺失应为 (nil,false): pd=%+v ok=%v", pd, ok)
	}
}

func TestPlanDeviation_CorruptFileNotOverwritten(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	path := pm.PlanDeviationPath(12)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte(`{"chapterNum":12,"missingEvents":[`)
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := pm.ReadPlanDeviation(12); err == nil {
		t.Fatal("损坏文件必须报错")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("原文件应仍在: %v", err)
	}
	if string(after) != string(corrupt) {
		t.Fatalf("损坏文件被改动（绝不允许覆盖）:\n got %s\nwant %s", after, corrupt)
	}
	assertNoTempLeftovers(t, filepath.Join(dir, "analysis", "plan-deviation"))
}

// TestPlanDeviation_WriteRejectsBadInput 无主档守卫：nil / 章号<1 直接报错，
// 不产生 000.json 之类脏档。
func TestPlanDeviation_WriteRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	pm := &Manager{Dir: dir}
	if err := pm.WritePlanDeviation(nil); err == nil {
		t.Fatal("nil 偏差应报错")
	}
	if err := pm.WritePlanDeviation(&types.PlanDeviation{ChapterNum: 0}); err == nil {
		t.Fatal("章号 0 应报错")
	}
	if _, err := os.Stat(pm.PlanDeviationPath(0)); !os.IsNotExist(err) {
		t.Fatalf("不应产生 000.json: %v", err)
	}
}
