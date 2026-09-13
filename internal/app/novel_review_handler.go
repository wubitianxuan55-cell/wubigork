package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gaeaconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/novelreview"
	"github.com/gaea/gaea/internal/project"
)

// ── 平台质量评审（oh-story 蒸馏 t1 落地面）───────────────────────────────
//
// 规格 docs/gaea-novel-ohstory-distill-2026-09.md §2/§4：把上游的「平台 rubric +
// S1~S4 分级」从给 LLM 的评审提纲，落成 gaea 的**确定性章节体检**——纯函数、
// 零 LLM、零网络，逐维度给 PASS/WARN/FAIL 与原文证据，供创作间面板直显。
// 阈值/词表是数据资产（internal/novelreview/rubric.json，可被工作区覆盖文件
// 整体替换）；本文件只做「读章 → 评审 → 载荷」的接线。

// novelReviewOnce 保证 rubric 覆盖文件每进程只探测一次（增强面：覆盖文件不在场
// 或解析失败都静默保内置默认，不挡评审主功能——与词表覆盖同款纪律）。
var novelReviewOnce sync.Once

// ensureNovelReviewRubric 按惯例目录加载 rubric 覆盖（<cwd>/.gaea|/.agents|/.agent|/.claude/
// skills/novel-review/rubric.json，.gaea 优先）。改门槛不改代码不发版。
func ensureNovelReviewRubric() {
	novelReviewOnce.Do(func() {
		for _, base := range gaeaconfig.ConventionDirs {
			p := filepath.Join(gaeaCwd(), base, "skills", "novel-review", "rubric.json")
			if _, err := os.Stat(p); err == nil {
				_ = novelreview.LoadRubricFile(p)
				return
			}
		}
	})
}

// ChapterReviewEvidenceView 单条证据（段落号 + 原文摘录，Go 侧截断免前端按 rune 切字）。
type ChapterReviewEvidenceView struct {
	Paragraph int    `json:"paragraph"`
	Excerpt   string `json:"excerpt"`
}

// ChapterReviewDimensionView 单维度评审结果（面板直显）。
type ChapterReviewDimensionView struct {
	ID       string                      `json:"id"`
	Label    string                      `json:"label"`
	Verdict  string                      `json:"verdict"` // pass/warn/fail/skip
	Severity string                      `json:"severity,omitempty"`
	Detail   string                      `json:"detail"`
	Advice   string                      `json:"advice,omitempty"`
	Evidence []ChapterReviewEvidenceView `json:"evidence,omitempty"`
}

// ChapterReviewPayload NovelChapterReview 返回。
type ChapterReviewPayload struct {
	ChapterNum    int                          `json:"chapterNum"`
	Platform      string                       `json:"platform"`
	PlatformLabel string                       `json:"platformLabel"`
	Words         int                          `json:"words"`
	Verdict       string                       `json:"verdict"` // APPROVE/CONCERNS/REJECT
	Counts        map[string]int               `json:"counts"`
	Dimensions    []ChapterReviewDimensionView `json:"dimensions"`
	Advisories    []string                     `json:"advisories,omitempty"`
}

// ReviewPlatformView 档位选择器条目。
type ReviewPlatformView struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Form  string `json:"form"`
}

// NovelReviewPlatforms 返回可用档位（前端选择器数据源；顺序即数据资产顺序）。
func (a *writingState) NovelReviewPlatforms() []ReviewPlatformView {
	ensureNovelReviewRubric()
	ids := novelreview.PlatformIDs()
	labels := novelreview.PlatformLabels()
	out := make([]ReviewPlatformView, 0, len(ids))
	for _, id := range ids {
		p, err := novelreview.PlatformByID(id)
		if err != nil {
			continue
		}
		out = append(out, ReviewPlatformView{ID: id, Label: labels[id], Form: p.Form})
	}
	return out
}

// protagonistNames 从角色库取主角名（RoleType=protagonist）；无数据返回 nil（维度 skip）。
func protagonistNames(pm *project.Manager) []string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil {
		return nil
	}
	var names []string
	for _, c := range cf.Characters {
		if c.RoleType == "protagonist" && strings.TrimSpace(c.Name) != "" {
			names = append(names, strings.TrimSpace(c.Name))
		}
	}
	return names
}

// NovelChapterReview 对指定章做平台质量评审（平台档位空/未知回落 general）。
func (a *writingState) NovelChapterReview(chapterNum int, platform string) (ChapterReviewPayload, error) {
	ensureNovelReviewRubric()
	pm := a.getPM()
	if pm == nil {
		return ChapterReviewPayload{}, fmt.Errorf("请先打开项目")
	}
	if chapterNum <= 0 {
		return ChapterReviewPayload{}, fmt.Errorf("章节号非法")
	}
	text, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil {
		return ChapterReviewPayload{}, fmt.Errorf("读取章节失败: %w", err)
	}
	if countNonSpaceRunesOf(text) == 0 {
		return ChapterReviewPayload{}, fmt.Errorf("第 %d 章为空，无可评审内容", chapterNum)
	}
	rep, err := novelreview.Review(text, platform, novelreview.Options{
		ChapterNum:       chapterNum,
		ProtagonistNames: protagonistNames(pm),
	})
	if err != nil {
		return ChapterReviewPayload{}, fmt.Errorf("评审失败: %w", err)
	}
	payload := ChapterReviewPayload{
		ChapterNum:    rep.ChapterNum,
		Platform:      rep.Platform,
		PlatformLabel: rep.PlatformLabel,
		Words:         rep.Words,
		Verdict:       rep.Verdict,
		Counts:        rep.Counts,
		Dimensions:    make([]ChapterReviewDimensionView, 0, len(rep.Dimensions)),
		Advisories:    rep.Advisories,
	}
	for _, d := range rep.Dimensions {
		view := ChapterReviewDimensionView{
			ID:       d.ID,
			Label:    d.Label,
			Verdict:  d.Verdict,
			Severity: d.Severity,
			Detail:   d.Detail,
			Advice:   d.Advice,
		}
		for _, ev := range d.Evidence {
			view.Evidence = append(view.Evidence, ChapterReviewEvidenceView{
				Paragraph: ev.Paragraph,
				Excerpt:   excerptOf(text, ev.Start, ev.End),
			})
		}
		payload.Dimensions = append(payload.Dimensions, view)
	}
	return payload, nil
}
