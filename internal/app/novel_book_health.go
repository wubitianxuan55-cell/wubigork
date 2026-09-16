package app

// GenerationGate 闭环收口（规格 进度计划/gaea-gen-gate-closure-20260916.md）：
//   1. runAutoGateAfterGeneration——生成 done 后异步自动门（确定性三路+分析路，
//      Q1-Q4）：让 t1-P2 伏笔自动回收、t3-P1 记忆回填、t4 V2 落盘三条已建成
//      管道在每次生成后真正过水；结果 emit chapter-gate 事件（V1 零前端消费，
//      t7 章节分析面板天然是消费面）；
//   2. RunBookHealthCheck——全书体检（作者点按，Q5/Q6）：纯确定性零 LLM，
//      逐章 outlineContract+质量+AI 味+字数聚合一份报告；伏笔 Lint 复用；
//      V2 覆盖率计数。无常驻、无定时、无夜间。

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
)

// ── 生成后自动门 ────────────────────────────────────────────

// runAutoGateAfterGeneration 生成 done 后的异步自动门（extractCharactersAfterChapter
// 同款 go 位，不阻塞 done 事件）：构建+分析+emit 都在 goroutine 内，任何失败只
// warn 降级，绝不 panic（recover 兜底）、绝不阻塞。结果 emit chapter-gate 事件
// （Q4）。构建逻辑在 buildAutoGateReport（同步可测）。
func (a *writingState) runAutoGateAfterGeneration(pm *project.Manager, chapterNum int, content, branch string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("生成后自动门 panic 已兜底", "chapter", chapterNum, "panic", r)
		}
	}()
	report := a.buildAutoGateReport(pm, chapterNum, content, branch)
	a.emit("chapter-gate", report)
	slog.Info("生成后自动门完成", "chapter", chapterNum,
		"outline", report["outlineIssues"], "quality", report["qualityIssues"],
		"aiTaste", report["aiTasteScore"], "analysis", report["analysisDone"])
}

