package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/gaea/gaea/internal/types"
)

// ── 伏笔生命周期清理与统计（t1-P3）──────────────────────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §8.1（三个清理入口）/ §5.1
// （ForeshadowStats）。关键不变量：source_type 是批量清理的唯一判据，
// 手动条目（source_type != analysis，含 manual_/空来源的存量登记）永不被
// 批量删除——只重置不删除（spec §1.1「手动登记=作者资产」）。
//
// gaea 适配（有意偏离，spec §8.3 D5 决策分支）：
//   - 不引入 SourceAnalysis 引用匹配分支（gaea 同步层从不写该字段——引入
//     匹配分支就是 MuMu D5 死代码的同款）；
//   - §8.3 单条删除联动依赖「持久化分析记录」，gaea 分析产物不落盘，无
//     联动目标，暂不做（若日后分析记录持久化再补）。

// ForeshadowCleanupResult 清理入口统一返回（wire camelCase，见 wire_shape_guard）。
type ForeshadowCleanupResult struct {
	Deleted     int `json:"deleted"`               // 删除的分析来源条数
	RolledBack  int `json:"rolledBack,omitempty"`  // 回退回收态的条数（Clean 专属）
	ResetManual int `json:"resetManual,omitempty"` // 重置为 pending 的手工条数（Reset 专属）
}

// foreshadowIsManual 报告条目是否手动登记（source_type 非 analysis 即手动——
// 存量 foreshadows.json 大多没有 source_type 字段，它们是用户资产，按手动保护）。
func foreshadowIsManual(f types.Foreshadow) bool {
	return f.SourceType != types.ForeshadowSourceAnalysis
}

// foreshadowAlive 可回收压力状态（D15 口径，与 analysis.foreshadowResolvable 同源）。
func foreshadowAlive(s types.ForeshadowStatus) bool {
	return s == types.ForeshadowPlanted || s == types.ForeshadowHinted || s == types.ForeshadowPartial
}

// DeleteChapterForeshadows 删除指定章节关联的伏笔条目（章节清空/重新生成时用）。
// 命中 = PlantedIn==chapterFile ∨ RevealedIn==chapterFile；onlyAnalysisSource=true
// （默认语义）时只删分析来源条目，手动条目原样保留。
func (a *writingState) DeleteChapterForeshadows(chapterFile string, onlyAnalysisSource bool) (ForeshadowCleanupResult, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("请先打开项目")
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ForeshadowCleanupResult{}, nil // 空登记=无事可清，正常态
		}
		return ForeshadowCleanupResult{}, fmt.Errorf("读取伏笔登记表失败: %w", err)
	}
	res := ForeshadowCleanupResult{}
	kept := ff.Items[:0:0]
	for _, it := range ff.Items {
		hit := it.PlantedIn == chapterFile || it.RevealedIn == chapterFile
		// onlyAnalysisSource=true：只删分析来源（手动保护）；false：整章关联全删
		if hit && (!onlyAnalysisSource || !foreshadowIsManual(it)) {
			res.Deleted++
			continue
		}
		kept = append(kept, it)
	}
	ff.Items = kept
	if err := pm.WriteForeshadows(ff); err != nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("写回伏笔登记表失败: %w", err)
	}
	return res, nil
}

// CleanChapterAnalysisForeshadows 重新分析前的章节清理（语义与 DeleteChapter 严格区分）：
// ① 只删「分析来源 ∧ 埋入本章」的条目（本章分析产物可由重分析再生）；
// ② 回退本章回收的条目（RevealedIn==chapterFile 且状态 ∈ 已回收/部分回收 →
// planted，清空回收痕迹），让重分析重新判定回收是否成立。
func (a *writingState) CleanChapterAnalysisForeshadows(chapterFile string) (ForeshadowCleanupResult, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("请先打开项目")
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ForeshadowCleanupResult{}, nil
		}
		return ForeshadowCleanupResult{}, fmt.Errorf("读取伏笔登记表失败: %w", err)
	}
	res := ForeshadowCleanupResult{}
	kept := ff.Items[:0:0]
	for i := range ff.Items {
		it := &ff.Items[i]
		if !foreshadowIsManual(*it) && it.PlantedIn == chapterFile {
			res.Deleted++ // ① 分析来源且埋入本章：删（可再生）
			continue
		}
		// ② 本章回收 → 回退到 planted（§8.2 系统内部迁移，不暴露其他 API；
		// 回退集 = 已回收 ∪ 部分回收，spec §8.1 ②）
		if it.RevealedIn == chapterFile && (types.IsResolvedStatus(it.Status) || it.Status == types.ForeshadowPartial) {
			it.Status = types.ForeshadowPlanted
			it.RevealedIn = ""
			it.ResolvedAt = ""
			it.ResolutionText = ""
			res.RolledBack++
		}
		kept = append(kept, *it)
	}
	ff.Items = kept
	if err := pm.WriteForeshadows(ff); err != nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("写回伏笔登记表失败: %w", err)
	}
	return res, nil
}

