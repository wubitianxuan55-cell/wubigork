package bookimport

// 篇幅路由与骨架聚合（oh-story T4：导入结构映射与篇幅路由，规格
// docs/gaea-novel-ohstory-distill-2026-09.md）。上游 length-routing.md（MIT）
// 以 短篇 30000 字为建议上界、章节数 ≥5 为强长篇信号；gaea 扩成三档骨架路由：
// 反推大纲的目标粒度随篇幅走——短篇章级细纲、中篇每 10 章一节点、长篇每 30 章
// 一节点（卷级粗纲）。阈值是数据判据不是代码机密：改这里即改路由。

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Tier 篇幅档位。
type Tier string

const (
	TierShort Tier = "short"
	TierMid   Tier = "mid"
	TierLong  Tier = "long"
)

// 路由阈值（字数口径 = utf8.RuneCountInString，与引擎全线一致）。
const (
	// shortWordsMax 短篇字数上界（上游 length-routing 建议值 30000）。
	shortWordsMax = 30000
	// shortChaptersMax 章数 <5 视为短篇（上游：有章节分隔且 ≥5 = 强长篇信号）。
	shortChaptersMax = 5
	// longWordsMin 长篇字数下界（百万字级大部头的下沿）。
	longWordsMin = 800000
	// longChaptersMin 长篇章数下界。
	longChaptersMin = 300
	// 中/长篇聚合粒度：每个大纲节点覆盖的章数。
	midSegmentSize  = 10
	longSegmentSize = 30
	// segmentSummaryMax 聚合节点摘要截断（rune）。
	segmentSummaryMax = 200
)

// RouteTier 按总字数与章数选骨架档位：
//   - 短篇：字数 <30000 或 章数 <5（含零章）——章级细纲；
//   - 长篇：字数 ≥800000 或 章数 ≥300——卷级粗纲；
//   - 其余为中篇——段级聚合。
//
// 判定顺序：长篇信号优先于短篇信号（大部头不因章数少而误判；零章除外）。
func RouteTier(totalWords, totalChapters int) Tier {
	if totalChapters <= 0 {
		return TierShort
	}
	if totalWords >= longWordsMin || totalChapters >= longChaptersMin {
		return TierLong
	}
	if totalWords < shortWordsMax || totalChapters < shortChaptersMax {
		return TierShort
	}
	return TierMid
}

// SegmentSize 返回该档位每个大纲节点覆盖的章数；短篇返回 0 = 不聚合（章级细纲）。
func SegmentSize(tier Tier) int {
	switch tier {
	case TierMid:
		return midSegmentSize
	case TierLong:
		return longSegmentSize
	default:
		return 0
	}
}

// Segment 聚合骨架节点：覆盖 [ChapterFrom, ChapterTo] 闭区间。
type Segment struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	ChapterFrom int    `json:"chapterFrom"`
	ChapterTo   int    `json:"chapterTo"`
}

// AggregateSkeleton 把章级条目按 segSize 顺序聚合成骨架节点；segSize<=0 返回 nil
// （章级档位不聚合）。摘要由各章标题拼接（反推条目的 Title 是逐章标题；
// Summary 更有信息量时优先），超出 segmentSummaryMax 截断。
func AggregateSkeleton(items []OutlineStructure, segSize int) []Segment {
	if segSize <= 0 || len(items) == 0 {
		return nil
	}
	var out []Segment
	for start := 0; start < len(items); start += segSize {
		end := start + segSize
		if end > len(items) {
			end = len(items)
		}
		grp := items[start:end]
		var b strings.Builder
		for i, it := range grp {
			t := strings.TrimSpace(it.Title)
			if t == "" {
				continue
			}
			if i > 0 && b.Len() > 0 {
				b.WriteString("；")
			}
			b.WriteString(t)
		}
		summary := b.String()
		if r := utf8.RuneCountInString(summary); r > segmentSummaryMax {
			summary = string([]rune(summary)[:segmentSummaryMax]) + "……"
		}
		from, to := grp[0].ChapterNumber, grp[len(grp)-1].ChapterNumber
		title := fmt.Sprintf("第%d-%d章", from, to)
		if from == to {
			title = fmt.Sprintf("第%d章", from)
		}
		out = append(out, Segment{Title: title, Summary: summary, ChapterFrom: from, ChapterTo: to})
	}
	return out
}
