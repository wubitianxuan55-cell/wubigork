package builtin

// md_validate_content.go — 内容保真对账（沿江包1 复盘 T4/P1-3 的刀4）：
// 以 PDF 原件文字层为基准，对派生 md 的表格单元格与段落做 12 字滑窗 n-gram
// 命中率对账。命中率低 = 转换侧有错字/丢字/多字——单看任一版本都发现不了
// （复盘实测：Word 侧独有错字 15 处、上标负号丢 218 处，都是对账抓出来的）。
//
// 算法照搬沿江包1 项目目录验证过的 verify_vs_pdf.py：归一（去图引/软换行/
// 空白、NFKC 全角半角与上下标统一、下划线视作连字符）→ 原件全文 12-gram 集 →
// 逐单元格/逐段落算命中率。表格命中率 <98%、段落缺失占比 >5% 列硬伤。
// 位置定位到 md 行号，附缺失片段样例；基线对比按表头签名/段首主键跨版本对齐。

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

const (
	mdvGramN          = 12   // 滑窗长度
	mdvTableRateMin   = 0.98 // 表格命中率阈值（复盘口径）
	mdvParaMissMax    = 0.05 // 段落缺失占比阈值
	mdvParaMinRunes   = 30   // 段落参与对账的最短归一字数
	mdvCellMinRunes   = 12   // 单元格参与对账的最短归一字数
	mdvMaxContentHits = 60   // 单次对账硬伤上限（防一页坏表刷屏）
)

var (
	reContentImgRef = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reContentDash   = regexp.MustCompile(`^-{2,}$`)
)

// mdvContentNorm 对账归一：去图引/软换行/md 管道与标题记号/全部空白，NFKC
// 统一全角半角与上下标（³→3、（→( ），WPS 用下划线代替上标负号的痕迹视为 -
// （PDF 侧 NFKC 后上标负号同样是 -）。两侧同函数，保证可比。
func mdvContentNorm(s string) string {
	s = reContentImgRef.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "<br>", "")
	s = strings.ReplaceAll(s, "\\|", "|")
	s = strings.NewReplacer("|", "", "#", "", "*", "", ">", "").Replace(s)
	s = norm.NFKC.String(s)
	s = strings.ReplaceAll(s, "_", "-")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\u00a0', '\u3000', '\u2028':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// mdvGrams 12 字滑窗。
func mdvGrams(s string) []string {
	rs := []rune(s)
	if len(rs) < mdvGramN {
		return nil
	}
	out := make([]string, 0, len(rs)-mdvGramN+1)
	for i := 0; i+mdvGramN <= len(rs); i++ {
		out = append(out, string(rs[i:i+mdvGramN]))
	}
	return out
}

// mdvContentIssue 内容对账的一条硬伤（含缺失片段样例）。
type mdvContentIssue struct {
	issue     mdvIssue
	hitRate   float64
	missParts string
}

// mdvCheckContent 对账一个派生文件（lines）对 PDF 原件文字（已归一全集 gramSet）。
// 返回表格与段落两层硬伤。
func mdvCheckContent(file string, lines []string, gramSet map[string]bool) []mdvIssue {
	// 切表格块（连续 | 行）。
	type tblBlock struct {
		line   int
		rows   []string
		header string
	}
	var blocks []tblBlock
	var cur tblBlock
	inTbl := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "|") {
			if !inTbl {
				cur = tblBlock{line: i + 1}
				inTbl = true
			}
			cur.rows = append(cur.rows, l)
			continue
		}
		if inTbl {
			blocks = append(blocks, cur)
			inTbl = false
		}
	}
	if inTbl {
		blocks = append(blocks, cur)
	}

	var out []mdvIssue
	add := func(is mdvContentIssue) {
		if len(out) >= mdvMaxContentHits {
			return
		}
		out = append(out, is.issue)
	}

	tblNo := 0
	for _, b := range blocks {
		tblNo++
		header := ""
		var cells []string
		for ri, row := range b.rows {
			if reContentDash.MatchString(strings.Trim(row, "| ")) || isMDVSeparatorRow(row) {
				continue
			}
			for _, c := range strings.Split(strings.Trim(strings.TrimSpace(row), "|"), "|") {
				cc := mdvContentNorm(c)
				if len([]rune(cc)) >= mdvCellMinRunes {
					cells = append(cells, cc)
				}
			}
			if ri == 0 {
				header = mdvTableKey(row)
			}
		}
		if len(cells) == 0 {
			continue
		}
		missTotal, gramTotal := 0, 0
		var worstMiss int
		var worstCell, worstPart string
		for _, c := range cells {
			gs := mdvGrams(c)
			if gs == nil {
				continue
			}
			var miss []string
			for _, g := range gs {
				if !gramSet[g] {
					miss = append(miss, g)
				}
			}
			missTotal += len(miss)
			gramTotal += len(gs)
			if len(miss) > worstMiss {
				worstMiss = len(miss)
				worstCell = c
				worstPart = strings.Join(miss[:min(mdvGramN*2, len(miss))], "")
			}
		}
		if gramTotal == 0 {
			continue
		}
		rate := 1 - float64(missTotal)/float64(gramTotal)
		if rate >= mdvTableRateMin {
			continue
		}
		add(mdvContentIssue{hitRate: rate, missParts: worstPart, issue: mdvIssue{
			kind: "content", file: file, line: b.line, severity: mdvHard,
			key: "content:tbl:" + header,
			summary: fmt.Sprintf("内容对账：表#%d（L%d，%d 格）对原件命中率 %.1f%%（缺 %d/%d 窗）——转换侧疑有错字/丢字，可疑片段「%s」",
				tblNo, b.line, len(cells), rate*100, missTotal, gramTotal, clipRunes(worstCell, 40)),
		}})
	}

	// 段落层：≥30 归一字、缺失占比 >5%。
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "|") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "![") {
			continue
		}
		cc := mdvContentNorm(t)
		if len([]rune(cc)) < mdvParaMinRunes {
			continue
		}
		gs := mdvGrams(cc)
		if gs == nil {
			continue
		}
		var miss []string
		for _, g := range gs {
			if !gramSet[g] {
				miss = append(miss, g)
			}
		}
		if float64(len(miss))/float64(len(gs)) <= mdvParaMissMax {
			continue
		}
		add(mdvContentIssue{issue: mdvIssue{
			kind: "content", file: file, line: i + 1, severity: mdvHard,
			key: "content:p:" + clipRunes(cc, 24),
			summary: fmt.Sprintf("内容对账：段落（L%d）对原件缺失占比 %.1f%%——可疑片段「%s」原文「%s」",
				i+1, float64(len(miss))/float64(len(gs))*100, strings.Join(miss[:min(2, len(miss))], ""), clipRunes(t, 40)),
		}})
	}
	return out
}

// mdvGramSet 原件文字的 12-gram 全集。
func mdvGramSet(pdfText string) map[string]bool {
	set := make(map[string]bool)
	for _, g := range mdvGrams(mdvContentNorm(pdfText)) {
		set[g] = true
	}
	return set
}

func clipRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
