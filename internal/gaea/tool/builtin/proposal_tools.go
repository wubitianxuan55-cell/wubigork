package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/proposal"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── 方案库三工具（2026-11-23 用户拍板立项）────────────────────────

// metaStr 方案库元数据字符串字段的宽容形态（2026-11-27 复盘 T7：模型偶尔把年度
// 传成数字，string 字段直接报「参数无效」逼用户重试）。string 原样收；数字/布尔
// 等 JSON 标量按字面收（2026 → "2026"）；null 与缺席同语义（不动零值）。
type metaStr string

func (s *metaStr) UnmarshalJSON(b []byte) error {
	t := bytes.TrimSpace(b)
	if len(t) == 0 || string(t) == "null" {
		return nil
	}
	if t[0] == '"' {
		var v string
		if err := json.Unmarshal(t, &v); err != nil {
			return err
		}
		*s = metaStr(v)
		return nil
	}
	*s = metaStr(t)
	return nil
}

// 录入与索引都在办公板块的方案模式内由模型完成，无独立解析入口：
//   - proposal_save：模型只定标题层级（outline），正文由工具从源文件逐字切出；
//     也支持直给正文（sections）。长文档分批续录。走 PersistWrite（hardAsk 审批红线）。
//   - proposal_search：全文路（精确词）+ 语义路（段落向量）→ 本地 reranker 精排。
//     检索一律复用现成的本地模型客户端（localEmbedder/localReranker），不另造检索工具。
//   - proposal_read：按需读取（整节 / 命中段落 / 连子节整读）。
// 存储与检索内核在 internal/gaea/proposal。

func init() {
	tool.RegisterBuiltin(proposalSaveTool{})
	tool.RegisterBuiltin(proposalSearchTool{})
	tool.RegisterBuiltin(proposalReadTool{})
}

func proposalStore() (*proposal.Store, error) {
	gdb := db.GetDatabase(config.MemoryUserDir())
	if gdb == nil {
		return nil, fmt.Errorf("方案库不可用（主库未就绪）")
	}
	return proposal.OpenWithAssets(gdb, config.MemoryUserDir())
}

// proposalSemantic 方案库混合检索的本地模型依赖：复用工具侧本地检索客户端
// （SetRetrievalRuntime 注入的 retrieval.Embedder / retrieval.Reranker，即 Herdsman
// 的 OpenAI 兼容 embedding / rerank 接口）。任一不可用即跳过对应环节。
func proposalSemantic() *proposal.Semantic {
	sem := &proposal.Semantic{}
	if e := localEmbedder(); e != nil {
		sem.Embed, sem.Model = e, retrievalEmbedModel()
	}
	if r := localReranker(); r != nil {
		sem.Rerank = r
	}
	if sem.Embed == nil && sem.Rerank == nil {
		return nil
	}
	return sem
}

// ProposalSemantic 供 app 绑定面（方案库板块的检索）取同一套本地模型依赖——
// 与工具面同模型，向量索引才不会两边各按各的模型互相清库。
func ProposalSemantic() *proposal.Semantic { return proposalSemantic() }

// retrievalEmbedModel 生效的 embedding 模型名（与 localEmbedder 的缺省一致）。
func retrievalEmbedModel() string {
	if m := strings.TrimSpace(retrievalRuntime.EmbedModel); m != "" {
		return m
	}
	return "bge-m3"
}

// ── proposal_save：录入（标题 1~4 级，模型定层级）────────────────────

type proposalSaveTool struct{ workDir string }

func (proposalSaveTool) Name() string { return "proposal_save" }
func (proposalSaveTool) Description() string {
	return "把方案录入方案库（标题 1~4 级，由你划分层级）。两种给法：①按源文件切正文（推荐，几百页的方案用这个）：先把原文件转成 Markdown（Word/PPT/PDF 用 format_convert 并设 preserve_images=true），你只列出认定的标题与层级 outline=[{level,title,line?,display_title?}]，正文由工具从源文件逐字切出，不必重打；正文引用的本地图片会复制为方案托管资产并改写稳定引用。源文件标题有 OCR 错字时：title 照源文件的写法定位，display_title 写修正后的（入库用显示名；给了 display_title，title 差一两个字也能对上）。源文件没有井号标题（扫描件转出的正文很常见）：先用 action=detect_headings（配合 source_file）按「第X章 / 1.1 / 一、/（一）」编号形态列出标题样行与建议层级；再给 outline_file 就把结果写成 outline 草稿文件，改完直接用 outline_file 录入，几百个标题不必抄进参数。长文档分批：每批 100~200 个标题，之后各批 mode=append 续录（title 不变），最后一批 finalize=true。②直给正文 sections=[{level,title,body}]，适合短文档或你自己写的内容。outline/sections 须按文档顺序、父级在前、逐级展开（不能跳级，第一项 1 级）；title 写标题原文（可带编号），拿不准时给 line。已有同名方案时整册覆盖须 overwrite=true。分批录入中断后用 action=status（配合 title）查已录到第几行、下一批从哪里接；dry_run=true 先演练一批（校验与定位照做、不写库）。回执的「方案库」一行是全库份数（含录入中、含本方案），与「本批」的录入进度是两回事。仅用户明确要求沉淀方案时调用。"
}
func (proposalSaveTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"方案标题（同一份方案续录时保持不变）"},
  "project_type":{"type":"string","description":"项目类型（如 技术标/修复方案/调查报告）"},
  "year":{"type":"string","description":"年度（如 2026）"},
  "summary":{"type":"string","description":"一段话摘要"},
  "mode":{"type":"string","enum":["replace","append"],"description":"replace=新建或整册覆盖（默认）；append=续录同一份方案"},
  "overwrite":{"type":"boolean","description":"方案库已有同名方案时，整册覆盖须显式 true"},
  "source_file":{"type":"string","description":"已转成 Markdown 的源文件路径（.md/.txt）；按 outline 录入、认标题时必填"},
  "outline_file":{"type":"string","description":"outline 的文件形态（JSON 数组，元素同 outline 条目）：录入时从这里读，几百个标题不必抄进参数；action=detect_headings 时反过来——把认出的标题草稿写到这个路径"},
  "outline":{"type":"array","description":"你认定的标题与层级，按文档顺序（不必带正文；与 outline_file 二选一）","items":{"type":"object","properties":{
    "level":{"type":"integer","description":"标题层级 1~4"},
    "title":{"type":"string","description":"定位用：源文件里标题的写法（照原文，OCR 错字也照写）"},
    "line":{"type":"integer","description":"该标题在源文件中的行号（可选，拿不准定位时给）"},
    "display_title":{"type":"string","description":"入库显示名（可选）：源文件标题有错字时写修正后的；给了它，title 差一两个字也能对上，源文件行自带的编号会补回去"}
  },"required":["level","title"]}},
  "sections":{"type":"array","description":"直给正文（与 outline 二选一）","items":{"type":"object","properties":{
    "level":{"type":"integer","description":"标题层级 1~4"},
    "title":{"type":"string","description":"标题"},
    "body":{"type":"string","description":"该节正文（Markdown）"}
  },"required":["level","title","body"]}},
  "finalize":{"type":"boolean","description":"源文件形态：本批是最后一批，末节正文截至文件尾"},
  "action":{"type":"string","enum":["status","detect_headings"],"description":"都不写库：status=查录入进度（配合 title：已录到源文件第几行、下一批从哪里接）；detect_headings=认标题（配合 source_file：源文件没有井号标题时，按编号形态列出标题样行与建议层级；再给 outline_file 就把草稿写成文件）"},
  "dry_run":{"type":"boolean","description":"演练：整批校验、标题定位、切正文照做，但不写库；确认无误后去掉它正式录入"}
}}`)
}
func (proposalSaveTool) ReadOnly() bool     { return false }
func (proposalSaveTool) PersistWrite() bool { return true }
func (proposalSaveTool) CompactDescription() string {
	return "方案入库(标题1~4级;outline按源文件切正文,title定位/display_title入库,可append分批;无井号标题先action=detect_headings认标题,可outline_file读写草稿;或直给sections;action=status查进度;dry_run演练不写库)"
}
func (proposalSaveTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"title":{"type":"string"},"project_type":{"type":"string"},"year":{"type":"string"},"summary":{"type":"string"},"mode":{"type":"string","enum":["replace","append"]},"overwrite":{"type":"boolean"},"source_file":{"type":"string"},"outline_file":{"type":"string"},"outline":{"type":"array"},"sections":{"type":"array"},"finalize":{"type":"boolean"},"action":{"type":"string","enum":["status","detect_headings"]},"dry_run":{"type":"boolean"}}}`)
}

