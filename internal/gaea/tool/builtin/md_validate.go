// Package builtin provides Tianxuan's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(mdValidate{}) }

// mdValidate Markdown/Office 文档编号机检（2026-11-20 实测反馈①：9 个子代理
// 并行写各自编各自的号——表1-4 撞 3 处、表1-8B 撞 3 处、图1-11 跳号，工具侧
// 无提示，只能事后写脚本全量重排返工 2 轮）。三类确定性检查，全只读：
//
//   - numbering：题注编号（表X-Y/图X-Y，含 Table/Figure）唯一性+同章内连续性
//   - orphans：正文引用↔题注双向对账（引用了不存在的表/图；题注从未被引用）
//   - table_columns：Markdown 表格块内各行列数一致性
//
// docx 输入经 docmd 转 md 后同管线校验（回执行号=md 行号）。
type mdValidate struct{ workDir string }

func (mdValidate) Name() string { return "md_validate" }

func (mdValidate) Description() string {
	return "对 Markdown 文档（或 docx，自动转 md）做编号机检（只读，确定性，不经模型）：①题注编号唯一性+同章连续性（表1-4/图1-11 类，含 Table/Figure）②引用↔题注双向对账（孤立引用=正文引了不存在的表/图；未引用=题注从未被正文提及）③表格块内列数一致性。多代理并行编写或合并章节后必跑——撞号/跳号/漏引用当场暴露，不要等交付后返工。checks 可选默认全跑。paths 传多文件=全册并集对账（跨章引用在他章题注中闭合即通过，撞号跨文件可检出）；单文件模式下本地无题注的引用归类为 cross_ref（跨章引用·待全册核验）不算硬伤——先跑单章不误报，交付前用 paths 全册复核。"
}

