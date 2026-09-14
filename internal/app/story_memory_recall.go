package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 故事记忆语义召回（t3-P2，长程一致性最后一环）────────────────
//
// 口径来源：docs/distill/03-long-range-consistency.md §3.3/§3.4/§12.4-1/§12.4-2。
// 生产者（t3-P1）已按章落盘 memories/MMM-<n>-memory.json；本文件在章节生成时
// 做四段流水线召回：结构化 query → 向量候选 → 阈值筛选（空则兜底）→ 注入。
// 全链容错：库不可用/embedder 不可用/无记忆/任何失败 → 返回 nil，绝不中断生成。

// 检索三常量（spec §12.1：gaea 用归一化余弦，阈值语义与 MuMu 未归一化 L2 的
// 0.6 不同，不可照抄；改参数=改这里）。
const (
	storyMemoryTopK      = 8    // 注入上限
	storyMemoryThreshold = 0.35 // 余弦阈值（归一化语义）
	storyMemoryFallback  = 3    // 无高分命中时的兜底条数（§3.4「空手不如少带」）
	memoryQueryMaxLen    = 800  // 结构化 query 总长（§12.4-1 [:800]）
	memoryLineMax        = 100  // 注入行内容截断（MuMu :789 content[:100]）
	memoryRecallTimeout  = 20 * time.Second
)

// storyMemoryKind 项目隔离的向量 kind（§12.3：单机桌面无 user 维度，
// 以项目目录隔离；semantic_vectors 全局共享表）。
func storyMemoryKind(pm *project.Manager) string {
	return "story_memory|" + pm.Dir
}

// buildMemoryQuery 结构化 query 构造（§12.4-1「本域最可借鉴的一段」）：
// 不嵌大纲原文，按高信号字段分段限额拼接——人物[:8]/关键事件[:6]/叙事目标/
// 情绪/本章要求[:150]，整体 [:800]，换行压空格。
func buildMemoryQuery(node *types.OutlineNode, plotReq string) string {
	if node == nil {
		return ""
	}
	seg := make([]string, 0, 5)
	if nc := len(node.Characters); nc > 0 {
		if nc > 8 {
			nc = 8
		}
		seg = append(seg, "人物："+strings.Join(node.Characters[:nc], "、"))
	}
	if nk := len(node.KeyPoints); nk > 0 {
		if nk > 6 {
			nk = 6
		}
		seg = append(seg, "关键事件："+strings.Join(node.KeyPoints[:nk], "、"))
	}
	if s := strings.TrimSpace(node.Summary); s != "" {
		seg = append(seg, "叙事目标："+s)
	}
	if s := strings.TrimSpace(node.Emotion); s != "" {
		seg = append(seg, "情绪："+s)
	}
	if s := strings.TrimSpace(plotReq); s != "" {
		seg = append(seg, "本章要求："+truncateBudget(s, 150))
	}
	q := strings.ReplaceAll(strings.Join(seg, "；"), "\n", " ")
	return truncateBudget(q, memoryQueryMaxLen)
}

// selectMemoryHits 四段流水线的筛选段（§3.4）：阈值之上按分降序取 topK；
// 全部低于阈值时保留 top fallback 条（「无高分不如少带」，补关键词召回
// 「一个不中就空手」的短板）。
func selectMemoryHits(hits []semantic.Hit) []semantic.Hit {
	var above []semantic.Hit
	for _, h := range hits {
		if h.Score >= storyMemoryThreshold {
			above = append(above, h)
		}
	}
	if len(above) > 0 {
		if len(above) > storyMemoryTopK {
			above = above[:storyMemoryTopK]
		}
		return above
	}
	if len(hits) > storyMemoryFallback {
		return hits[:storyMemoryFallback]
	}
	return hits
}

// formatMemoryLines 渲染注入行：`- (相关度:0.82) 内容[:100]`（MuMu :789 同型）。
func formatMemoryLines(hits []semantic.Hit) []string {
	lines := make([]string, 0, len(hits))
	for _, h := range hits {
		lines = append(lines, fmt.Sprintf("- (相关度:%.2f) %s", h.Score, truncateBudget(h.Text, memoryLineMax)))
	}
	return lines
}

// recallStoryMemories 召回入口（生成链路容错增强）：读全部章节记忆 →
// 只保留本章之前的条目（当前章记忆描述的是正在写的内容，召回无益）→
// 增量向量化（Ensure 正文快照比对）→ 语义检索 → Stale 清理陈旧向量
// （章节重写后条目减少时不留死向量，规避 MuMu D1 的向量侧残留）。
func (a *writingState) recallStoryMemories(pm *project.Manager, node *types.OutlineNode, plotReq string, currentChapter int) []string {
	store := a.app.hubSemanticStore()
	if store == nil || !store.Available() {
		return nil
	}
	e := a.app.localSearchEmbedder()
	if e == nil {
		return nil // 本地 embedding 引擎未配置（Herdsman 未启用）
	}
	all, err := pm.ReadAllChapterMemories()
	if err != nil || len(all) == 0 {
		return nil
	}
	docs := make([]semantic.Doc, 0, len(all))
	keep := make(map[string]bool, len(all))
	for _, m := range all {
		keep[m.ID] = true
		if m.ChapterNum <= 0 || m.ChapterNum >= currentChapter {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if t := strings.TrimSpace(m.Title); t != "" {
			text = t + "：" + text
		}
		if text == "" {
			continue
		}
		docs = append(docs, semantic.Doc{ID: m.ID, Text: text})
	}
	if len(docs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), memoryRecallTimeout)
	defer cancel()
	if !e.Available(ctx) {
		return nil // 本地 embedding 模型未启动
	}
	kind := storyMemoryKind(pm)
	query := buildMemoryQuery(node, plotReq)
	if query == "" {
		return nil
	}
	hits, err := store.Search(ctx, e, kind, docs, query, storyMemoryTopK)
	if err != nil {
		return nil
	}
	_, _ = store.Stale(kind, keep) // 陈旧向量清理尽力而为（重写减条目的残留）
	// Search 已按分降序；再次显式排序保证确定性（不依赖实现细节）
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return formatMemoryLines(selectMemoryHits(hits))
}

// findOutlineNodeByID 在大纲树（含分支子节点）中按 ID 找节点（召回 query 需要
// 节点的结构化字段；找不到返回 nil，调用方降级）。
func findOutlineNodeByID(nodes []types.OutlineNode, id string) *types.OutlineNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
		if got := findOutlineNodeByID(nodes[i].Children, id); got != nil {
			return got
		}
	}
	return nil
}