// buildAutoGateReport 自动门报告构建（同步，测试直调）：确定性三路（写前契约/
// 写后质量/AI 味）必出零 LLM；分析路（analysisAgent.Analyze=V2 落盘+伏笔同步+
// 记忆回填一次 LLM）仅主线章（Q3：分支章 upsert 会污染主线条目）。
func (a *writingState) buildAutoGateReport(pm *project.Manager, chapterNum int, content, branch string) map[string]interface{} {
	report := map[string]interface{}{
		"type":           "report",
		"chapterNum":     chapterNum,
		"branch":         branch,
		"outlineIssues":  len(gateOutlineIssues(pm, chapterNum)),
		"qualityIssues":  len(novelgate.ChapterQualityIssues(content)),
		"aiTasteScore":   nil,
		"analysisDone":   false,
		"foreshadowSync": nil,
	}
	if ts, err := novelstyle.ScoreTextNoRef(content); err == nil && ts != nil {
		novelstyle.ApplyWhitelist(ts, content, bookWhitelist(pm))
		report["aiTasteScore"] = ts.Score
	}

	if branch != "" {
		return report // 分支章：登记表/V2 是主线口径，跳分析路（Q3）
	}
	if a.analysisAgent == nil {
		slog.Warn("生成后自动门: analysisAgent 未初始化，分析路缺省", "chapter", chapterNum)
		return report
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := a.analysisAgent.Analyze(ctx, chapterNum, content); err != nil {
		slog.Warn("生成后自动门: 分析失败（V2/伏笔同步/记忆回填本次未发生）", "chapter", chapterNum, "error", err)
		return report
	}
	report["analysisDone"] = true
	if res, ok := a.analysisAgent.LastSync(); ok {
		report["foreshadowSync"] = map[string]interface{}{
			"plantedCount":  res.PlantedCount,
			"resolvedCount": res.ResolvedCount,
			"createdCount":  res.CreatedCount,
			"skippedCount":  res.SkippedResolveCount,
		}
	}
	return report
}

// ── 全书体检 ────────────────────────────────────────────────

// BookHealthChapter 逐章体检行（全确定性，零 LLM）。
type BookHealthChapter struct {
	ChapterNum    int `json:"chapterNum"`
	Words         int `json:"words"`         // rune 口径
	OutlineIssues int `json:"outlineIssues"` // 写前契约问题数（含 warn 级）
	OutlineErrors int `json:"outlineErrors"` // 契约 error 级（S1/S2）
	QualityIssues int `json:"qualityIssues"` // 写后质量问题数
	QualityErrors int `json:"qualityErrors"` // 质量 error 级（S1/S2）
	AiTasteScore  int `json:"aiTasteScore"`  // AI 味 0-100（白名单豁免后；-1=打分失败）
}

// BookHealthReport 全书体检报告（Q6：camelCase typed struct）。
type BookHealthReport struct {
	TotalChapters         int                  `json:"totalChapters"`
	Chapters              []BookHealthChapter  `json:"chapters"`
	ContractIssueChapters int                  `json:"contractIssueChapters"`  // outlineIssues>0 的章数
	QualityIssueChapters  int                  `json:"qualityIssueChapters"`   // qualityIssues>0 的章数
	WorstAiTaste          *BookHealthChapter   `json:"worstAiTaste,omitempty"` // 分数最高的章（>=60 才填）
	AnalyzedChapters      int                  `json:"analyzedChapters"`       // V2 分析覆盖章数
	Foreshadow            ForeshadowLintReport `json:"foreshadow"`
}

// RunBookHealthCheck 全书体检（触发式全量编译，作者点按；Q5）：逐章确定性三路
// + 字数；伏笔 Lint 复用（lint 是 writingState 方法，内部读同一 pm）；V2 覆盖率。
// 无项目报错；零章=空报告不算错（新书正常态）。
func (a *writingState) RunBookHealthCheck() (BookHealthReport, error) {
	pm := a.getPM()
	if pm == nil {
		return BookHealthReport{}, fmt.Errorf("请先打开项目")
	}
	report := BookHealthReport{Chapters: []BookHealthChapter{}}

	for i := 1; ; i++ {
		content, err := pm.ReadChapterAsStitch(i)
		if err != nil {
			break // 读取失败即停（countWrittenChapters 同口径）
		}
		if content == "" {
			continue
		}
		row := BookHealthChapter{
			ChapterNum:   i,
			Words:        len([]rune(content)),
			AiTasteScore: -1,
		}
		for _, is := range gateOutlineIssues(pm, i) {
			row.OutlineIssues++
			if is.Severity == "S1" || is.Severity == "S2" {
				row.OutlineErrors++
			}
		}
		for _, is := range novelgate.ChapterQualityIssues(content) {
			row.QualityIssues++
			if is.Severity == "S1" || is.Severity == "S2" {
				row.QualityErrors++
			}
		}
		if ts, terr := novelstyle.ScoreTextNoRef(content); terr == nil && ts != nil {
			novelstyle.ApplyWhitelist(ts, content, bookWhitelist(pm))
			row.AiTasteScore = ts.Score
		}
		report.Chapters = append(report.Chapters, row)
		report.TotalChapters++
		if row.OutlineIssues > 0 {
			report.ContractIssueChapters++
		}
		if row.QualityIssues > 0 {
			report.QualityIssueChapters++
		}
		if row.AiTasteScore >= 60 && (report.WorstAiTaste == nil || row.AiTasteScore > report.WorstAiTaste.AiTasteScore) {
			worst := row
			report.WorstAiTaste = &worst
		}
	}

	if af, err := pm.ReadAnalysisV2File(); err == nil && af != nil {
		for _, it := range af.Items {
			if it.ChapterNum > 0 && it.ChapterNum <= report.TotalChapters {
				report.AnalyzedChapters++
			}
		}
	}

	if lint, err := a.LintForeshadows(); err == nil {
		report.Foreshadow = lint
	} else {
		slog.Warn("全书体检: 伏笔 Lint 失败缺省", "error", err)
	}
	return report, nil
}
