package app

// xlsx_audit — 灵犀 optimize plan P1-2 的只读质量门禁（错误值扫描+循环引用
// 最小面+未重算清单）。灵犀的纪律是「交付前公式错误必须为零」：xlsx_apply
// 应用后自动重算，本工具是交付前的复核面；xlsx_recalc 是它的修复面。
//
// 只读：不实现 WritePathResolver，读目标经 ReadTargeter 交 AccessPolicy 校验
// （路径解析同 xlsxProfileResolvePath：相对=ResolveRelWithin，绝对=原样）。

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

type xlsxAuditTool struct{ a *App }

func (t xlsxAuditTool) Name() string   { return "xlsx_audit" }
func (t xlsxAuditTool) ReadOnly() bool { return true }

func (t xlsxAuditTool) Description() string {
	return "对既有 .xlsx 做交付前质量审计（只读，不改文件）：扫描公式错误值（#REF!/#DIV/0!/#VALUE!/#N/A/#NAME? 等）、" +
		"自引用循环（A→B→A 深循环由 xlsx_recalc 的 LibreOffice 重算链报）、有公式但未重算的单元格，外加打印版面的确定性检查。" +
		"回执第一行是健康体检：打开性（zip 与 XML 是否完好，坏了直接报打不开）、公式格里有多少没有缓存值、定义名（含表级归属）/ 数据验证 / 合并格等结构面。" +
		"交付纪律：公式错误必须为零——xlsx_recalc 的回执已附同款审计，重算后零错误即达交付门，不必紧接着重复调用；" +
		"重算后又改过文件、或要复核不是自己重算的文件时用本工具。" +
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
	// 健康体检（2026-11-27 AI 反馈 #15③）：「打不开」类问题此前排障全靠手撕 zip/XML。
	// 先做打开性探测（zip 结构 + 工作簿/工作表 XML），坏了直接报，不去扫单元格。
	health, herr := xlsxedit.Inspect(path)
	if herr != nil {
		return "", fmt.Errorf("打不开：%v——Excel/WPS 多半也打不开；先核实文件是否写完整（是否被别的程序占着、脚本是否中途退出），不要去扫单元格", herr)
	}
	healthLine := "健康体检：可打开｜" + health.CacheLine() + "｜" + health.StructureLine()
	if health.NoCachedValue > 0 {
		healthLine += "——没重算，交付前 xlsx_recalc"
	}
	rep, err := xlsxedit.Audit(path)
	if err != nil {
		return "", err
	}
	head := xlsxFileFingerprint(path)
	layout, lerr := xlsxLayoutNotes(path)
	if lerr != nil {
		layout = nil // 版面检查失败不拦审计（诚实缺省，无版面段）
	}
	return tool.WrapText(head + healthLine + "\n" + formatAuditReport(rep, in.Limit) + strings.Join(layout, "")), nil
}

// xlsxFileFingerprint 文件指纹（2026-11-21 造价门实测 5.1：同轮改文件后复检，
// 审计文本可能逐字节相同而被上层「结果同一性去重」折叠成 cached——指纹含
// size/mtime/sha16，文件一变指纹必变，新旧结果可分辨）。
func xlsxFileFingerprint(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("文件指纹：size=%d mtime=%s（sha 未读）\n", fi.Size(), fi.ModTime().Format("01-02 15:04:05"))
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("文件指纹：size=%d mtime=%s sha16=%x\n", fi.Size(), fi.ModTime().Format("01-02 15:04:05"), sum[:8])
}

