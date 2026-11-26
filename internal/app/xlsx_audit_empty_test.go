package app

// xlsx_audit_empty_test.go — 「空串结果 ≠ 未重算」在回执层的钉（2026-11-26 造价
// 反馈三批）：测算样表的未用槽位挂着取价公式，480 格空结果此前把交付门判成
// 「未达」，回执又说「结果本应为空的不算问题」，自相矛盾；未重算清单只给数量和
// 前 20 条，定位靠逐格核对。

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/office/xlsxedit"
)

func emptyFindings(n int) []xlsxedit.AuditFinding {
	fs := make([]xlsxedit.AuditFinding, 0, n)
	for i := 0; i < n/2; i++ {
		row := 6 + i
		fs = append(fs,
			xlsxedit.AuditFinding{Sheet: "综合单价", Cell: fmt.Sprintf("J%d", row), Kind: "empty_result",
				Formula: fmt.Sprintf(`IFERROR(VLOOKUP(I%d,工料机价格!$A$4:$G$123,4,FALSE()),"")`, row)},
			xlsxedit.AuditFinding{Sheet: "综合单价", Cell: fmt.Sprintf("K%d", row), Kind: "empty_result",
				Formula: fmt.Sprintf(`IFERROR(VLOOKUP(I%d,工料机价格!$A$4:$G$123,6,FALSE()),"")`, row)},
		)
	}
	return fs
}

func TestRecalcReceiptEmptyResultsPassTheGate(t *testing.T) {
	audit := xlsxedit.AuditReport{Sheets: 6, TotalFormulas: 1230, Findings: emptyFindings(480)}
	out := formatRecalcReceipt(xlsxedit.RecalcReport{TotalFormulas: 1230, TotalErrors: 0}, audit, nil)
	for _, want := range []string{
		"零错误，达到交付门",
		"另有 480 个公式的结果是空串",
		"综合单价!J×240、综合单价!K×240",
		"clear_where",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：\n%s", want, out)
		}
	}
	if strings.Contains(out, "未达交付门") || strings.Contains(out, "原样再调一次 xlsx_recalc") {
		t.Fatalf("空结果不该判未达、也不该催着重算（三场实测就是这样白耗了十几次重算）：\n%s", out)
	}
}

func TestRecalcReceiptRealMissingCacheStillBlocks(t *testing.T) {
	audit := xlsxedit.AuditReport{Sheets: 2, TotalFormulas: 20, Findings: append(emptyFindings(8),
		xlsxedit.AuditFinding{Sheet: "清单", Cell: "H4", Kind: "missing_cache", Formula: "=综合单价!S4"},
		xlsxedit.AuditFinding{Sheet: "清单", Cell: "H5", Kind: "missing_cache", Formula: "=综合单价!S15"},
	)}
	out := formatRecalcReceipt(xlsxedit.RecalcReport{TotalFormulas: 20, TotalErrors: 0}, audit, nil)
	for _, want := range []string{"未达交付门", "[未重算] 清单!H4", "另有 8 个空结果公式"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：\n%s", want, out)
		}
	}
}

func TestAuditReportClustersMissingCacheAndSeparatesEmptyResults(t *testing.T) {
	var fs []xlsxedit.AuditFinding
	for i := 0; i < 6; i++ {
		fs = append(fs, xlsxedit.AuditFinding{Sheet: "清单", Cell: fmt.Sprintf("H%d", 4+i), Kind: "missing_cache",
			Formula: fmt.Sprintf("=综合单价!S%d", 4+i*11)})
	}
	fs = append(fs, emptyFindings(4)...)
	out := formatAuditReport(xlsxedit.AuditReport{Sheets: 2, TotalFormulas: 30, Findings: fs}, 40)
	for _, want := range []string{
		"发现 6 条问题（错误值 0 / 自引用 0 / 未重算 6），另有 4 个空结果公式（不阻塞）",
		"未重算分布：位置 清单!H×6；公式 `=综合单价!S#`×6",
		"另有 4 个公式的结果是空串",
		"clear_where",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("报告缺 %q：\n%s", want, out)
		}
	}
	// 只有空结果：独立复核口径同样算达到交付门。
	only := formatAuditReport(xlsxedit.AuditReport{Sheets: 1, TotalFormulas: 9, Findings: emptyFindings(8)}, 40)
	if !strings.Contains(only, "达到交付门") || strings.Contains(only, "xlsx_recalc") {
		t.Fatalf("只有空结果时应判达到交付门、不催重算：\n%s", only)
	}
}