const (
	// proposalSourceMaxBytes 源文件上限。
	proposalSourceMaxBytes = 64 << 20
	// proposalLongSectionHint 小节字数超过它就在回执里提示补下级标题。
	proposalLongSectionHint = 8000
	// proposalIndexBudget 录入后向量化本方案语义块的时间预算。
	proposalIndexBudget = 90 * time.Second
	// proposalOutlineCap 标题树一次最多列出的节点数。
	proposalOutlineCap = 300
)

func (t proposalSaveTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Title       string               `json:"title"`
		ProjectType string               `json:"project_type"`
		Year        metaStr              `json:"year"`
		Summary     string               `json:"summary"`
		Mode        string               `json:"mode"`
		Overwrite   bool                 `json:"overwrite"`
		SourceFile  string               `json:"source_file"`
		OutlineFile string               `json:"outline_file"`
		Outline     []proposalOutlineArg `json:"outline"`
		Sections    []struct {
			Level int    `json:"level"`
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"sections"`
		Finalize bool   `json:"finalize"`
		Action   string `json:"action"`
		DryRun   bool   `json:"dry_run"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	// 只读的进度查询走同一个工具（不另设工具占提示词位）：分批续录被打断后，模型据此
	// 知道「已录到第几行、下一批从哪个标题接」，不必靶回执末尾的行号人工推算。
	switch action := strings.ToLower(strings.TrimSpace(p.Action)); action {
	case "":
	case "status":
		if p.DryRun || p.Mode != "" || len(p.Outline) > 0 || len(p.Sections) > 0 {
			return "", errors.New("action=status 只查进度，不能同时带录入内容（outline/sections/mode/dry_run）")
		}
		return tool.WrapText(proposalIngestStatus(t.workDir, strings.TrimSpace(p.Title))), nil
	case "detect_headings":
		// 认标题（2026-11-27 AI 反馈 #12）：扫描件转出的正文没有井号标题时，此前每次
		// 导入都要手写脚本按编号形态找标题行。只读源文件、不写库；带了 outline_file
		// 才在工作区落一份 outline 草稿（非交付件）。
		if p.DryRun || p.Mode != "" || strings.TrimSpace(p.Title) != "" || len(p.Outline) > 0 || len(p.Sections) > 0 {
			return "", errors.New("action=detect_headings 只认标题，不能同时带录入内容（title/outline/sections/mode/dry_run）")
		}
		text, err := proposalDetectHeadings(t.workDir, p.SourceFile, p.OutlineFile)
		if err != nil {
			return "", err
		}
		return tool.WrapText(text), nil
	default:
		return "", fmt.Errorf("action 只支持 status（查录入进度）与 detect_headings（认标题），都不写库；收到 %q", p.Action)
	}
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	switch mode {
	case "", "replace":
	case "append":
	default:
		return "", fmt.Errorf("mode 只能是 replace（默认，新建或整册覆盖）或 append（续录同一份方案），收到 %q", p.Mode)
	}
	in := proposal.SaveInput{
		Meta: proposal.Proposal{
			Title: p.Title, ProjectType: strings.TrimSpace(p.ProjectType),
			Year: strings.TrimSpace(string(p.Year)), Summary: strings.TrimSpace(p.Summary),
		},
		Overwrite: p.Overwrite, Append: mode == "append", Finalize: p.Finalize,
		SourceFile:   strings.TrimSpace(p.SourceFile),
		AssetBaseDir: t.workDir,
		DryRun:       p.DryRun,
	}
	if p.OutlineFile != "" && len(p.Outline) > 0 {
		return "", errors.New("outline 与 outline_file 只能给一个")
	}
	if p.OutlineFile != "" {
		loaded, err := proposalOutlineFromFile(t.workDir, p.OutlineFile)
		if err != nil {
			return "", err
		}
		p.Outline = loaded
	}
	for _, e := range p.Outline {
		in.Outline = append(in.Outline, proposal.OutlineEntry{Level: e.Level, Title: e.Title, Line: e.Line, DisplayTitle: e.DisplayTitle})
	}
	for _, s := range p.Sections {
		in.Sections = append(in.Sections, proposal.Section{Level: s.Level, Title: s.Title, Body: s.Body})
	}
	if len(in.Outline) > 0 || (in.Append && in.Finalize && in.SourceFile != "") {
		abs, lines, err := proposalSourceLines(t.workDir, in.SourceFile)
		if err != nil {
			return "", err
		}
		in.Lines = lines
		in.AssetBaseDir = filepath.Dir(abs)
		if in.Meta.Source == "" {
			in.Meta.Source = filepath.Base(abs)
		}
		if in.Meta.Workspace == "" {
			in.Meta.Workspace = filepath.Dir(abs)
		}
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	res, err := st.Save(in)
	if err != nil {
		return "", err
	}
	return tool.WrapText(proposalSaveReceipt(ctx, st, in, res)), nil
}

// proposalOutlineArg outline 条目的参数形态：内联 outline 与 outline_file 文件同构。
type proposalOutlineArg struct {
	Level        int    `json:"level"`
	Title        string `json:"title"`
	Line         int    `json:"line"`
	DisplayTitle string `json:"display_title"`
}

// proposalDetectRowCap 认标题回执最多列多少行（再多的给 outline_file 写成文件）。
const proposalDetectRowCap = 600

// proposalOutlineFromFile 读 outline_file：标准形态是 JSON 数组（元素同 outline 条目，
// 草稿每行一条，便于逐行编辑）；每行一个对象的写法也认。
func proposalOutlineFromFile(workDir, path string) ([]proposalOutlineArg, error) {
	abs := resolveIn(workDir, path)
	if SessionDirFenceRule(abs) {
		return nil, fmt.Errorf("SESSION_DENIED: .gaea/sessions/ 是引擎内部事件流，不能当 outline 文件")
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("读不到 outline_file %s: %w", path, err)
	}
	var entries []proposalOutlineArg
	if err := json.Unmarshal(raw, &entries); err != nil {
		entries = nil
		for n, line := range strings.Split(string(raw), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var one proposalOutlineArg
			if jerr := json.Unmarshal([]byte(line), &one); jerr != nil {
				return nil, fmt.Errorf("outline_file %s 第 %d 行不是 outline 条目（元素形如 {level,title,line,display_title}）：%v", path, n+1, jerr)
			}
			entries = append(entries, one)
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("outline_file %s 是空的", path)
	}
	return entries, nil
}

// proposalDetectHeadings 认标题（proposal_save action=detect_headings）：按编号形态
// 列出标题样行与建议层级。给了 outlineFile 就把拿得准的那些写成 outline 草稿
// （每行一条 JSON），拿不准的只在回执里列出来。
func proposalDetectHeadings(workDir, sourceFile, outlineFile string) (string, error) {
	_, lines, err := proposalSourceLines(workDir, sourceFile)
	if err != nil {
		return "", err
	}
	cands := proposal.DetectHeadings(lines)
	var b strings.Builder
	fmt.Fprintf(&b, "## 认标题：%s（共 %d 行）\n\n", strings.TrimSpace(sourceFile), len(lines))
	if len(cands) == 0 {
		b.WriteString("没认出编号形态的标题样行。只认井号标题与「第X章 / 第X节 / 1.1 / 一、/（一）/ 1.」；目录行、代码块、表格里的正文不算。源文件用的是别的体例时请自己列 outline（可先 dry_run=true 演练）。\n")
		return b.String(), nil
	}
	var levelCount [proposal.MaxLevels + 1]int
	kindCount := map[string]int{}
	var kindOrder []string
	sure := make([]proposal.HeadingCandidate, 0, len(cands))
	unsure := make([]proposal.HeadingCandidate, 0, len(cands))
	for _, c := range cands {
		if kindCount[c.Kind] == 0 {
			kindOrder = append(kindOrder, c.Kind)
		}
		kindCount[c.Kind]++
		if c.Weak {
			unsure = append(unsure, c)
			continue
		}
		levelCount[c.Level]++
		sure = append(sure, c)
	}
	fmt.Fprintf(&b, "**认出 %d 个标题样行**（拿得准 %d、拿不准 %d）", len(cands), len(sure), len(unsure))
	if len(sure) > 0 {
		var levels []string
		for lv := 1; lv <= proposal.MaxLevels; lv++ {
			if levelCount[lv] > 0 {
				levels = append(levels, fmt.Sprintf("%d 级 %d", lv, levelCount[lv]))
			}
		}
		fmt.Fprintf(&b, "；层级建议：%s", strings.Join(levels, "、"))
	}
	var kinds []string
	for _, k := range kindOrder {
		kinds = append(kinds, fmt.Sprintf("%s %d", k, kindCount[k]))
	}
	fmt.Fprintf(&b, "；形态：%s\n", strings.Join(kinds, "、"))

	if outlineFile != "" {
		out := resolveIn(workDir, outlineFile)
		if SessionDirFenceRule(out) {
			return "", fmt.Errorf("SESSION_DENIED: .gaea/sessions/ 是引擎内部事件流，不能把 outline 草稿写进去")
		}
		if workDir != "" {
			if err := confine([]string{workDir}, out); err != nil {
				return "", err
			}
		}
		var buf strings.Builder
		buf.WriteString("[\n")
		for i, c := range sure {
			one, _ := json.Marshal(map[string]any{"level": c.Level, "title": c.Title, "line": c.Line})
			buf.Write(one)
			if i < len(sure)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString("]\n")
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", fmt.Errorf("建 outline 草稿目录失败: %w", err)
		}
		if err := os.WriteFile(out, []byte(buf.String()), 0o644); err != nil {
			return "", fmt.Errorf("写 outline 草稿失败: %w", err)
		}
		fmt.Fprintf(&b, "\n**草稿已写出**：%s（%d 条，每行一条 level / title / line）\n", strings.TrimSpace(outlineFile), len(sure))
		proposalListCandidates(&b, "开头", sure[:min(5, len(sure))])
		if len(sure) > 5 {
			proposalListCandidates(&b, "末尾", sure[max(5, len(sure)-3):])
		}
		if len(unsure) > 0 {
			fmt.Fprintf(&b, "**没写进草稿的 %d 行**（像标题但拿不准：单个数字开头、或藏在表格行里；要用就自己加进草稿）：\n", len(unsure))
			for i, c := range unsure {
				if i >= 12 {
					fmt.Fprintf(&b, "- …共 %d 行\n", len(unsure))
					break
				}
				fmt.Fprintf(&b, "- 第 %d 行「%s」（%s）\n", c.Line, c.Title, c.Kind)
			}
		}
		b.WriteString("\n下一步：层级是按编号形态与上下文推的，以你的认定为准。用 edit_file 改草稿（层级不对的、多认的删掉、错字加 display_title），再 proposal_save 带 source_file + outline_file 录入（先加 dry_run=true 演练）。\n")
		return b.String(), nil
	}

	b.WriteString("\n| 行 | 级 | 形态 | 标题 |\n|---|---|---|---|\n")
	for i, c := range cands {
		if i >= proposalDetectRowCap {
			fmt.Fprintf(&b, "…（共 %d 行，只列前 %d 行；给 outline_file 可把结果写成草稿文件）\n", len(cands), proposalDetectRowCap)
			break
		}
		kind := c.Kind
		if c.InTable {
			kind += "·表格行"
		} else if c.Weak {
			kind += "·拿不准"
		}
		fmt.Fprintf(&b, "| %d | %d | %s | %s |\n", c.Line, c.Level, kind, cell(c.Title))
	}
	b.WriteString("\n下一步：层级是按编号形态与上下文推的，以你的认定为准。标题多时给 outline_file 把这份清单写成草稿再改；也可以直接抄进 outline（带 line）。先加 dry_run=true 演练。\n")
	return b.String(), nil
}

// proposalListCandidates 回执里列几条候选：「第 12 行 1 级「…」」。
func proposalListCandidates(b *strings.Builder, label string, cs []proposal.HeadingCandidate) {
	if len(cs) == 0 {
		return
	}
	fmt.Fprintf(b, "**%s**：", label)
	for i, c := range cs {
		if i > 0 {
			b.WriteString("、")
		}
		fmt.Fprintf(b, "第 %d 行 %d 级「%s」", c.Line, c.Level, c.Title)
	}
	b.WriteString("\n")
}

// proposalSourceLines 读源文件为行数组（行号与 read_file/grep -n 一致）。只收
// Markdown/文本——Word/PDF 先经 format_convert 转 Markdown，转换只做一次，
// 分批续录时行号才稳定。编码按 read_file 同一套检测（UTF-8/UTF-16/GBK）。
func proposalSourceLines(workDir, path string) (abs string, lines []string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil, errors.New("按 outline 录入需要 source_file（已转成 Markdown 的源文件路径）")
	}
	abs = resolveIn(workDir, path)
	if SessionDirFenceRule(abs) {
		return "", nil, fmt.Errorf("SESSION_DENIED: .gaea/sessions/ 是引擎内部事件流，不能作为方案库的来源文件")
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".md", ".markdown", ".txt", "":
	default:
		return "", nil, fmt.Errorf("source_file 只收 Markdown/文本（.md/.txt/.markdown）；Word/PDF 请先用 format_convert 转成 .md 再录入（收到 %s）", filepath.Ext(abs))
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", nil, fmt.Errorf("读不到源文件 %s: %w", abs, err)
	}
	if info.IsDir() {
		return "", nil, fmt.Errorf("source_file 是目录，不是文件：%s", abs)
	}
	if info.Size() > proposalSourceMaxBytes {
		return "", nil, fmt.Errorf("源文件 %s 有 %d MB，超过上限 %d MB", abs, info.Size()>>20, proposalSourceMaxBytes>>20)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, fmt.Errorf("读源文件失败: %w", err)
	}
	kind, _ := fileenc.Detect(raw)
	text := string(fileenc.Decode(raw, kind))
	text = strings.TrimPrefix(text, "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines = strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // 末尾换行不算一行
	}
	return abs, lines, nil
}

// proposalSaveReceipt 录入回执：结构统计、源文件进度、语义索引进度、过长小节、
// 弱匹配点名与下一步指引。
func proposalSaveReceipt(ctx context.Context, st *proposal.Store, in proposal.SaveInput, res *proposal.SaveResult) string {
	title := strings.TrimSpace(in.Meta.Title)
	var b strings.Builder
	switch {
	case res.DryRun && res.Done:
		fmt.Fprintf(&b, "🔍 演练通过（未写库）：%s\n\n", title)
	case res.DryRun:
		fmt.Fprintf(&b, "🔍 演练通过（未写库；末节正文待收尾）：%s\n\n", title)
	case res.Done:
		fmt.Fprintf(&b, "✅ 已录入方案库：%s\n\n", title)
	default:
		fmt.Fprintf(&b, "✅ 已录入方案库（录入中，末节正文待收尾）：%s\n\n", title)
	}
	if in.Meta.ProjectType != "" || in.Meta.Year != "" {
		fmt.Fprintf(&b, "**类型/年度**: %s / %s\n", orDash(in.Meta.ProjectType), orDash(in.Meta.Year))
	}
	// 落点写进回执（2026-11-24 实弹：方案归进哪个目录只字未提，界面上「少了一份」
	// 却无从查起）。2026-11-27 AI 反馈 #5'：同一方案分 4 批录入，「方案库共 N 份」
	// 逐批 3→2→1，被读成本方案的录入进度——份数与落点拆成两行，份数写明是全库
	// 口径（现行 / 录入中、含不含本方案），录入进度只看「本批」「源文件」两行。
	if list, err := st.List(); err == nil {
		n, ingesting := len(list), 0
		for i := range list {
			if list[i].Status == proposal.StatusIngesting {
				ingesting++
			}
		}
		fmt.Fprintf(&b, "**落点**: %s\n", proposalLanding(st, in))
		if res.DryRun && res.Created {
			fmt.Fprintf(&b, "**方案库**: 现有 %d 份（现行 %d、录入中 %d），不含本方案；正式写入后 %d 份\n", n, n-ingesting, ingesting, n+1)
		} else {
			fmt.Fprintf(&b, "**方案库**: 全库共 %d 份（现行 %d、录入中 %d），含本方案\n", n, n-ingesting, ingesting)
		}
	}
	stt := res.Stats
	if res.DryRun {
		fmt.Fprintf(&b, "**本批（演练）**: 将新增 %d 个标题节点（压到 4 级 %d 个）；写入后共 %d 个节点、%s（1 级 %d / 2 级 %d / 3 级 %d / 4 级 %d）\n",
			res.Added, res.Clamped, stt.Sections, fmtChars(stt.Chars), stt.LevelCounts[1], stt.LevelCounts[2], stt.LevelCounts[3], stt.LevelCounts[4])
	} else {
		fmt.Fprintf(&b, "**本批**: 新增 %d 个标题节点（压到 4 级 %d 个）；累计 %d 个节点、%s（1 级 %d / 2 级 %d / 3 级 %d / 4 级 %d）\n",
			res.Added, res.Clamped, stt.Sections, fmtChars(stt.Chars), stt.LevelCounts[1], stt.LevelCounts[2], stt.LevelCounts[3], stt.LevelCounts[4])
	}
	fmt.Fprintf(&b, "**字数口径**: 库内=%s 为 md 字符数（含空白与标记）；成稿去空白口径会低约 8%%，与 PDF 原件字数三者不等属正常\n", fmtChars(stt.Chars))
	if res.TotalLines > 0 {
		fmt.Fprintf(&b, "**源文件**: %s（共 %d 行，已录到第 %d 行）", in.SourceFile, res.TotalLines, res.IngestLine)
		if res.SkippedLines > 0 {
			fmt.Fprintf(&b, "；首个标题前 %d 行（封面/目录）未入库", res.SkippedLines)
		}
		b.WriteString("\n")
	}
	if res.DryRun {
		b.WriteString("**语义索引**: 演练不写库、不建索引\n")
	} else {
		fmt.Fprintf(&b, "**语义索引**: %s\n", proposalIndexStatus(ctx, st, res.ProposalID))
	}
	if res.Assets > 0 {
		fmt.Fprintf(&b, "**图片资产**: 已托管 %d 张（正文引用已改为 gaea-asset:// 稳定 ID）\n", res.Assets)
	}
	if len(res.MissingAssets) > 0 {
		fmt.Fprintf(&b, "**图片未托管**（%d 处，原引用保留；修正路径或重新转换后再录入）：\n", len(res.MissingAssets))
		for i, item := range res.MissingAssets {
			if i >= 8 {
				fmt.Fprintf(&b, "- …共 %d 处\n", len(res.MissingAssets))
				break
			}
			b.WriteString("- " + item + "\n")
		}
	}
	if long := proposalLongSections(st, res.ProposalID); long != "" {
		fmt.Fprintf(&b, "\n**过长小节**（超过 %s，建议在其内部补下级标题后重录）：\n%s\n", fmtChars(proposalLongSectionHint), long)
	}
	if len(res.WeakMatches) > 0 {
		b.WriteString("\n**标题定位待核对**（落在正文句或长行上；若是误判，用 line 指明标题行号后重发该批）：\n")
		for i, w := range res.WeakMatches {
			if i >= 12 {
				fmt.Fprintf(&b, "- …共 %d 处\n", len(res.WeakMatches))
				break
			}
			b.WriteString("- " + w + "\n")
		}
	}
	if len(res.TitleFixes) > 0 {
		b.WriteString("\n**标题修正**（title 定位到源文件行，入库用的是 display_title；请核对）：\n")
		for i, f := range res.TitleFixes {
			if i >= 12 {
				fmt.Fprintf(&b, "- …共 %d 处\n", len(res.TitleFixes))
				break
			}
			b.WriteString("- " + f + "\n")
		}
	}
	b.WriteString("\n")
	switch {
	case res.DryRun:
		b.WriteString("下一步：确认无误后去掉 dry_run=true 正式录入这一批（写入前会请你确认）。\n")
	case !res.Done:
		fmt.Fprintf(&b, "下一步：继续 mode=append 传下一批 outline（title 保持 %q）；全部传完用 mode=append + finalize=true 收尾。\n", title)
	case res.Replaced:
		b.WriteString("已整册覆盖旧内容。用 proposal_search 抽查几个关键词核对结构。\n")
	default:
		b.WriteString("用 proposal_search 抽查几个关键词核对结构；写新方案时按标题树参照骨架、按段落取材。\n")
	}
	return b.String()
}

// proposalLanding 回执里的落点：已入库的取库里记的工作区（续录时沿用原值），演练新方案
// 取本次给的；都没有就写「未注明」。工作区只是归档信息，不参与分组与检索。
func proposalLanding(st *proposal.Store, in proposal.SaveInput) string {
	ws := strings.TrimSpace(in.Meta.Workspace)
	if p, err := st.Resolve(strings.TrimSpace(in.Meta.Title)); err == nil && p != nil {
		ws = strings.TrimSpace(p.Workspace)
	}
	return "工作区 " + orDash(ws)
}

// proposalIngestStatus 分批续录的进度（proposal_save action=status）：已录到源文件第
// 几行、末几个标题、下一批从哪里接。
func proposalIngestStatus(workDir, title string) string {
	if title == "" {
		return "action=status 需要同时给 title（方案标题）。"
	}
	st, err := proposalStore()
	if err != nil {
		return err.Error()
	}
	p, err := st.Resolve(title)
	if err != nil {
		return fmt.Sprintf("%s: %v", proposal.ErrStoreUnreadable, err)
	}
	if p == nil {
		return fmt.Sprintf("方案库里没有《%s》。不填参数调用 proposal_search 可看方案清单。", title)
	}
	secs, err := st.Outline(p.ID)
	if err != nil {
		return fmt.Sprintf("%s: %v", proposal.ErrStoreUnreadable, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 《%s》录入状态\n\n", p.Title)
	if p.Status != proposal.StatusIngesting {
		fmt.Fprintf(&b, "**状态**: 现行——已录入完成，%d 个标题节点。\n", len(secs))
		b.WriteString("要改结构请整册重录（mode=replace + overwrite=true）；只改个别标题或某一节正文用 proposal_update。\n")
		return b.String()
	}
	b.WriteString("**状态**: 录入中——最后一批的末节正文还没截取，全部录完前这份方案在库里不算完整。\n")
	if p.SourceFile != "" {
		if abs, lines, err := proposalSourceLines(workDir, p.SourceFile); err == nil {
			fmt.Fprintf(&b, "**源文件**: %s（共 %d 行，已录到第 %d 行）\n", abs, len(lines), p.IngestLine)
		} else {
			fmt.Fprintf(&b, "**源文件**: %s（现在读不到：%v）\n", p.SourceFile, err)
		}
	}
	fmt.Fprintf(&b, "**已录**: %d 个标题节点", len(secs))
	if n := len(secs); n > 0 {
		from := n - 3
		if from < 0 {
			from = 0
		}
		fmt.Fprintf(&b, "；末 %d 个：", n-from)
		for i := from; i < n; i++ {
			if i > from {
				b.WriteString("、")
			}
			fmt.Fprintf(&b, "%d 级「%s」（第 %d 行）", secs[i].Level, secs[i].Title, secs[i].SrcLine)
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "**下一步**: 从源文件第 %d 行之后接着录——mode=append、title 保持 %q，传下一批 outline（首个标题须在第 %d 行之后）；全部传完用 mode=append + finalize=true 收尾。拿不准下一批怎么切，可先加 dry_run=true 演练。\n",
		p.IngestLine+1, p.Title, p.IngestLine)
	return b.String()
}

// proposalIndexStatus 录入后把本方案的语义块向量化（尽力而为：本地 embedding
// 不可用或超时都不影响已落库的内容，余量检索时自动补齐），返回一句状态。
func proposalIndexStatus(ctx context.Context, st *proposal.Store, pid int64) string {
	sem := proposalSemantic()
	if sem == nil || sem.Embed == nil {
		return "本地 embedding 不可用，暂未向量化（检索时只按全文路）"
	}
	if cur, err := st.EmbedModel(); err == nil && cur != "" && cur != sem.Model {
		_ = st.ResetEmbedding(sem.Model) // 模型变了：旧向量不可比，整库重嵌（本方案先补，其余检索时补）
	}
	_, pending, err := st.EmbedPending(ctx, sem.Embed, sem.Model, pid, proposalIndexBudget)
	if err != nil {
		return fmt.Sprintf("向量化中断（%v），余量检索时自动补齐", err)
	}
	_, total := st.PendingCount(pid)
	if total == 0 {
		return "无可向量化的正文"
	}
	if pending > 0 {
		return fmt.Sprintf("已向量化 %d/%d 块，余 %d 块检索时自动补齐（%s）", total-pending, total, pending, sem.Model)
	}
	return fmt.Sprintf("已向量化 %d/%d 块（%s）", total, total, sem.Model)
}

func proposalLongSections(st *proposal.Store, pid int64) string {
	long, err := st.Longest(pid, 6)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, s := range long {
		if s.Chars <= proposalLongSectionHint {
			break
		}
		fmt.Fprintf(&b, "- %s（%s）\n", s.TitlePath, fmtChars(s.Chars))
	}
	return b.String()
}

// ── proposal_search：全文 + 语义 → 本地精排 ─────────────────────────

type proposalSearchTool struct{}

func (proposalSearchTool) Name() string { return "proposal_search" }
func (proposalSearchTool) Description() string {
	return "检索方案库。query 可以是关键词（多个词用空格分隔，都需出现），也可以是自然语言问法——全文路管精确词（规范号、术语），语义路（本地 embedding）管换了说法的问法，再由本地 reranker 精排。scope=title 只在 1~4 级标题里找（参照历史方案的章节骨架）。可按 level/project_type/year/proposal 过滤。不填 query：给 proposal 就返回该方案的标题树，否则返回方案清单。命中给出小节 id、标题路径、命中位置摘要和子节清单，用 proposal_read 读取。"
}
func (proposalSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"关键词或自然语言问法（可选）"},
  "scope":{"type":"string","enum":["title"],"description":"title=只在标题树里找（章节骨架参照）"},
  "level":{"type":"integer","description":"标题层级过滤 1~4（可选）"},
  "project_type":{"type":"string","description":"项目类型过滤（可选）"},
  "year":{"type":"string","description":"年度过滤（可选）"},
  "proposal":{"type":"string","description":"限定在某份方案内（标题；可选）"},
  "limit":{"type":"integer","description":"返回条数（默认 10，上限 30）"}
}
}`)
}
func (proposalSearchTool) ReadOnly() bool { return true }
func (proposalSearchTool) CompactDescription() string {
	return "检索方案库(全文+本地语义召回+精排;scope=title查标题树;可按层级/类型/年度/方案过滤)"
}
func (proposalSearchTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"scope":{"type":"string","enum":["title"]},"level":{"type":"integer"},"project_type":{"type":"string"},"year":{"type":"string"},"proposal":{"type":"string"},"limit":{"type":"integer"}}}`)
}

func (proposalSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query       string  `json:"query"`
		Scope       string  `json:"scope"`
		Level       int     `json:"level"`
		ProjectType string  `json:"project_type"`
		Year        metaStr `json:"year"`
		Proposal    string  `json:"proposal"`
		Limit       int     `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(p.Query) == "" {
		text := ""
		if name := strings.TrimSpace(p.Proposal); name != "" {
			text, err = proposalOutlineText(st, name, p.Level)
		} else {
			text, err = proposalOverview(st)
		}
		if err != nil {
			return "", err
		}
		return tool.WrapText(text), nil
	}
	in := proposal.SearchInput{
		Query: p.Query, Level: p.Level, TopN: p.Limit,
		ProjectType: strings.TrimSpace(p.ProjectType), Year: strings.TrimSpace(string(p.Year)),
		Proposal: strings.TrimSpace(p.Proposal),
	}
	switch strings.ToLower(strings.TrimSpace(p.Scope)) {
	case "", "all", "body":
	case "title":
		in.Scope = proposal.ScopeTitle
	default:
		return "", fmt.Errorf("scope 只能是 title（只在标题树里找）或不填，收到 %q", p.Scope)
	}
	res, err := st.HybridSearch(ctx, in, proposalSemantic())
	if err != nil {
		return "", fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
	}
	if len(res.Hits) == 0 {
		return tool.WrapText("方案库未命中。换词（两字以上的专业词最稳）、换说法，或用 scope=title 按章节标题找；不填 query 可看方案清单与标题树。"), nil
	}
	return tool.WrapText(proposalHitsText(res)), nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// fmtChars 字数的人读形式（过万折成「x.x 万字」）。
// 口径（沿江包1 复盘 T6）：库内字数 = SUM(LENGTH(body))，即 md 字符数**含空白
// 与标记**；成稿去空白口径会低约 8%，PDF 原件口径另计——三个数都对但不等，
// 回执里注明防误读。
func fmtChars(n int) string {
	if n >= 10000 {
		return fmt.Sprintf("%.1f 万字", float64(n)/10000)
	}
	return fmt.Sprintf("%d 字", n)
}

func ingestingTag(status string) string {
	if status == proposal.StatusIngesting {
		return "〔录入中，尚未收尾〕"
	}
	return ""
}

// quoteLines 把可能是多行的文本排成 Markdown 引用块。
func quoteLines(s string) string {
	s = strings.TrimRight(s, "\n")
	return "> " + strings.ReplaceAll(s, "\n", "\n> ")
}

func proposalOverview(st *proposal.Store) (string, error) {
	list, err := st.List()
	if err != nil {
		return "", fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
	}
	if len(list) == 0 {
		return "方案库为空。把历史方案录进来：在方案模式里先 format_convert 转成 Markdown，再用 proposal_save 按标题层级录入。", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 方案库概览（%d 份）\n\n", len(list))
	b.WriteString("| 标题 | 类型 | 年度 | 状态 | 标题节点 | 字数 | 语义索引 | 目录 | 来源 |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, it := range list {
		idx := "—"
		if it.IndexTotal > 0 {
			idx = fmt.Sprintf("%d/%d", it.IndexDone, it.IndexTotal)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %s | %s | %s | %s |\n",
			cell(it.Title), cell(orDash(it.ProjectType)), cell(orDash(it.Year)), it.Status,
			it.Sections, fmtChars(it.Chars), idx, cell(proposalDirName(it.Workspace)), cell(orDash(it.Source)))
	}
	b.WriteString("\n目录=来源工作区的末级目录名（只是归档信息，不参与分组与检索）。用 query 检索；给 proposal（标题）看它的标题树。")
	return b.String(), nil
}

// proposalDirName 工作区路径的末级目录名（清单里只放这一段，完整路径在 proposal_update 的回执里）。
func proposalDirName(ws string) string {
	ws = strings.TrimSpace(strings.TrimRight(strings.ReplaceAll(ws, "/", "\\"), "\\"))
	if ws == "" {
		return "—"
	}
	return filepath.Base(ws)
}

// proposalOutlineText 某方案的标题树（level 1~4 时只列该级）。
func proposalOutlineText(st *proposal.Store, name string, level int) (string, error) {
	p, err := st.Resolve(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
	}
	if p == nil {
		return fmt.Sprintf("方案库里没有《%s》。不填 proposal 可看方案清单。", name), nil
	}
	secs, err := st.Outline(p.ID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
	}
	if level >= 1 && level <= proposal.MaxLevels {
		kept := secs[:0:0]
		for _, s := range secs {
			if s.Level == level {
				kept = append(kept, s)
			}
		}
		secs = kept
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 《%s》标题树（共 %d 节%s）\n\n", p.Title, len(secs), ingestingTag(p.Status))
	for i, s := range secs {
		if i >= proposalOutlineCap {
			fmt.Fprintf(&b, "…（共 %d 节，仅列前 %d 个；用 level 过滤，或 proposal_read 看某节的子节）\n", len(secs), proposalOutlineCap)
			break
		}
		fmt.Fprintf(&b, "%s[%d] %s（%s）\n", strings.Repeat("  ", s.Level-1), s.ID, s.Title, fmtChars(s.Chars))
	}
	b.WriteString("\n方括号里是 section_id；proposal_read section_id=… 读该节（scope=subtree 连子节，query 只取相关段落）。")
	return b.String(), nil
}

func proposalHitsText(res *proposal.SearchResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## 方案库命中 %d 条（%s）\n\n", len(res.Hits), res.Mode)
	if res.Note != "" {
		fmt.Fprintf(&b, "%s\n\n", res.Note)
	}
	for i, h := range res.Hits {
		fmt.Fprintf(&b, "### %d. %s\n\n", i+1, h.Title)
		fmt.Fprintf(&b, "- **方案**: %s（%s / %s）%s\n", h.ProposalTitle, orDash(h.ProjectType), orDash(h.Year), ingestingTag(h.Status))
		fmt.Fprintf(&b, "- **层级**: L%d　**路径**: %s　**本节**: %s\n", h.Level, h.TitlePath, fmtChars(h.Chars))
		if len(h.Matched) > 0 || h.SemanticScore > 0 || h.RerankScore != 0 {
			fmt.Fprintf(&b, "- **命中**: ")
			if len(h.Matched) > 0 {
				fmt.Fprintf(&b, "全文词 %s", strings.Join(h.Matched, "、"))
			}
			if h.SemanticScore > 0 {
				fmt.Fprintf(&b, "　语义相似度 %.2f", h.SemanticScore)
			}
			if h.RerankScore != 0 {
				fmt.Fprintf(&b, "　精排分 %.2f", h.RerankScore)
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s\n\n", quoteLines(h.Snippet))
		if len(h.Children) > 0 {
			b.WriteString("- **子节**（可下钻）: ")
			for j, c := range h.Children {
				if j >= 12 {
					fmt.Fprintf(&b, "…共 %d 个", h.ChildCount)
					break
				}
				if j > 0 {
					b.WriteString("；")
				}
				fmt.Fprintf(&b, "[%d] %s（%s）", c.ID, c.Title, fmtChars(c.Chars))
			}
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "\n（proposal_read section_id=%d 读整节；加 query 只取相关段落）\n\n---\n\n", h.SectionID)
	}
	return b.String()
}

// ── proposal_read：按需读取 ─────────────────────────────────────────

type proposalReadTool struct{}

func (proposalReadTool) Name() string { return "proposal_read" }
func (proposalReadTool) Description() string {
	return "读取方案库某小节：默认返回整节正文（过长自动分页，回执给出续读 offset）；query 只取含这些词的段落；scope=subtree 连同全部子节整读。section_id 来自 proposal_search 命中或标题树（方括号里的数字）。引用其中的内容时注明方案名与标题路径——别的项目里的工程量、人员、设备等专属数据不得照搬。"
}
func (proposalReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "section_id":{"type":"integer","description":"小节 id（proposal_search 命中或标题树方括号里的数字）"},
  "scope":{"type":"string","enum":["subtree"],"description":"subtree=连同全部子节整读（默认只读本节）"},
  "query":{"type":"string","description":"只取含这些词的段落（可选）"},
  "offset":{"type":"integer","description":"分页起点（字，默认 0）"},
  "limit":{"type":"integer","description":"本次最多返回的字数（默认 12000，上限 30000）"}
},
"required":["section_id"]}`)
}
func (proposalReadTool) ReadOnly() bool { return true }
func (proposalReadTool) CompactDescription() string {
	return "读方案库小节(整节自动分页;query只取相关段落;scope=subtree连子节)"
}
func (proposalReadTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"section_id":{"type":"integer"},"scope":{"type":"string","enum":["subtree"]},"query":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"}},"required":["section_id"]}`)
}

