package app

// ── 阶段（幕）边界：每 stageLength 章为一阶段，阶段开篇章先合账再开新账 ──
//
// 作者口径：每 20 章剧情进入下一个阶段。阶段开篇章（第 21/41/…章）生成时
// 注入两段：①上一阶段总结（LLM 合成，素材取自大纲节点摘要+章节摘要文件，
// 失败回落逐章摘要清单）②新阶段开篇纪律（静态约束）。非阶段开篇章零注入。
// 与既有增强区段同一纪律：任何失败都降级（回落摘要清单/空串），绝不因注入
// 失败中断章节生成主链路。

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// stageLength 每阶段章数（作者口径：每 20 章一幕）。
const stageLength = 20

// stageRecapBudgetRunes 阶段总结注入与摘要素材的合计预算（rune）。
const stageRecapBudgetRunes = 2000

// isStageOpener 判断是否阶段开篇章：第 21/41/… 章。第一章是全书开篇
// （create-chapter-first），不算阶段开篇。
func isStageOpener(chapterNum int) bool {
	return chapterNum > 1 && (chapterNum-1)%stageLength == 0
}

// stageRange 阶段开篇章对应的上一阶段章号区间 [start, end]。
func stageRange(chapterNum int) (int, int) {
	end := chapterNum - 1
	start := end - stageLength + 1
	if start < 1 {
		start = 1
	}
	return start, end
}

// buildStageDigest 收集上一阶段逐章摘要（大纲节点 Summary → 章节摘要文件回退，
// 解析链与 buildPrevSummaryWindow 同源）；单条截断、合计受 stageRecapBudgetRunes
// 预算约束。无任何素材返回空串。
func buildStageDigest(pm *project.Manager, start, end int) string {
	if pm == nil {
		return ""
	}
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		of = &types.OutlineFile{Nodes: []types.OutlineNode{}}
	}
	byNum := make(map[int]*types.OutlineNode)
	var walk func(nodes []types.OutlineNode)
	walk = func(nodes []types.OutlineNode) {
		for i := range nodes {
			n := &nodes[i]
			byNum[n.OrderIndex] = n
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(of.Nodes)

	var b strings.Builder
	used := 0
	for num := start; num <= end; num++ {
		summary := ""
		if n, ok := byNum[num]; ok && n != nil {
			summary = strings.TrimSpace(n.Summary)
		}
		if summary == "" {
			if cs, err := pm.ReadChapterSummary(num); err == nil && cs != nil {
				summary = strings.TrimSpace(cs.Summary)
			}
		}
		if summary == "" {
			continue
		}
		line := fmt.Sprintf("第%d章：%s", num, truncateRunes(summary, 120))
		if used+len(line) > stageRecapBudgetRunes {
			break
		}
		b.WriteString(line)
		b.WriteString("\n")
		used += len(line)
	}
	return strings.TrimRight(b.String(), "\n")
}

// stageRecapSection 阶段开篇章注入段：上一阶段总结（LLM 三段式合成，失败回落
// 逐章摘要清单）+ 新阶段开篇纪律。非阶段开篇章或素材为空返回空串（零注入）。
func (a *writingState) stageRecapSection(pm *project.Manager, chapterNum int) string {
	if !isStageOpener(chapterNum) {
		return ""
	}
	start, end := stageRange(chapterNum)
	digest := buildStageDigest(pm, start, end)
	if digest == "" {
		return ""
	}
	recap := strings.TrimSpace(a.synthesizeStageRecap(start, end, digest))
	if recap == "" {
		recap = digest
	}

	var b strings.Builder
	fmt.Fprintf(&b, "【上一阶段总结（第%d~%d章）】\n%s\n\n", start, end, truncateRunes(recap, stageRecapBudgetRunes))
	fmt.Fprintf(&b, "【新阶段开篇（第%d章）纪律】\n", chapterNum)
	b.WriteString("- 上一阶段的高潮刚过：先承接余波与代价（世界/人物因它发生了什么），不假装无事发生；\n")
	b.WriteString("- 本章是新阶段的起点：章末前把新阶段的核心驱动（新目标/新矛盾/新悬念）摆上台面；\n")
	b.WriteString("- 上一阶段的遗留线头自然延续，不得当作不存在；可以换场景/换目标，但主线承诺不变。")
	return b.String()
}

// synthesizeStageRecap 用 LLM 把逐章摘要合成三段式阶段复盘；任何失败返回空串
// （调用方回落逐章摘要清单），绝不阻断章节生成。
func (a *writingState) synthesizeStageRecap(start, end int, digest string) string {
	if a.client == nil || a.eng == nil || a.ctx == nil {
		return ""
	}
	tmpl := a.eng.Get("stage-recap")
	if tmpl == nil {
		return ""
	}
	eng, model, _ := a.routeModel("novel")
	sys := tmpl.BuildSystemPrompt("")
	user := tmpl.BuildUserPrompt(map[string]string{
		"stage_range":     fmt.Sprintf("第%d~%d章", start, end),
		"chapter_digests": digest,
	})
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model, sys, user, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.3, MaxTokens: 700,
	})
	if err != nil {
		slog.Warn("阶段总结合成失败，回落逐章摘要", "stage", fmt.Sprintf("%d~%d", start, end), "error", err)
		return ""
	}
	return strings.TrimSpace(reply)
}