func (mdValidate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"文档路径（相对工作区或绝对）：.md 或 .docx"},
  "paths":{"type":"array","items":{"type":"string"},"description":"多文件全册模式（覆盖 path）：各章 md/docx 一起对账，跨章引用闭合即通过、撞号跨文件检出"},
  "checks":{"type":"array","items":{"type":"string","enum":["numbering","orphans","table_columns"]},"description":"可选，默认全跑"}
},
"required":["path"]
}`)
}

func (mdValidate) ReadOnly() bool { return true }

// SpaceTag 自声明办公域（与静态表同口径；自声明优先进生效标签）。
func (mdValidate) SpaceTag() string { return spaces.SpaceWork }

func (v mdValidate) ReadTarget(args json.RawMessage) (string, bool) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(v.workDir, p.Path), true
}

func (mdValidate) CompactDescription() string {
	return compactDesc["md_validate"]
}
func (mdValidate) CompactSchema() json.RawMessage { return compactSchema["md_validate"] }

// ── 编号形态 ──────────────────────────────────────────────────
//
// 题注行：行首（容忍引用块 "> " 前缀）「表/图/Table/Figure + 编号」开头。
// 编号主干：数字段(-或–或—连接)+可选字母尾（表1-13A）；en/em-dash 归一为 -
//（2026-11-21 复盘 B6：中文排版长破折号「表1—3」此前只读到「表1」）。

var reMDVCaption = regexp.MustCompile(`^(?:>\s*)?(表|图|Table|Figure)\s*([0-9]+(?:[-–—][0-9]+)*[A-Za-z]?)\b`)
var reMDVRef = regexp.MustCompile(`(表|图|Table|Figure)\s*([0-9]+(?:[-–—][0-9]+)*[A-Za-z]?)`)

// mdvCompoundPrefixes 复合词前缀（复盘 B6：「附图8」被读作内部引用「图8」、
// 一次报 13 项假阳性——这些是附录/外部编号体系，不参与本文档题注对账，引用
// 扫描前先行中性化）。题注行按行首锚定不受影响（「附图8 …」不以 图/表 起头）。
var mdvCompoundPrefixes = []string{"附图", "附表", "插图", "图幅"}

func mdvNeutralizeCompounds(ln string) string {
	for _, w := range mdvCompoundPrefixes {
		ln = strings.ReplaceAll(ln, w, "▢▢")
	}
	return ln
}

// mdvNormNum 编号归一（–/— → -）。
func mdvNormNum(s string) string {
	s = strings.ReplaceAll(s, "–", "-")
	return strings.ReplaceAll(s, "—", "-")
}

// mdvCaption 一个题注记录。
type mdvCaption struct {
	kind string // 归一化：表/Table→表；图/Figure→图
	num  string // 归一编号（en-dash→-）
	line int    // 1-based
	file string // 所属文件基名（多文件模式定位跨章题注）
}

// mdvRefLoc 一处引用的位置。
type mdvRefLoc struct {
	file string
	line int
}

// mdvKey 归一化主键（kind+num），撞号判定用。
func (c mdvCaption) key() string { return c.kind + c.num }

// mdvMain 编号主干数字段（"1-13A"→[1,13]，尾缀丢弃）。
func (c mdvCaption) main() []int {
	parts := strings.Split(mdvNormNum(c.num), "-")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		d := strings.TrimRight(p, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
		n, err := strconv.Atoi(d)
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}

// mdvNormalizeKind 中英归一。
func mdvNormalizeKind(k string) string {
	switch k {
	case "Table":
		return "表"
	case "Figure":
		return "图"
	}
	return k
}

// mdvIssue 一条检出项。
type mdvIssue struct {
	kind    string // numbering / orphans / cross_ref / table_columns
	file    string // 多文件模式的归属（单文件可空）
	line    int
	summary string
}

// Execute 跑所选检查，聚合回执。paths 多文件=全册并集对账（2026-11-21 反馈
// P1-4：单章模式把合法跨章引用误报「孤立引用」9 条，诱发把有效引用改成模糊
// 描述——单文件时本地无题注的引用改判 cross_ref 待核类，多文件时他章闭合即过）。
func (v mdValidate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path   string   `json:"path"`
		Paths  []string `json:"paths,omitempty"`
		Checks []string `json:"checks,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("md_validate 参数解析失败: %w", err)
	}
	if strings.TrimSpace(p.Path) == "" && len(p.Paths) == 0 {
		return "", fmt.Errorf("path is required")
	}
	// 多文件模式（paths 覆盖 path）；去重保序。
	files := p.Paths
	if len(files) == 0 {
		files = []string{p.Path}
	}
	seen := map[string]bool{}
	var uniq []string
	for _, f := range files {
		if strings.TrimSpace(f) == "" || seen[f] {
			continue
		}
		seen[f] = true
		uniq = append(uniq, f)
	}
	multi := len(uniq) > 1

	type mdvLoaded struct {
		name  string
		lines []string
	}
	var loaded []mdvLoaded
	for _, f := range uniq {
		src := resolveIn(v.workDir, f)
		ext := strings.ToLower(filepath.Ext(src))
		var text string
		switch ext {
		case ".md", ".markdown", ".txt":
			raw, err := os.ReadFile(src)
			if err != nil {
				return "", fmt.Errorf("读取 %s 失败: %w%s", src, err, posixStylePathHint(f))
			}
			text = string(raw)
		case ".docx":
			md, _, _, err := docmd.ConvertLimit(src, "", 0)
			if err != nil {
				return "", fmt.Errorf("docx 转 md 失败: %w", err)
			}
			text = md
		default:
			return "", fmt.Errorf("暂支持 md/txt/docx（当前 %s）", ext)
		}
		loaded = append(loaded, mdvLoaded{name: filepath.Base(src), lines: strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")})
	}

	run := map[string]bool{}
	for _, c := range p.Checks {
		run[c] = true
	}
	if len(run) == 0 {
		run["numbering"], run["orphans"], run["table_columns"] = true, true, true
	}

	var issues []mdvIssue
	var nTable, nFigure int
	var bNote strings.Builder // 对账汇总注记（跨章闭合数等，非 issue）

	if run["numbering"] || run["orphans"] {
		var captions []mdvCaption
		refs := map[string][]mdvRefLoc{} // 引用 key → 位置们
		for _, lf := range loaded {
			for i, ln := range lf.lines {
				trimmed := strings.TrimSpace(ln)
				if m := reMDVCaption.FindStringSubmatch(trimmed); m != nil {
					captions = append(captions, mdvCaption{
						kind: mdvNormalizeKind(m[1]),
						num:  mdvNormNum(m[2]),
						line: i + 1,
						file: lf.name,
					})
					continue
				}
				for _, m := range reMDVRef.FindAllStringSubmatch(mdvNeutralizeCompounds(ln), -1) {
					k := mdvNormalizeKind(m[1]) + mdvNormNum(m[2])
					refs[k] = append(refs[k], mdvRefLoc{file: lf.name, line: i + 1})
				}
			}
		}
		for _, c := range captions {
			if c.kind == "表" {
				nTable++
			} else {
				nFigure++
			}
		}
		if run["numbering"] {
			issues = append(issues, mdvCheckDuplicates(captions)...)
			issues = append(issues, mdvCheckContinuity(captions)...)
		}
		if run["orphans"] {
			orphanIssues, crossClosed := mdvCheckOrphans(captions, refs, multi)
			issues = append(issues, orphanIssues...)
			if multi && crossClosed > 0 {
				fmt.Fprintf(&bNote, "跨章引用 %d 处已在全册并集闭合（不误报）。\n", crossClosed)
			}
		}
	}
	if run["table_columns"] {
		for _, lf := range loaded {
			for _, is := range mdvCheckTables(lf.lines) {
				is.file = lf.name
				issues = append(issues, is)
			}
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].file != issues[j].file {
			return issues[i].file < issues[j].file
		}
		if issues[i].line != issues[j].line {
			return issues[i].line < issues[j].line
		}
		return issues[i].kind < issues[j].kind
	})

	scope := loaded[0].name
	if multi {
		scope = fmt.Sprintf("%d 文件并集（%s 等）", len(loaded), loaded[0].name)
	}
	var b strings.Builder
	if len(issues) == 0 {
		fmt.Fprintf(&b, "OK: %s — 编号唯一且连续；引用↔题注对账通过；表格列数一致", scope)
		if multi {
			b.WriteString("（撞号跨文件可检出）")
		} else if !run["numbering"] || !run["orphans"] {
			b.WriteString("（本次为所选 checks 的子集）")
		}
		if s := strings.TrimRight(bNote.String(), "\n"); s != "" {
			b.WriteString("\n" + s)
		}
		return b.String(), nil
	}
	fmt.Fprintf(&b, "md_validate: %s — %d issue(s)\n", scope, len(issues))
	for _, is := range issues {
		if multi && is.file != "" {
			fmt.Fprintf(&b, "[%s] %s L%d %s\n", is.kind, is.file, is.line, is.summary)
		} else {
			fmt.Fprintf(&b, "[%s] L%d %s\n", is.kind, is.line, is.summary)
		}
	}
	if s := strings.TrimRight(bNote.String(), "\n"); s != "" {
		b.WriteString(s + "\n")
	}
	b.WriteString("修复后重跑本工具确认清零；撞号须全局重排（勿只改当前行——其他子代理可能引用了旧号）。")
	return b.String(), nil
}