func (proposalReadTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		SectionID int64  `json:"section_id"`
		Scope     string `json:"scope"`
		Query     string `json:"query"`
		Offset    int    `json:"offset"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.SectionID <= 0 {
		return "", fmt.Errorf("section_id 为必填项（proposal_search 命中里的小节 id，或标题树方括号里的数字）")
	}
	opt := proposal.ReadOpt{Offset: p.Offset, Limit: p.Limit, Query: p.Query}
	switch strings.ToLower(strings.TrimSpace(p.Scope)) {
	case "", "self":
	case "subtree":
		opt.Scope = "subtree"
	default:
		return "", fmt.Errorf("scope 只能是 subtree（连子节整读）或不填，收到 %q", p.Scope)
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	r, err := st.Read(p.SectionID, opt)
	if err != nil {
		if strings.Contains(err.Error(), proposal.ErrStoreUnreadable) {
			return "", fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
		}
		return "", err
	}
	return tool.WrapText(proposalReadText(r, opt)), nil
}

func proposalReadText(r *proposal.ReadResult, opt proposal.ReadOpt) string {
	sec := r.Section
	var b strings.Builder
	if len(r.Passages) > 0 {
		fmt.Fprintf(&b, "## %s（按「%s」取到 %d 处）\n\n", sec.Title, strings.TrimSpace(opt.Query), len(r.Passages))
	} else {
		fmt.Fprintf(&b, "## %s\n\n", sec.Title)
	}
	fmt.Fprintf(&b, "**方案**: %s%s　**路径**: %s\n", r.Proposal.Title, ingestingTag(r.Proposal.Status), sec.TitlePath)
	if len(r.Parts) > 1 {
		fmt.Fprintf(&b, "**范围**: 本节 %s，连子节 %d 个共 %s\n", fmtChars(sec.Chars), len(r.Parts)-1, fmtChars(r.Total))
	} else {
		fmt.Fprintf(&b, "**本节**: %s\n", fmtChars(r.Total))
	}
	if len(r.Children) > 0 {
		b.WriteString("**子节**: ")
		for j, c := range r.Children {
			if j >= 20 {
				fmt.Fprintf(&b, "…共 %d 个", len(r.Children))
				break
			}
			if j > 0 {
				b.WriteString("；")
			}
			fmt.Fprintf(&b, "[%d] %s（%s）", c.ID, c.Title, fmtChars(c.Chars))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(r.Passages) > 0 {
		for i, ps := range r.Passages {
			fmt.Fprintf(&b, "**段落 %d**", i+1)
			if ps.SectionID != sec.ID {
				fmt.Fprintf(&b, "（出自子节 %s）", ps.Title)
			}
			fmt.Fprintf(&b, "（第 %d 字起）\n%s\n\n", ps.Start+1, quoteLines(ps.Text))
		}
		b.WriteString("---\n只返回了含检索词的段落；本节")
		if len(r.Parts) > 1 {
			b.WriteString("（连子节）")
		}
		b.WriteString("其余内容未返回，去掉 query 可顺序分页读全文。")
		return b.String()
	}
	if strings.TrimSpace(r.Text) == "" {
		b.WriteString("（本节标题下没有正文")
		if len(r.Children) > 0 {
			fmt.Fprintf(&b, "，内容在 %d 个子节里，见上方子节清单", len(r.Children))
		}
		if opt.Offset > 0 && r.Total > 0 {
			fmt.Fprintf(&b, "；offset=%d 已到节尾（共 %s）", opt.Offset, fmtChars(r.Total))
		}
		b.WriteString("）\n")
	}
	if r.Text != "" {
		b.WriteString(r.Text + "\n\n")
	}
	b.WriteString("---\n")
	fmt.Fprintf(&b, "共 %s，本次返回第 %d~%d 字。", fmtChars(r.Total), r.Offset+1, r.Offset+len([]rune(r.Text)))
	if r.HasMore {
		fmt.Fprintf(&b, "未完，继续读：offset=%d；只看某方面内容可加 query。", r.Offset+len([]rune(r.Text)))
	} else if r.Total > proposal.DefaultReadLimit {
		b.WriteString("（已到节尾）")
	}
	return b.String()
}
