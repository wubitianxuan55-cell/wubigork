package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 伏笔一致性体检（并行池小说线：Continuity Linter，登记表已在产的另一半）────
//
// 纯确定性检查、零 LLM：只看登记表内部的序/状态自洽与登记表 ↔ 已写章节的
// 指向有效性，不做「回收章文本是否回收了该伏笔」类语义判断（描述是自由文本，
// 确定性关键词匹配误报率高，宁缺勿滥）。

// 悬置判定：非长线条目埋设后超过该章数仍未回收则提示。
const foreshadowStaleAfterChapters = 10

// ForeshadowLintFinding 一条体检发现。
type ForeshadowLintFinding struct {
	Code         string `json:"code"`              // ordering / status-mismatch / dangling / stale / duplicate
	Severity     string `json:"severity"`          // high / medium / low
	ForeshadowID string `json:"foreshadowId"`      // 命中条目 ID
	ItemDesc     string `json:"itemDesc"`          // 条目描述（面板直显免二次查）
	Message      string `json:"message"`           // 中文说明
	Chapter      string `json:"chapter,omitempty"` // 相关章节文件名
}

// ForeshadowLintReport LintForeshadows 返回。
type ForeshadowLintReport struct {
	TotalChapters int                     `json:"totalChapters"`
	Items         int                     `json:"items"`
	Planted       int                     `json:"planted"`
	Hinted        int                     `json:"hinted"`
	Revealed      int                     `json:"revealed"`
	LongTerm      int                     `json:"longTerm"`
	Findings      []ForeshadowLintFinding `json:"findings"`
}

// chapterNumOf 章节文件名 → 章号（取前导数字："001.md"→1，"001a.md"→1；空/无数字→0）。
func chapterNumOf(filename string) int {
	n := strings.IndexFunc(filename, func(r rune) bool { return r < '0' || r > '9' })
	digits := filename
	if n >= 0 {
		digits = filename[:n]
	}
	v, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return v
}

// normalizeForeshadowDesc 描述归一（重复判定用）：去首尾空白。
func normalizeForeshadowDesc(s string) string {
	return strings.TrimSpace(s)
}

