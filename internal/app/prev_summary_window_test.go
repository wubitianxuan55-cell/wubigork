package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// makePrevNodes 生成 n 个大纲节点，OrderIndex 1..n，摘要「摘要N」+ 填充长度。
func makePrevNodes(n, padRunes int) []types.OutlineNode {
	nodes := make([]types.OutlineNode, 0, n)
	for i := 1; i <= n; i++ {
		nodes = append(nodes, types.OutlineNode{
			OrderIndex: i,
			Summary:    fmt.Sprintf("摘要%d%s", i, strings.Repeat("事", padRunes)),
		})
	}
	return nodes
}

// TestBuildPrevSummaryWindow_Window t3 首刀：只取本章之前最近 10 章（spec §12.1，
// MuMu :1375），修复「200 rune×全部前章」的无界 prompt 前缀（§11.3 缺口 1）。
func TestBuildPrevSummaryWindow_Window(t *testing.T) {
	// 200 章的书写第 201 章：只应出现 191~200，共 10 条
	nodes := makePrevNodes(200, 5)
	out := buildPrevSummaryWindow(nodes, 201, func(n types.OutlineNode) string { return n.Summary })

	for cn := 191; cn <= 200; cn++ {
		if !strings.Contains(out, fmt.Sprintf("第%d章：", cn)) {
			t.Fatalf("窗口应含第 %d 章", cn)
		}
	}
	for cn := 1; cn <= 190; cn++ {
		if strings.Contains(out, fmt.Sprintf("第%d章：", cn)) {
			t.Fatalf("窗口外第 %d 章不应出现", cn)
		}
	}
	if got := strings.Count(out, "\n\n"); got != ctxPrevWindowChapters { // 头+10 条之间
		t.Fatalf("应为头+10 条（9 个空行分隔），实际分隔 %d", got)
	}
	// 窗口声明头必须告知「部分视图」语义
	if !strings.Contains(out, "更早章节的剧情同样有效") {
		t.Fatalf("缺窗口声明头: %s", out)
	}
}

// TestBuildPrevSummaryWindow_Truncate 单章 180 rune 截断（MuMu :1408 同参）。
func TestBuildPrevSummaryWindow_Truncate(t *testing.T) {
	nodes := makePrevNodes(3, 300) // 每章摘要远超 180 rune
	out := buildPrevSummaryWindow(nodes, 4, func(n types.OutlineNode) string { return n.Summary })

	for _, line := range strings.Split(out, "\n\n")[1:] {
		if runeLen(line) > ctxPrevChapterLen+runeLen("第200章：") {
			t.Fatalf("单章行超预算: %q (%d rune)", line, runeLen(line))
		}
		if strings.Count(line, "事") > ctxPrevChapterLen {
			t.Fatalf("截断未生效: %q", line)
		}
	}
}

// TestBuildPrevSummaryWindow_FallbackChain 回退链：大纲 Summary → 章节摘要文件 → 跳过。
func TestBuildPrevSummaryWindow_FallbackChain(t *testing.T) {
	nodes := []types.OutlineNode{
		{OrderIndex: 1, Summary: ""},  // 大纲空 → 回退章节摘要「章摘1」
		{OrderIndex: 2, Summary: "大纲摘要2"},
		{OrderIndex: 3, Summary: ""},  // 两路都空 → 整章跳过
		{OrderIndex: 4, Summary: "大纲摘要4"},
	}
	// 解析器模拟回退链：大纲空时返回章摘（章 1 有、章 3 无）
	resolve := func(n types.OutlineNode) string {
		if n.OrderIndex == 1 {
			return "章摘1"
		}
		if n.OrderIndex == 3 {
			return ""
		}
		return n.Summary
	}
	out := buildPrevSummaryWindow(nodes, 5, resolve)

	if strings.Contains(out, "第1章：大纲") || !strings.Contains(out, "第1章：章摘1") {
		t.Fatalf("章 1 应回退到章节摘要: %s", out)
	}
	if !strings.Contains(out, "第2章：大纲摘要2") || !strings.Contains(out, "第4章：大纲摘要4") {
		t.Fatalf("有摘要章应原样注入: %s", out)
	}
	if strings.Contains(out, "第3章") {
		t.Fatalf("两路皆空的章 3 应整章跳过: %s", out)
	}
}

// TestBuildPrevSummaryWindow_OrderAndEmpty 节点乱序仍按章号升序；无可注入返回 ""；
// limitChapter 很小（开新书）时窗口钳到第 1 章起。
func TestBuildPrevSummaryWindow_OrderAndEmpty(t *testing.T) {
	nodes := []types.OutlineNode{
		{OrderIndex: 3, Summary: "三"},
		{OrderIndex: 1, Summary: "一"},
		{OrderIndex: 2, Summary: "二"},
	}
	out := buildPrevSummaryWindow(nodes, 4, func(n types.OutlineNode) string { return n.Summary })
	i1, i2, i3 := strings.Index(out, "第1章"), strings.Index(out, "第2章"), strings.Index(out, "第3章")
	if i1 < 0 || i2 < 0 || i3 < 0 || !(i1 < i2 && i2 < i3) {
		t.Fatalf("应按章号升序: %s", out)
	}

	// 空：无节点 / resolver 恒空 / limitChapter<=0
	if got := buildPrevSummaryWindow(nil, 5, func(n types.OutlineNode) string { return n.Summary }); got != "" {
		t.Fatalf("空节点应返回空串: %q", got)
	}
	if got := buildPrevSummaryWindow(makePrevNodes(3, 1), 5, func(types.OutlineNode) string { return "" }); got != "" {
		t.Fatalf("全空摘要应返回空串: %q", got)
	}
	if got := buildPrevSummaryWindow(makePrevNodes(3, 1), 0, func(n types.OutlineNode) string { return n.Summary }); got != "" {
		t.Fatalf("非法 limitChapter 应返回空串: %q", got)
	}

	// 开新书：写第 3 章（limit=3），窗口钳到第 1 章起
	out = buildPrevSummaryWindow(makePrevNodes(2, 1), 3, func(n types.OutlineNode) string { return n.Summary })
	if !strings.Contains(out, "第1章") || !strings.Contains(out, "第2章") {
		t.Fatalf("小 limit 窗口应钳到第 1 章起: %s", out)
	}
}
