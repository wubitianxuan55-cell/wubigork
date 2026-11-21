package app

// office_longdoc_image_test.go — 2026-11-21 方案模式复盘批的图引面与防删源
// 回归钉（A1/A2/A3/B9/C3）：
//
//   - longdocCheck 图引机检：缺失/外链=硬伤，存在=放行；
//   - longdocAbsolutizeImages：相对路径按章节目录改写绝对（正斜杠），绝对/外链不动；
//   - compactLongdocFindings：拒绝理由单行（下游按首行截断不再吃掉明细）；
//   - longdocCountEmbeddedImages：docx zip 数 drawing；
//   - exportDocx 同 stem 防删源：python 成败皆不得触碰同 stem 源 md。

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// ormRepoRootForSkills 走到仓根（findSkillScript 依赖 ga.cfg.Workspace）。
func ormRepoRootForSkills(t *testing.T) string {
	t.Helper()
	root, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(root, ".gaea", "skills", "docx", "scripts", "create_docx.py")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("未找到仓库 .gaea/skills")
		}
		root = parent
	}
}

func TestLongdocCheckImageFindings(t *testing.T) {
	dir := t.TempDir()
	chDir := filepath.Join(dir, "ch1")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chDir, "ok.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	md := "# 第一章\n\n![好图](ok.png)\n\n![缺图](missing.png)\n\n![外链](https://example.com/x.png)\n"
	if err := os.WriteFile(filepath.Join(chDir, "第1章.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &longdocManifestFile{Title: "整册", Chapters: []longdocChapter{{No: 1, Title: "第一章", Path: filepath.Join(chDir, "第1章.md")}}}
	findings := longdocCheck(m, filepath.Join(dir, "manifest.json"))
	hard := 0
	var details string
	for _, f := range findings {
		if f.Hard {
			hard++
			details += f.Detail + "\n"
		}
	}
	if hard != 2 {
		t.Fatalf("缺图+外链应各报 1 项硬伤（共 2），got %d：%s", hard, details)
	}
	if !strings.Contains(details, "missing.png") || !strings.Contains(details, "外链") {
		t.Fatalf("硬伤应点名缺失文件与外链：%s", details)
	}
	if strings.Contains(details, "ok.png") {
		t.Fatalf("存在的图不应报：%s", details)
	}
}

func TestLongdocAbsolutizeImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := "# 第一章\n\n![题](a.png)\n\n![缺](nope.png)\n\n![外](https://e/x.png)\n\n![绝](C:/abs/x.png)\n"
	got := text
	n := longdocAbsolutizeImages(&got, dir)
	if n != 1 {
		t.Fatalf("只应改写 1 处（存在的相对路径），got %d：%s", n, got)
	}
	want := filepath.ToSlash(filepath.Join(dir, "a.png"))
	if !strings.Contains(got, "("+want+")") {
		t.Fatalf("相对路径应改写为绝对（正斜杠）：%s", got)
	}
	for _, keep := range []string{"(nope.png)", "(https://e/x.png)", "(C:/abs/x.png)"} {
		if !strings.Contains(got, keep) {
			t.Fatalf("%s 不应被改写：%s", keep, got)
		}
	}
}

func TestCompactLongdocFindingsSingleLine(t *testing.T) {
	fs := []longdocFinding{
		{Code: "a", Detail: "第一项", Hard: true},
		{Code: "b", Detail: "第二项", Hard: true},
		{Code: "c", Detail: "信息项", Hard: false},
	}
	s := compactLongdocFindings(fs)
	if strings.ContainsAny(s, "\n\r") {
		t.Fatalf("拒绝理由必须单行（B9 首行截断）：%q", s)
	}
	if !strings.Contains(s, "[a] 第一项") || !strings.Contains(s, "[b] 第二项") || strings.Contains(s, "信息项") {
		t.Fatalf("应含全部硬伤且不含信息项：%s", s)
	}
}

func TestLongdocCountEmbeddedImages(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.docx")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte("<w:body><w:p><w:drawing/></w:p><w:p><w:drawing/></w:p><w:p><w:drawing/></w:p></w:body>"))
	_ = zw.Close()
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := longdocCountEmbeddedImages(p)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("应数出 3 个 drawing，got %d", n)
	}
}

// TestExportDocxSameStemNeverTouchesSourceMD A1/A2 回归钉：输出 docx 与章节
// 源 md 同 stem 时，导出成败皆不得覆写/删除源 md（python 失败路径同样覆盖）。
func TestExportDocxSameStemNeverTouchesSourceMD(t *testing.T) {
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ormRepoRootForSkills(t)}
	defer func() { ga.cfg = oldCfg }() // 全局态必须归还（隔离并行用例）
	dir := t.TempDir()
	src := filepath.Join(dir, "第1章 项目实施方案.md")
	if err := os.WriteFile(src, []byte("章节源内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "第1章 项目实施方案.docx")
	_ = exportDocx(ExportDeliverableInput{Markdown: "整册内容"}, "第1章", out) // python 成败皆可：契约与成败无关
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("源 md 被删除（A1 复发）: %v", err)
	}
	if string(b) != "章节源内容" {
		t.Fatalf("源 md 被覆写（A1 复发）: %q", string(b))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".gaea-tmp-") {
			t.Fatalf("中间临时件泄漏: %s", e.Name())
		}
	}
}

// TestLongdocBuildRejectMessageSingleLine B9 回归钉：拒绝组装的错误单行且带
// 硬伤明细（不再「冒号后为空」）。走 Execute 的机检闸前段：清单缺章即拒。
func TestLongdocBuildRejectMessageSingleLine(t *testing.T) {
	dir := t.TempDir()
	mf := filepath.Join(dir, "manifest.json")
	m := longdocManifestFile{Title: "整册", Chapters: []longdocChapter{{No: 1, Title: "第一章", Path: filepath.Join(dir, "missing.md")}}}
	b, _ := json.Marshal(m)
	if err := os.WriteFile(mf, b, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := longdocBuildTool{a: &App{}}.Execute(nil, json.RawMessage(`{"manifest":"`+filepath.ToSlash(mf)+`","format":"docx","output_path":"`+filepath.ToSlash(filepath.Join(dir, "out.docx"))+`"}`))
	if err == nil {
		t.Fatal("缺章应拒绝组装")
	}
	if strings.ContainsAny(err.Error(), "\n\r") {
		t.Fatalf("拒绝理由必须单行（B9）：%q", err.Error())
	}
	if !strings.Contains(err.Error(), "missing_chapter") {
		t.Fatalf("拒绝理由应带硬伤明细：%v", err)
	}
}
