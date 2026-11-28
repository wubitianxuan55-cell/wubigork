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
//   - content：内容保真对账（沿江包1 复盘刀4）——以 source_pdf 原件文字层为基准
//     做 12 字滑窗 n-gram 命中率对账（表格 <98%、段落缺失 >5% 列硬伤，附缺失片段）
//
// docx 输入经 docmd 转 md 后同管线校验（回执行号=md 行号）。
type mdValidate struct{ workDir string }

func (mdValidate) Name() string { return "md_validate" }

func (mdValidate) Description() string {
	return "对 Markdown 文档（或 docx，自动转 md）做编号机检（只读，确定性，不经模型）：①题注编号唯一性+同章连续性（表1-4/图1-11 类，含 Table/Figure）②引用↔题注双向对账（孤立引用=正文引了不存在的表/图；未引用=题注从未被正文提及）③表格块内列数一致性。多代理并行编写或合并章节后必跑——撞号/跳号/漏引用当场暴露，不要等交付后返工。checks 可选默认全跑。paths 传多文件=全册并集对账（跨章引用在他章题注中闭合即通过，撞号跨文件可检出）；单文件模式下本地无题注的引用归类为 cross_ref（跨章引用·待全册核验）不算硬伤——先跑单章不误报，交付前用 paths 全册复核。回执按硬伤/提示分级（撞号/跳号/孤立引用/列数不一致=硬伤，未引用题注=提示，hints=false 折叠提示）；传 baseline=旧版文档路径只报新增缺陷，直接回答「问题是原有的还是本次引入的」；传 source_pdf 跑内容保真对账（12 字滑窗 n-gram，表格命中率<98%/段落缺失>5% 列硬伤附缺失片段——错字丢字单看任一版本发现不了，录入派生 md 后必跑）。"
}

func (mdValidate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"文档路径（相对工作区或绝对）：.md 或 .docx；与 paths 至少给一个"},
  "paths":{"type":"array","items":{"type":"string"},"description":"多文件全册模式（可单独给）：各章 md/docx 一起对账，跨章引用闭合即通过、撞号跨文件检出"},
  "checks":{"type":"array","items":{"type":"string","enum":["numbering","orphans","table_columns","content"]},"description":"可选，默认全跑（content 需 source_pdf，没给则跳过）"},
  "source":{"type":"string","description":"基准件路径（content 内容保真对账必填）：PDF 直读文字层；文字层乱码/扫描的原件先备一份基准 md/txt 再传。对派生 md 做逐表/逐段 12 字滑窗 n-gram 命中率对账，表格 <98%、段落缺失 >5% 列硬伤并附缺失片段——错字/丢字单看任一版本发现不了"},
  "hints":{"type":"boolean","description":"false=只报硬伤（撞号/跳号/孤立引用/列数/内容对账）；提示类（未引用题注/跨章待核）只报总数。默认 true 全报"},
  "baseline":{"type":"string","description":"基线文档路径（旧版 md/docx）：给出后按内容主键做差集，只报相对基线新增的缺陷——直接回答「这批问题是原有的还是本次引入的」"}
},
"required":[]
}`)
}

func (mdValidate) ReadOnly() bool { return true }

// SpaceTag 自声明办公域（与静态表同口径；自声明优先进生效标签）。
func (mdValidate) SpaceTag() string { return spaces.SpaceWork }

// mdvReadTargets 解析本次调用要读的全部本地路径：path、paths、source、baseline
// （相对路径按工作区）。paths 可单独给——此前 ReadTarget 只认 path，只传 paths
// 时在路径权限闸就被判「cannot resolve read target」（2026-11-28 黄甲反馈：主线程
// 实测被拦 4 次，而参数说明写的是「paths 覆盖 path」）。
func (v mdValidate) mdvReadTargets(args json.RawMessage) ([]string, error) {
	var p struct {
		Path     string   `json:"path"`
		Paths    []string `json:"paths"`
		Source   string   `json:"source"`
		Baseline string   `json:"baseline"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	var out []string
	for _, s := range append(append([]string(nil), p.Path, p.Source, p.Baseline), p.Paths...) {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, resolveIn(v.workDir, s))
		}
	}
	return out, nil
}

func (v mdValidate) ReadPaths(_ context.Context, args json.RawMessage) ([]string, error) {
	return v.mdvReadTargets(args)
}

