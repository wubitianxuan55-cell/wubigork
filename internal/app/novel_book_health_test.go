package app

// GenerationGate 闭环收口·线 B 测试（规格 进度计划/gaea-gen-gate-closure-20260916.md §3）：
// 自动门（分支跳分析/agent 缺省降级/确定性三路必出）+ 全书体检（干净+脏章聚合、
// 空书零章）。fixture：newFingerprintTestApp 项目 + analysis.New(nil,...) 的
// nil-client agent（Analyze 走 error 路径=降级可测）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// mustAnalysisAgentNoEngine 装配「无可用引擎」的分析 agent（真 client+真模板
// 引擎；Analyze 的 LLM 调用走 error 返回 → 降级路径可测——nil client 会在
// agent 内部 nil 解引用 panic，不可用）。
func mustAnalysisAgentNoEngine(t *testing.T, a *App) {
	t.Helper()
	cfg := &config.Config{}
	a.analysisAgent = analysis.New(ai.NewClient(cfg), a.getPM(), cfg, prompt.NewEngine("../../prompts"))
}

func TestAutoGateReport_DeterministicPathsAlwaysPresent(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteChapter(t, a, 1, "干净样本。")
	pm := a.getPM()
	content, err := pm.ReadChapter(1)
	if err != nil {
		t.Fatalf("读章: %v", err)
	}
	// 无 agent：确定性三路仍必出，分析路降级 false
	report := a.buildAutoGateReport(pm, 1, content, "")
	if report["analysisDone"] != false {
		t.Fatalf("无 agent 应降级 false: %+v", report)
	}
	if _, ok := report["outlineIssues"].(int); !ok {
		t.Fatalf("outlineIssues 应为 int: %+v", report)
	}
	if _, ok := report["qualityIssues"].(int); !ok {
		t.Fatalf("qualityIssues 应为 int: %+v", report)
	}
	if report["aiTasteScore"] == nil {
		t.Fatalf("干净长文本应有 AI 味分: %+v", report)
	}
}

func TestAutoGateReport_BranchSkipsAnalysis(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustAnalysisAgentNoEngine(t, a)
	mustWriteChapter(t, a, 1, "分支正文样本。")
	pm := a.getPM()
	content, _ := pm.ReadChapter(1)
	report := a.buildAutoGateReport(pm, 1, content, "B1")
	if report["branch"] != "B1" || report["analysisDone"] != false {
		t.Fatalf("分支章应跳分析路: %+v", report)
	}
}

func TestAutoGateReport_MainlineAgentFailureDegrades(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustAnalysisAgentNoEngine(t, a) // nil client → Analyze error → analysisDone=false 不 panic
	mustWriteChapter(t, a, 1, "主线正文样本。")
	pm := a.getPM()
	content, _ := pm.ReadChapter(1)
	report := a.buildAutoGateReport(pm, 1, content, "")
	if report["analysisDone"] != false {
		t.Fatalf("分析失败应降级 false: %+v", report)
	}
	if report["foreshadowSync"] != nil {
		t.Fatalf("分析未成不应带同步结果: %+v", report)
	}
}

func TestBookHealthCheck_CleanAndDirtyChapters(t *testing.T) {
	a := newFingerprintTestApp(t)
	// 第 1 章：干净（mustWriteChapter 样本句式有起伏，质量路应零问题）
	mustWriteChapter(t, a, 1, "干净样本。")
	pm := a.getPM()
	c1, _ := pm.ReadChapter(1)

	// 第 2 章：脏——超长段落（>400 rune 一段）触发 S3
	var sb strings.Builder
	for i := 0; i < 300; i++ {
		sb.WriteString("他缓缓地走向前方那扇门，门后的黑暗仿佛在呼吸。")
	}
	if err := pm.WriteChapter(2, sb.String()); err != nil {
		t.Fatalf("写脏章: %v", err)
	}

	// 第 1 章给一个「只有标题」的大纲节点：summary/keypoints/emotion 缺项 →
	// 契约问题计入该章（S2×1+S3+S4=3 条，error 级 S2=1）；第 2 章无节点=无契约可违。
	of := &types.OutlineFile{}
	of.Nodes = []types.OutlineNode{{ID: "n1", Title: "第一章", ChapterFile: "001.md"}}
	if err := pm.WriteOutlines(of); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	report, err := a.RunBookHealthCheck()
	if err != nil {
		t.Fatalf("体检: %v", err)
	}
	if report.TotalChapters != 2 || len(report.Chapters) != 2 {
		t.Fatalf("章数不对: %+v", report)
	}
	row1, row2 := report.Chapters[0], report.Chapters[1]
	if row1.Words != len([]rune(c1)) {
		t.Fatalf("字数 rune 口径不对: %d vs %d", row1.Words, len([]rune(c1)))
	}
	if row1.QualityIssues != 0 {
		t.Fatalf("干净章不应有质量问题: %+v", row1)
	}
	if row2.QualityIssues == 0 || row2.QualityErrors > 0 {
		t.Fatalf("超长段落应触发 S3（warn 级非 error）: %+v", row2)
	}
	if row1.AiTasteScore < 0 || row2.AiTasteScore < 0 {
		t.Fatalf("AI 味分应有值: %+v %+v", row1, row2)
	}
	// 第 1 章大纲缺项（summary/keypoints/emotion）→ 契约问题章=1；第 2 章无
	// 节点=无契约可违不计。
	if report.ContractIssueChapters != 1 || row1.OutlineIssues != 3 || row1.OutlineErrors != 1 {
		t.Fatalf("契约计数不对: %+v / row1=%+v", report, row1)
	}
	if row2.OutlineIssues != 0 {
		t.Fatalf("无大纲节点章不应有契约问题: %+v", row2)
	}
	if report.QualityIssueChapters != 1 {
		t.Fatalf("只有第 2 章有质量问题: %+v", report)
	}
}

func TestBookHealthCheck_AnalysisCoverageAndEmptyBook(t *testing.T) {
	a := newFingerprintTestApp(t)
	// 空书：零章报告不算错
	report, err := a.RunBookHealthCheck()
	if err != nil || report.TotalChapters != 0 || report.Chapters == nil {
		t.Fatalf("空书应零章非 nil: %+v err=%v", report, err)
	}

	// V2 覆盖率：写一章 + 落一条 V2 → AnalyzedChapters=1
	mustWriteChapter(t, a, 1, "覆盖率样本。")
	if err := a.getPM().UpsertAnalysisV2(types.ChapterAnalysisResult{ChapterNum: 1}); err != nil {
		t.Fatalf("落 V2: %v", err)
	}
	report, err = a.RunBookHealthCheck()
	if err != nil {
		t.Fatalf("体检: %v", err)
	}
	if report.AnalyzedChapters != 1 {
		t.Fatalf("V2 覆盖应为 1: %+v", report)
	}
}