// xlsxLayoutNotes 打印版面确定性检查（2026-11-21 造价门实测 5.2：doc_render 出
// 图后问视觉模型「是否被截断/列宽是否合理」，整册纵向 A4 压成单列它照样秒回
// 「版式清晰规范」——版面问题用确定性判据：读 page_setup 与已用列数，长表纵向
// 且未按宽适应=必压扁；vision 只作参考）。
func xlsxLayoutNotes(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []string
	out = append(out, "版面检查（确定性判据，非视觉判断）：\n")
	warned := 0
	for _, sheet := range f.GetSheetList() {
		ps, perr := f.GetPageLayout(sheet)
		if perr != nil {
			continue
		}
		cols, cerr := f.GetCols(sheet)
		if cerr != nil {
			continue
		}
		used := 0
		for _, col := range cols {
			if len(col) > 0 {
				used++
			}
		}
		orientation := ""
		if ps.Orientation != nil {
			orientation = *ps.Orientation
		}
		display := orientation
		if display == "" {
			display = "portrait(默认)"
		}
		fit := 0
		if ps.FitToWidth != nil {
			fit = *ps.FitToWidth
		}
		note := ""
		if used >= 10 && orientation != "landscape" && fit < 1 {
			note = " ⚠ 长表纵向且未按宽适应——打印/导出会被压成单列或截断，建议横向+FitToWidth=1"
			warned++
		}
		out = append(out, fmt.Sprintf("  - %s：%d 列，%s，FitToWidth=%d%s\n", sheet, used, display, fit, note))
	}
	if warned > 0 {
		out = append(out, fmt.Sprintf("  版面告警 %d 张表（⚠ 项修打印设置后重查；视觉模型看不出整体压扁，以此为准）\n", warned))
	} else {
		out = append(out, "  版面无告警（长表均横向或已按宽适应）。\n")
	}
	return out, nil
}

// formatAuditReport 把审计报告排版成给模型看的文本（纯函数，便于单测）。
// 错误值优先展示（交付门的硬伤），自引用次之，未重算折叠成计数+样例。
// xlsx_audit 独立复核口径（尾注指向「重算后复审」）。
func formatAuditReport(rep xlsxedit.AuditReport, limit int) string {
	return formatAuditWith(rep, limit, auditTailStandalone)
}

// formatAuditFindings 同一份主体的重算回执口径（xlsx_recalc 2026-11-25 B4：
// 回执直接附审计结果）——尾注指向「修复后再次重算 / 全量清单用 xlsx_audit」。
func formatAuditFindings(rep xlsxedit.AuditReport, limit int) string {
	return formatAuditWith(rep, limit, auditTailRecalc)
}

// auditTail 是排版尾注的两种口径：主体（计数 + 逐条发现）同一份，只换结尾。
type auditTail struct {
	clean  string                               // 零发现时接在计数行后
	rest   func(total, shown, limit int) string // 有未展示条目
	cached string                               // 全部展示完、仍有未重算
	fixed  string                               // 全部展示完、无未重算
}

var auditTailStandalone = auditTail{
	clean: "，零错误零异常，达到交付门。\n",
	rest: func(total, shown, limit int) string {
		return fmt.Sprintf("…其余 %d 条未展示（limit=%d）；先修复错误值与自引用，再 xlsx_recalc 后复审。\n", total-shown, limit)
	},
	cached: "下一步：xlsx_recalc 重算刷新缓存值，再跑本工具复核，零错误才算交付。\n",
	fixed:  "下一步：修复上述问题后 xlsx_recalc + 本工具复核，零错误才算交付。\n",
}

var auditTailRecalc = auditTail{
	clean: "，零错误零异常。\n",
	rest: func(total, shown, limit int) string {
		return fmt.Sprintf("…其余 %d 条未展示（limit=%d）；全部清单用 xlsx_audit。\n", total-shown, limit)
	},
	cached: "下一步：重算后仍有未重算的单元格——确认本机 LibreOffice 重算链可用后再次重算；仍不刷新就用 xlsx_audit 看全量清单。\n",
	fixed:  "下一步：修复上述问题后再次重算。Err:522=循环引用，Err:508=括号/配对缺失，Err:507=除零。\n",
}

