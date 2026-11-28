package tool

// 2026-11-28 黄甲反馈回归钉：发布边界认「随迁」。工作区整理时 bash mv 目录，目录
// 里的既有 Office 件跟着搬家——此前一律按 created 越界件整体隔离，主控只能逐件
// move_file 取回（实测 7 批拦截、120 次取回，占全会话工具调用 28%）。随迁=不是
// 发布；但豁免面→正式面、复制、mv 覆盖同名正式件三类照旧拦。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func relocationPairs(rs []PublishRelocation) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.From+" → "+r.To)
	}
	return out
}

func renameWs(t *testing.T, ws, from, to string) {
	t.Helper()
	dst := filepath.Join(ws, filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(ws, filepath.FromSlash(from)), dst); err != nil {
		t.Fatal(err)
	}
}

// 目录整体 mv：目录里的既有正式件全是随迁件，不隔离、不进事后体检面。
func TestPublishGuardRelocationWithinWorkspacePasses(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "施工日志", "SG-01.docx"), "日志一")
	writePublishFile(t, filepath.Join(ws, "施工日志", "SG-02.docx"), "日志二")
	writePublishFile(t, filepath.Join(ws, "施工日志", "说明.md"), "非 durable")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	renameWs(t, ws, "施工日志", "03_实施过程/施工日志")

	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("工作区内目录移动不该判越界：%+v", vs)
	}
	rels := g.Relocations()
	if len(rels) != 2 {
		t.Fatalf("应认出 2 件随迁件：%+v", rels)
	}
	got := strings.Join(relocationPairs(rels), "|")
	want := "施工日志/SG-01.docx → 03_实施过程/施工日志/SG-01.docx|施工日志/SG-02.docx → 03_实施过程/施工日志/SG-02.docx"
	if got != want {
		t.Fatalf("随迁件清单不对\n got %s\nwant %s", got, want)
	}
	if touched := g.Touched(); len(touched) != 0 {
		t.Fatalf("随迁件不是写出来的，不该进事后体检面：%+v", touched)
	}
}

// 单件改名（含改扩展名大小写之外的纯改名）同属随迁。
func TestPublishGuardRenameFormalFilePasses(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "开工筹备计划.docx"), "计划")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	renameWs(t, ws, "开工筹备计划.docx", "归档/历史草稿/开工筹备计划.docx")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 || len(g.Relocations()) != 1 {
		t.Fatalf("既有正式件改名应放行并记为随迁：vs=%+v rels=%+v", vs, g.Relocations())
	}
}

// 源头在豁免面（.gaea/work/、_draft/）：过程件经 bash 落进正式名是发布，照拦。
func TestPublishGuardRelocationFromExemptStillCaught(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, ".gaea", "work", "测算.xlsx"), "过程件")
	writePublishFile(t, filepath.Join(ws, "_draft", "报告.docx"), "草稿")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	renameWs(t, ws, ".gaea/work/测算.xlsx", "交付/测算.xlsx")
	renameWs(t, ws, "_draft/报告.docx", "交付/报告.docx")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(violationRels(vs), "|")
	if got != "交付/报告.docx|交付/测算.xlsx" {
		t.Fatalf("豁免面 → 正式面应照拦：got %s", got)
	}
	if len(g.Relocations()) != 0 {
		t.Fatalf("不该有随迁件：%+v", g.Relocations())
	}
}

// 复制：源头还在，新路径上多出一份正式件就是发布。
func TestPublishGuardCopyStillCaught(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "a", "方案.docx"), "正文")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(ws, "b", "方案.docx"), "正文")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(violationRels(vs), "|"); got != "b/方案.docx" {
		t.Fatalf("cp 应照拦：got %s", got)
	}
	if len(g.Relocations()) != 0 {
		t.Fatalf("复制不是随迁：%+v", g.Relocations())
	}
}

// mv 覆盖同名正式件：原内容已被盖掉，是 modified，照拦。
func TestPublishGuardMoveOverExistingStillCaught(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "a", "方案.docx"), "新一点的正文")
	writePublishFile(t, filepath.Join(ws, "b", "方案.docx"), "旧")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(ws, "b", "方案.docx")); err != nil {
		t.Fatal(err)
	}
	renameWs(t, ws, "a/方案.docx", "b/方案.docx")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Kind != "modified" || vs[0].RelPath != "b/方案.docx" {
		t.Fatalf("mv 覆盖同名正式件应判 modified：%+v", vs)
	}
	if len(g.Relocations()) != 0 {
		t.Fatalf("覆盖不是随迁：%+v", g.Relocations())
	}
}

// 同一批里既有随迁又有真发布：随迁放行，真发布照拦。
func TestPublishGuardRelocationMixedWithRealPublish(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "旧", "既有.xlsx"), "既有")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	renameWs(t, ws, "旧/既有.xlsx", "新/既有.xlsx")
	writePublishFile(t, filepath.Join(ws, "新", "脚本现写.xlsx"), "现写")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(violationRels(vs), "|"); got != "新/脚本现写.xlsx" {
		t.Fatalf("只该拦脚本现写的那件：got %s", got)
	}
	if len(g.Relocations()) != 1 {
		t.Fatalf("应认出 1 件随迁件：%+v", g.Relocations())
	}
}
