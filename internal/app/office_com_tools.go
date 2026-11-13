package app

// office_com_tools.go — dsh-office COM 实时协同工具族（波1，蒸馏自
// github.com/copylee711/dsh-office v1.2.0，调研论证见
// docs/gaea-dsh-office-distill-research-2026-11.md）。
//
// 定位：与无头 OOXML 手术管线（office_generate/docx_apply/pptx_apply/
// xlsx_apply）**互补不替代**——本族让文档在用户正开着的 Word/Excel/
// PowerPoint/WPS 窗口里被实时编辑（改动即时上屏、可协同、格式不走样、
// 文件被占用也能改）；无头档继续负责无人值守/批量/无 Office 环境。
//
// 与上游工具层的差异（协议层偏离见 comhelper/doc.go）：
//   - path 参数接受工作区相对路径（上游仅绝对路径）——agent 更顺手，
//     解析成绝对路径再发 helper。
//   - 波1 硬编码体验开关（上游设置页 9 项）：suite=auto, follow=true,
//     typing=true, card=false（迷你卡片挂拍板池第 3 项）, silent=false,
//     showOnOpen=true, renderWidth=1100, finalCheck=false（保存后自动总览
//     附图挂波3；模型可用 office_render {overview:true} 手动核对）。
//   - office_render 的成品图以 PNG 路径交付，模型经 vision 工具读取
//     （doc_render 同款管线——gaea agent 上下文是纯文本，无图片附件通道）。
//   - 波1 不带 WPS 公式预转 math 腿（MS Word 公式完整可用）。
//   - office_save 落 Journal 证据卡（appendOfficeEvidence 单点，与四类写
//     同账；基线快照不适用=诚实缺省）；未做 unseen 追踪（挂波3）。
//
// 装配：gaeaExtraTools 条件注册（officeComAvailable——本机无 Office/WPS
// 则整族不出现，无头管线不受影响；上游 installedApps 同思路）。三处纪律：
// RegistryMap 契约行 + spacetags 静态表 + docs/gaea-tool-registry-map 重导。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
	comhelper "github.com/gaea/gaea/internal/office/comhelper"
)

// ── 波1 硬编码体验开关（接 gaea 设置挂波3）────────────────────────────────
const (
	officeComSuite       = "auto"
	officeComFollow      = true
	officeComTyping      = true
	officeComCard        = false // 迷你卡片挂拍板池第 3 项（波二）
	officeComSilent      = false
	officeComRenderWidth = 1100
)

// 命令超时（对齐上游 tools.ts 各调用点）。
const (
	officeComTimeoutOpen   = 80 * time.Second
	officeComTimeoutRead   = 50 * time.Second
	officeComTimeoutEdit   = 270 * time.Second
	officeComTimeoutRender = 50 * time.Second
	officeComTimeoutTile   = 110 * time.Second
	officeComTimeoutSave   = 110 * time.Second
	officeComTimeoutStatus = 30 * time.Second
	officeComTimeoutClose  = 50 * time.Second
)

// officeComCaller seam：测试注入假 client（协议测试跑真 Client，工具测试跑桩）。
type officeComCaller interface {
	Call(ctx context.Context, cmd string, args map[string]any, timeout time.Duration) (json.RawMessage, error)
}

type officeComState struct {
	mu     sync.Mutex
	caller officeComCaller
	// current 当前文档（最后 open/read/edit/save 的目标）：doc 缺省时复用
	//（上游 current 语义）。app 为 kind，doc 为绝对路径或窗口名。
	currentApp string
	currentDoc string
	// own 我们自己 open/创建的文档 key（app:文件名小写）——office_close 只许
	// 关这些，用户本来开着的文档一律拒绝（上游 own Set 同语义）。
	own map[string]bool
	// told 每会话已发放过操作清单的应用（按需帮助：不进 schema 省上下文）。
	toldMu sync.Mutex
	told   map[string]map[string]bool
}

var (
	officeCom officeComState
	// officeComNewCaller seam：测试注入假 client（协议测试跑真 Client，
	// 工具测试跑桩）。
	officeComNewCaller = func() officeComCaller { return comhelper.NewClient() }
)

// officeComClient 惰性单例（首次 office_* 调用才编译并拉起 helper）。
func (s *officeComState) client() (officeComCaller, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.caller == nil {
		s.caller = officeComNewCaller()
	}
	if s.caller == nil {
		return nil, fmt.Errorf("office helper 初始化失败")
	}
	return s.caller, nil
}

// officeComCall 注入 suite 后发命令（与上游 host 包装同位）。
func (s *officeComState) call(ctx context.Context, cmd string, args map[string]any, timeout time.Duration) (json.RawMessage, error) {
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	if args == nil {
		args = map[string]any{}
	}
	args["suite"] = officeComSuite
	return client.Call(ctx, cmd, args, timeout)
}