// lintForeshadowItems 对登记条目跑五类确定性检查（纯函数）。
func lintForeshadowItems(items []types.Foreshadow, totalChapters int) []ForeshadowLintFinding {
	findings := []ForeshadowLintFinding{}
	add := func(f ForeshadowLintFinding) { findings = append(findings, f) }

	firstByDesc := map[string]string{} // 归一描述 → 首个条目 ID
	for _, it := range items {
		desc := it.Description
		plantedNum := chapterNumOf(it.PlantedIn)
		revealedNum := chapterNumOf(it.RevealedIn)

		// ① 回收先于埋设（序颠倒，登记错误类）
		if plantedNum > 0 && revealedNum > 0 && revealedNum < plantedNum {
			add(ForeshadowLintFinding{Code: "ordering", Severity: "high", ForeshadowID: it.ID, ItemDesc: desc,
				Message: fmt.Sprintf("回收章（%s）早于埋设章（%s），登记序颠倒", it.RevealedIn, it.PlantedIn),
				Chapter: it.RevealedIn})
		}
		// ② 状态与回收章不一致
		if types.IsResolvedStatus(it.Status) && it.RevealedIn == "" {
			add(ForeshadowLintFinding{Code: "status-mismatch", Severity: "medium", ForeshadowID: it.ID, ItemDesc: desc,
				Message: "状态已是「已回收」但未填回收章节"})
		}
		// partial 豁免：部分回收天然可能同时带回收章，不算自相矛盾（宁缺勿误）。
		if it.RevealedIn != "" && !types.IsResolvedStatus(it.Status) && it.Status != types.ForeshadowPartial {
			add(ForeshadowLintFinding{Code: "status-mismatch", Severity: "medium", ForeshadowID: it.ID, ItemDesc: desc,
				Message: fmt.Sprintf("已填回收章（%s）但状态仍是「%s」，应改为「已回收」", it.RevealedIn, foreshadowStatusLabel(it.Status)),
				Chapter: it.RevealedIn})
		}
		// ③ 指向空缺章节（登记指向未写的章）
		if plantedNum > totalChapters {
			add(ForeshadowLintFinding{Code: "dangling", Severity: "high", ForeshadowID: it.ID, ItemDesc: desc,
				Message: fmt.Sprintf("埋设章 %s 不存在（当前写至第 %d 章）", it.PlantedIn, totalChapters),
				Chapter: it.PlantedIn})
		}
		if revealedNum > totalChapters {
			add(ForeshadowLintFinding{Code: "dangling", Severity: "high", ForeshadowID: it.ID, ItemDesc: desc,
				Message: fmt.Sprintf("回收章 %s 不存在（当前写至第 %d 章）", it.RevealedIn, totalChapters),
				Chapter: it.RevealedIn})
		}
		// ④ 悬置未回收（长线条目豁免——长线本来就是跨书埋设）
		if !it.IsLongTerm && (it.Status == types.ForeshadowPlanted || it.Status == types.ForeshadowHinted) &&
			plantedNum > 0 && plantedNum <= totalChapters {
			if age := totalChapters - plantedNum; age >= foreshadowStaleAfterChapters {
				add(ForeshadowLintFinding{Code: "stale", Severity: "medium", ForeshadowID: it.ID, ItemDesc: desc,
					Message: fmt.Sprintf("已悬置 %d 章未回收（第 %d 章埋设），考虑回收或标记长线", age, plantedNum),
					Chapter: it.PlantedIn})
			}
		}
		// ⑤ 疑似重复登记（归一描述相同）
		if d := normalizeForeshadowDesc(desc); d != "" {
			if firstID, ok := firstByDesc[d]; ok {
				add(ForeshadowLintFinding{Code: "duplicate", Severity: "low", ForeshadowID: it.ID, ItemDesc: desc,
					Message: fmt.Sprintf("与条目 %s 描述相同，疑似重复登记", firstID)})
			} else {
				firstByDesc[d] = it.ID
			}
		}
	}
	return findings
}

func foreshadowStatusLabel(s types.ForeshadowStatus) string {
	if types.IsResolvedStatus(s) {
		return "已回收"
	}
	switch s {
	case types.ForeshadowPending:
		return "已规划"
	case types.ForeshadowPlanted:
		return "已埋设"
	case types.ForeshadowHinted:
		return "已暗示"
	case types.ForeshadowPartial:
		return "部分回收"
	case types.ForeshadowAbandoned:
		return "已废弃"
	}
	return string(s)
}

// countWrittenChapters 统计已写章节数（v4 场景工程走 Stitch，遍历口径与
// collectChapterSamples 一致：读取失败即停）。
func countWrittenChapters(pm *project.Manager) int {
	n := 0
	for i := 1; ; i++ {
		content, err := pm.ReadChapterAsStitch(i)
		if err != nil {
			break
		}
		if content == "" {
			continue
		}
		n++
	}
	return n
}

// LintForeshadows 伏笔一致性体检：登记表序/状态自洽 + 指向已写章节的有效性。
// 登记表为空/文件缺失是正常态（空报告），不算错。
func (a *writingState) LintForeshadows() (ForeshadowLintReport, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowLintReport{}, fmt.Errorf("请先打开项目")
	}
	total := countWrittenChapters(pm)
	report := ForeshadowLintReport{
		TotalChapters: total,
		Findings:      []ForeshadowLintFinding{},
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return report, nil
		}
		return ForeshadowLintReport{}, fmt.Errorf("读取伏笔登记表失败: %w", err)
	}
	report.Items = len(ff.Items)
	for _, it := range ff.Items {
		switch {
		case types.IsResolvedStatus(it.Status):
			report.Revealed++
		case it.Status == types.ForeshadowPlanted:
			report.Planted++
		case it.Status == types.ForeshadowHinted:
			report.Hinted++
		}
		if it.IsLongTerm {
			report.LongTerm++
		}
	}
	report.Findings = lintForeshadowItems(ff.Items, total)
	return report, nil
}