// mdvCheckDuplicates 撞号：同 key 多个题注（多文件模式下跨文件撞号直接点名两侧）。
func mdvCheckDuplicates(captions []mdvCaption) []mdvIssue {
	seen := map[string]mdvCaption{}
	var out []mdvIssue
	for _, c := range captions {
		if first, dup := seen[c.key()]; dup {
			where := fmt.Sprintf("L%d", first.line)
			if first.file != c.file {
				where = fmt.Sprintf("%s L%d", first.file, first.line)
			}
			out = append(out, mdvIssue{kind: "numbering", file: c.file, line: c.line,
				summary: fmt.Sprintf("%s 撞号：与 %s 重复（多代理并行编号未对账的典型形态）", c.key(), where)})
			continue
		}
		seen[c.key()] = c
	}
	return out
}

// mdvCheckContinuity 跳号：同 kind+同章（首段相同）内序号排序后相邻差>1。
// 仅对两段以上主干（X-Y 形态）判；单段编号（表3）形态太自由不判。
func mdvCheckContinuity(captions []mdvCaption) []mdvIssue {
	groups := map[string][]mdvCaption{} // kind+章 → 题注
	for _, c := range captions {
		main := c.main()
		if len(main) < 2 {
			continue
		}
		key := fmt.Sprintf("%s-%d", c.kind, main[0])
		groups[key] = append(groups[key], c)
	}
	var out []mdvIssue
	for _, g := range groups {
		sorted := append([]mdvCaption(nil), g...)
		sort.Slice(sorted, func(i, j int) bool {
			mi, mj := sorted[i].main(), sorted[j].main()
			for k := 0; k < len(mi) && k < len(mj); k++ {
				if mi[k] != mj[k] {
					return mi[k] < mj[k]
				}
			}
			return sorted[i].num < sorted[j].num
		})
		for i := 1; i < len(sorted); i++ {
			prev, cur := sorted[i-1].main(), sorted[i].main()
			if len(prev) >= 2 && len(cur) >= 2 && cur[1]-prev[1] > 1 {
				out = append(out, mdvIssue{kind: "numbering", file: sorted[i].file, line: sorted[i].line,
					summary: fmt.Sprintf("%s%d 章内跳号：序号 %d(L%d) 之后直接 %d(L%d)，缺 %s%d-%d..%s%d-%d",
						sorted[i].kind, cur[0],
						prev[1], sorted[i-1].line, cur[1], sorted[i].line,
						sorted[i].kind, cur[0], prev[1]+1, sorted[i].kind, cur[0], cur[1]-1)})
			}
		}
	}
	return out
}