// officeComKey 文档去重键（app:文件名小写——路径与窗口名共用文件名收口）。
func officeComKey(app, doc string) string {
	name := doc
	if i := strings.LastIndexAny(doc, `/\`); i >= 0 {
		name = doc[i+1:]
	}
	return app + ":" + strings.ToLower(name)
}

// officeComMarkTold 记录会话已发放清单；返回是否首次。
func (s *officeComState) markTold(session, kind string) bool {
	s.toldMu.Lock()
	defer s.toldMu.Unlock()
	if s.told == nil {
		s.told = map[string]map[string]bool{}
	}
	set, ok := s.told[session]
	if !ok {
		set = map[string]bool{}
		s.told[session] = set
	}
	if set[kind] {
		return false
	}
	set[kind] = true
	return true
}

// officeGuide 首次使用发放操作清单（上游 guide() 同文）。
func (s *officeComState) officeGuide(session, kind string) string {
	if !s.markTold(session, kind) {
		return ""
	}
	return fmt.Sprintf("\n\n%s operations for office_edit (this list is given once; office_help {app: \"%s\"} gives it again):\n%s",
		comhelper.AppDisplayName(kind, ""), kind, officeGuides[kind])
}

// ── 目标解析（上游 appOf/target 直译）────────────────────────────────────

var officeComExts = map[string]string{
	".docx": "word", ".doc": "word", ".docm": "word", ".rtf": "word", ".dotx": "word",
	".xlsx": "excel", ".xls": "excel", ".xlsm": "excel", ".xlsb": "excel", ".csv": "excel",
	".pptx": "ppt", ".ppt": "ppt", ".pptm": "ppt",
}

var officeComAbsRe = regexp.MustCompile(`^[a-zA-Z]:[\\/]`)

// officeComResolvePath 工作区相对路径 → 绝对路径（相对=ResolveRelWithin；
// 绝对=必须在工作区内）。波1 口径：office 实时协同档只支持工作区内文档
// （写边界由 WritePaths+AccessPolicy 强制，读边界在此单点收口——helper 能
// 附着任意绝对路径，不在这一层挡住就绕过了读策略）。helper 只吃绝对路径。
func officeComResolvePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少文件路径")
	}
	var p string
	if filepath.IsAbs(raw) || officeComAbsRe.MatchString(raw) {
		p = filepath.Clean(raw)
		if !wspath.Within(gaeaCwd(), p) {
			return "", fmt.Errorf("office 实时协同档目前只支持工作区内文档：%s 在工作区外。把文件复制进工作区后重试", p)
		}
		return p, nil
	}
	resolved, err := wspath.ResolveRelWithin(gaeaCwd(), raw)
	if err != nil {
		return "", fmt.Errorf("非法工作区相对路径: %s", raw)
	}
	return resolved, nil
}

// officeComAppOf 指定 app 或从文件扩展名推断 kind。
func officeComAppOf(app, path string) (string, error) {
	if app != "" {
		if app != "word" && app != "excel" && app != "ppt" {
			return "", fmt.Errorf("app 必须是 word / excel / ppt")
		}
		return app, nil
	}
	ext := strings.ToLower(filepath.Ext(path))
	if kind, ok := officeComExts[ext]; ok {
		return kind, nil
	}
	return "", fmt.Errorf("请指明应用：传 app（word/excel/ppt），或用带扩展名的文件名（doc）")
}

// officeComTarget 解析本次调用针对哪个文档：显式 app/doc 优先，否则复用
// current；都没有则报错（上游 target() 同语义）。返回 kind 与绝对路径/窗口名。
func (s *officeComState) target(app, doc string) (kind, target string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if doc == "" && app == "" {
		if s.currentDoc == "" {
			return "", "", fmt.Errorf("还没有目标文档：先调 office_open，或传 doc")
		}
		return s.currentApp, s.currentDoc, nil
	}
	if doc == "" {
		if s.currentApp == app && s.currentDoc != "" {
			return app, s.currentDoc, nil
		}
		return app, "", fmt.Errorf("只有 doc 没有路径时需要 app：当前没有 %s 文档", app)
	}
	abs, rerr := officeComResolvePath(doc)
	if rerr != nil {
		return "", "", rerr
	}
	kind, aerr := officeComAppOf(app, abs)
	if aerr != nil {
		return "", "", aerr
	}
	s.currentApp, s.currentDoc = kind, abs
	return kind, abs, nil
}

// ── office_open ──────────────────────────────────────────────────────────

type officeComOpenTool struct{}

func (officeComOpenTool) Name() string        { return "office_open" }
func (officeComOpenTool) ReadOnly() bool      { return false }
func (officeComOpenTool) Description() string { return officeComOpenDescription }

const officeComOpenDescription = "在 Word / Excel / PowerPoint（或 WPS，本机装了什么就用什么）里打开文档让用户实时看到 AI 干活：" +
	"改动逐步出现在窗口里、格式由 Office 自己排版不走样，用户可以同时在里面编辑。已打开的文档直接接着用（文件被占用也能改）；" +
	"路径不存在的会新建。先调本工具再使用其余 office_ 工具。编辑既有文档的另一种方式是 docx_apply 等无头工具（不打开窗口，适合无人值守批量）；" +
	"要让用户看到过程或文件正被占用时用本工具族。"

func (officeComOpenTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "path":{"type":"string","description":"文档路径（工作区相对或绝对）。路径不存在则新建该文件；省略则新建未保存文档（此时 app 必填）"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在无 path 时必填，或 doc 无扩展名时用于挑选应用"},
    "template":{"type":"string","description":"模板/样例文档路径：复制到 path（path 必须不存在）后打开副本，原模板不动。用户给了模板要沿用其样式时用"},
    "show":{"type":"boolean","description":"true=把文档窗口带到前台（静默场景下调出窗口用）"}
  }
}`)
}

