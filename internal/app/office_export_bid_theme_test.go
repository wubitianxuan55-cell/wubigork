package app

// office_export_bid_theme_test.go — 2026-11-21 反馈 P0-2 回归钉：标书模板口径。
// 报告主题的微软雅黑直接改变页数口径（实测同一内容 472→419 页，页数是投标
// 显性指标），主题蓝标题不符标书口径。标书主题必须：宋体正文/黑体黑标题
// （样式层+run 双保险）/章标题另起页（样式层 pageBreakBefore，longdoc_verify
// book 读数所查面）/页眉下框线+缺省取投标人/封面五要素不残留占位文字。

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestExportDocxBidTheme(t *testing.T) {
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ormRepoRootForSkills(t)}
	defer func() { ga.cfg = oldCfg }() // 全局态必须归还（隔离并行用例）
	dir := t.TempDir()
	out := filepath.Join(dir, "技术标.docx")
	err := exportDocx(ExportDeliverableInput{
		Markdown: "# 第一章 总体方案\n\n正文段落。\n\n## 1.1 编制依据\n\n依据内容。\n",
		Template: "标书",
		Cover:    true,
		TOC:      true,
		Subtitle: "CQS26C02978 包3",
		Author:   "某某环境科技有限公司",
	}, "沿江一公里技术标", out)
	if err != nil {
		t.Skipf("python 链不可用，跳过实弹：%v", err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	read := func(name string) string {
		for _, f := range zr.File {
			if f.Name == name {
				rc, oerr := f.Open()
				if oerr != nil {
					t.Fatal(oerr)
				}
				defer rc.Close()
				b, _ := io.ReadAll(rc)
				return string(b)
			}
		}
		return ""
	}
	var hdr strings.Builder
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "word/header") {
			rc, oerr := f.Open()
			if oerr == nil {
				b, _ := io.ReadAll(rc)
				_ = rc.Close()
				hdr.Write(b)
			}
		}
	}
	doc, sty, hdrStr := read("word/document.xml"), read("word/styles.xml"), hdr.String()
	checks := map[string]bool{
		"正文 eastAsia 宋体": strings.Contains(doc, `w:eastAsia="宋体"`),
		"标题黑体":           strings.Contains(doc, `w:eastAsia="黑体"`),
		"标题色黑":           strings.Contains(doc, `<w:color w:val="000000"`),
		"无主题蓝标题":         !strings.Contains(doc, "365F91") && !strings.Contains(doc, "4F81BD"),
		"封面投标人":          strings.Contains(doc, "投标人：某某环境科技有限公司"),
		"封面包号":           strings.Contains(doc, "CQS26C02978 包3"),
		"封面正本标记":         strings.Contains(doc, "正本"),
		"H1 样式另起页（样式层）":  strings.Contains(sty, "pageBreakBefore"),
		"H1 样式黑体黑（样式层）":  strings.Contains(sty, `w:eastAsia="黑体"`) && strings.Contains(sty, `<w:color w:val="000000"`),
		"页眉下框线":          strings.Contains(hdrStr, "<w:pBdr>") && strings.Contains(hdrStr, "w:bottom"),
		"页眉缺省取投标人":       strings.Contains(hdrStr, "某某环境科技有限公司"),
		"无占位封面文字":        !strings.Contains(doc, "gaea 交付物"),
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("标书主题缺：%s", name)
		}
	}
}
