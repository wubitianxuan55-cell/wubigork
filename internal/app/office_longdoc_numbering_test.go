package app

// office_longdoc_numbering_test.go — 2026-11-24 编号联动回归钉：docx 标题编号
// 改样式层多级列表（create_docx.py）后，字面前缀从标题文字剥离，机检对账走
// 裸题名兜底。本文件钉住：前缀剥离口径（含日期/序次语义不误剥）、章标题在册
// 裸题名对账、静态目录量页裸题名兜底。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLongdocStripHeadingNumber(t *testing.T) {
	cases := [][2]string{
		{"第1章 项目实施方案", "项目实施方案"},
		{"第一章 总体方案", "总体方案"},
		{"第12篇 编制说明", "编制说明"},
		{"1.1 项目背景", "项目背景"},
		{"1.2.3 修复技术筛选", "修复技术筛选"},
		{"1、总体要求", "总体要求"},
		{"1 概述", "概述"},
		{"（一）组织保障", "组织保障"},
		{"(三)安全目标", "安全目标"},
		{"一、编制依据", "编制依据"},
		{"第1章 1.1 双前缀", "双前缀"},
		// 数字开头的非编号标题不误剥（日期/序次/量纲语义）。
		{"2026 年度工作计划", "2026 年度工作计划"},
		{"12 月进度安排", "12 月进度安排"},
		{"2026年度工作计划", "2026年度工作计划"},
		{"GB 36600 相关标准", "GB 36600 相关标准"},
		{"3D 建模流程", "3D 建模流程"},
		{"项目概述与背景", "项目概述与背景"},
		{"", ""},
	}
	for _, c := range cases {
		if got := longdocStripHeadingNumber(c[0]); got != c[1] {
			t.Errorf("strip(%q)=%q want %q", c[0], got, c[1])
		}
	}
}

// 章标题在册：docx 标题文字不带字面编号（编号引擎渲染）时按裸题名对账。
func TestLongdocVerifyBookBareTitleFallback(t *testing.T) {
	dir := t.TempDir()
	ch := filepath.Join(dir, "第1章.md")
	os.WriteFile(ch, []byte("# 第一章 总体方案\n\n正文。"), 0o644)
	// docx 只含裸题名（编号联动后的真实形态：编号在多级列表里，不在 w:t）。
	dx := filepath.Join(dir, "整册.docx")
	ldvFakeDocx(t, dx, `<w:p><w:t>总体方案</w:t></w:p>`)

	out, err := ldvRun(t, `{"action":"book","docx":`+ldvJSON(dx)+`,"paths":[`+ldvJSON(ch)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "✗") || strings.Contains(out, "章标题未出现") {
		t.Fatalf("裸题名应兜底通过：\n%s", out)
	}
	if !strings.Contains(out, "按裸题名对账") {
		t.Fatalf("回执应标注裸题名对账口径：\n%s", out)
	}
	// 连裸题名都不在 → 仍 fail。
	dx2 := filepath.Join(dir, "整册2.docx")
	ldvFakeDocx(t, dx2, `<w:p><w:t>无关内容</w:t></w:p>`)
	out2, err := ldvRun(t, `{"action":"book","docx":`+ldvJSON(dx2)+`,"paths":[`+ldvJSON(ch)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "章标题未出现在整册正文") {
		t.Fatalf("题名漂移应 fail：\n%s", out2)
	}
}

// 静态目录量页：md 字面前缀异写（第一章）时裸题名兜底仍能定位页码。
func TestMeasureTocEntriesBareFallback(t *testing.T) {
	in := ExportDeliverableInput{TocEntries: []ExportTocEntry{
		{Text: "第一章 总体方案", Level: 1, Page: "—"},
	}}
	export := func(in ExportDeliverableInput) (ExportDeliverableResult, error) {
		return ExportDeliverableResult{Path: "unused"}, nil
	}
	resolve := func(string) (string, error) { return "unused", nil }
	// 换掉 PDF 抽页腿：渲染文本只有引擎口径「第1章 总体方案」（无字面前缀）。
	origConv, origPages := longdocConvertToPdf, longdocPDFPageTexts
	longdocConvertToPdf = func(string, string) error { return nil }
	longdocPDFPageTexts = func(string) ([]string, error) {
		return []string{"封面", "目录 第1章 总体方案……3", "第1章 总体方案 正文 3"}, nil
	}
	defer func() { longdocConvertToPdf, longdocPDFPageTexts = origConv, origPages }()
	entries, err := measureTocEntries(in, export, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Page == "—" {
		t.Fatalf("裸题名兜底应回填页码：%+v", entries)
	}
}
