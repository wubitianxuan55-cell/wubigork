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
//   - 体验开关（上游设置页 9 项）：波1 曾硬编码，波3 上收为 gaea 配置
//     [office_com] 段（internal/gaea/config OfficeComConfig，经
//     officeComSettings 单点读取）：suite=auto, silent=false, follow=true,
//     typing=true, render_width=1100 为上游 DEFAULTS 口径；card 恒 false
//     （迷你卡片由前端壳内状态条替代，波3 拍板不接设置）；showOnOpen=true
//     缺省不变（silent 模式下缺省改后台）；finalCheck=false（保存后自动总览
//     附图不做，模型可用 office_render {overview:true} 手动核对）。
//   - office_render 的成品图以 PNG 路径交付，模型经 vision 工具读取
//     （doc_render 同款管线——gaea agent 上下文是纯文本，无图片附件通道）。
//   - 波2 WPS 公式预转 math 腿已带（internal/office/omml 子集直转，fail-soft：
//     子集外公式不进 math map，helper 保留原文；Word 档走 Word 自带解析不受影响）。
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
	"github.com/gaea/gaea/internal/office/omml"
)

// ── 体验设置（波3：从 gaea 配置 [office_com] 段单点读取）──────────────────

// officeComPrefs 是 COM 档一次调用快照的体验开关（波1 顶部硬编码常量的
// 波3 上收——ga.cfg 的 [office_com] 段经 config.OfficeComNormalized 归一后
// 透传，helper 命令语义不变；类型名不与读取函数 officeComSettings 重名）。
type officeComPrefs struct {
	suite       string // auto | office | wps（进 officeCom.call 的 args["suite"]）
	silent      bool   // 静默模式：后台无窗口完成，回合结束 settle 自动保存关闭
	follow      bool   // 跟随：窗口滚到 AI 正在改的位置
	typing      bool   // 逐字写入（像打字）
	renderWidth int    // office_render 出图宽度像素（600-2000）
}

// officeComSettings 单点读取体验设置（所有消费点只经此处，禁止散读 ga.cfg）。
// ga.cfg==nil（测试态/启动早期）回上游 DEFAULTS 口径默认值；card 不在设置内——
// 迷你卡片由前端壳内状态条替代（2026-11-13 波3 拍板），open/edit 恒发 card=false
// （helper 侧 cardStarted 恒 false，settle 时的 card 收尾是容错 no-op）。
func officeComSettings() officeComPrefs {
	// 上游 DEFAULTS：suite=auto、silent=false、follow=true、typing=true、
	// render_width=1100（与 config.OfficeComNormalized 的兜底同源口径）。
	s := officeComPrefs{suite: "auto", follow: true, typing: true, renderWidth: 1100}
	ga.mu.Lock()
	cfg := ga.cfg
	ga.mu.Unlock()
	if cfg == nil {
		return s
	}
	oc := cfg.OfficeComNormalized()
	s.suite, s.silent, s.follow, s.typing, s.renderWidth =
		oc.Suite, oc.Silent, oc.Follow, oc.Typing, oc.RenderWidth
	return s
}

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
	// unseen 编辑后未看渲染图（波3.5 unseen 交付前检查闭环，上游 unseen Set
	// 同语义）：office_edit 成功置位、office_render 成功清除、office_save 时
	// 仍置位则保存后自动附整档总览+Final check 指令、close 清除。key=officeComKey。
	unseenMu sync.Mutex
	unseen   map[string]bool
	told     map[string]map[string]bool
	// ── 波3 settle 收尾记账（office_com_settle.go 消费）─────────────────
	// lastActive 是 open/edit/save/close 最近一次成功调用的时刻（读/渲染/
	// 状态不 stamp——它们不产生需要收尾的写面）；lastSettle 是上次 settle
	// 尝试（含失败）的时刻。maybeOfficeComSettle 只在「lastActive 晚于
	// lastSettle（本回合内有活动）或存在静默打开文档」时才发 helper 命令
	//（TurnDone 每回合都触发，无活动必须零 ping helper）。
	lastActive time.Time
	lastSettle time.Time
	// silentOpen 记录静默模式后台打开的文档（key=officeComKey，value=打开时
	// 的绝对路径）：settle 守卫与证据卡 Target 兜底用。save 后文档仍在后台
	// 打开不清（另存为时随 current 指向更新）；office_close 成功即清。
	silentOpen map[string]string
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

