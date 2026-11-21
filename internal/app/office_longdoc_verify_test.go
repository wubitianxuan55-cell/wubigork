package app

// office_longdoc_verify_test.go — 验证体系工具化四门单测（沿江1公里复盘 §七
// 脚本内核的回归钉）。夹具全自造（md/假 docx zip/程序化 PNG），零外部依赖。

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ldvRun(t *testing.T, args string) (string, error) {
	t.Helper()
	return longdocVerifyTool{a: &App{}}.Execute(context.Background(), json.RawMessage(args))
}

func TestLongdocVerifyChapterFullChain(t *testing.T) {
	dir := t.TempDir()
	// 源图存在（图引可解析）。
	if err := os.WriteFile(filepath.Join(dir, "fig1.png"), []byte("fake-png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	md := "# 第三章 质量保障\n\n" +
		"布置见图3-1，参数见表3-1。\n\n" +
		"![平面布置](fig1.png)\n\n" +
		"图3-1 平面布置图\n\n" +
		"表3-1 参数表\n\n" +
		"正文不得使用项目经理称谓。\n\n" +
		"## 本章图表索引\n\n" +
		"| 图3-1 |\n| 表3-1 |\n"
	p := filepath.Join(dir, "第3章.md")
	if err := os.WriteFile(p, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	// 通过面：题注↔引用↔索引闭环+图引存在+禁用词只有否定语境。
	out, err := ldvRun(t, `{"action":"chapter","path":`+ldvJSON(p)+`,"forbidden":["项目经理"],"keys":["图3-1"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "✓ 全部通过") {
		t.Fatalf("干净章应通过：\n%s", out)
	}
	if !strings.Contains(out, "否定语境排除 1") {
		t.Fatalf("否定语境应排除 1 处：\n%s", out)
	}
	// 破坏面：跳号+孤立引用+索引缺条+禁用词真命中+旧口径。
	md2 := "# 第三章\n\n" +
		"见 图3-1 与 图3-3 和 图9-9。\n\n" +
		"图3-1 甲\n\n图3-3 丙\n\n" +
		"本方案由项目经理负责。\n\n" +
		"旧口径 588 m²。\n\n" +
		"## 本章图表索引\n\n| 图3-1 |\n"
	p2 := filepath.Join(dir, "第3章b.md")
	if err := os.WriteFile(p2, []byte(md2), 0o644); err != nil {
		t.Fatal(err)
	}
	out2, err := ldvRun(t, `{"action":"chapter","path":`+ldvJSON(p2)+`,"forbidden":["项目经理"],"bad_keys":["588 m²"]}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"跳号", "索引缺条目", "真命中 1", "旧口径残留", "跨章引用"} {
		if !strings.Contains(out2, want) {
			t.Errorf("破坏面应报 %q：\n%s", want, out2)
		}
	}
	if strings.Contains(out2, "孤立引用") {
		t.Errorf("跨章引用（图9-9）chapter 门不裁决（cross 门并集裁决）：\n%s", out2)
	}
	if strings.Contains(out2, "✓ 全部通过") {
		t.Errorf("破坏面不应通过：\n%s", out2)
	}
}

func TestLongdocVerifyCrossUniqueness(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "第3章.md")
	b := filepath.Join(dir, "第4章.md")
	os.WriteFile(a, []byte("见 图3-1。\n\n图3-1 甲\n"), 0o644)
	os.WriteFile(b, []byte("见 图3-1。\n\n图3-1 乙\n"), 0o644) // 跨章重号
	out, err := ldvRun(t, `{"action":"cross","paths":[`+ldvJSON(a)+`,`+ldvJSON(b)+`],"keys":["图3-1"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "跨章重复") || strings.Contains(out, "✓ 全部通过") {
		t.Fatalf("跨章重号应 fail：\n%s", out)
	}
	// 唯一面通过（引用↔题注并集闭环）。
	os.WriteFile(b, []byte("见 图4-1。\n\n图4-1 乙\n"), 0o644)
	out2, err := ldvRun(t, `{"action":"cross","paths":[`+ldvJSON(a)+`,`+ldvJSON(b)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "✓ 全部通过") {
		t.Fatalf("编号唯一且并集闭环应通过：\n%s", out2)
	}
	// 并集孤儿：引用全册不存在的编号 → cross 门裁决（chapter 门放行的跨章引用在此闭合）。
	os.WriteFile(b, []byte("见 图9-9（应为笔误）。\n\n图4-1 乙\n"), 0o644)
	out3, err := ldvRun(t, `{"action":"cross","paths":[`+ldvJSON(a)+`,`+ldvJSON(b)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
		if !strings.Contains(out3, "孤立引用（全册并集）") {
			t.Fatalf("全册孤儿引用应 fail：\n%s", out3)
		}
}

// 语义初筛（2026-11-21 反馈 P0-3 回归钉）：编号存在/对账齐但内容错换的「引用
// 错表」——低重合标记不判 fail（初筛有误报率），重合的正常过筛。实测形态：
// 第3章「节点日期取自第2章表2-1」而表2-1 已是职责分工表（里程碑表重排走了）。
func TestLongdocVerifyCrossSemanticPrescreen(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "第2章.md")
	b := filepath.Join(dir, "第3章.md")
	os.WriteFile(a, []byte("职责划分见表2-1；里程碑安排见表2-2。\n\n表2-1 主要职责分工表\n\n表2-2 里程碑节点计划表\n"), 0o644)
	os.WriteFile(b, []byte("N4～N9 节点日期取自第2章表2-1，形成总控节点。\n\n施工里程碑服从表2-2 的节点安排。\n"), 0o644)
	out, err := ldvRun(t, `{"action":"cross","paths":[`+ldvJSON(a)+`,`+ldvJSON(b)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[1s] 跨章引用语义初筛：2 处") {
		t.Fatalf("应列出 2 处跨章引用初筛：\n%s", out)
	}
	if !strings.Contains(out, "表2-1") || !strings.Contains(out, "低重合·请核对") {
		t.Fatalf("引用错表（节点日期→职责分工表）应标记低重合：\n%s", out)
	}
	if !strings.Contains(out, "被引「表2-2 里程碑节点计划表」") {
		t.Fatalf("初筛应输出被引对象真实标题：\n%s", out)
	}
	if !strings.Contains(out, "重合3（") { // 施工里程碑…节点安排 ↔ 里程碑节点计划表：程碑/节点/里程 共享 3 个 bigram
		t.Fatalf("里程碑句对里程碑表应过筛（重合3）：\n%s", out)
	}
	if !strings.Contains(out, "✓ 全部通过") {
		t.Fatalf("语义初筛不判 fail，本夹具无其他硬伤应全过：\n%s", out)
	}
}

// ldvFakeDocx 造最小合法 docx zip（document/styles/header/footer）。
func ldvFakeDocx(t *testing.T, path, bodyText string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w := func(name, content string) {
		f, _ := zw.Create(name)
		_, _ = f.Write([]byte(content))
	}
	w("word/document.xml", `<w:document><w:body>`+bodyText+`<w:sectPr/></w:body></w:document>`)
	w("word/styles.xml", `<w:styles><w:style w:styleId="3"><w:name w:val="heading 1"/><w:color w:val="000000"/><w:sz w:val="32"/><w:pPr><w:pageBreakBefore/></w:pPr></w:style></w:styles>`)
	w("word/header1.xml", `<w:hdr><w:t>某公司</w:t></w:hdr>`)
	w("word/footer1.xml", `<w:ftr><w:instrText> PAGE </w:instrText></w:ftr>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLongdocVerifyBook(t *testing.T) {
	dir := t.TempDir()
	ch := filepath.Join(dir, "第1章.md")
	os.WriteFile(ch, []byte("# 第一章 总体方案\n\n正文。"), 0o644)
	dx := filepath.Join(dir, "整册.docx")
	ldvFakeDocx(t, dx, `<w:p><w:t>第一章 总体方案 与残留占位【待补充】</w:t></w:p>`)

	out, err := ldvRun(t, `{"action":"book","docx":`+ldvJSON(dx)+`,"paths":[`+ldvJSON(ch)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"章标题在册：第一章 总体方案", "heading 1", "颜色=000000", "页眉 有", "PAGE", "分节数 1", "【待补充】"} {
		if !strings.Contains(out, want) {
			t.Errorf("book 回执应含 %q：\n%s", want, out)
		}
	}
	if strings.Contains(out, "✓ 全部通过") {
		t.Errorf("占位残留应 fail：\n%s", out)
	}
	// 章标题不在册 → fail。
	dx2 := filepath.Join(dir, "整册2.docx")
	ldvFakeDocx(t, dx2, `<w:p><w:t>另一份内容</w:t></w:p>`)
	out2, err := ldvRun(t, `{"action":"book","docx":`+ldvJSON(dx2)+`,"paths":[`+ldvJSON(ch)+`]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "章标题未出现在整册正文") {
		t.Fatalf("章标题漂移应 fail：\n%s", out2)
	}
}

func TestLongdocVerifyInk(t *testing.T) {
	dir := t.TempDir()
	// 100×100：上带纯黑（判黑），下带纯蓝（判彩）。
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if y < 50 {
				img.Set(x, y, color.RGBA{10, 10, 10, 255})
			} else {
				img.Set(x, y, color.RGBA{40, 70, 200, 255})
			}
		}
	}
	p := filepath.Join(dir, "page.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	out, err := ldvRun(t, `{"action":"ink","png":`+ldvJSON(p)+`,"bands":[{"label":"标题行","top":0.1,"bottom":0.4},{"label":"正文带","top":0.6,"bottom":0.9}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "标题行") || !strings.Contains(out, "→ 黑色") {
		t.Fatalf("黑带应判黑色：\n%s", out)
	}
	if !strings.Contains(out, "→ 蓝/彩色") {
		t.Fatalf("蓝带应判彩色：\n%s", out)
	}
	if !strings.Contains(out, "✓ 全部通过") {
		t.Fatalf("判色本身不产生 fail：\n%s", out)
	}
}

func TestLongdocVerifyReadTarget(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "第3章.md")
	os.WriteFile(p, []byte("# x"), 0o644)
	got, ok := longdocVerifyTool{}.ReadTarget(json.RawMessage(`{"action":"chapter","path":` + ldvJSON(p) + `}`))
	if !ok || got != p {
		t.Fatalf("ReadTarget=%q,%v", got, ok)
	}
	if _, ok2 := (longdocVerifyTool{}).ReadTarget(json.RawMessage(`{"action":"cross"}`)); ok2 {
		t.Fatal("cross 无 paths 应 false")
	}
}

func ldvJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
