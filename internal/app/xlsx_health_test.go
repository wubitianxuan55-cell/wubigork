package app

// xlsx_health_test.go — xlsx 健康体检三处接线的回归钉（2026-11-27 AI 反馈 #7 /
// #15③）：xlsx_audit 的体检行（含打不开即报）、xlsx_recalc 前后的结构对账、
// xlsx_apply 重算没做成时缺缓存值如实报出。体检引擎本身在 xlsxedit 包有测试。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
)

// TestRecalcStructureNote 结构对账纯函数：没变给确认、变了点名、任一侧体检没跑成
// 不下结论。
func TestRecalcStructureNote(t *testing.T) {
	before := xlsxedit.Health{Sheets: []string{"封面", "综合单价"}, DefinedNames: 15, SheetScopedNames: 15, DataValidations: 2}
	note := recalcStructureNote(before, nil, before, nil)
	if !strings.HasPrefix(note, "\n结构对账：") || !strings.Contains(note, "一致") || strings.Contains(note, "⚠") {
		t.Fatalf("结构没变应给一句确认：%q", note)
	}
	after := before
	after.SheetScopedNames, after.DataValidations = 0, 1
	note = recalcStructureNote(before, nil, after, nil)
	for _, want := range []string{"⚠ 结构对账", "工作表级归属 15 → 0", "数据有效性 2 → 1", "打印标题"} {
		if !strings.Contains(note, want) {
			t.Errorf("结构对账应含 %q：%s", want, note)
		}
	}
	if note := recalcStructureNote(before, fmt.Errorf("open 失败"), after, nil); note != "" {
		t.Errorf("重算前体检没跑成不下结论：%q", note)
	}
	if note := recalcStructureNote(before, nil, after, fmt.Errorf("open 失败")); note != "" {
		t.Errorf("重算后体检没跑成不下结论：%q", note)
	}
}

// TestXlsxAuditHealthLine 体检行：有公式没缓存值 → 点名并指向重算；重算过的样表 →
// 不催；坏文件 → 直接报打不开，不进单元格扫描。
func TestXlsxAuditHealthLine(t *testing.T) {
	audit := xlsxAuditTool{a: &App{}}
	dir := t.TempDir()
	nocache := filepath.Join(dir, "nocache.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", 2)
	_ = f.SetCellFormula(sheet, "A2", "A1*3")
	if err := f.SaveAs(nocache); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	out, err := audit.Execute(context.Background(), json.RawMessage(`{"path":`+jsonStr(nocache)+`}`))
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	line := healthLineOf(t, out)
	for _, want := range []string{"健康体检：可打开｜公式 1 格，其中 1 格没有缓存值｜", "——没重算，交付前 xlsx_recalc"} {
		if !strings.Contains(line, want) {
			t.Errorf("体检行缺 %q：%s", want, line)
		}
	}

	sample, err := filepath.Abs(filepath.Join("..", "gaea", "skill", "assets", "cost-estimate-sample.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	out, err = audit.Execute(context.Background(), json.RawMessage(`{"path":`+jsonStr(sample)+`}`))
	if err != nil {
		t.Fatalf("样表 audit 失败：%v", err)
	}
	line = healthLineOf(t, out)
	if !strings.Contains(line, "健康体检：可打开｜公式 1") || !strings.Contains(line, "格｜6 表｜定义名 15（表级 15）") {
		t.Errorf("样表体检行应给公式数与结构面：%s", line)
	}
	if strings.Contains(line, "没重算") {
		t.Errorf("重算过的样表不该催重算：%s", line)
	}

	junk := filepath.Join(dir, "junk.xlsx")
	if err := os.WriteFile(junk, []byte("not a zip at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := audit.Execute(context.Background(), json.RawMessage(`{"path":`+jsonStr(junk)+`}`)); err == nil ||
		!strings.Contains(err.Error(), "打不开") || !strings.Contains(err.Error(), "不要去扫单元格") {
		t.Fatalf("坏文件应直接报打不开：out=%q err=%v", out, err)
	}
}

// TestGaeaXlsxApplyEditReportsMissingCache 重算没做成时（本用例显式关掉重算），缺
// 缓存值的格数必须如实报出——此前只有「环境缺失」一种失败会出备注。没有公式的
// 工作簿不出这条（防误报）。
func TestGaeaXlsxApplyEditReportsMissingCache(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GAEA_RECALC_DISABLE", "1")
	a := &App{core: &core{}}
	dir := filepath.Join(".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.ToSlash(filepath.Join(dir, "cache.xlsx"))
	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", 100)
	_ = f.SetCellValue("Sheet1", "A2", 200)
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	r, err := a.GaeaXlsxApplyEdit(rel, `[{"type":"set_formula","sheet":"Sheet1","target":"A3","formula":"SUM(A1:A2)"}]`)
	if err != nil {
		t.Fatalf("应用失败：%v", err)
	}
	for _, want := range []string{"公式重算跳过", "⚠ 1 个公式格没有缓存值", "xlsx_recalc"} {
		if !strings.Contains(r.Summary, want) {
			t.Errorf("重算没做成的回执缺 %q：%s", want, r.Summary)
		}
	}

	plain := filepath.ToSlash(filepath.Join(dir, "plain.xlsx"))
	f2 := excelize.NewFile()
	_ = f2.SetCellValue("Sheet1", "A1", 1)
	if err := f2.SaveAs(plain); err != nil {
		t.Fatal(err)
	}
	_ = f2.Close()
	r, err = a.GaeaXlsxApplyEdit(plain, `[{"type":"set_value","sheet":"Sheet1","target":"A2","value":5}]`)
	if err != nil {
		t.Fatalf("应用失败：%v", err)
	}
	if strings.Contains(r.Summary, "没有缓存值") {
		t.Errorf("没有公式的工作簿不该报缺缓存值：%s", r.Summary)
	}
}

// healthLineOf 取回执里的体检行：办公工具回执是 tool.WrapText 的 JSON 信封，正文
// 在 message 字段（第一段是文件指纹，体检行紧随其后）。
func healthLineOf(t *testing.T, out string) string {
	t.Helper()
	var env struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil || env.Message == "" {
		t.Fatalf("回执应是 WrapText 信封：%s", out)
	}
	for _, line := range strings.Split(env.Message, "\n") {
		if strings.HasPrefix(line, "健康体检：") {
			return line
		}
	}
	t.Fatalf("回执缺体检行：\n%s", env.Message)
	return ""
}