// officeComCall 注入 suite 后发命令（与上游 host 包装同位）。写面命令
// （open/edit/save/close）成功返回时 stamp 活动时刻——settle 守卫的记账点。
func (s *officeComState) call(ctx context.Context, cmd string, args map[string]any, timeout time.Duration) (json.RawMessage, error) {
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	if args == nil {
		args = map[string]any{}
	}
	args["suite"] = officeComSettings().suite
	raw, err := client.Call(ctx, cmd, args, timeout)
	if err == nil {
		switch cmd {
		case "open", "edit", "save", "close":
			s.mu.Lock()
			s.lastActive = time.Now()
			s.mu.Unlock()
		}
	}
	return raw, err
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

// officeGuide 首次使用发放操作清单（上游 guide() 同文；ppt 发基础版——
// 2026-11-13 波2 拍板 PPT 成品版式引擎取消，教学面只发基础 ops，见
// guides_ppt_basic.go；word/excel 仍全量上游对齐）。
func (s *officeComState) officeGuide(session, kind string) string {
	if !s.markTold(session, kind) {
		return ""
	}
	guide, ok := officeGuides[kind]
	if !ok {
		guide = officeGuidesPptBasic // ppt：基础 ops 清单（成品版式教学摘除）
	}
	return fmt.Sprintf("\n\n%s operations for office_edit (this list is given once; office_help {app: \"%s\"} gives it again):\n%s",
		comhelper.AppDisplayName(kind, ""), kind, guide)
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

// officeComDefaultDirName 交互文件的默认落点目录（2026-11-13 二轮实弹反馈：
// 交互文件固定在工作空间的一个位置，不要乱放——模型每次 office_open 随意给
// 路径/文件名，交互卡因此出现一堆不同路径的相同文件）。目录名显式中文（工作
// 区可见、用户好找），与 .gaea/work（过程件）/exports（交付物）互补：COM 实时
// 交互稿在完成前持续被编辑，放独立目录不污染 exports。
const officeComDefaultDirName = "交互文件"

// officeComNormalizeOpenPath 归一 office_open 的 path（纯函数，可测）：
//   - 空 → 默认目录 + "Office-<kind>-<时间戳>.<ext>"（同秒内重复 open 同 kind
//     =同路径=接着用，不再每次开新副本）；
//   - 纯文件名（无目录分隔、非绝对）→ 默认目录内归一（模型只给了名字=没想好
//     放哪，收编到固定位置）；
//   - 带目录/绝对路径 → 尊重原意，仅走工作区边界校验。
//
// 返回（工作区内绝对路径, 是否由本函数改写过）。
// 2026-11-13 二轮反馈修正：默认命名用 **title（任务主题语义名）**——时间戳
// 命名用户无法按主题查找；title 也缺省才回退 Office-<kind>-<日期>.<ext>。
// 同名已存在=接着用（office_open 附着语义），天然按主题续写。
func officeComNormalizeOpenPath(raw, kind, title string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	ext := map[string]string{"word": ".docx", "excel": ".xlsx", "ppt": ".pptx"}[kind]
	defaultIn := func(name string) (string, bool, error) {
		p, err := wspath.ResolveRelWithin(gaeaCwd(), filepath.Join(officeComDefaultDirName, name))
		if err != nil {
			return "", false, err
		}
		return p, true, nil
	}
	if raw == "" {
		// title 语义名优先（清洗文件名非法字符）；同主题文档已存在=附着续写。
		if t := officeComSafeFileName(title); t != "" {
			return defaultIn(t + ext)
		}
		return defaultIn(fmt.Sprintf("Office-%s-%s%s", kind, time.Now().Format("20060102"), ext))
	}
	if filepath.IsAbs(raw) || officeComAbsRe.MatchString(raw) {
		p, err := officeComResolvePath(raw)
		return p, false, err
	}
	if !strings.ContainsAny(raw, `/\`) {
		// 纯文件名：收编进默认目录（扩展名缺失时按 kind 补）。
		name := raw
		if filepath.Ext(name) == "" {
			name += ext
		}
		return defaultIn(name)
	}
	p, err := officeComResolvePath(raw)
	return p, false, err
}

// officeComSafeFileName 清洗语义名里的文件名非法字符（Windows 集合+空白收敛
// 为连字符；超长截 60），返回空串表示无可用语义名。
func officeComSafeFileName(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return ""
	}
	repl := strings.NewReplacer(
		`\`, "-", "/", "-", ":", "-", "*", "-", "?", "-", `"`, "-", "<", "-", ">", "-", "|", "-",
	)
	t = repl.Replace(t)
	t = strings.Join(strings.Fields(t), "-")
	if len([]rune(t)) > 60 {
		t = string([]rune(t)[:60])
	}
	return strings.Trim(t, "-.")
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
	"路径不存在的会新建。**落点约定**：path 省略时自动落到工作区「交互文件」目录并用 title 作语义文件名（如 交互文件/季度总结.docx）——title 请按任务主题起名，方便用户查找；" +
	"只给文件名（无目录）也会收编进「交互文件」目录；带目录的路径尊重原意。同一文档继续编辑请沿用首次 open 的路径（省略 doc 参数即沿用最近文档），" +
	"不要每次换文件名另开副本。先调本工具再使用其余 office_ 工具。编辑既有文档的另一种方式是 docx_apply 等无头工具（不打开窗口，适合无人值守批量）；" +
	"要让用户看到过程或文件正被占用时用本工具族。"

func (officeComOpenTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "path":{"type":"string","description":"文档路径（工作区相对或绝对）。路径不存在则新建该文件；省略则自动落到工作区「交互文件」目录并用 title 作文件名；只给文件名也会收编进该目录。同一文档继续编辑请沿用首次路径，不要换名另开副本"},
    "title":{"type":"string","description":"任务主题语义名（如 季度总结/设计方案）——path 省略时作为默认文件名（交互文件/季度总结.docx），方便用户按主题查找文件"},
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
		// 声明面与 Execute 同判据：path 归一（默认目录/纯文件名收编）后声明。
		kind0, kerr := officeComAppOf("", in.Path)
		if kerr != nil {
			return nil, kerr
		}
		var titleField struct {
			Title string `json:"title"`
		}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &titleField)
		}
		p, _, nerr := officeComNormalizeOpenPath(in.Path, kind0, titleField.Title)
		if nerr != nil {
			return nil, nerr
		}
		if _, statErr := os.Stat(p); statErr != nil {
			return []string{p}, nil // 新建场景：保存时落盘
		}
		return nil, nil // 已存在文件=附着打开（open 本身不写盘）
	}
	// path 缺省：归一生成默认落点（新建场景，保存时落盘）——与 Execute 同判据。
	// kind 需要从 args.app 取（缺省路径必填 app——Execute 同守卫）。
	var appField struct {
		App string `json:"app"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &appField)
	}
	kind0, kerr := officeComAppOf(appField.App, "")
	if kerr != nil {
		return nil, kerr
	}
	var titleField2 struct {
		Title string `json:"title"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &titleField2)
	}
	p, _, nerr := officeComNormalizeOpenPath("", kind0, titleField2.Title)
	if nerr != nil {
		return nil, nerr
	}
	return []string{p}, nil
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
		Title    string `json:"title"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	// 波3.5 二轮实弹：path 归一（默认目录/纯文件名收编/带目录尊重）——
	// 修「交互卡一堆不同路径的相同文件」。
	var abs string
	var normalized bool
	{
		var err error
		// kind 先行：缺省路径需要 kind 生成文件名。
		kind0, kerr := officeComAppOf(in.App, in.Path)
		if kerr != nil {
			return "", kerr
		}
		abs, normalized, err = officeComNormalizeOpenPath(in.Path, kind0, in.Title)
		if err != nil {
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
	settings := officeComSettings()
	// show 缺省 true（上游 showOnOpen 默认开：打开即见窗口是本档的核心体验）；
	// 静默模式下缺省改后台（「后台无窗口完成」是 silent 的本意），模型显式
	// show=true 才带回前台——schema 的 show 描述（静默场景下调出窗口用）同款
	// 语义。card 恒 false：迷你卡片由前端壳内状态条替代（波3 拍板，不接设置）。
	show := in.Show == nil || *in.Show
	if settings.silent {
		show = in.Show != nil && *in.Show
	}
	silent := settings.silent && !show
	openArgs := map[string]any{
		"app": kind, "show": show,
		"follow": settings.follow, "typing": settings.typing,
		"card": false, "silent": silent,
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
	// 静默记账：本次以 silent=true 后台打开的文档记入 silentOpen（settle 守卫
	// 与证据卡 Target 兜底用）。记账是 helper Hidden 集的超集——若文档本就开在
	// 用户可见窗口，OpenQuietly 原地附着不隐藏，settle 侧只动真隐藏文档，无害。
	if silent {
		if officeCom.silentOpen == nil {
			officeCom.silentOpen = map[string]string{}
		}
		officeCom.silentOpen[officeComKey(kind, docPath)] = docPath
	}
	officeCom.mu.Unlock()
	// 归一告知（模型知情=后续 edit/save 沿用同一路径，不再另起炉灶）。
	normalizedNote := ""
	if normalized {
		normalizedNote = fmt.Sprintf("\n落点已归一到工作区默认交互目录：%s（后续 edit/save 省略 doc 即沿用它；用户指定了具体位置时会尊重原意）。",
			officeComDefaultDirName+`\`+filepath.Base(docPath))
	}
	return comhelper.FormatOpened(doc) + normalizedNote + officeCom.officeGuide(session, kind), nil
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
		// 波3.5 实弹：Excel 大表显式 range 输出可达数百 KB（361KB 实锤），
		// 工具层主动截断+结构化指引（profile 概况/缩小 range/分块），不吃
		// agent 的 48KiB 硬截断（那个只有英文 elided、无操作指引）。
		const excelReadByteCap = 38 * 1024
		if len(text) > excelReadByteCap {
			cut := strings.LastIndex(text[:excelReadByteCap], "\n")
			if cut <= 0 {
				cut = excelReadByteCap
			}
			text = text[:cut] + "\n\n[输出超限已截断：不要整表读——用 profile:true 看每列概况后再定清洗策略，或缩小 range 分块读取（如 A1:F200 逐块）。]"
		}
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
	settings := officeComSettings()
	editArgs := map[string]any{
		"app": kind, "ops": json.RawMessage(argsOfOps(in.Ops)),
		"follow": settings.follow, "typing": settings.typing,
		// card 恒 false（迷你卡片由前端壳内状态条替代）；silent 透传设置——
		// helper 侧对静默打开（Hidden）或 silent 声明的文档按 quiet 模式编辑
		//（不滚动跟随、不打字、不弹卡）。
		"card": false, "silent": settings.silent,
	}
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		editArgs["doc"] = targetDoc
	}
	// 波2 WPS 公式预转 math 腿：把 ops 里的 $…$ / $$…$$ 预转成 OMML 随 edit
	// 下发（helper 仅 WPS 档按 d:/i: 键查表 InsertXML；Word 档用自带 LaTeX
	// 解析器，不受影响）。fail-soft：转不动的公式不进 math map，helper 保留
	// 原文不报错；本步任何失败（含 ops JSON 还原异常）都不阻断 edit 本身。
	var opsAny any
	if err := json.Unmarshal([]byte(argsOfOps(in.Ops)), &opsAny); err == nil {
		if math := omml.FormulasOf(opsAny, nil); len(math) > 0 {
			editArgs["math"] = math
		}
	}
	raw, err := officeCom.call(ctx, "edit", editArgs, officeComTimeoutEdit)
	if err != nil {
		return "", err
	}
	var result comhelper.EditResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("office helper edit 回执解析失败: %w", err)
	}
	// 波3.5 unseen：有成功修改 → 标记「改过没看过」，office_save 时触发交付前
	// 总览检查（office_render 成功会清除）。
	if len(result.Done) > 0 {
		officeCom.unseenMu.Lock()
		if officeCom.unseen == nil {
			officeCom.unseen = map[string]bool{}
		}
		officeCom.unseen[officeComKey(kind, targetDoc)] = true
		officeCom.unseenMu.Unlock()
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

// officeComRenderOverview 整档总览拼图（上游 overview() 直译）：Word/PPT 全部
// 页按 6 页一张拼图（带页码，最多 18 页=3 张），PNG 落 dir。render 工具的
// overview:true 与 office_save 的 unseen 交付前检查共用（波3.5）。
func officeComRenderOverview(ctx context.Context, kind, targetDoc, dir string) ([]string, []string, error) {
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
	total := 1
	for from := 1; from <= min(total, 18); from += 6 {
		out := filepath.Join(dir, fmt.Sprintf("sheet-%d.png", from))
		res, err := renderOnce(map[string]any{"sheet": true, "from": from, "to": from + 5}, out, 640, officeComTimeoutTile)
		if err != nil {
			return nil, nil, err
		}
		total = res.Total
		pages = append(pages, out)
		lines = append(lines, fmt.Sprintf("%s, tiled in one picture.", res.What))
	}
	if total > 18 {
		lines = append(lines, fmt.Sprintf("Only the first 18 of %d are shown.", total))
	}
	return lines, pages, nil
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
	// 出图宽度：显式 600-2000 优先，否则用设置缺省（officeComSettings 已归一到
	// [600,2000]，缺省 1100）。
	width := officeComSettings().renderWidth
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
		lines, pages, err = officeComRenderOverview(ctx, kind, targetDoc, dir)
		if err != nil {
			return "", err
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
	// 波3.5 unseen：看过渲染图（调 render 即算——上游同口径，以工具调用行为
	// 代理视觉检查）→ 清除标记，office_save 不再强制总览。
	if len(pages) > 0 {
		officeCom.unseenMu.Lock()
		delete(officeCom.unseen, officeComKey(kind, targetDoc))
		officeCom.unseenMu.Unlock()
	}
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
	// 静默记账维护：保存后文档仍在后台打开，不清账——另存为/未命名文档首次
	// 落盘时改键跟随新路径（value 刷成真实落盘路径，settle 证据卡 Target 用）；
	// 只有 office_close 成功才清（见下）。
	officeCom.mu.Lock()
	if _, ok := officeCom.silentOpen[officeComKey(kind, targetDoc)]; ok {
		delete(officeCom.silentOpen, officeComKey(kind, targetDoc))
		officeCom.silentOpen[officeComKey(kind, saved.Path)] = saved.Path
	}
	officeCom.mu.Unlock()
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
	// 波3.5 unseen 交付前检查（上游 finalCheck 同语义）：编辑后没看过渲染图
	// 就保存 → 自动附整档总览+Final check 指令，逼模型逐页核对再宣告完成。
	// 导出 PDF 不检查（那是导出不是交付核对）；excel 走区域渲染单图。
	officeCom.unseenMu.Lock()
	wasUnseen := officeCom.unseen[officeComKey(kind, targetDoc)]
	delete(officeCom.unseen, officeComKey(kind, targetDoc))
	officeCom.unseenMu.Unlock()
	if !wasUnseen || strings.EqualFold(filepath.Ext(absSavePath), ".pdf") {
		return text + tidy, nil
	}
	checkDir := filepath.Join(gaeaCwd(), ".gaea", "renders", "office",
		fmt.Sprintf("final-%d", time.Now().UnixMilli()))
	if err := os.MkdirAll(checkDir, 0o755); err != nil {
		return text + tidy, nil // 检查图失败不拦交付回执（保存已成功）
	}
	var shownLines, shownPages []string
	if kind == "excel" {
		out := filepath.Join(checkDir, "final.png")
		finalArgs := map[string]any{"app": kind, "out": out, "width": min(officeComSettings().renderWidth, 1000)}
		if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
			finalArgs["doc"] = targetDoc
		}
		fraw, ferr := officeCom.call(ctx, "render", finalArgs, officeComTimeoutRender)
		if ferr == nil {
			var res comhelper.RenderResult
			if json.Unmarshal(fraw, &res) == nil {
				shownPages = append(shownPages, out)
				shownLines = append(shownLines, fmt.Sprintf("%s, %g×%g.", res.What, res.Width, res.Height))
			}
		}
	} else {
		shownLines, shownPages, _ = officeComRenderOverview(ctx, kind, saved.Path, checkDir)
	}
	if len(shownPages) == 0 {
		return text + tidy, nil // 总览失败=诚实缺省，回执不带检查段
	}
	text += "\nFinal check: you edited after you last looked at the document, so here is the whole of it as it is now. " +
		strings.Join(shownLines, " ") + "\n用 vision 工具读取以下 PNG 逐页核对（内容出框/字体行距不一/公式成纯文本/表格过宽/题注脱位/模板残留/半空页——有问题用 office_render 细看、office_edit 修、再 office_save；都没问题就向用户宣告完成）:" +
		"\n" + strings.Join(shownPages, "\n")
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
	officeCom.unseenMu.Lock()
	delete(officeCom.unseen, key)
	officeCom.unseenMu.Unlock()
	delete(officeCom.silentOpen, key) // 静默文档被显式关闭：清收尾记账
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
	return "列出某个应用（word/excel/ppt）的 office_edit 全部操作与字段：Word（文字/公式/表格/图片/引用/样式/页面设置）、Excel（单元格/格式/工作表/图表/数据清洗）、PowerPoint（页面/文字/图片/SmartArt/形状/表格）。不确定操作名或字段时先调本工具。"
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
	guide, ok := officeGuides[kind]
	if !ok {
		guide = officeGuidesPptBasic // ppt：基础 ops 清单（同 officeGuide 口径）
	}
	return fmt.Sprintf("%s operations for office_edit:\n%s\n\nColours are \"#RRGGBB\"; align is left | center | right | justify; file paths are absolute.",
		comhelper.AppDisplayName(kind, ""), guide), nil
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
