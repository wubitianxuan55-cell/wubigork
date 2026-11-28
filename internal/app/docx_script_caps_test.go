package app

// docx_script_caps_test.go — 2026-11-28 黄甲反馈回归钉：create_docx.py 能力探测。
// 脚本查找在工作区没播种 docx 技能时回落到用户 skills 镜像，那里的旧副本不认
// --toc-spec 等扩展参数——此前原样发过去就是 argparse 报错整单失败（实测一次
// 硬失败、四次降级成空目录域）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// fakeOldCreateDocx 旧版镜像的形态：argparse 只认基础参数，收到未知参数 exit 2；
// 把实际收到的参数写进「输出文件」供断言。
const fakeOldCreateDocx = `import argparse, json, sys
ap = argparse.ArgumentParser()
ap.add_argument("input")
ap.add_argument("output")
ap.add_argument("--spec", default="")
ap.add_argument("--title", default="")
ap.add_argument("--cover", action="store_true")
ap.add_argument("--toc", action="store_true")
ap.add_argument("--header", default="")
ap.add_argument("--footer", default="")
ap.add_argument("--font", default="")
ap.add_argument("--template", default="")
ns = ap.parse_args()
with open(ns.output, "w", encoding="utf-8") as f:
    f.write(json.dumps(sys.argv[2:], ensure_ascii=False))
`

// fakeCurrentCreateDocx 现行脚本形态：在旧版基础上补齐七个扩展参数（插在
// parse_args 之前，argparse 才认）。
func fakeCurrentCreateDocx() string {
	extra := ""
	for _, f := range docxExtFlags {
		extra += "ap.add_argument('" + f.flag + "', default='')\n"
	}
	return strings.Replace(fakeOldCreateDocx, "ns = ap.parse_args()", extra+"ns = ap.parse_args()", 1)
}