// WritePaths 声明持久写：模板复制必然写盘；路径不存在的新建在保存时落盘。
// 已存在文件的普通打开不声明（不写盘）。
// ReadTargeter 不实现：新建场景（path 不存在）没有可读目标，实现会把
// 「新建文档」在访问检查阶段硬拒绝（ReadTarget false=denied）。读边界由
// officeComResolvePath 单点收口。
func (t officeComOpenTool) WritePaths(args json.RawMessage) ([]string, error) {
	var in struct {
		Path     string `json:"path"`
		Template string `json:"template"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, fmt.Errorf("参数无效: %w", err)
		}
	}
	if strings.TrimSpace(in.Template) != "" {
		if strings.TrimSpace(in.Path) == "" {
			return nil, fmt.Errorf("用 template 时必须给 path（副本落点）")
		}
		p, err := officeComResolvePath(in.Path)
		if err != nil {
			return nil, err
		}
		return []string{p}, nil
	}
	if strings.TrimSpace(in.Path) != "" {
		p, err := officeComResolvePath(in.Path)
		if err != nil {
			return nil, err
		}
		if _, statErr := os.Stat(p); statErr != nil {
			return []string{p}, nil // 新建场景：保存时落盘
		}
	}
	return nil, nil
}

// ContextualTool：会话级操作清单发放。
func (t officeComOpenTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	return t.execute(ctx, tc.SessionID, args)
}
func (t officeComOpenTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.execute(ctx, "", args)
}

func (t officeComOpenTool) execute(ctx context.Context, session string, args json.RawMessage) (string, error) {
	var in struct {
		Path     string `json:"path"`
		App      string `json:"app"`
		Template string `json:"template"`
		Show     *bool  `json:"show"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	var abs string
	if strings.TrimSpace(in.Path) != "" {
		var err error
		if abs, err = officeComResolvePath(in.Path); err != nil {
			return "", err
		}
	}
	kind, err := officeComAppOf(in.App, abs)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Template) != "" {
		if abs == "" {
			return "", fmt.Errorf("用 template 时必须给 path（副本落点）")
		}
		tplAbs, err := officeComResolvePath(in.Template)
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(filepath.Ext(tplAbs), filepath.Ext(abs)) {
			return "", fmt.Errorf("template 与 path 必须是同类型文件（扩展名一致）")
		}
		if _, statErr := os.Stat(abs); statErr == nil {
			return "", fmt.Errorf("%s 已存在：换一个不存在的 path，模板会复制到那里", abs)
		}
		if _, statErr := os.Stat(tplAbs); statErr != nil {
			return "", fmt.Errorf("读取模板失败: %w", statErr)
		}
		data, err := os.ReadFile(tplAbs)
		if err != nil {
			return "", fmt.Errorf("读取模板失败: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return "", fmt.Errorf("创建目标目录失败: %w", err)
		}
		if err := os.WriteFile(abs, data, 0o644); err != nil {
			return "", fmt.Errorf("复制模板失败: %w", err)
		}
	}
	openArgs := map[string]any{
		// show 缺省 true（上游 showOnOpen 默认开：打开即见窗口是本档的核心体验）。
		"app": kind, "show": in.Show == nil || *in.Show,
		"follow": officeComFollow, "typing": officeComTyping,
		"card": officeComCard, "silent": officeComSilent,
	}
	if abs != "" {
		openArgs["path"] = abs
	}
	raw, err := officeCom.call(ctx, "open", openArgs, officeComTimeoutOpen)
	if err != nil {
		return "", err
	}
	var doc comhelper.DocInfo
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("office helper open 回执解析失败: %w", err)
	}
	officeCom.mu.Lock()
	docPath := doc.Name
	if doc.Path != nil {
		docPath = *doc.Path
	}
	officeCom.currentApp, officeCom.currentDoc = kind, docPath
	if doc.How != "attached" && doc.Path != nil {
		if officeCom.own == nil {
			officeCom.own = map[string]bool{}
		}
		officeCom.own[officeComKey(kind, *doc.Path)] = true
	}
	officeCom.mu.Unlock()
	return comhelper.FormatOpened(doc) + officeCom.officeGuide(session, kind), nil
}

// ── office_status ────────────────────────────────────────────────────────

type officeComStatusTool struct{}

func (officeComStatusTool) Name() string   { return "office_status" }
func (officeComStatusTool) ReadOnly() bool { return true }
func (officeComStatusTool) Description() string {
	return "列出 Word / Excel / PowerPoint（含 WPS）现在打开了哪些文档、哪些有未保存修改、应用是否安装/运行。"
}

