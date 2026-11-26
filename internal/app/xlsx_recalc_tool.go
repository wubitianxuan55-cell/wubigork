package app

// xlsx_recalc — 灵犀 optimize plan P1-2 的公式重算工具：把 LibreOffice
// 重算链（xlsxedit.Recalc——重算脚本随 gaea 二进制内嵌，2026-11-26 起不再取
// 自技能目录；每次调用一份独立配置目录）暴露成 agent 工具面。原 GaeaXlsxRecalc
// 是前端「重算」钮用的（返回预览投影），本工具给 agent 返回重算报告文本。
//
// 写面：重算会用缓存值改写文件（公式本身不动），因此实现 WritePathResolver
// 声明 path；不落证据卡——刷新的是派生数据（公式结果缓存），公式与用户数据
// 不变，审计面在 xlsx_audit。
//
// 2026-11-25 办公工具梳理二批 B4：回执直接附重算后的审计结果（与 xlsx_audit
// 同一份 formatAuditFindings）。依据=69 个真实会话里 xlsx_recalc 25 次、
// xlsx_audit 22 次，几乎总是「重算→审计」成对再调一次。两件不真合并：一个是
// 写、一个是只读，合了只读子代理就没法做纯审计（复核别人改过的文件）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

type xlsxRecalcTool struct{ a *App }

func (t xlsxRecalcTool) Name() string   { return "xlsx_recalc" }
func (t xlsxRecalcTool) ReadOnly() bool { return false }

func (t xlsxRecalcTool) Description() string {
	return "用 LibreOffice 重算 .xlsx 里全部公式并写回缓存值（公式本身不改，刷新的是结果缓存）。" +
		"openpyxl/excelize 生成的公式没有缓存值（Excel/WPS 打开时才显示结果），交付前必须重算。" +
		"回执直接附审计结果（公式错误值/自引用/未重算），写明是否达到交付门，不必紧接着再调 xlsx_audit；" +
		"重算后又改了文件、或只复核不改文件时才用 xlsx_audit。依赖本机 LibreOffice——未安装时报错如实说明。"
}

