package noveltext

// 切分工具单测（审计 IN1-04）：段落 Idx/Line 双口径、rune 区间往返、句切分口径。
// 行为基线 = 被替换的 novelreview.splitParagraphs 与 novelgate.splitSentences 原实现。

import (
	"strings"
	"testing"
)

func TestSplitParagraphs_DoubleNumbering(t *testing.T) {
	// 布局：第1行有段；第2行空；第3行全空白（占行号不占段号）；第4行带缩进空格；
	// 第5行正常。Idx 只数非空段，Line 空行也占号——novelreview 证据用 Idx、
	// novelgate「第 N 行」证据用 Line，两口径同源不漂移。
	text := "第一段。\n\n　\t \n  第二段。  \n第三段，继续。"
	paras := SplitParagraphs([]rune(text))
	if len(paras) != 3 {
		t.Fatalf("应切出 3 段，实际 %d: %+v", len(paras), paras)
	}
	want := []struct {
		idx, line int
		seg       string // Start..End 对应的原始片段
		text      string // 去空白后的段文
	}{
		{1, 1, "第一段。", "第一段。"},
		{2, 4, "  第二段。  ", "第二段。"},
		{3, 5, "第三段，继续。", "第三段，继续。"},
	}
	runes := []rune(text)
	for i, w := range want {
		pa := paras[i]
		if pa.Idx != w.idx || pa.Line != w.line {
			t.Fatalf("段 %d 编号不符: got Idx=%d Line=%d, want Idx=%d Line=%d", i, pa.Idx, pa.Line, w.idx, w.line)
		}
		if pa.Text != w.text {
			t.Fatalf("段 %d 文本不符: got %q, want %q", i, pa.Text, w.text)
		}
		if pa.Start < 0 || pa.End > len(runes) || pa.Start >= pa.End {
			t.Fatalf("段 %d 区间非法: [%d,%d)，全文 %d rune", i, pa.Start, pa.End, len(runes))
		}
		if got := string(runes[pa.Start:pa.End]); got != w.seg {
			t.Fatalf("段 %d 区间往返不符: got %q, want %q", i, got, w.seg)
		}
	}
}

func TestSplitParagraphs_CRLFAndTail(t *testing.T) {
	// \r\n 行尾：\r 计入段文区间、Text 去掉；末行无换行符也要收段。
	text := "甲段。\r\n乙段，未收尾。"
	paras := SplitParagraphs([]rune(text))
	if len(paras) != 2 {
		t.Fatalf("应切出 2 段，实际 %d: %+v", len(paras), paras)
	}
	if paras[0].Text != "甲段。" || paras[1].Text != "乙段，未收尾。" {
		t.Fatalf("段文不符: %q / %q", paras[0].Text, paras[1].Text)
	}
	if paras[1].Line != 2 {
		t.Fatalf("末行行号应为 2，实际 %d", paras[1].Line)
	}
}

func TestSplitParagraphs_EmptyAndBlank(t *testing.T) {
	for _, text := range []string{"", "   ", "\n\n\n", "　"} {
		if got := SplitParagraphs([]rune(text)); len(got) != 0 {
			t.Fatalf("空白输入 %q 应得 0 段，实际 %d", text, len(got))
		}
	}
}

func TestSplitSentences_SplitAndDropEmpty(t *testing.T) {
	got := SplitSentences("他站住！等等……她回头了？！门开了。")
	want := []string{"他站住", "等等", "她回头了", "门开了"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("切句不符: got %v, want %v", got, want)
	}
	// 连续句末标点算一个分隔（不产生空句）；无句末标点整段算一句。
	if got := SplitSentences("！！？！？"); len(got) != 0 {
		t.Fatalf("纯标点应得 0 句，实际 %v", got)
	}
	if got := SplitSentences("没有句末标点的一段话"); len(got) != 1 || got[0] != "没有句末标点的一段话" {
		t.Fatalf("无句末标点应整段一句，实际 %v", got)
	}
	if got := SplitSentences("   "); len(got) != 0 {
		t.Fatalf("空白输入应得 0 句，实际 %v", got)
	}
}

// TestTrigramRepeatRate_Baseline 阶段八 8.7 重复率基准：复读文本显著高于自然
// 行文（复用病历夹具同型语料——flavor 的 Repeat 大段复读 vs healthy 句式轮换）。
func TestTrigramRepeatRate_Baseline(t *testing.T) {
	// 空/超短：0
	if got := TrigramRepeatRate(""); got != 0 {
		t.Fatalf("空文本: %v", got)
	}
	if got := TrigramRepeatRate("十二"); got != 0 {
		t.Fatalf("短文本: %v", got)
	}
	// 复读（flavor 病历同型）：大段 Repeat → 高重复率
	flavor := strings.Repeat("他感到无比的震惊。此外，值得注意的是。", 8)
	flavorRate := TrigramRepeatRate(flavor)
	if flavorRate < 0.8 {
		t.Fatalf("复读文本重复率应 ≥0.8: %v", flavorRate)
	}
	// 自然行文（healthy 同型）：句式轮换 → 低重复率
	healthy := "落了一夜的雨。他把伞收在门边。「你还回来。」她说，声音没有起伏。他喉头一紧，把到嘴的话咽了回去。灯花跳了一下，两个人都没动。账都清了，屋里只剩灯芯的响。"
	healthyRate := TrigramRepeatRate(healthy)
	if healthyRate > 0.05 {
		t.Fatalf("自然行文重复率应 ≤0.05: %v", healthyRate)
	}
	if flavorRate <= healthyRate {
		t.Fatalf("复读应显著高于自然行文: flavor=%v healthy=%v", flavorRate, healthyRate)
	}
}