func (v mdValidate) ReadTarget(args json.RawMessage) (string, bool) {
	paths, err := v.mdvReadTargets(args)
	if err != nil || len(paths) == 0 {
		return "", false
	}
	return paths[0], true
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
// 2026-11-28 黄甲反馈补两批：正文复合词（「代表 12 家单位」「施工总平面布置图 2 张」
// 这类被读成图表引用）与标准条文里的表图号（见 reMDVStdRef）。
var mdvCompoundPrefixes = []string{
	"附图", "附表", "插图", "图幅",
	"示意图", "流程图", "布置图", "路线图", "剖面图", "立面图", "断面图", "平面图",
	"管网图", "透视图", "效果图", "鸟瞰图", "框图", "意图", "企图", "宏图", "蓝图",
	"地图", "视图", "制图", "绘图", "试图", "版图",
	"代表", "列表", "报表", "表达", "表明", "表现", "表示", "表述", "表征", "表面",
	"表率", "表白", "表单", "填表", "发表", "秒表", "手表", "钟表", "电表",
}

// reMDVStdRef 标准条文里的表/图号前缀：「GB 36600-2018 表 1」「HJ 25.1 表 2」
// 「DB51/2362 表 4.2」引的是外部标准的表图，不参与本文档题注对账（实测两条被判
// 孤立引用硬伤）。前缀须含数字，「附录A 表 3」这类不受影响。
var reMDVStdRef = regexp.MustCompile(`[A-Z]{1,8}[0-9]*(?:/[A-Z0-9]{1,8})?(?:[. ]?[0-9]+)*(?:[-–—][0-9]+)*\s*(表|图)`)

func mdvNeutralizeCompounds(ln string) string {
	for _, w := range mdvCompoundPrefixes {
		ln = strings.ReplaceAll(ln, w, "▢▢")
	}
	ln = reMDVStdRef.ReplaceAllStringFunc(ln, func(m string) string {
		r := []rune(m)
		if !strings.ContainsAny(string(r[:len(r)-1]), "0123456789") {
			return m
		}
		return string(r[:len(r)-1]) + "▢"
	})
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
	kind     string // numbering / orphans / cross_ref / table_columns
	file     string // 多文件模式的归属（单文件可空）
	line     int
	summary  string
	severity string // "hard" 硬伤（撞号/跳号/孤立引用/列数不一致）｜"hint" 提示（未引用题注/跨章待核）——沿江包1 复盘 T2 分级
	key      string // 内容主键（版本无关）：基线对比按它去重，行号漂移不影响
}

const (
	mdvHard = "hard"
	mdvHint = "hint"
)

// Execute 跑所选检查，聚合回执。paths 多文件=全册并集对账（2026-11-21 反馈
// P1-4：单章模式把合法跨章引用误报「孤立引用」9 条，诱发把有效引用改成模糊
// 描述——单文件时本地无题注的引用改判 cross_ref 待核类，多文件时他章闭合即过）。
// mdvLoaded 一份已载入文本（md 行数组）。
type mdvLoaded struct {
	name  string
	lines []string
}

// mdvLoadText 读一个 md/txt/docx 为文本（docx 经 docmd 转 md）。
func mdvLoadText(src string) (string, error) {
	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".md", ".markdown", ".txt":
		raw, err := os.ReadFile(src)
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", src, err)
		}
		return string(raw), nil
	case ".docx":
		md, _, _, err := docmd.ConvertLimit(src, "", 0)
		if err != nil {
			return "", fmt.Errorf("docx 转 md 失败: %w", err)
		}
		return md, nil
	case ".pdf":
		text, _, err := docmd.PdfTextLayer(src, 0)
		if err != nil {
			return "", fmt.Errorf("读 PDF 文字层失败（乱码/扫描件可先备一份基准 md/txt 作 source）: %w", err)
		}
		return text, nil
	default:
		return "", fmt.Errorf("暂支持 md/txt/docx（当前 %s）", ext)
	}
}

// mdvAnalyze 跑所选检查（题注编号/引用对账/表格列数/内容保真），返回检出项与
// 对账注记。contentGrams 非 nil 且 run[content] 时对每个派生文件做原件对账。
func mdvAnalyze(loaded []mdvLoaded, run map[string]bool, multi bool, contentGrams map[string]bool) ([]mdvIssue, strings.Builder) {
	var issues []mdvIssue
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
	if run["content"] && len(contentGrams) > 0 {
		for _, lf := range loaded {
			issues = append(issues, mdvCheckContent(lf.name, lf.lines, contentGrams)...)
		}
	}
	return issues, bNote
}