func (officeComStatusTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t officeComStatusTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	raw, err := officeCom.call(ctx, "status", nil, officeComTimeoutStatus)
	if err != nil {
		return "", err
	}
	var apps []comhelper.AppStatus
	if err := json.Unmarshal(raw, &apps); err != nil {
		return "", fmt.Errorf("office helper status 回执解析失败: %w", err)
	}
	return comhelper.FormatStatus(apps), nil
}

// ── office_read ──────────────────────────────────────────────────────────

type officeComReadTool struct{}

func (officeComReadTool) Name() string   { return "office_read" }
func (officeComReadTool) ReadOnly() bool { return true }

func (officeComReadTool) Description() string {
	return "把打开中的文档按现状读成文本（包括用户刚改过的内容）：Word=带编号段落/标题级别/表格（含原生脚注）；Excel=工作表+区域单元格值与公式（>60 行长表自动给每列概况）；PowerPoint=每页形状（名称/类型/位置/文字）。读到的就是 Office 窗口里此刻的样子。"
}

func (officeComReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名（如 report.docx）。缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在要读另一应用的活动文档、或 doc 无扩展名时需要"},
    "from":{"type":"integer","description":"Word：起始段落号（缺省 1）"},
    "count":{"type":"integer","description":"Word：读取段落数（缺省 120，最大 400）"},
    "full":{"type":"boolean","description":"Word/PowerPoint：输出完整文本（缺省每段/形状只取前 100/200 字符）"},
    "footnotes":{"type":"boolean","description":"Word：读取全部原生脚注（含正文范围之外的）；full:true 取完整脚注文本"},
    "table":{"type":"integer","description":"Word：读取第 N 个表格的单元格"},
    "sheet":{"type":"string","description":"Excel：工作表名"},
    "range":{"type":"string","description":"Excel：A1 风格区域，如 A1:F40（缺省活动表已用区域）"},
    "profile":{"type":"boolean","description":"Excel：给出区域内每列概况（取值种类/范围/可疑项）——表太长读不全时用它"},
    "slide":{"type":"integer","description":"PowerPoint：只读这一页"}
  }
}`)
}

func (t officeComReadTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	return t.execute(ctx, tc.SessionID, args)
}
func (t officeComReadTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.execute(ctx, "", args)
}

func (t officeComReadTool) execute(ctx context.Context, session string, args json.RawMessage) (string, error) {
	var in map[string]any
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	doc, _ := in["doc"].(string)
	appNameArg, _ := in["app"].(string)
	kind, targetDoc, err := officeCom.target(appNameArg, doc)
	if err != nil {
		return "", err
	}
	// doc/app 已消费（target 归一），其余字段原样透传 helper。
	delete(in, "doc")
	delete(in, "app")
	in["app"] = kind
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		in["doc"] = targetDoc
	}
	raw, err := officeCom.call(ctx, "read", in, officeComTimeoutRead)
	if err != nil {
		return "", err
	}
	var text string
	switch kind {
	case "word":
		var read comhelper.WordRead
		if err := json.Unmarshal(raw, &read); err != nil {
			return "", fmt.Errorf("office helper read 回执解析失败: %w", err)
		}
		text = comhelper.FormatWord(read)
	case "excel":
		var read comhelper.ExcelRead
		if err := json.Unmarshal(raw, &read); err != nil {
			return "", fmt.Errorf("office helper read 回执解析失败: %w", err)
		}
		text = comhelper.FormatExcel(read)
	default:
		var read comhelper.PptRead
		if err := json.Unmarshal(raw, &read); err != nil {
			return "", fmt.Errorf("office helper read 回执解析失败: %w", err)
		}
		text = comhelper.FormatPpt(read)
	}
	return text + officeCom.officeGuide(session, kind), nil
}

// ── office_edit ──────────────────────────────────────────────────────────

type officeComEditTool struct{}

func (officeComEditTool) Name() string   { return "office_edit" }
func (officeComEditTool) ReadOnly() bool { return false }

func (officeComEditTool) Description() string {
	return "在打开中的 Word / Excel / PowerPoint / WPS 里按顺序执行一批修改（实时出现在窗口里，不落盘；保存用 office_save）。" +
		"每个 op 形如 {\"op\": 操作名, ...字段}；某一步失败即停下并报告已完成哪些。Word 的一批修改在用户侧是一次 Ctrl+Z 可整体撤销。" +
		"各应用的操作清单随首次 office_open/office_read 发放，office_help 可再取——写 op 前先读清单，不要猜操作名/字段。" +
		"文本里的 $...$ / $$...$$ LaTeX 在 Word 会成为原生公式。这是办公文档的实时协同编辑档；无人值守批量仍用 docx_apply/xlsx_apply 等无头工具。"
}

func (officeComEditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["ops"],
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名。缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在要编辑另一应用的活动文档、或 doc 无扩展名时需要"},
    "ops":{"type":"array","items":{"type":"object"},"description":"按序执行的操作，如 [{\"op\":\"set_text\",\"para\":3,\"expect\":\"原开头\",\"text\":\"新文本\"}]；单次最多 200 条"}
  }
}`)
}

