package app

// xlsx_audit — 灵犀 optimize plan P1-2 的只读质量门禁（错误值扫描+循环引用
// 最小面+未重算清单）。灵犀的纪律是「交付前公式错误必须为零」：xlsx_apply
// 应用后自动重算，本工具是交付前的复核面；xlsx_recalc 是它的修复面。
//
// 只读：不实现 WritePathResolver，读目标经 ReadTargeter 交 AccessPolicy 校验
// （路径解析同 xlsxProfileResolvePath：相对=ResolveRelWithin，绝对=原样）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

type xlsxAuditTool struct{ a *App }

func (t xlsxAuditTool) Name() string   { return "xlsx_audit" }
func (t xlsxAuditTool) ReadOnly() bool { return true }

func (t xlsxAuditTool) Description() string {
	return "对既有 .xlsx 做交付前质量审计（只读，不改文件）：扫描公式错误值（#REF!/#DIV/0!/#VALUE!/#N/A/#NAME? 等）、" +
		"自引用循环（A→B→A 深循环由 xlsx_recalc 的 LibreOffice 重算链报）、有公式但未重算的单元格。" +
		"交付纪律：公式错误必须为零——xlsx_apply 应用后先 xlsx_recalc 重算，再本工具复核，零错误才算交付；" +
		"生成公式表时公式优先，不要用脚本算完把数值硬编码进单元格。"
}

func (t xlsxAuditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path"],
  "properties":{
    "path":{"type":"string","description":"目标 .xlsx 工作区相对路径"},
    "limit":{"type":"integer","description":"最多展示多少条发现（缺省 40，超出部分给计数）"}
  }
}`)
}

func (t xlsxAuditTool) ReadTarget(args json.RawMessage) (string, bool) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", false
	}
	p, err := xlsxProfileResolvePath(in.Path)
	if err != nil {
		return "", false
	}
	return p, true
}

func (t xlsxAuditTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("xlsx_audit 参数解析失败: %w", err)
	}
	if strings.TrimSpace(in.Path) == "" {
		return "", fmt.Errorf("缺少参数 path（目标 .xlsx 工作区相对路径）")
	}
	path, err := xlsxProfileResolvePath(in.Path)
	if err != nil {
		return "", err
	}
	if !xlsxedit.AuditFileExists(path) {
		return "", fmt.Errorf("文件不存在：%s", in.Path)
	}
	rep, err := xlsxedit.Audit(path)
	if err != nil {
		return "", err
	}
	return tool.WrapText(formatAuditReport(rep, in.Limit)), nil
}

// formatAuditReport 把审计报告排版成给模型看的文本（纯函数，便于单测）。
// 错误值优先展示（交付门的硬伤），自引用次之，未重算折叠成计数+样例。
func formatAuditReport(rep xlsxedit.AuditReport, limit int) string {
	if limit <= 0 {
		limit = 40
	}
	var b strings.Builder
	if rep.Truncated {
		fmt.Fprintf(&b, "（表过大，已扫描前 %d 单元格截断）\n", xlsxedit.AuditMaxCells)
	}
	fmt.Fprintf(&b, "审计结果：%d 个工作表，%d 个公式", rep.Sheets, rep.TotalFormulas)

	errs, refs, caches := splitFindings(rep.Findings)
	if len(errs) == 0 && len(refs) == 0 && len(caches) == 0 {
		b.WriteString("，零错误零异常，达到交付门。\n")
		return b.String()
	}
	fmt.Fprintf(&b, "，发现 %d 条问题（错误值 %d / 自引用 %d / 未重算 %d）\n",
		len(errs)+len(refs)+len(caches), len(errs), len(refs), len(caches))

	shown := 0
	budget := limit
	for _, f := range errs {
		if budget <= 0 {
			break
		}
		fmt.Fprintf(&b, "- [错误值] %s!%s = %s（公式 %s）——必须修复后重算再交付\n",
			f.Sheet, f.Cell, f.Value, f.Formula)
		shown++
		budget--
	}
	for _, f := range refs {
		if budget <= 0 {
			break
		}
		fmt.Fprintf(&b, "- [自引用] %s!%s 公式引用了自身（%s）——循环引用，改成引用其它单元格或改用迭代设置\n",
			f.Sheet, f.Cell, f.Formula)
		shown++
		budget--
	}
	for _, f := range caches {
		if budget <= 0 {
			break
		}
		fmt.Fprintf(&b, "- [未重算] %s!%s（%s）——跑 xlsx_recalc 刷新缓存值\n",
			f.Sheet, f.Cell, f.Formula)
		shown++
		budget--
	}
	if total := len(errs) + len(refs) + len(caches); total > shown {
		fmt.Fprintf(&b, "…其余 %d 条未展示（limit=%d）；先修复错误值与自引用，再 xlsx_recalc 后复审。\n",
			total-shown, limit)
	} else if len(caches) > 0 {
		b.WriteString("下一步：xlsx_recalc 重算刷新缓存值，再跑本工具复核，零错误才算交付。\n")
	} else {
		b.WriteString("下一步：修复上述问题后 xlsx_recalc + 本工具复核，零错误才算交付。\n")
	}
	return b.String()
}

// splitFindings 按类分组，错误值按表内位置排序（稳定输出）。
func splitFindings(fs []xlsxedit.AuditFinding) (errs, refs, caches []xlsxedit.AuditFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Sheet != fs[j].Sheet {
			return fs[i].Sheet < fs[j].Sheet
		}
		return fs[i].Cell < fs[j].Cell
	})
	for _, f := range fs {
		switch f.Kind {
		case "error_value":
			errs = append(errs, f)
		case "self_ref":
			refs = append(refs, f)
		default:
			caches = append(caches, f)
		}
	}
	return errs, refs, caches
}

var _ tool.Tool = xlsxAuditTool{}
var _ tool.ReadTargeter = xlsxAuditTool{}