func (t xlsxRecalcTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path"],
  "properties":{
    "path":{"type":"string","description":"目标 .xlsx 工作区相对路径"}
  }
}`)
}

func (t xlsxRecalcTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t xlsxRecalcTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("xlsx_recalc 参数解析失败: %w", err)
	}
	if strings.TrimSpace(in.Path) == "" {
		return "", fmt.Errorf("缺少参数 path（目标 .xlsx 工作区相对路径）")
	}
	path := officeTranslateShellPath(in.Path)
	if !filepath.IsAbs(path) {
		p, perr := wspath.ResolveRelWithin(gaeaCwd(), path)
		if perr != nil {
			return "", fmt.Errorf("非法工作区相对路径: %s", path)
		}
		path = p
	}
	if !xlsxedit.AuditFileExists(path) {
		return "", fmt.Errorf("文件不存在：%s", in.Path)
	}
	rep, err := xlsxedit.Recalc(path)
	if err != nil {
		switch {
		case errors.Is(err, xlsxedit.ErrRecalcNotWorkbook):
			return "", fmt.Errorf("文件不是能打开的 .xlsx（zip 结构坏了），重算没做：%v——先核实文件是否写完整，不要去 xlsx_audit 找单元格", err)
		case errors.Is(err, xlsxedit.ErrRecalcRead):
			return "", fmt.Errorf("读取重算结果失败（瞬时读取错误，已自动重试 1 次仍失败）：%v——"+
				"这是文件读取环节的问题，不是公式问题，不要去 xlsx_audit 找单元格；文件多半没坏，原样再调一次 xlsx_recalc", err)
		case errors.Is(err, xlsxedit.ErrRecalcUnavailable):
			return "", fmt.Errorf("公式重算不可用：%v（需要本机 python+openpyxl 与 LibreOffice）。"+
				"重算没做成，文件里的公式缓存值就没刷新——不要声称「公式已计算/结果已校验」，如实告知用户本机缺重算环境；"+
				"确实需要结果数值时，用 excelize CalcCellValue 类思路在脚本里逐格算并明确标注「脚本估算值，非重算缓存」", err)
		default:
			return "", fmt.Errorf("公式重算失败：%v——不要声称重算完成；用 xlsx_audit 看具体单元格后修复再试", err)
		}
	}
	// 重算已成功落盘：随后的审计只是给回执附证据，跑不成也不把这次调用
	// 判成失败（文件确实重算过了），只在回执里如实说明并指向 xlsx_audit。
	audit, auditErr := xlsxedit.Audit(path)
	return formatRecalcReceipt(rep, audit, auditErr), nil
}

// formatRecalcReceipt 渲染重算回执：重算报告 + 交付门结论 + 审计段（纯函数，
// 便于单测）。结论同时看两头——LibreOffice 报的公式错误数，与重算后文件的
// 审计发现（错误值/自引用/未重算）；审计没跑成（auditErr 非 nil，此时 audit
// 是零值、不可信）时不下「已达交付门」的结论。
func formatRecalcReceipt(rep xlsxedit.RecalcReport, audit xlsxedit.AuditReport, auditErr error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "重算完成：%d 个公式，%d 个错误", rep.TotalFormulas, rep.TotalErrors)
	if len(rep.ErrorSummary) > 0 {
		b.WriteString("（错误分布：")
		first := true
		for k, v := range rep.ErrorSummary {
			if !first {
				b.WriteString("，")
			}
			first = false
			fmt.Fprintf(&b, "%s=%v", k, v)
		}
		b.WriteString("）")
	}
	// 空串结果的公式（重算过、结果就是空）不进阻塞面：测算样表的未用槽位挂着
	// 取价公式，480 格空结果此前把交付门判成「未达」，回执又说「不算问题」
	// （2026-11-26 造价反馈三批，三场实测 414/278/114 条）。
	blocking, empties := 0, []xlsxedit.AuditFinding{}
	for _, f := range audit.Findings {
		if f.Kind == "empty_result" {
			empties = append(empties, f)
		} else {
			blocking++
		}
	}
	switch {
	case auditErr != nil:
		fmt.Fprintf(&b, "\n重算后的审计没跑成（%v）——确认前不要声称「结果已校验」，用 xlsx_audit 复核。", auditErr)
		if rep.TotalErrors > 0 {
			b.WriteString("\n有公式错误未达交付门：Err:522=循环引用，Err:508=括号/配对缺失，Err:507=除零；修复后再次重算。")
		}
	case rep.TotalErrors == 0 && len(audit.Findings) == 0:
		if audit.Truncated {
			fmt.Fprintf(&b, "\n重算报告零错误；表过大，重算后审计只扫描了前 %d 个单元格、未发现问题——全量复核用 xlsx_audit。", xlsxedit.AuditMaxCells)
		} else {
			b.WriteString("\n零错误，达到交付门（重算后审计：错误值/自引用/未重算均为零）。")
		}
	case rep.TotalErrors == 0 && blocking == 0:
		b.WriteString("\n零错误，达到交付门（重算后审计：错误值/自引用/未重算均为零）。")
		b.WriteString("\n" + emptyResultNote(empties))
	default:
		if n := countAuditKind(audit.Findings, "missing_cache"); n > 0 && n == len(audit.Findings) && rep.TotalErrors == 0 {
			// 没有错误值、没有自引用，只是公式没缓存值：多半是重算没生效（文件没被
			// LibreOffice 真正写过），也可能是公式结果本身就是空字符串——两种都不该
			// 按「公式写错了」去改（2026-11-26 宁丰反馈 #1）。
			fmt.Fprintf(&b, "\n未达交付门——%d 个公式重算后没有缓存值。先原样再调一次 xlsx_recalc（上一次重算可能没生效）；仍有则逐格核对，结果本应为空的公式不算问题。清单：\n", n)
		} else {
			b.WriteString("\n未达交付门——按下面审计结果修复后再次重算。重算后审计：\n")
		}
		b.WriteString(formatAuditFindings(audit, recalcAuditLimit))
	}
	return b.String()
}

// countAuditKind 数审计发现里某一类的条数。
func countAuditKind(findings []xlsxedit.AuditFinding, kind string) int {
	n := 0
	for _, f := range findings {
		if f.Kind == kind {
			n++
		}
	}
	return n
}

// recalcAuditLimit 是重算回执里审计发现的最多个数（全文清单用 xlsx_audit）。
const recalcAuditLimit = 20

var _ tool.Tool = xlsxRecalcTool{}
var _ tool.WritePathResolver = xlsxRecalcTool{}