func (t officeComEditTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	return t.execute(ctx, tc.SessionID, args)
}
func (t officeComEditTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.execute(ctx, "", args)
}

func (t officeComEditTool) execute(ctx context.Context, session string, args json.RawMessage) (string, error) {
	var in struct {
		Doc string `json:"doc"`
		App string `json:"app"`
		Ops []json.RawMessage
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if len(in.Ops) == 0 {
		return "", fmt.Errorf("ops 必须是非空数组")
	}
	if len(in.Ops) > 200 {
		return "", fmt.Errorf("单次最多 200 条操作")
	}
	kind, targetDoc, err := officeCom.target(in.App, in.Doc)
	if err != nil {
		return "", err
	}
	editArgs := map[string]any{
		"app": kind, "ops": json.RawMessage(argsOfOps(in.Ops)),
		"follow": officeComFollow, "typing": officeComTyping,
		"card": officeComCard, "silent": officeComSilent,
	}
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		editArgs["doc"] = targetDoc
	}
	raw, err := officeCom.call(ctx, "edit", editArgs, officeComTimeoutEdit)
	if err != nil {
		return "", err
	}
	var result comhelper.EditResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("office helper edit 回执解析失败: %w", err)
	}
	return comhelper.FormatEdit(result) + officeCom.officeGuide(session, kind), nil
}

// argsOfOps 把 ops 原样还原成 JSON 数组文本（透传给 helper，不做字段级校验——
// 未知名段会被 helper 的 Bag.Unread 点名，不悄悄失效）。
func argsOfOps(ops []json.RawMessage) string {
	buf := strings.Builder{}
	buf.WriteByte('[')
	for i, op := range ops {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(op)
	}
	buf.WriteByte(']')
	return buf.String()
}

// ── office_render ────────────────────────────────────────────────────────

type officeComRenderTool struct{}

func (officeComRenderTool) Name() string   { return "office_render" }
func (officeComRenderTool) ReadOnly() bool { return true }

func (officeComRenderTool) Description() string {
	return "把打开中文档的真实排版渲染成 PNG（按 Office 自己的排版，含未保存修改，窗口在后台也能出图）：Word 的一页、PowerPoint 的一页（to 可一次多页）、Excel 的区域或单张图表；overview:true 把全部页拼图（每张 6 页，最多 18 页）。" +
		"返回 PNG 路径，用 vision 工具逐张读取核对。改完的每页都要看：溢出文本框、过宽表格、字体行距不一、公式成了纯文本、题注没跟图表在一起、模板残留文字、半空页……office_read 看不到这些。Excel 区域渲染经剪贴板取得（渲染期间剪贴板文字保留，图片/文件等内容会丢）。"
}