// mdvCheckOrphans 引用↔题注双向对账。multi=全册并集模式：他章闭合的跨章引用
// 通过（计一条汇总），并集仍无=真孤立（error 面）；单文件模式：本地无题注的
// 引用归类 cross_ref（跨章引用·待全册核验）不算硬伤——2026-11-21 反馈 P1-4：
// 单章模式曾把合法跨章引用混进 orphans 报 9 条误报，诱发把有效引用改成模糊描述。
func mdvCheckOrphans(captions []mdvCaption, refs map[string][]mdvRefLoc, multi bool) ([]mdvIssue, int) {
	var out []mdvIssue
	capWhere := map[string]string{} // key → 题注所在文件
	for _, c := range captions {
		capWhere[c.key()] = c.file
	}
	crossClosed := 0
	keys := make([]string, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		where, ok := capWhere[k]
		if !ok {
			locs := refs[k]
			first := locs[0]
			if multi {
				// 全册并集仍无此题注：真失锚（引用错号或题注被改号）。
				out = append(out, mdvIssue{kind: "orphans", file: first.file, line: first.line,
					summary: fmt.Sprintf("孤立引用（全册并集）：引用 %s（L%d 等 %d 处）但全册无此题注", k, first.line, len(locs))})
			} else {
				// 单文件模式：可能是跨章引用（合法）也可能是失锚笔误——
				// 不冒充硬伤，归类待全册核验。
				lines := make([]int, 0, len(locs))
				for _, l := range locs {
					lines = append(lines, l.line)
				}
				out = append(out, mdvIssue{kind: "cross_ref", line: first.line,
					summary: fmt.Sprintf("跨章引用（待全册核验）：本地无题注 %s（L%v）——若为跨章/附录编号属正常，勿改成文字描述；用 paths 多文件模式闭合核验，失锚则修号", k, lines)})
			}
			continue
		}
		if multi {
			// 他章闭合的跨章引用：通过，不进 issue（汇总一条在回执尾）。
			for _, l := range refs[k] {
				if l.file != where {
					crossClosed++
				}
			}
		}
	}
	// 未引用：题注存在但正文零引用（warning 面：疑似漏挂引用或孤儿图表）。
	for _, c := range captions {
		if _, ok := refs[c.key()]; !ok {
			out = append(out, mdvIssue{kind: "orphans", file: c.file, line: c.line,
				summary: fmt.Sprintf("未引用：题注 %s 存在但正文从未引用（孤立图表或漏挂引用）", c.key())})
		}
	}
	return out, crossClosed
}

// mdvCheckTables Markdown 表格块内列数一致性。表格块=连续含 | 的行；块内
// 每行按 | 分隔计数（首尾空段剔除），分隔行（全 -/:/空格）跳过。
func mdvCheckTables(lines []string) []mdvIssue {
	var out []mdvIssue
	i := 0
	for i < len(lines) {
		if !strings.Contains(lines[i], "|") {
			i++
			continue
		}
		j := i
		for j < len(lines) && strings.Contains(lines[j], "|") {
			j++
		}
		if j-i >= 2 { // 单行竖线不算表格
			var cols int
			var firstLine int
			for k := i; k < j; k++ {
				if isMDVSeparatorRow(lines[k]) {
					continue
				}
				n := mdvColumnCount(lines[k])
				if cols == 0 {
					cols, firstLine = n, k+1
					continue
				}
				if n != cols {
					out = append(out, mdvIssue{kind: "table_columns", line: k + 1,
						summary: fmt.Sprintf("表格块内列数不一致：L%d=%d 列，L%d=%d 列（同表各行必须同列数）", firstLine, cols, k+1, n)})
				}
			}
		}
		i = j
	}
	return out
}

// posixStylePathHint POSIX 形态路径的报错提示（2026-11-21 复盘 B5：bash 的
// /d/... 写法在非 bash 工具里被当相对路径拼错，报错不给正确写法导致反复试错）。
// md_validate 与 doc_render 共用（同包）。
func posixStylePathHint(p string) string {
	if strings.HasPrefix(strings.TrimSpace(p), "/") {
		return "（提示：/d/... 是 bash 的 POSIX 写法，本工具不认——传 Windows 盘符绝对路径 D:\\...\\文件.md（正斜杠 D:/ 亦可）或工作区相对路径）"
	}
	return ""
}

func isMDVSeparatorRow(ln string) bool {
	s := strings.ReplaceAll(strings.ReplaceAll(ln, "|", ""), " ", "")
	if s == "" {
		return true
	}
	for _, ch := range s {
		if ch != '-' && ch != ':' {
			return false
		}
	}
	return true
}

func mdvColumnCount(ln string) int {
	trimmed := strings.Trim(strings.TrimSpace(ln), "|")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "|") + 1
}