func formatAuditWith(rep xlsxedit.AuditReport, limit int, tail auditTail) string {
	if limit <= 0 {
		limit = 40
	}
	var b strings.Builder
	if rep.Truncated {
		fmt.Fprintf(&b, "（表过大，已扫描前 %d 单元格截断）\n", xlsxedit.AuditMaxCells)
	}
	fmt.Fprintf(&b, "审计结果：%d 个工作表，%d 个公式", rep.Sheets, rep.TotalFormulas)

	errs, refs, caches, empties := splitFindings(rep.Findings)
	if len(errs) == 0 && len(refs) == 0 && len(caches) == 0 {
		// 没有阻塞项；空串结果的公式不算问题（2026-11-26 造价反馈三批）。
		b.WriteString(tail.clean)
		b.WriteString(emptyResultNote(empties))
		return b.String()
	}
	fmt.Fprintf(&b, "，发现 %d 条问题（错误值 %d / 自引用 %d / 未重算 %d）",
		len(errs)+len(refs)+len(caches), len(errs), len(refs), len(caches))
	if len(empties) > 0 {
		fmt.Fprintf(&b, "，另有 %d 个空结果公式（不阻塞）", len(empties))
	}
	b.WriteString("\n")

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
	// 未重算先给聚类（按表+列、按公式模式），再给清单：只给数量和前 N 条时，
	// 「114 条未重算」要逐格核对才知道全是同一列的同一类公式（旺平反馈 #2）。
	if len(caches) > 3 {
		fmt.Fprintf(&b, "未重算分布：%s\n", clusterAuditFindings(caches))
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
	b.WriteString(emptyResultNote(empties))
	if total := len(errs) + len(refs) + len(caches); total > shown {
		b.WriteString(tail.rest(total, shown, limit))
	} else if len(caches) > 0 {
		b.WriteString(tail.cached)
	} else {
		b.WriteString(tail.fixed)
	}
	return b.String()
}

// emptyResultNote 「重算过、结果就是空串」的公式摘要：不阻塞交付门，按位置与
// 公式模式聚类，并给出清法。
func emptyResultNote(empties []xlsxedit.AuditFinding) string {
	if len(empties) == 0 {
		return ""
	}
	return fmt.Sprintf("另有 %d 个公式的结果是空串（重算过、结果就是空，不算问题，不阻塞交付）：%s——多半是未用槽位上挂着的取价公式；交付前想清掉用 xlsx_apply 的 clear_where（按条件列清空区域内的公式，条件列填了的行不碰）。\n",
		len(empties), clusterAuditFindings(empties))
}

// clusterAuditFindings 把一类发现折成一行：按「表+列」与按公式模式（行号抹成 #）
// 各取前几类计数。
func clusterAuditFindings(fs []xlsxedit.AuditFinding) string {
	if len(fs) == 0 {
		return ""
	}
	cols := map[string]int{}
	pats := map[string]int{}
	for _, f := range fs {
		col := f.Cell
		if i := strings.IndexAny(f.Cell, "0123456789"); i > 0 {
			col = f.Cell[:i]
		}
		cols[f.Sheet+"!"+col]++
		pats[formulaPattern(f.Formula)]++
	}
	return "位置 " + topCounts(cols, 4) + "；公式 " + topCounts(pats, 3)
}

// topCounts 按次数降序（同次数按名称）取前 n 类，写成「名×次数」。
func topCounts(m map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v != list[j].v {
			return list[i].v > list[j].v
		}
		return list[i].k < list[j].k
	})
	var b strings.Builder
	for i, e := range list {
		if i >= n {
			fmt.Fprintf(&b, "…（共 %d 类）", len(list))
			break
		}
		if i > 0 {
			b.WriteString("、")
		}
		fmt.Fprintf(&b, "%s×%d", e.k, e.v)
	}
	return b.String()
}

// auditRefRowRe 单元格引用里的行号部分（聚类时抹成 #）。
var auditRefRowRe = regexp.MustCompile(`(\$?[A-Za-z]{1,3})\$?[0-9]{1,7}`)

// formulaPattern 公式模板：行号抹成 #，截到 60 字。
func formulaPattern(f string) string {
	p := auditRefRowRe.ReplaceAllString(strings.ReplaceAll(f, "$", ""), "${1}#")
	if r := []rune(p); len(r) > 60 {
		p = string(r[:60]) + "…"
	}
	return "`" + p + "`"
}

// splitFindings 按类分组，错误值按表内位置排序（稳定输出）。empties 是「重算过、
// 结果就是空串」的公式格——不进交付门的阻塞面。
func splitFindings(fs []xlsxedit.AuditFinding) (errs, refs, caches, empties []xlsxedit.AuditFinding) {
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
		case "empty_result":
			empties = append(empties, f)
		default:
			caches = append(caches, f)
		}
	}
	return errs, refs, caches, empties
}

var _ tool.Tool = xlsxAuditTool{}
var _ tool.ReadTargeter = xlsxAuditTool{}