func (officeComRenderTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名。缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在 doc 无扩展名时需要"},
    "page":{"type":"integer","description":"Word：页码（缺省 1）"},
    "slide":{"type":"integer","description":"PowerPoint：页码（overview 时可省）"},
    "to":{"type":"integer","description":"Word/PowerPoint：结束页（与起始页一次最多 4 页）"},
    "overview":{"type":"boolean","description":"true=全部页拼图（Word/PowerPoint，每张 6 页带页码，最多 18 页）；看完总览再用 page/slide 细看"},
    "sheet":{"type":"string","description":"Excel：工作表名"},
    "range":{"type":"string","description":"Excel：要渲染的区域（覆盖其上的图表会一起入图）"},
    "chart":{"type":"string","description":"Excel：单张图表名（代替 range）"},
    "width":{"type":"integer","description":"出图宽度像素（600-2000，缺省 1100；越大越清晰越费 vision token）"}
  }
}`)
}

func (t officeComRenderTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Doc      string `json:"doc"`
		App      string `json:"app"`
		Page     *int   `json:"page"`
		Slide    *int   `json:"slide"`
		To       *int   `json:"to"`
		Overview bool   `json:"overview"`
		Sheet    string `json:"sheet"`
		Range    string `json:"range"`
		Chart    string `json:"chart"`
		Width    int    `json:"width"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	kind, targetDoc, err := officeCom.target(in.App, in.Doc)
	if err != nil {
		return "", err
	}
	width := officeComRenderWidth
	if in.Width >= 600 && in.Width <= 2000 {
		width = in.Width
	}
	// 渲染产物目录（doc_render 同款 .gaea/renders 约定，vision 读取需要路径存活）。
	stem := "current"
	if targetDoc != "" {
		stem = strings.TrimSuffix(filepath.Base(targetDoc), filepath.Ext(targetDoc))
	}
	stem = regexp.MustCompile(`[^\w一-龥]+`).ReplaceAllString(stem, "-")
	dir := filepath.Join(gaeaCwd(), ".gaea", "renders", "office",
		fmt.Sprintf("%s-%d", stem, time.Now().UnixMilli()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建渲染目录失败: %w", err)
	}

	baseArgs := map[string]any{"app": kind}
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		baseArgs["doc"] = targetDoc
	}

	renderOnce := func(extra map[string]any, out string, w int, timeout time.Duration) (comhelper.RenderResult, error) {
		callArgs := make(map[string]any, len(baseArgs)+len(extra)+2)
		for k, v := range baseArgs {
			callArgs[k] = v
		}
		for k, v := range extra {
			callArgs[k] = v
		}
		callArgs["out"] = out
		callArgs["width"] = w
		raw, err := officeCom.call(ctx, "render", callArgs, timeout)
		if err != nil {
			return comhelper.RenderResult{}, err
		}
		var res comhelper.RenderResult
		if err := json.Unmarshal(raw, &res); err != nil {
			return comhelper.RenderResult{}, fmt.Errorf("office helper render 回执解析失败: %w", err)
		}
		return res, nil
	}

	var lines []string
	var pages []string
	if in.Overview && kind != "excel" {
		total := 1
		for from := 1; from <= min(total, 18); from += 6 {
			out := filepath.Join(dir, fmt.Sprintf("sheet-%d.png", from))
			res, err := renderOnce(map[string]any{"sheet": true, "from": from, "to": from + 5}, out, 640, officeComTimeoutTile)
			if err != nil {
				return "", err
			}
			total = res.Total
			pages = append(pages, out)
			lines = append(lines, fmt.Sprintf("%s, tiled in one picture.", res.What))
		}
		if total > 18 {
			lines = append(lines, fmt.Sprintf("Only the first 18 of %d are shown.", total))
		}
	} else {
		switch kind {
		case "word":
			first := 1
			if in.Page != nil {
				first = *in.Page
			}
			last := first
			if in.To != nil {
				last = min(max(*in.To, first), first+3)
			}
			for number := first; number <= last; number++ {
				out := filepath.Join(dir, fmt.Sprintf("page-%d.png", number))
				res, err := renderOnce(map[string]any{"page": number}, out, width, officeComTimeoutRender)
				if err != nil {
					if len(pages) == 0 {
						return "", err
					}
					lines = append(lines, err.Error())
					break
				}
				pages = append(pages, out)
				lines = append(lines, fmt.Sprintf("%s, %g×%g.", res.What, res.Width, res.Height))
			}
		case "ppt":
			if in.Slide == nil {
				return "", fmt.Errorf("PowerPoint 需要 slide（或 overview:true）")
			}
			first := *in.Slide
			last := first
			if in.To != nil {
				last = min(max(*in.To, first), first+3)
			}
			for number := first; number <= last; number++ {
				out := filepath.Join(dir, fmt.Sprintf("slide-%d.png", number))
				res, err := renderOnce(map[string]any{"slide": number}, out, width, officeComTimeoutRender)
				if err != nil {
					if len(pages) == 0 {
						return "", err
					}
					break
				}
				pages = append(pages, out)
				lines = append(lines, fmt.Sprintf("%s, %g×%g.", res.What, res.Width, res.Height))
			}
		default: // excel
			extra := map[string]any{}
			if in.Sheet != "" {
				extra["sheet"] = in.Sheet
			}
			if in.Range != "" {
				extra["range"] = in.Range
			}
			if in.Chart != "" {
				extra["chart"] = in.Chart
			}
			out := filepath.Join(dir, "range.png")
			res, err := renderOnce(extra, out, min(width, 1000), officeComTimeoutRender)
			if err != nil {
				return "", err
			}
			pages = append(pages, out)
			lines = append(lines, fmt.Sprintf("%s, %g×%g.", res.What, res.Width, res.Height))
		}
	}
	payload, _ := json.Marshal(map[string]any{
		"ok":     true,
		"doc":    targetDoc,
		"what":   lines,
		"pages":  pages,
		"count":  len(pages),
		"vision": "用 vision 工具读取以上 PNG（image_path 逐张）",
	})
	return string(payload), nil
}

// ── office_save ──────────────────────────────────────────────────────────

type officeComSaveTool struct{}

func (officeComSaveTool) Name() string   { return "office_save" }
func (officeComSaveTool) ReadOnly() bool { return false }

func (officeComSaveTool) Description() string {
	return "保存打开中的文档到磁盘；给出 path 时另存为（扩展名 .pdf 导出 PDF 且原文档不动，其他扩展名=文档从此就是那个文件）。" +
		"AI 的修改只在 office_save 时写入文件，不会替用户关闭文档。保存 Word 文档前会自动做一次排版补空（fill_gaps：图表移页留下的页尾空白把正文提上来填满）。" +
		"保存后建议 office_render {overview:true} 总览核对再交付。落盘后自动登记证据卡（可复核）。"
}

