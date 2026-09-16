package app

// t7 前端接线·线 B 测试（规格 进度计划/gaea-analysis-v2-panel-t7-20260916.md §3）：
// NovelChapterAnalysisV2 三例——命中回九维+meta 直连 / 无条目诚实报错 / 无分析
// agent 报错。fixture：analysisAgent 非_nil（foreshadow_view_test 先例）+
// UpsertAnalysisV2（novel_rewrite_handler_test 先例）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/types"
)

func newAnalysisReadApp(t *testing.T) *App {
	t.Helper()
	a := newFingerprintTestApp(t)
	a.analysisAgent = analysis.New(nil, a.getPM(), &config.Config{}, nil)
	return a
}

func TestNovelChapterAnalysisV2_HitReturnsFullResult(t *testing.T) {
	a := newAnalysisReadApp(t)
	mustWriteChapter(t, a, 2, "样本正文。")
	v2 := types.AnalysisResultV2{
		Hooks:       []types.Hook{{Type: "悬念", Content: "夜半敲门声", Strength: 7, Position: "开头"}},
		Suggestions: []string{"节奏可收紧"},
	}
	v2.Scores = types.AnalysisScores{Pacing: 8, Engagement: 7, Coherence: 9, Overall: 8}
	if err := a.getPM().UpsertAnalysisV2(types.ChapterAnalysisResult{
		ChapterNum: 2, ChapterFile: "002.md", AnalyzerSource: "llm", Engine: "herdsman", Model: "test-model", Result: v2,
	}); err != nil {
		t.Fatalf("落分析: %v", err)
	}
	got, err := a.NovelChapterAnalysisV2(2)
	if err != nil {
		t.Fatalf("读取: %v", err)
	}
	if got.ChapterNum != 2 || got.ChapterFile != "002.md" || got.Engine != "herdsman" || got.Model != "test-model" {
		t.Fatalf("meta 不对: %+v", got)
	}
	if len(got.Result.Hooks) != 1 || got.Result.Hooks[0].Content != "夜半敲门声" {
		t.Fatalf("九维直连不对: %+v", got.Result.Hooks)
	}
	if got.Result.Scores.Overall != 8 || len(got.Result.Suggestions) != 1 {
		t.Fatalf("评分/建议不对: %+v", got.Result.Scores)
	}
}

func TestNovelChapterAnalysisV2_MissingChapterErrors(t *testing.T) {
	a := newAnalysisReadApp(t)
	if _, err := a.NovelChapterAnalysisV2(9); err == nil || !strings.Contains(err.Error(), "尚未分析") {
		t.Fatalf("缺章应诚实报错，得到: %v", err)
	}
}

func TestNovelChapterAnalysisV2_NoAgentErrors(t *testing.T) {
	a := newFingerprintTestApp(t) // analysisAgent nil
	if _, err := a.NovelChapterAnalysisV2(1); err == nil || !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("无 agent 应报错，得到: %v", err)
	}
}