func writeFakeDocxScript(t *testing.T, ws, src string) string {
	t.Helper()
	dir := filepath.Join(ws, ".gaea", "skills", "docx", "scripts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "create_docx.py")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDocxScriptMissing(t *testing.T) {
	ws := t.TempDir()
	old := writeFakeDocxScript(t, ws, fakeOldCreateDocx)
	got := docxScriptMissing(old)
	want := "--subtitle --author --bid-copy --bid-date --toc-spec --heading-numbering --template-file"
	if strings.Join(got, " ") != want {
		t.Fatalf("旧版镜像缺的扩展参数\n got %v\nwant %s", got, want)
	}
	if docxScriptSupports(old, "--toc-spec") || !docxScriptSupports(old, "--template") {
		t.Fatal("能力判定不对")
	}
	note := docxScriptDegradeNote(old, []string{"--subtitle", "--toc-spec"})
	for _, want := range []string{"旧版", "--subtitle --toc-spec", "封面缺副标题", "F9", ".gaea/skills/docx/scripts/create_docx.py"} {
		if !strings.Contains(note, want) {
			t.Errorf("降级备注应含 %q：%s", want, note)
		}
	}
}

func TestDocxScriptMissingCurrentScriptAndNonScript(t *testing.T) {
	current := fakeCurrentCreateDocx()
	path := writeFakeDocxScript(t, t.TempDir(), current)
	if missing := docxScriptMissing(path); missing != nil {
		t.Fatalf("现行脚本不该缺参数：%v", missing)
	}

	// 不是 create_docx.py 形态（没有 add_argument）：不裁剪。
	other := filepath.Join(t.TempDir(), "x.py")
	os.WriteFile(other, []byte("print('hi')\n"), 0o644)
	if missing := docxScriptMissing(other); missing != nil {
		t.Fatalf("非 create_docx.py 形态不该裁剪：%v", missing)
	}
	if missing := docxScriptMissing(filepath.Join(t.TempDir(), "nope.py")); missing != nil {
		t.Fatalf("读不到脚本不该裁剪：%v", missing)
	}

	// 缓存跟着文件变：补上参数后重新探测。
	os.WriteFile(path, []byte(current), 0o644)
	trimmed := strings.ReplaceAll(current, "ap.add_argument('--toc-spec', default='')\n", "")
	os.WriteFile(path, []byte(trimmed), 0o644)
	if docxScriptSupports(path, "--toc-spec") {
		t.Fatal("脚本改了，能力缓存应失效")
	}
}

// 对旧版镜像导出：不支持的参数不发（发了就是 exit 2 整单失败），导出成功并带回
// 降级备注；支持的基础参数照发。
func TestExportDocxNotesSkipsUnsupportedFlags(t *testing.T) {
	ws := t.TempDir()
	script := writeFakeDocxScript(t, ws, fakeOldCreateDocx)
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	defer func() { ga.cfg = oldCfg }() // 全局态必须归还（隔离并行用例）

	out := filepath.Join(ws, "报告.docx")
	note, err := exportDocxNotes(ExportDeliverableInput{
		Markdown:         "# 第一章 概述\n\n正文。\n",
		Template:         "标书",
		Cover:            true,
		TOC:              true,
		Subtitle:         "副标题",
		Author:           "编制单位",
		BidDate:          "2026 年 10 月",
		TocEntries:       []ExportTocEntry{{Text: "第一章 概述", Level: 1, Page: "3"}},
		HeadingNumbering: "on",
	}, "测试交付", out)
	if err != nil {
		t.Fatalf("对旧版镜像导出不该整单失败：%v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("应产出文件：%v", err)
	}
	var argv []string
	if err := json.Unmarshal(raw, &argv); err != nil {
		t.Fatalf("假脚本应写回参数清单：%v (%s)", err, raw)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"--cover", "--toc", "--template 标书", "--title 测试交付"} {
		if !strings.Contains(joined, want) {
			t.Errorf("基础参数应照发，缺 %q：%s", want, joined)
		}
	}
	for _, banned := range []string{"--subtitle", "--author", "--bid-date", "--toc-spec", "--heading-numbering", "--template-file"} {
		if strings.Contains(joined, banned) {
			t.Errorf("旧版不认的参数不该发：%q (%s)", banned, joined)
		}
	}
	for _, want := range []string{"旧版", script, "--subtitle --author --bid-date --toc-spec --heading-numbering", "封面缺副标题", "F9", "标题编号联动未生效"} {
		if !strings.Contains(note, want) {
			t.Errorf("降级备注应含 %q：%s", want, note)
		}
	}

	// 现行脚本：参数全发、无降级备注。
	ws2 := t.TempDir()
	writeFakeDocxScript(t, ws2, fakeCurrentCreateDocx())
	ga.cfg = &gaeaConfig.Config{Workspace: ws2}
	out2 := filepath.Join(ws2, "报告.docx")
	note, err = exportDocxNotes(ExportDeliverableInput{
		Markdown: "# t\n\nx\n", Template: "标书", Subtitle: "副标题", TocEntries: []ExportTocEntry{{Text: "t", Level: 1, Page: "1"}},
	}, "标题", out2)
	if err != nil {
		t.Fatalf("现行脚本导出应成功：%v", err)
	}
	raw, _ = os.ReadFile(out2)
	_ = json.Unmarshal(raw, &argv)
	if !strings.Contains(strings.Join(argv, " "), "--subtitle 副标题") || !strings.Contains(strings.Join(argv, " "), "--toc-spec") {
		t.Errorf("现行脚本应收到全部参数：%s", raw)
	}
	if note != "" {
		t.Fatalf("现行脚本不该有降级备注：%s", note)
	}
}

func TestBuiltinDocxTemplateName(t *testing.T) {
	for name, want := range map[string]string{"标书": "标书", "公文": "公文", "": "通用", "客户甲格式": "通用"} {
		if got := builtinDocxTemplateName(name); got != want {
			t.Errorf("%q → %q, want %q", name, got, want)
		}
	}
}