func (officeComSaveTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名。缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在 doc 无扩展名时需要"},
    "path":{"type":"string","description":"另存为路径（工作区相对或绝对）；省略=原地保存"}
  }
}`)
}

// WritePaths 声明落盘路径：显式 path 优先；doc 缺省时用当前文档路径
// （从未保存过的新文档必须给 path）。checkToolAccess 在 Execute 前调用，
// 会话态此刻可用（与 Execute 同序）。
func (t officeComSaveTool) WritePaths(args json.RawMessage) ([]string, error) {
	var in struct {
		Doc  string `json:"doc"`
		Path string `json:"path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, fmt.Errorf("参数无效: %w", err)
		}
	}
	if strings.TrimSpace(in.Path) != "" {
		p, err := officeComResolvePath(in.Path)
		if err != nil {
			return nil, err
		}
		return []string{p}, nil
	}
	if strings.TrimSpace(in.Doc) != "" {
		p, err := officeComResolvePath(in.Doc)
		if err != nil {
			return nil, err
		}
		if _, statErr := os.Stat(p); statErr == nil {
			return []string{p}, nil
		}
		return nil, fmt.Errorf("文档从未保存过：office_save 必须给 path（另存为落点）")
	}
	officeCom.mu.Lock()
	current := officeCom.currentDoc
	officeCom.mu.Unlock()
	if current != "" && officeComAbsRe.MatchString(current) {
		return []string{current}, nil
	}
	return nil, fmt.Errorf("缺少落盘路径：传 path（或先 office_open 过的文档）")
}

func (t officeComSaveTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Doc  string `json:"doc"`
		App  string `json:"app"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	var absSavePath string
	if strings.TrimSpace(in.Path) != "" {
		var err error
		if absSavePath, err = officeComResolvePath(in.Path); err != nil {
			return "", err
		}
	}
	kind, targetDoc, err := officeCom.target(in.App, in.Doc)
	if err != nil {
		return "", err
	}
	// Word 保存前的排版补空（上游 fill_gaps pre-save pass；失败不拦保存；
	// 导出 PDF 跳过——那是导出不是整理）。
	tidy := ""
	if kind == "word" && (absSavePath == "" || !strings.EqualFold(filepath.Ext(absSavePath), ".pdf")) {
		gapArgs := map[string]any{"app": kind, "ops": json.RawMessage(`[{"op":"fill_gaps"}]`)}
		if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
			gapArgs["doc"] = targetDoc
		}
		if raw, gerr := officeCom.call(ctx, "edit", gapArgs, 100*time.Second); gerr == nil {
			var pass comhelper.EditResult
			if json.Unmarshal(raw, &pass) == nil && len(pass.Done) > 0 && !regexp.MustCompile(`^no page`).MatchString(pass.Done[0]) {
				tidy = " Before saving, " + pass.Done[0] + "."
			}
		}
	}
	saveArgs := map[string]any{"app": kind}
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		saveArgs["doc"] = targetDoc
	}
	if absSavePath != "" {
		saveArgs["path"] = absSavePath
	}
	raw, err := officeCom.call(ctx, "save", saveArgs, officeComTimeoutSave)
	if err != nil {
		return "", err
	}
	var saved struct {
		Path  string `json:"path"`
		Bytes int64  `json:"bytes"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return "", fmt.Errorf("office helper save 回执解析失败: %w", err)
	}
	moved := absSavePath != "" && !strings.EqualFold(filepath.Ext(absSavePath), ".pdf")
	if moved {
		officeCom.mu.Lock()
		if officeCom.own != nil && targetDoc != "" {
			if officeCom.own[officeComKey(kind, targetDoc)] {
				officeCom.own[officeComKey(kind, saved.Path)] = true
			}
		}
		officeCom.currentApp, officeCom.currentDoc = kind, saved.Path
		officeCom.mu.Unlock()
	}
	// Journal 证据卡：COM 实时编辑没有基线快照（诚实缺省，不冒充通过）。
	if rel := officeComJournalRel(saved.Path); rel != "" {
		appendOfficeEvidence("office_save", rel,
			"COM 实时编辑（Office/WPS 窗口内，无基线快照）",
			fmt.Sprintf("saved %s (%d bytes)%s", saved.Path, saved.Bytes, tidy),
			"", "")
	}
	text := fmt.Sprintf("Saved %s (%d bytes).", saved.Path, saved.Bytes)
	if moved {
		text += " The open document is now this file."
	}
	return text + tidy, nil
}

// officeComJournalRel 绝对路径 → Journal 证据卡 target（工作区内=相对路径；
// 区外=绝对路径如实落账）。
func officeComJournalRel(abs string) string {
	if abs == "" {
		return ""
	}
	if rel, err := wspath.ResolveRelWithin(gaeaCwd(), abs); err == nil {
		return rel
	}
	// ResolveRelWithin 只处理相对入参；绝对路径手工求工作区内相对。
	cwd := gaeaCwd()
	if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return abs
}

// ── office_close ─────────────────────────────────────────────────────────

type officeComCloseTool struct{}

func (officeComCloseTool) Name() string   { return "office_close" }
func (officeComCloseTool) ReadOnly() bool { return false }

func (officeComCloseTool) Description() string {
	return "关闭 AI 自己用 office_open 打开或新建的文档（临时稿/用完的素材）。用户本来开着的文档归用户，一律拒绝关闭。" +
		"有未保存修改时必须明确传 save:true（保存并关闭）或 save:false（放弃修改关闭）。"
}

