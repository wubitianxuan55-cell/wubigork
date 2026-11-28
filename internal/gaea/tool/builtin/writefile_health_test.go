package builtin

// writefile_health_test.go — 2026-11-28 黄甲反馈回归钉：生成层退化的内容（同一行
// 连续重复）在落盘前被拦下；结构性/装饰性重复与合法的高重复占比不误伤。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeArgs(t *testing.T, m map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInspectContent_PollutedOutlineIsDegenerate(t *testing.T) {
	// 取自实测污染件的两种形态："12 tables"×5 与 "discard"×4（中间隔着别的行）。
	var b strings.Builder
	b.WriteString("# 目录大纲\n\n## 一、配额总表\n\n正文一段。\n\n")
	for i := 0; i < 5; i++ {
		b.WriteString("12 tables\n")
	}
	b.WriteString("\n别的行\n\n")
	for i := 0; i < 4; i++ {
		b.WriteString("discard\n")
	}
	h := inspectContent(b.String())
	if !h.degenerate() {
		t.Fatalf("应判退化：%+v", h)
	}
	if h.MaxRun != 5 || h.MaxRunAt != 7 || h.MaxRunLine != "12 tables" {
		t.Fatalf("应点名最长的那一段：%+v", h)
	}
}

func TestInspectContent_StructuralRepeatsAreFine(t *testing.T) {
	cases := map[string]string{
		"分隔线":    strings.Repeat("==========\n", 6),
		"表格分隔行":  strings.Repeat("| --- | --- | --- |\n", 5),
		"空表格行":   strings.Repeat("|    |    |    |\n", 5),
		"破折号占位行": strings.Repeat("| — | — | — |\n", 5),
		"短行":     strings.Repeat("end\n", 6),
		"空行打断":   strings.Repeat("## 相同的标题行\n\n", 5),
	}
	for name, body := range cases {
		if h := inspectContent(body); h.degenerate() {
			t.Errorf("%s：结构性/装饰性重复不该判退化：%+v", name, h)
		}
	}
}

// 重复行占比高但无相邻重复（清单、表格行天然重复）是合法内容：不拦，占比只进体检行。
func TestInspectContent_HighDupRatioWithoutRunIsFine(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 20; i++ {
		b.WriteString("| 过程记录表格实际填写件 | 见附件册 | 待补 |\n")
		b.WriteString("| 监测报告原件及 OCR 识别件 | 见检测报告册 | 待补 |\n")
	}
	h := inspectContent(b.String())
	if h.degenerate() {
		t.Fatalf("无相邻重复不该判退化：%+v", h)
	}
	if h.DupRatio < 0.99 {
		t.Fatalf("两种行各重复 20 次，占比应接近 100%%：%+v", h)
	}
	if !strings.Contains(h.receiptLine(), "重复行 100%") || strings.Contains(h.receiptLine(), "回读核对") {
		t.Fatalf("高占比只进体检行、不带重复警示：%q", h.receiptLine())
	}
}

func TestInspectContent_LegitChecklistNotBlocked(t *testing.T) {
	var b strings.Builder
	b.WriteString("# 待补资料统计\n\n| 序号 | 资料 | 状态 |\n| --- | --- | --- |\n")
	for i := 1; i <= 20; i++ {
		b.WriteString("| " + strconv.Itoa(i) + " | 过程记录表格实际填写件（第 " + strconv.Itoa(i) + " 册） | 待补 |\n")
	}
	b.WriteString("\n- 以上各项资料到位后逐项销号，销号情况见本章末表。\n")
	for i := 1; i <= 10; i++ {
		b.WriteString("- 第 " + strconv.Itoa(i) + " 项：待补，责任单位见上表。\n")
	}
	h := inspectContent(b.String())
	if h.degenerate() {
		t.Fatalf("清单类合法内容不该判退化：%+v", h)
	}
	if h.Lines < 30 || !strings.Contains(h.receiptLine(), "内容体检：") || strings.Contains(h.receiptLine(), "回读核对") {
		t.Fatalf("长内容应有体检行且不带重复警示：%q", h.receiptLine())
	}
}

func TestWriteFileBlocksDegenerateContentBeforeWriting(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "outline.md")
	if err := os.WriteFile(path, []byte("干净版本"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := writeFile{workDir: ws, roots: []string{ws}}

	var b strings.Builder
	b.WriteString("# 目录大纲\n\n")
	for i := 0; i < 5; i++ {
		b.WriteString("12 tables\n")
	}
	_, err := w.Execute(context.Background(), writeArgs(t, map[string]any{"path": path, "content": b.String()}))
	if err == nil || !strings.Contains(err.Error(), "WRITE_DEGENERATE_CONTENT") || !strings.Contains(err.Error(), "allow_repeats:true") {
		t.Fatalf("退化内容应被拦并给出绕法：%v", err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != "干净版本" {
		t.Fatalf("被拦时不得写盘：%q", raw)
	}

	// 确属有意重复：显式声明后照常写入。
	if _, err := w.Execute(context.Background(), writeArgs(t, map[string]any{
		"path": path, "content": b.String(), "allow_repeats": true,
	})); err != nil {
		t.Fatalf("声明 allow_repeats 后应写入：%v", err)
	}
}

// 回执体检行：CRLF 内容同样统计；小文件不印。
func TestWriteFileReceiptHealthLine(t *testing.T) {
	ws := t.TempDir()
	w := writeFile{workDir: ws, roots: []string{ws}}

	small := filepath.Join(ws, "small.txt")
	out, err := w.Execute(context.Background(), writeArgs(t, map[string]any{"path": small, "content": "a\nb\n"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "内容体检") {
		t.Fatalf("小文件不该印体检行：%q", out)
	}

	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("第 " + strconv.Itoa(i) + " 行：各不相同的正文内容。\r\n")
	}
	big := filepath.Join(ws, "big.md")
	out, err = w.Execute(context.Background(), writeArgs(t, map[string]any{"path": big, "content": b.String()}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "内容体检：40 行") || !strings.Contains(out, "最长连续重复 1") {
		t.Fatalf("长内容应印体检行：%q", out)
	}

	// 低于阈值的相邻重复：不拦，但体检行点名提醒回读。
	repeat := filepath.Join(ws, "repeat.md")
	out, err = w.Execute(context.Background(), writeArgs(t, map[string]any{
		"path": repeat, "content": "短文\n\n同一行出现三次\n同一行出现三次\n同一行出现三次\n",
	}))
	if err != nil {
		t.Fatalf("未达阈值不该拦：%v", err)
	}
	if !strings.Contains(out, "最长连续重复 3") || !strings.Contains(out, "回读核对") {
		t.Fatalf("未达阈值的相邻重复应在体检行提醒：%q", out)
	}
}
