package builtin

// 写入内容体检（2026-11-28 黄甲反馈）：一次重写约 500 行的大纲，生成层输出退化
// ——同一行连续重复（"12 tables"×5、"discard"×4）、整份文档从头重启三遍、夹满
// "garbage lines" 一类元注释——write_file 照常落盘且回执 ok=true，约 52KB 垃圾
// 盖掉了干净文件（靠写前快照救回）。
//
// 硬拦信号取「相邻相同行」，不取重复行占比：该场会话 22 次合法写入的重复行占比
// 是 9%~14%（待补清单、表格行天然重复），污染件是 9.5%——占比分不开；而合法写入
// 的最长相邻重复全是 1，污染件是 5。占比只进回执的体检行，不参与拦截。

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	// repeatRunHardLimit 相邻相同行达到这个次数即判退化、硬拦。
	repeatRunHardLimit = 4
	// repeatMinRunes 短于此的行不参与相邻重复判定（"end"、"--" 这类天然重复）。
	repeatMinRunes = 6
	// dupLongLineRunes 重复行占比只统计不短于此的行。
	dupLongLineRunes = 15
	// healthMinLines 非空行少于这个数且没有相邻重复时不印体检行（小文件不添噪音）。
	healthMinLines = 30
)

// contentHealth 一份待写内容的重复行画像。
type contentHealth struct {
	Lines      int     // 非空行数
	DupRatio   float64 // 非短行里「出现不止一次的行」的占比
	MaxRun     int     // 相邻相同行的最长连续次数（按 repeatCandidate 口径）
	MaxRunAt   int     // 那一段的起始行号（1 起；MaxRun<2 时为 0）
	MaxRunLine string  // 那一行
}

// inspectContent 统计重复行画像。行按去首尾空白后的文本比较；CRLF 归一成 LF。
func inspectContent(content string) contentHealth {
	lines := splitContentLines(content)
	h := contentHealth{MaxRun: 1}
	counts := map[string]int{}
	longLines := 0
	run, runStart, prev := 0, 0, ""
	for i, raw := range lines {
		s := strings.TrimSpace(raw)
		if s == "" {
			prev = "" // 空行打断相邻
			continue
		}
		h.Lines++
		if utf8RuneLen(s) >= dupLongLineRunes {
			counts[s]++
			longLines++
		}
		if !repeatCandidate(s) {
			prev = ""
			continue
		}
		if s == prev {
			run++
		} else {
			run, runStart = 1, i+1
		}
		prev = s
		if run > h.MaxRun {
			h.MaxRun, h.MaxRunAt, h.MaxRunLine = run, runStart, s
		}
	}
	if longLines > 0 {
		dup := 0
		for _, c := range counts {
			if c > 1 {
				dup += c
			}
		}
		h.DupRatio = float64(dup) / float64(longLines)
	}
	return h
}

// repeatCandidate 这一行参与相邻重复判定：长度够、含字母或数字（排除 "====="、
// "| --- | --- |"、空表格行这类结构性/装饰性重复）。
func repeatCandidate(s string) bool {
	if utf8RuneLen(s) < repeatMinRunes {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// degenerate 是否达到硬拦线。
func (h contentHealth) degenerate() bool { return h.MaxRun >= repeatRunHardLimit }

// blockError 硬拦报错：说清是哪一行、怎么绕（确属有意重复时显式声明）。
func (h contentHealth) blockError() error {
	line := h.MaxRunLine
	if r := []rune(line); len(r) > 40 {
		line = string(r[:40]) + "…"
	}
	return codedError("WRITE_DEGENERATE_CONTENT",
		"写入被拦，未写盘：内容疑似生成层退化——第 %d 行起「%s」连续重复 %d 次（阈值 %d）。重新生成本段（长内容分片写、每片不超过 80 行、写后回读核对）；确属有意重复请传 allow_repeats:true",
		h.MaxRunAt, line, h.MaxRun, repeatRunHardLimit)
}

// receiptLine 回执里的体检行；内容不大又没有相邻重复时返回空串。
func (h contentHealth) receiptLine() string {
	if h.Lines < healthMinLines && h.MaxRun < 2 {
		return ""
	}
	line := fmt.Sprintf("\n内容体检：%d 行，重复行 %.0f%%，最长连续重复 %d", h.Lines, h.DupRatio*100, h.MaxRun)
	if h.MaxRun >= 2 {
		line += fmt.Sprintf("（第 %d 行起「%s」，回读核对）", h.MaxRunAt, truncateRunes(h.MaxRunLine, 24))
	}
	return line
}

func splitContentLines(content string) []string {
	return strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
}

func utf8RuneLen(s string) int { return len([]rune(s)) }

func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