// ClearProjectForeshadowsForReset 全新生成/项目重置清理：
// ① 删全部分析来源条目；② 手动条目重置为 pending 并清空章节关联与时间戳
// （记录本身保留——作者资产只重置不删）。
func (a *writingState) ClearProjectForeshadowsForReset() (ForeshadowCleanupResult, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("请先打开项目")
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ForeshadowCleanupResult{}, nil
		}
		return ForeshadowCleanupResult{}, fmt.Errorf("读取伏笔登记表失败: %w", err)
	}
	res := ForeshadowCleanupResult{}
	kept := ff.Items[:0:0]
	for i := range ff.Items {
		it := &ff.Items[i]
		if !foreshadowIsManual(*it) {
			res.Deleted++ // ① 分析来源：整批删（可再生副产物）
			continue
		}
		// ② 手动条目：重置不删除
		it.Status = types.ForeshadowPending
		it.PlantedIn = ""
		it.RevealedIn = ""
		it.TargetResolveIn = ""
		it.PlantedAt = ""
		it.ResolvedAt = ""
		res.ResetManual++
		kept = append(kept, *it)
	}
	ff.Items = kept
	if err := pm.WriteForeshadows(ff); err != nil {
		return ForeshadowCleanupResult{}, fmt.Errorf("写回伏笔登记表失败: %w", err)
	}
	return res, nil
}

// ForeshadowStatsReport 伏笔登记表统计（spec §5.1 ForeshadowStats）。
type ForeshadowStatsReport struct {
	Total             int `json:"total"`
	Pending           int `json:"pending"`
	Planted           int `json:"planted"`
	Hinted            int `json:"hinted"`
	Resolved          int `json:"resolved"` // gaea 写入口径 revealed（resolved 别名归一并入）
	PartiallyResolved int `json:"partiallyResolved"`
	Abandoned         int `json:"abandoned"`
	LongTermCount     int `json:"longTermCount"`
	OverdueCount      int `json:"overdueCount"` // 仅 currentChapter>0 时计算
	CurrentChapter    int `json:"currentChapter,omitempty"`
}

// GetForeshadowStats 伏笔统计。currentChapter<=0 时自动按已写章节数计算
// （超期数才有意义）；登记表缺失是正常态（全 0 报告）。
func (a *writingState) GetForeshadowStats(currentChapter int) (ForeshadowStatsReport, error) {
	pm := a.getPM()
	if pm == nil {
		return ForeshadowStatsReport{}, fmt.Errorf("请先打开项目")
	}
	stats := ForeshadowStatsReport{}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return stats, nil
		}
		return ForeshadowStatsReport{}, fmt.Errorf("读取伏笔登记表失败: %w", err)
	}
	if currentChapter <= 0 {
		currentChapter = countWrittenChapters(pm)
	}
	stats.CurrentChapter = currentChapter
	for _, it := range ff.Items {
		switch types.NormalizeForeshadowStatus(it.Status) {
		case types.ForeshadowPending:
			stats.Pending++
		case types.ForeshadowPlanted:
			stats.Planted++
		case types.ForeshadowHinted:
			stats.Hinted++
		case types.ForeshadowRevealed:
			stats.Resolved++
		case types.ForeshadowPartial:
			stats.PartiallyResolved++
		case types.ForeshadowAbandoned:
			stats.Abandoned++
		default:
			// 未知状态不计入任何分桶（Total 按分桶和计，与 spec 一致；
			// 脏数据交由 Lint/校验面暴露，不在统计里猜）
		}
		if it.IsLongTerm {
			stats.LongTermCount++
		}
		// 超期 = 存活（planted/hinted/partial）且计划回收章 < 当前章（D15 口径，
		// 判定走 types.ClassifyResolve 唯一入口）
		if currentChapter > 0 && foreshadowAlive(it.Status) &&
			types.ClassifyResolve(it, currentChapter) == types.ResolveOverdue {
			stats.OverdueCount++
		}
	}
	stats.Total = stats.Pending + stats.Planted + stats.Hinted + stats.Resolved +
		stats.PartiallyResolved + stats.Abandoned
	return stats, nil
}