func (officeComCloseTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名。缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在 doc 无扩展名时需要"},
    "save":{"type":"boolean","description":"true=保存并关闭；false=放弃未保存修改关闭；有未保存修改时必填"}
  }
}`)
}

// WritePaths save=true 时声明落盘路径（关闭内含保存=持久写）。
func (t officeComCloseTool) WritePaths(args json.RawMessage) ([]string, error) {
	var in struct {
		Doc  string `json:"doc"`
		Save *bool  `json:"save"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, fmt.Errorf("参数无效: %w", err)
		}
	}
	if in.Save == nil || !*in.Save {
		return nil, nil
	}
	if strings.TrimSpace(in.Doc) != "" {
		p, err := officeComResolvePath(in.Doc)
		if err != nil {
			return nil, err
		}
		return []string{p}, nil
	}
	officeCom.mu.Lock()
	current := officeCom.currentDoc
	officeCom.mu.Unlock()
	if current != "" && officeComAbsRe.MatchString(current) {
		return []string{current}, nil
	}
	return nil, fmt.Errorf("save:true 需要可落盘的路径：传 doc（文档路径）")
}

func (t officeComCloseTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Doc  string `json:"doc"`
		App  string `json:"app"`
		Save *bool  `json:"save"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	kind, targetDoc, err := officeCom.target(in.App, in.Doc)
	if err != nil {
		return "", err
	}
	officeCom.mu.Lock()
	key := officeComKey(kind, targetDoc)
	isOwn := officeCom.own[key]
	officeCom.mu.Unlock()
	if !isOwn {
		return "", fmt.Errorf("该文档在 AI 接手前就已打开（或不是 AI 打开的）：关闭与否交给用户")
	}
	if in.Save == nil {
		// 上游同语义：未保存修改不表态就拒绝，逼模型明确 save。
		raw, serr := officeCom.call(ctx, "status", nil, officeComTimeoutStatus)
		if serr == nil {
			var apps []comhelper.AppStatus
			if json.Unmarshal(raw, &apps) == nil {
				name := targetDoc
				if i := strings.LastIndexAny(targetDoc, `/\`); i >= 0 {
					name = targetDoc[i+1:]
				}
				for _, app := range apps {
					for _, doc := range app.Documents {
						docName := doc.Name
						if doc.Path != nil {
							docName = *doc.Path
						}
						if app.App == kind && strings.EqualFold(filepath.Base(docName), name) && !doc.Saved {
							return "", fmt.Errorf("该文档有未保存修改：传 save:true 保留或 save:false 丢弃")
						}
					}
				}
			}
		}
	}
	closeArgs := map[string]any{"app": kind, "save": in.Save != nil && *in.Save}
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		closeArgs["doc"] = targetDoc
	}
	if _, err := officeCom.call(ctx, "close", closeArgs, officeComTimeoutClose); err != nil {
		return "", err
	}
	officeCom.mu.Lock()
	delete(officeCom.own, key)
	if officeCom.currentApp == kind && officeCom.currentDoc == targetDoc {
		officeCom.currentApp, officeCom.currentDoc = "", ""
	}
	officeCom.mu.Unlock()
	name := targetDoc
	if i := strings.LastIndexAny(targetDoc, `/\`); i >= 0 {
		name = targetDoc[i+1:]
	}
	return fmt.Sprintf("Closed %s.", name), nil
}

// ── office_help ──────────────────────────────────────────────────────────

type officeComHelpTool struct{}

func (officeComHelpTool) Name() string   { return "office_help" }
func (officeComHelpTool) ReadOnly() bool { return true }

func (officeComHelpTool) Description() string {
	return "列出某个应用（word/excel/ppt）的 office_edit 全部操作与字段：Word（文字/公式/表格/图片/引用/样式/页面设置）、Excel（单元格/格式/工作表/图表/数据清洗）、PowerPoint（成品版式/SmartArt/形状/画布）。不确定操作名或字段时先调本工具。"
}

func (officeComHelpTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["app"],
  "properties":{
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"哪个应用"}
  }
}`)
}

func (t officeComHelpTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	var in struct {
		App string `json:"app"`
	}
	if err := json.Unmarshal(args, &in); err != nil || in.App == "" {
		return "", fmt.Errorf("app 必须是 word / excel / ppt")
	}
	kind, err := officeComAppOf(in.App, "")
	if err != nil {
		return "", err
	}
	officeCom.markTold(tc.SessionID, kind)
	return fmt.Sprintf("%s operations for office_edit:\n%s\n\nColours are \"#RRGGBB\"; align is left | center | right | justify; file paths are absolute.",
		comhelper.AppDisplayName(kind, ""), officeGuides[kind]), nil
}
func (t officeComHelpTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.ExecuteWithContext(ctx, tool.ToolContext{}, args)
}

// ── 可用性探测（build 标签文件分实现）─────────────────────────────────────

var (
	officeComProbeOnce sync.Once
	officeComProbeOK   bool
)

// officeComAvailable 本机是否有可用的 Office 套件（MS Office 或 WPS）。
// 无则整族不注册（gaeaExtraTools 条件装配），无头管线照旧。
func officeComAvailable() bool {
	officeComProbeOnce.Do(func() {
		officeComProbeOK = officeComProbeSuite()
	})
	return officeComProbeOK
}
