package app

// xlsx_audit_recalc_test.go — P1-2 质量门禁两件的工具层测试。
// audit 引擎的文件扫描在 xlsxedit 包有 hermetic 测试；这里钉：
//   - 工具契约（名/只读性/schema/WritePaths/ReadTarget）；
//   - formatAuditReport 排版纯函数（错误值优先、未重算折叠、截断计数、交付门文案）。

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
)

func TestXlsxAuditRecalcContract(t *testing.T) {
	audit := xlsxAuditTool{a: &App{}}
	if audit.Name() != "xlsx_audit" || !audit.ReadOnly() {
		t.Fatal("xlsx_audit 契约：只读工具名固定")
	}
	recalc := xlsxRecalcTool{a: &App{}}
	if recalc.Name() != "xlsx_recalc" || recalc.ReadOnly() {
		t.Fatal("xlsx_recalc 契约：写工具名固定且不标只读")
	}
	if !json.Valid(audit.Schema()) || !json.Valid(recalc.Schema()) {
		t.Fatal("schema 非法 JSON")
	}
	// recalc 写面：WritePaths 从 args.path 声明。
	wp, err := recalc.WritePaths(json.RawMessage(`{"path":"表.xlsx"}`))
	if err != nil || len(wp) != 1 || !strings.HasSuffix(wp[0], "表.xlsx") {
		t.Fatalf("xlsx_recalc WritePaths 应声明 path：%v %v", wp, err)
	}
}

func TestXlsxAuditMissingArgs(t *testing.T) {
	audit := xlsxAuditTool{a: &App{}}
	if _, err := audit.Execute(context.Background(), json.RawMessage(`{bad`)); err == nil ||
		!strings.Contains(err.Error(), "xlsx_audit 参数解析失败") {
		t.Fatalf("坏参应带工具名（错误三要素守卫），实际：%v", err)
	}
	if _, err := audit.Execute(context.Background(), json.RawMessage(`{}`)); err == nil ||
		!strings.Contains(err.Error(), "缺少参数") {
		t.Fatalf("缺 path 应给字段名，实际：%v", err)
	}
	recalc := xlsxRecalcTool{a: &App{}}
	if _, err := recalc.Execute(context.Background(), json.RawMessage(`{"path":""}`)); err == nil ||
		!strings.Contains(err.Error(), "缺少参数 path") {
		t.Fatalf("recalc 缺 path 应给字段名，实际：%v", err)
	}
}

func TestFormatAuditReportClean(t *testing.T) {
	out := formatAuditReport(xlsxedit.AuditReport{Sheets: 2, TotalFormulas: 12, Findings: []xlsxedit.AuditFinding{}}, 0)
	if !strings.Contains(out, "零错误零异常") || !strings.Contains(out, "12 个公式") {
		t.Fatalf("clean 报告应达交付门：\n%s", out)
	}
}

func TestFormatAuditReportPrioritizesErrors(t *testing.T) {
	rep := xlsxedit.AuditReport{
		Sheets: 1, TotalFormulas: 5,
		Findings: []xlsxedit.AuditFinding{
			{Sheet: "Sheet1", Cell: "C3", Kind: "missing_cache", Formula: "=A1*B1"},
			{Sheet: "Sheet1", Cell: "B2", Kind: "error_value", Value: "#DIV/0!", Formula: "=1/0"},
			{Sheet: "Sheet1", Cell: "D4", Kind: "self_ref", Formula: "=D4+1"},
		},
	}
	out := formatAuditReport(rep, 0)
	// 错误值行必须在自引用行之前（硬伤优先）。
	if strings.Index(out, "[错误值]") > strings.Index(out, "[自引用]") {
		t.Fatalf("错误值应排在自引用之前：\n%s", out)
	}
	for _, want := range []string{"#DIV/0!", "[自引用]", "D4+1", "[未重算]", "xlsx_recalc", "零错误才算交付"} {
		if !strings.Contains(out, want) {
			t.Fatalf("报告缺 %q：\n%s", want, out)
		}
	}
}

func TestFormatAuditReportLimitShowsRemainder(t *testing.T) {
	var findings []xlsxedit.AuditFinding
	for i := 0; i < 8; i++ {
		findings = append(findings, xlsxedit.AuditFinding{
			Sheet: "Sheet1", Cell: cellNameN(i+1), Kind: "missing_cache", Formula: "=A1",
		})
	}
	out := formatAuditReport(xlsxedit.AuditReport{Sheets: 1, TotalFormulas: 8, Findings: findings}, 3)
	if strings.Count(out, "[未重算]") != 3 || !strings.Contains(out, "其余 5 条未展示") {
		t.Fatalf("limit 截断应展示 3 条+剩余计数：\n%s", out)
	}
}

func TestFormatAuditReportTruncated(t *testing.T) {
	out := formatAuditReport(xlsxedit.AuditReport{Truncated: true, Sheets: 1}, 0)
	if !strings.Contains(out, "截断") {
		t.Fatalf("截断应明说：\n%s", out)
	}
}

// TestXlsxAuditExecuteOnFixture 端到端：合成文件（有公式无缓存）走真实 Execute。
func TestXlsxAuditExecuteOnFixture(t *testing.T) {
	dir := t.TempDir()
	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", 10)
	_ = f.SetCellValue("Sheet1", "A2", 20)
	_ = f.SetCellFormula("Sheet1", "A3", "=SUM(A1:A2)")
	p := filepath.Join(dir, "f.xlsx")
	if err := f.SaveAs(p); err != nil {
		t.Fatalf("保存夹具失败：%v", err)
	}
	_ = f.Close()

	audit := xlsxAuditTool{a: &App{}}
	out, err := audit.Execute(context.Background(), json.RawMessage(`{"path":` + jsonStr(p) + `}`))
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	if !strings.Contains(out, "未重算") || !strings.Contains(out, "A3") {
		t.Fatalf("端到端应报 A3 未重算：\n%s", out)
	}
}

func cellNameN(col int) string {
	return string(rune('A'+col-1)) + "1"
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
