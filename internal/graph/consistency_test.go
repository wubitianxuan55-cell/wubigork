package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 测试工具 ─────────────────────────────────────────────────

// newConsistencyTestProject 创建一个空的测试项目
func newConsistencyTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建测试项目失败: %v", err)
	}
	return pm
}

func writeCharacterFixture(t *testing.T, pm *project.Manager, id, name, status string) {
	t.Helper()
	err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{ID: id, Name: name, Status: status}},
	})
	if err != nil {
		t.Fatalf("写入角色失败: %v", err)
	}
}

func findIssue(report *ConsistencyReport, pred func(ConsistencyIssue) bool) *ConsistencyIssue {
	for i := range report.Issues {
		if pred(report.Issues[i]) {
			return &report.Issues[i]
		}
	}
	return nil
}

// ── 章节枚举 ─────────────────────────────────────────────────

func TestListChapterFiles_GapBranchAndFiltering(t *testing.T) {
	pm := newConsistencyTestProject(t)

	// 断档：主线 1,2,5 存在（3、4 缺失）；分支 3a、3b 存在
	chapters := []chapterFile{
		{num: 1, branch: ""},
		{num: 2, branch: ""},
		{num: 3, branch: "a"},
		{num: 3, branch: "b"},
		{num: 5, branch: ""},
	}
	for _, cf := range chapters {
		var err error
		if cf.branch == "" {
			err = pm.WriteChapter(cf.num, "正文")
		} else {
			err = pm.WriteChapterBranch(cf.num, cf.branch, "分支正文")
		}
		if err != nil {
			t.Fatalf("写入章节失败: %v", err)
		}
	}

	// 干扰项：摘要文件 / v4 场景子目录 / 杂项 md / 原子写残留临时文件，都不应算章节
	if err := pm.WriteChapterSummary(5, &types.ChapterSummary{Title: "摘要干扰项"}); err != nil {
		t.Fatalf("写入摘要失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pm.Dir, "chapters", "006"), 0755); err != nil {
		t.Fatalf("创建场景子目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "chapters", "notes.md"), []byte("x"), 0644); err != nil {
		t.Fatalf("写入杂项文件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "chapters", "007.md.tmp-123"), []byte("x"), 0644); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}

	got := listChapterFiles(pm)
	if len(got) != len(chapters) {
		t.Fatalf("expected %d chapter files, got %d: %+v", len(chapters), len(got), got)
	}
	for i, want := range chapters {
		if got[i] != want {
			t.Fatalf("got[%d] = %+v, want %+v", i, got[i], want)
		}
	}

	if got[4].Place() != "第5章" {
		t.Fatalf("主线位置标签错误: %s", got[4].Place())
	}
	if got[2].Place() != "第3章分支a" {
		t.Fatalf("分支位置标签错误: %s", got[2].Place())
	}
}

func TestListChapterFiles_NoChaptersDir(t *testing.T) {
	pm := newConsistencyTestProject(t)
	if got := listChapterFiles(pm); len(got) != 0 {
		t.Fatalf("expected empty list, got %+v", got)
	}
}

// ── Bug 修复回归：断档章节继续扫描 ──────────────────────────

func TestCheckConsistency_ContinuesAfterChapterGap(t *testing.T) {
	pm := newConsistencyTestProject(t)
	writeCharacterFixture(t, pm, "c1", "林凡", "Alive")

	// 章节 1、2、5 存在（3、4 断档）；仅第 5 章含死亡暗示且摘要含林凡
	if err := pm.WriteChapter(1, "林凡走在山道上。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapter(2, "林凡在殿中修炼。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapter(5, "林凡在决战中陨落，再也没有站起来。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterSummary(5, &types.ChapterSummary{
		Title:              "第五章",
		Summary:            "林凡决战落败。",
		CharactersAppeared: []string{"林凡"},
	}); err != nil {
		t.Fatal(err)
	}

	report, err := CheckConsistency(pm)
	if err != nil {
		t.Fatalf("CheckConsistency 失败: %v", err)
	}

	// 旧实现在第 3 章断档处 break，第 5 章永远不会被扫描；修复后必须出告警
	if len(report.Issues) != 1 {
		t.Fatalf("expected exactly 1 issue, got %d: %+v", len(report.Issues), report.Issues)
	}
	issue := report.Issues[0]
	if issue.Category != "status" || issue.EntityName != "林凡" {
		t.Fatalf("unexpected issue: %+v", issue)
	}
	if issue.Severity != "warning" {
		t.Fatalf("expected warning, got %s", issue.Severity)
	}
	if issue.Location != "第5章" {
		t.Fatalf("expected 第5章, got %s", issue.Location)
	}
	// 主线告警文案保持旧行为，Branch 为空
	if issue.Description != "林凡 在第5章似乎死亡，但当前状态仍为 Alive" {
		t.Fatalf("unexpected description: %s", issue.Description)
	}
	if issue.Branch != "" {
		t.Fatalf("主线告警 Branch 应为空, got %q", issue.Branch)
	}
}

// ── Bug 修复回归：分支章节纳入扫描并带标记 ──────────────────

func TestCheckConsistency_BranchChapterScannedAndMarked(t *testing.T) {
	pm := newConsistencyTestProject(t)
	writeCharacterFixture(t, pm, "c1", "林凡", "Alive")

	// 主线第 1 章无异常；分支 1a 有死亡暗示（旧实现完全不扫分支章节）
	if err := pm.WriteChapter(1, "林凡推开山门。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterBranch(1, "a", "林凡在分支线上力战而死。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterBranchSummary(1, "a", &types.ChapterSummary{
		Title:              "第一章（分支a）",
		Summary:            "分支线开场。",
		CharactersAppeared: []string{"林凡"},
	}); err != nil {
		t.Fatal(err)
	}

	report, err := CheckConsistency(pm)
	if err != nil {
		t.Fatalf("CheckConsistency 失败: %v", err)
	}

	if len(report.Issues) != 1 {
		t.Fatalf("expected exactly 1 issue, got %d: %+v", len(report.Issues), report.Issues)
	}
	issue := report.Issues[0]
	if issue.Branch != "a" {
		t.Fatalf("分支告警应带分支标记 a, got %q", issue.Branch)
	}
	if issue.Location != "第1章分支a" {
		t.Fatalf("expected 第1章分支a, got %s", issue.Location)
	}
	if issue.Description != "林凡 在第1章分支a似乎死亡，但当前状态仍为 Alive" {
		t.Fatalf("unexpected description: %s", issue.Description)
	}
}

// ── 分支不与主线混判 ─────────────────────────────────────────

func TestCheckConsistency_DeadCharacterBranchNotMergedWithMain(t *testing.T) {
	pm := newConsistencyTestProject(t)
	writeCharacterFixture(t, pm, "c1", "苏婉", "Dead")

	// 主线只出场 1 次（第1章）；分支 a 出场 2 次（1a、2a）
	if err := pm.WriteChapter(1, "苏婉守着灯。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterBranch(1, "a", "苏婉走在长街上。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapterBranch(2, "a", "苏婉再次出现。"); err != nil {
		t.Fatal(err)
	}
	appearances := []chapterFile{
		{num: 1, branch: ""},
		{num: 1, branch: "a"},
		{num: 2, branch: "a"},
	}
	for _, cf := range appearances {
		sum := &types.ChapterSummary{Summary: "出场。", CharactersAppeared: []string{"苏婉"}}
		var err error
		if cf.branch == "" {
			err = pm.WriteChapterSummary(cf.num, sum)
		} else {
			err = pm.WriteChapterBranchSummary(cf.num, cf.branch, sum)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	report, err := CheckConsistency(pm)
	if err != nil {
		t.Fatalf("CheckConsistency 失败: %v", err)
	}

	// 分支线内多次出场 → 带分支标记的 error
	branchIssue := findIssue(report, func(i ConsistencyIssue) bool {
		return i.EntityName == "苏婉" && i.Branch == "a"
	})
	if branchIssue == nil {
		t.Fatalf("分支线内 Dead 角色重复出场未告警: %+v", report.Issues)
	}
	if branchIssue.Severity != "error" {
		t.Fatalf("expected error, got %s", branchIssue.Severity)
	}
	if branchIssue.Location != "第1-2章（分支a）" {
		t.Fatalf("expected 第1-2章（分支a）, got %s", branchIssue.Location)
	}

	// 分支出场不得并入主线跨章结论：主线仅出场 1 次，不应有主线告警
	mainIssue := findIssue(report, func(i ConsistencyIssue) bool {
		return i.EntityName == "苏婉" && i.Branch == ""
	})
	if mainIssue != nil {
		t.Fatalf("分支出场不应作为主线依据: %+v", *mainIssue)
	}
}

func TestCheckConsistency_DeadCharacterMainLineStillDetected(t *testing.T) {
	pm := newConsistencyTestProject(t)
	writeCharacterFixture(t, pm, "c1", "苏婉", "Dead")

	// 主线出场 2 次（第 1、3 章，含断档）→ 旧行为保持：主线 error
	if err := pm.WriteChapter(1, "苏婉守着灯。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapter(3, "苏婉再次出现。"); err != nil {
		t.Fatal(err)
	}
	for _, num := range []int{1, 3} {
		if err := pm.WriteChapterSummary(num, &types.ChapterSummary{
			Summary:            "出场。",
			CharactersAppeared: []string{"苏婉"},
		}); err != nil {
			t.Fatal(err)
		}
	}

	report, err := CheckConsistency(pm)
	if err != nil {
		t.Fatalf("CheckConsistency 失败: %v", err)
	}

	issue := findIssue(report, func(i ConsistencyIssue) bool {
		return i.EntityName == "苏婉" && i.Category == "status"
	})
	if issue == nil {
		t.Fatalf("主线 Dead 角色跨章出场未告警: %+v", report.Issues)
	}
	if issue.Branch != "" {
		t.Fatalf("expected 主线告警 (Branch=\"\"), got %q", issue.Branch)
	}
	if issue.Location != "第1-3章" {
		t.Fatalf("expected 第1-3章, got %s", issue.Location)
	}
}
