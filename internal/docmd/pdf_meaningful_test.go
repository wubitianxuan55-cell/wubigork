package docmd

import (
	"strings"
	"testing"
)

// meaningfulText 双闸守卫（2026-10-28 实弹回归）：Chrome 图片型 PDF 的解压流垃圾
// 30 万字符里随机蹦出 1088 个汉字（占比 0.37%、56% 行含 3+ 连续符号），旧闸
// cjk>=5 被放行入库成知识库乱码正文，OCR 回退被短路。垃圾不得放行；
// 真中文/真英文/字面串提取产物不得误杀。
func TestMeaningfulText(t *testing.T) {
	// 垃圾模拟：与实弹同型——高熵符号串行占绝对多数 + 稀疏随机汉字（6 个 ≥5，
	// 旧绝对数闸会放行；新垃圾行占比闸与密度闸都拦下）。
	var junk strings.Builder
	junk.WriteString("%PDF-1.4\n%\n/Producer (Skia/PDF m116)\n/ca 1\n/LC 0\n/SA true\n")
	for i := 0; i < 100; i++ {
		junk.WriteString("xy<;tnKERQ\tm*)T$T#$VdKi071T#|wv(p#CCC09<Z@kqcWZ0@kh-`!Z\\Zh-C\n")
	}
	junk.WriteString("缟a1SH$kijj\n矕R@kqh-qHs?`_|\n缟矕缟\n")

	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"真中文散文", "福建省环境监测行业指导价适用于全省环境监测机构。水质监测每点位每次收费三百元至八百元，收费标准按照监测点位、频次和项目复杂程度确定。", true},
		{"中文表格混排", "类别 项目 收费标准\n水质 pH值 80\n噪声 厂界噪声 200\n大气 颗粒物 180", true},
		{"真英文散文", "The environmental monitoring guidance price applies to all institutions in Fujian province.\nThe fee standard is determined by monitoring points, frequency and project complexity.\nWater quality monitoring costs 300 to 800 yuan per point per time.", true},
		{"压缩流垃圾含稀疏随机汉字", junk.String(), false},
		{"PDF结构字典", "obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\ntrailer\nstartxref\n%%EOF", false},
		{"空串", "", false},
		{"纯空白", "   \n\t\n", false},
	}
	for _, c := range cases {
		if got := meaningfulText(c.in); got != c.want {
			t.Errorf("%s: meaningfulText = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestExtractRawTextRejectsBinaryJunk 图片型 PDF 文本提取为空时的兜底提取：
// 解压流垃圾（长符号串+稀疏随机汉字）必须回空串让上层走 OCR，不得当正文。
func TestExtractRawTextRejectsBinaryJunk(t *testing.T) {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n%\n/Producer (Skia/PDF m116)\n/CreationDate (D:20260920133528+00'00')\n")
	b.WriteString("/ca 1\n/LC 0\n/LJ 0\n/LW 1\n/ML 4\n/SA true\n")
	for i := 0; i < 60; i++ {
		b.WriteString("xy<;tnKERQ\tm*)T$T#$VdKi071T#|wv(p#CCC09<Z@kqcWZ0@kh-`!Z\\Zh-C\n")
	}
	b.WriteString("缟a1SH$kijj\n矕R@kqh-qHs?`_|\n")
	got := extractRawText([]byte(b.String()))
	if got != "" {
		r := []rune(got)
		t.Fatalf("二进制垃圾应回空串（走 OCR），实际 %d 字符，head=%q", len(r), r[:min(80, len(r))])
	}
}

// TestExtractRawTextKeepsLiteralText 正向对照：字面串文本（中/英）必须照常提取。
// 注：正文行不能以 " Tj" 结尾——isPDFKeyword 按 " Tj" 后缀滤指令行（既有口径），
// 只有操作符独立成行的生成器产物才会走到 extractRawText 的可读文本出口。
func TestExtractRawTextKeepsLiteralText(t *testing.T) {
	data := []byte("%PDF-1.4\n1 0 obj\n<</Type/Page>>\nendobj\n" +
		"BT\n/F1 12 Tf\n(Fujian Environmental Monitoring Fee Guide)\nTj\nET\n" +
		"BT\n(Water quality pH COD total nitrogen)\nTj\nET\n" +
		"BT\n(水质监测每点位每次收费三百元至八百元)\nTj\nET\n" +
		"trailer\n%%EOF\n")
	got := extractRawText(data)
	if !strings.Contains(got, "Fujian Environmental") {
		t.Fatalf("英文字面串应保留: %q", got)
	}
	if !strings.Contains(got, "水质监测每点位每次收费三百元至八百元") {
		t.Fatalf("中文字面串应保留: %q", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
