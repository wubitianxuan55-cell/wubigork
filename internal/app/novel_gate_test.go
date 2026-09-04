package app

import (
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 章节生成门（GenerationGate）测试 ───────────────────────────

// newGateEmptyApp 构造一个「无 agent、client nil」的最小 App（writingState 已初始化，pm 未设）。
func newGateEmptyApp() *App {
	a := &App{core: &core{}}
	a.writingState = &writingState{core: a.core, app: a}
	return a
}

// newGateProject 在临时目录创建一个真实小说项目。
func newGateProject(t *testing.T) *project.Manager {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "门禁测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	return pm
}

// TestRunChapterGate_NoProject getPM()==nil → 返回 error。
func TestRunChapterGate_NoProject(t *testing.T) {
	a := newGateEmptyApp()
	// 未设置 pm：getPM() 返回 nil
	if pm := a.getPM(); pm != nil {
		t.Fatalf("getPM() 应为 nil，得到 %v", pm)
	}

	_, err := a.RunChapterGate(1)
	if err == nil {
		t.Fatalf("无项目时应返回 error，得到 nil")
	}
	t.Logf("预期错误: %v", err)
}

// TestRunChapterGate_AllFail 无 agent、client nil（全部 AI 路失败/缺省），
// 仅有一章正文 + 一章摘要 → 不 panic、返回含四键的合并报告。
func TestRunChapterGate_AllFail(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)

	if err := pm.WriteChapter(1, "第一章正文：主角在夜色中走入青云宗。"); err != nil {
		t.Fatalf("写章节失败: %v", err)
	}
	if err := pm.WriteChapterSummary(1, &types.ChapterSummary{
		Title:   "第一章 初入青云",
		Summary: "主角叶辰初入青云宗，结识同门，埋下宗门大比伏笔。",
	}); err != nil {
		t.Fatalf("写章节摘要失败: %v", err)
	}

	a.setPM(pm)

	result, err := a.RunChapterGate(1)
	if err != nil {
		t.Fatalf("RunChapterGate(1) 不应返回 error: %v", err)
	}
	if result == nil {
		t.Fatalf("返回结果不应为 nil")
	}
	if result["chapterNum"] != 1 {
		t.Errorf("chapterNum 应为 1，得到 %v", result["chapterNum"])
	}

	for _, k := range []string{"analysis", "review", "consistency", "aiTaste"} {
		if _, ok := result[k]; !ok {
			t.Errorf("结果应包含键 %q，缺失", k)
		}
	}

	// 无 agent / client nil 时，analysis 与 review 应诚实缺省为 nil
	if result["analysis"] != nil {
		t.Errorf("无 analysisAgent 时 analysis 应为 nil，得到 %v", result["analysis"])
	}
	if result["review"] != nil {
		t.Errorf("无 chapterAgent 时 review 应为 nil，得到 %v", result["review"])
	}
	// consistency 走规则层（空项目应安全返回非 nil）；aiTaste 为确定性打分（非 nil）
	if result["consistency"] == nil {
		t.Errorf("consistency 应返回规则层结果（空项目安全），得到 nil")
	}
	if result["aiTaste"] == nil {
		t.Errorf("aiTaste 应返回确定性打分，得到 nil")
	}
}