func (v mdValidate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path     string   `json:"path"`
		Paths    []string `json:"paths,omitempty"`
		Checks   []string `json:"checks,omitempty"`
		Source   string   `json:"source,omitempty"`
		Hints    *bool    `json:"hints,omitempty"`
		Baseline string   `json:"baseline,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("md_validate 参数解析失败: %w", err)
	}
	if strings.TrimSpace(p.Path) == "" && len(p.Paths) == 0 {
		return "", fmt.Errorf("path 与 paths 至少给一个")
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
	// 内容保真对账（刀4）：要原件文字层。默认全跑时没给 source_pdf 就跳过 content
	//（显式点名 content 却没给原件=参数错，当场报）。
	contentGrams := map[string]bool{}
	if run["content"] {
		if strings.TrimSpace(p.Source) == "" {
			if len(p.Checks) > 0 {
				return "", fmt.Errorf("content 检查需要 source（基准件：PDF 原件或基准 md/txt）——没有基准就别点名 content")
			}
			delete(run, "content")
		} else {
			src := resolveIn(v.workDir, p.Source)
			baseText, err := mdvLoadText(src)
			if err != nil {
				return "", err
			}
			contentGrams = mdvGramSet(baseText)
		}
	}

	issues, bNote := mdvAnalyze(loaded, run, multi, contentGrams)

	// 基线对比（沿江包1 复盘 T2：直接回答「这批问题是原有的还是转换引入的」）：
	// 基线文档跑同一套检查，按内容主键（非行号）做差集，只列新增。
	hasBaseline := strings.TrimSpace(p.Baseline) != ""
	baseKeys := map[string]bool{}
	if strings.TrimSpace(p.Baseline) != "" {
		baseSrc := resolveIn(v.workDir, p.Baseline)
		baseText, err := mdvLoadText(baseSrc)
		if err != nil {
			return "", err
		}
		baseLoaded := []mdvLoaded{{name: filepath.Base(baseSrc), lines: strings.Split(strings.ReplaceAll(baseText, "\r\n", "\n"), "\n")}}
		baseIssues, _ := mdvAnalyze(baseLoaded, run, false, contentGrams)
		for _, is := range baseIssues {
			if is.key != "" {
				baseKeys[is.key] = true
			}
		}
	}
	if hasBaseline {
		fresh := issues[:0]
		kept := 0
		for _, is := range issues {
			if is.key != "" && baseKeys[is.key] {
				kept++
				continue
			}
			fresh = append(fresh, is)
		}
		issues = fresh
		// 基线把行号冲掉后按内容主键去重一遍（同一缺陷两处行可能都报）。
		seenKey := map[string]bool{}
		deduped := issues[:0]
		for _, is := range issues {
			if is.key != "" {
				if seenKey[is.key] {
					continue
				}
				seenKey[is.key] = true
			}
			deduped = append(deduped, is)
		}
		issues = deduped
		_ = kept
	}
	// 提示项折叠（hints=false）：硬伤照列，提示只报总数（复盘 P0-1「可关闭」）。
	hintsOn := p.Hints == nil || *p.Hints
	hintCount := 0
	if !hintsOn {
		kept := issues[:0]
		for _, is := range issues {
			if is.severity == mdvHint {
				hintCount++
				continue
			}
			kept = append(kept, is)
		}
		issues = kept
	}

	// 硬伤排前（复盘 T2：别让真硬伤淹没在体例噪声里），同级按文件/行号。
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].severity != issues[j].severity {
			return issues[i].severity == mdvHard
		}
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
	hardCount, hintListed := 0, 0
	for _, is := range issues {
		if is.severity == mdvHint {
			hintListed++
		} else {
			hardCount++
		}
	}
	var b strings.Builder
	if len(issues) == 0 {
		if hasBaseline {
			fmt.Fprintf(&b, "OK: %s — 较基线无新增缺陷（基线已有 %d 条，本次均未复发）", scope, len(baseKeys))
		} else {
			fmt.Fprintf(&b, "OK: %s — 编号唯一且连续；引用↔题注对账通过；表格列数一致", scope)
			if multi {
				b.WriteString("（撞号跨文件可检出）")
			} else if !run["numbering"] || !run["orphans"] {
				b.WriteString("（本次为所选 checks 的子集）")
			}
		}
		if s := strings.TrimRight(bNote.String(), "\n"); s != "" {
			b.WriteString("\n" + s)
		}
		return b.String(), nil
	}
	if hasBaseline {
		fmt.Fprintf(&b, "md_validate: %s — 较基线新增 %d 条（硬伤 %d / 提示 %d）；基线已有 %d 条未逐条列\n", scope, len(issues), hardCount, hintListed, len(baseKeys))
	} else {
		fmt.Fprintf(&b, "md_validate: %s — 共 %d 条（硬伤 %d / 提示 %d）\n", scope, len(issues), hardCount, hintListed)
	}
	if !hintsOn && hintCount > 0 {
		fmt.Fprintf(&b, "（另有提示 %d 条已折叠——hints=false；要逐条看传 hints=true）\n", hintCount)
	}
	for _, is := range issues {
		sev := "提示"
		if is.severity == mdvHard {
			sev = "硬伤"
		}
		if multi && is.file != "" {
			fmt.Fprintf(&b, "[%s·%s] %s L%d %s\n", is.kind, sev, is.file, is.line, is.summary)
		} else {
			fmt.Fprintf(&b, "[%s·%s] L%d %s\n", is.kind, sev, is.line, is.summary)
		}
	}
	if s := strings.TrimRight(bNote.String(), "\n"); s != "" {
		b.WriteString(s + "\n")
	}
	if hasBaseline {
		b.WriteString("新增项才是本次引入的，先修它们；基线已有项属源文档固有（要逐条看就跑无基线版）。")
	} else {
		b.WriteString("修复后重跑本工具确认清零；撞号须全局重排（勿只改当前行——其他子代理可能引用了旧号）。")
	}
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
			out = append(out, mdvIssue{kind: "numbering", file: c.file, line: c.line, severity: mdvHard, key: "dup:" + c.key(),
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
				out = append(out, mdvIssue{kind: "numbering", file: sorted[i].file, line: sorted[i].line, severity: mdvHard,
					key: fmt.Sprintf("skip:%s-%d:%d", sorted[i].kind, cur[0], cur[1]),
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
				out = append(out, mdvIssue{kind: "orphans", file: first.file, line: first.line, severity: mdvHard, key: "missing:" + k,
					summary: fmt.Sprintf("孤立引用（全册并集）：引用 %s（L%d 等 %d 处）但全册无此题注", k, first.line, len(locs))})
			} else {
				// 单文件模式：可能是跨章引用（合法）也可能是失锚笔误——
				// 不冒充硬伤，归类待全册核验。
				lines := make([]int, 0, len(locs))
				for _, l := range locs {
					lines = append(lines, l.line)
				}
				out = append(out, mdvIssue{kind: "cross_ref", line: first.line, severity: mdvHint, key: "cross:" + k,
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
	// 撞号的同 key 题注只报一次——同一缺陷报 N 遍正是体例噪声的来源（复盘 T2）。
	unrefSeen := map[string]bool{}
	for _, c := range captions {
		if _, ok := refs[c.key()]; !ok {
			if unrefSeen[c.key()] {
				continue
			}
			unrefSeen[c.key()] = true
			out = append(out, mdvIssue{kind: "orphans", file: c.file, line: c.line, severity: mdvHint, key: "unref:" + c.key(),
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
			var headerSig string
			for k := i; k < j; k++ {
				if isMDVSeparatorRow(lines[k]) {
					continue
				}
				n := mdvColumnCount(lines[k])
				if cols == 0 {
					cols, firstLine = n, k+1
					headerSig = "cols:" + mdvTableKey(lines[k])
					continue
				}
				if n != cols {
					out = append(out, mdvIssue{kind: "table_columns", line: k + 1, severity: mdvHard, key: headerSig,
						summary: fmt.Sprintf("表格块内列数不一致：L%d=%d 列，L%d=%d 列（同表各行必须同列数）", firstLine, cols, k+1, n)})
				}
			}
		}
		i = j
	}
	return out
}

// mdvTableKey 表格块的内容签名（表头行归一）：基线对比按它对齐同一张表——
// 行号在两版之间必然漂移，表头不会。
func mdvTableKey(headerLine string) string {
	fields := strings.FieldsFunc(headerLine, func(r rune) bool { return r == '|' || r == '\n' })
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(strings.TrimSpace(f))
	}
	runes := []rune(b.String())
	if len(runes) > 60 {
		runes = runes[:60]
	}
	return string(runes)
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
