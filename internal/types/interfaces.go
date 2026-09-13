package types

import (
	"sort"
	"strconv"
	"strings"
)

// ── 共享接口契约（依赖倒置）────────────────────────────────────
//
// 本文件定义各能力域之间的**反向依赖边界**：消费方依赖这里的小接口，
// 提供方（promptstore / novelcontext / 伏笔 Lint 实现）在各自包内实现，
// 从而满足两条工程约束：
//   1. `internal/types` 必须保持叶子包（**不 import 任何 internal 内其它包**），
//      否则会形成 types ↔ project/analysis 的循环依赖；
//   2. 5 个实施域无需修改共享类型即可开工（acceptance 第 3 条）。
//
// 因此这里的接口刻意只用 Go 内建类型 + 本包类型做参数/返回值，
// 需要的结构体在本包内定义一次（Template / ContextSection / ResolvedTemplate ...）。

// ── ① 模板解析接口（t6 提示词工坊）──────────────────────────

// InputSectionDef 模板输入节定义（对齐 prompts/*.json 的 input_sections）。
//
// Order 是**新增字段**（omitempty，旧 JSON 缺省为 0，零迁移）：
// 渲染顺序必须先按 Priority（P0→P1→P2）再按 Order 升序、Order 相同按 Key 字典序，
// 保证同一模板两次渲染输出完全一致。旧实现直接遍历 map → 输出随机（t6 要求修复）。
type InputSectionDef struct {
	Priority string `json:"priority"` // P0 / P1 / P2
	Label    string `json:"label"`
	Order    int    `json:"order,omitempty"`
}

// OutputFormatDef 模板输出节定义。
type OutputFormatDef struct {
	Format      string `json:"format"`
	Description string `json:"description"`
}

// TemplateConstraintsDef 模板约束定义（RTCO 的 C）。
type TemplateConstraintsDef struct {
	Must      []string `json:"must,omitempty"`
	Forbidden []string `json:"forbidden,omitempty"`
	Style     []string `json:"style,omitempty"`
}

// PromptTemplate 提示词模板的**共享契约形状**。
//
// 前 6 个字段与 gaea 现有 `internal/prompt.Template` 完全一致（同名同 JSON tag），
// 后 3 个字段为新增且可选 → 现有 15 个模板 JSON 原样可读，无需迁移脚本。
//
// 实现方（internal/prompt）可继续持有自己的 Template 类型，
// 在实现 TemplateResolver/TemplateParser 时做字段级转换即可。
type PromptTemplate struct {
	Name        string                     `json:"name"`
	System      string                     `json:"system"`
	Task        string                     `json:"task"`
	Inputs      map[string]InputSectionDef `json:"input_sections"`
	Output      OutputFormatDef            `json:"output"`
	Constraints TemplateConstraintsDef     `json:"constraints"`
	// ── 新增可选字段（handoff §5-t6 迁移结论）──
	Version     string `json:"version,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

// TemplateSource 模板来源，取值越靠前优先级越高（t6 两级三源）。
type TemplateSource string

const (
	TemplateSourceProjectOverride TemplateSource = "project_override" // 项目级覆盖（最高）
	TemplateSourceGlobalOverride  TemplateSource = "global_override"  // 全局覆盖
	TemplateSourceDisk            TemplateSource = "disk"             // 磁盘 prompts/*.json
	TemplateSourceEmbedded        TemplateSource = "embedded"         // 内嵌 JSON（兜底）
	TemplateSourceBuiltin         TemplateSource = "builtin"          // 代码内置常量（最后兜底）
)

// TemplateSourceRank 给出来源的解析优先序（数值越小越优先）。
func TemplateSourceRank(s TemplateSource) int {
	switch s {
	case TemplateSourceProjectOverride:
		return 0
	case TemplateSourceGlobalOverride:
		return 1
	case TemplateSourceDisk:
		return 2
	case TemplateSourceEmbedded:
		return 3
	case TemplateSourceBuiltin:
		return 4
	}
	return 5
}

// ResolvedTemplate 解析结果：模板 + 来源（UI 需显示「项目/全局/磁盘/内置」徽标）。
type ResolvedTemplate struct {
	Template PromptTemplate `json:"template"`
	Source   TemplateSource `json:"source"`
	// Path 磁盘来源时的文件路径（其它来源为空）。
	Path string `json:"path,omitempty"`
}

// TemplateParser 按名称解析模板（含来源归属）。实现方：internal/promptstore。
type TemplateParser interface {
	// Resolve 按优先级解析模板：项目覆盖 → 全局覆盖 → 磁盘 JSON → 内嵌 JSON。
	// 全部缺失时返回 ok=false（不返回 error——「无模板」是正常状态，由调用方兜底）。
	Resolve(name string) (resolved ResolvedTemplate, ok bool)
	// List 列出全部可用模板（**不含正文的完整下发给前端**由上层裁剪）。
	List() []string
}

// TemplateResolver 把磁盘/内嵌模板叠加覆盖层解析出最终模板。
//
// 与 TemplateParser 的区别：Parser 面向「按名取用」，Resolver 面向「覆盖层合成」。
type TemplateResolver interface {
	// ResolveTemplate 返回最终生效模板；覆盖层缺失时退回 base。
	ResolveTemplate(name string) (PromptTemplate, TemplateSource, error)
}

// ValidationIssue 保存模板时的一条校验问题。
//
// ★ 存在的意义（handoff §3-12）：MuMu 保存时零校验，用户写错一个 `{变量}`
// 保存成功、下次生成整条链路 500 且无法自动恢复。gaea 必须在**保存时**报错。
type ValidationIssue struct {
	Field    string `json:"field"`    // template_content / parameters / name
	Message  string `json:"message"`  // 中文说明
	Severity string `json:"severity"` // error / warning
	// Placeholder 相关问题时给出涉事占位符名。
	Placeholder string `json:"placeholder,omitempty"`
}

// TemplateValidator 保存期校验。
//
// 校验依据必须是**真实扫描占位符**，不得使用模板的 `parameters` 声明字段
// （handoff §3-13：MuMu 36 模板中 11 个声明与实际不符，7 个有运行时风险）。
type TemplateValidator interface {
	// ValidateTemplate 校验模板；返回的 issues 中有 severity=error 即拒绝保存。
	ValidateTemplate(t PromptTemplate) []ValidationIssue
	// ExtractPlaceholders 扫描模板文本中的 `{{name}}` 占位符（返回去重后的名字）。
	ExtractPlaceholders(text string) []string
}

// TemplateWarning 渲染期的一条降级告警（**不是错误**）。
//
// ★ 语义（handoff §3-12）：缺失变量必须「保留原文 + 返回 warning」，不得抛错。
// MuMu 的 str.format() 缺失键直接抛错，整条链路 500。
type TemplateWarning struct {
	Placeholder string `json:"placeholder"`
	Message     string `json:"message"`
	KeptLiteral bool   `json:"kept_literal"` // true = 已按原样保留 `{{name}}`
}

// TemplateRenderResult 渲染结果。
type TemplateRenderResult struct {
	Text     string            `json:"text"`
	Warnings []TemplateWarning `json:"warnings,omitempty"`
	// Used 实际替换成功的占位符名。
	Used []string `json:"used,omitempty"`
}

// TemplateEngine 模板引擎的完整能力（三个子接口的组合）。
//
// 消费方按需只依赖子接口（ISP）：只读模板的依赖 TemplateParser，
// 保存路径的依赖 TemplateValidator，渲染路径的依赖 TemplateRenderer。
type TemplateEngine interface {
	TemplateParser
	TemplateValidator
	TemplateRenderer
}

// TemplateRenderer 占位符渲染（语法 `{{name}}`）。
type TemplateRenderer interface {
	// RenderTemplate 渲染占位符；缺失变量保留原文并记入 Warnings，不抛错。
	RenderTemplate(text string, vars map[string]string) TemplateRenderResult
}

// PlaceholderToken 占位符语法约定（唯一口径，供 promptstore 与调用点共用）。
const (
	PlaceholderOpen  = "{{"
	PlaceholderClose = "}}"
)

// ── ② 上下文组装接口（t3 长程一致性）────────────────────────

// SectionPriority 上下文节的优先级（复用既有 P0/P1/P2 口径）。
type SectionPriority string

const (
	SectionP0 SectionPriority = "P0" // 必须保留
	SectionP1 SectionPriority = "P1" // 重要
	SectionP2 SectionPriority = "P2" // 可裁剪
)

// ContextSection 上下文的一个可组装节。
type ContextSection struct {
	ID       string          `json:"id"`
	Title    string          `json:"title,omitempty"`
	Content  string          `json:"content"`
	Priority SectionPriority `json:"priority"`
	// Order 同优先级内的稳定顺序（升序；相同则按 ID 字典序）。
	Order int `json:"order,omitempty"`
	// MaxRunes >0 时该节单独受字符预算约束（rune 计）。
	MaxRunes int `json:"max_runes,omitempty"`
}

// SectionSortKey 返回 (priorityRank, order, id) 三元排序键。
func SectionSortKey(s ContextSection) (int, int, string) {
	rank := 3
	switch s.Priority {
	case SectionP0:
		rank = 0
	case SectionP1:
		rank = 1
	case SectionP2:
		rank = 2
	}
	return rank, s.Order, s.ID
}

// SortContextSections 就地把节排成确定性顺序（P0→P1→P2，再 Order，再 ID）。
//
// 这是「渲染顺序稳定」在上下文层的对应物：禁止任何调用方依赖 map 迭代顺序。
func SortContextSections(sections []ContextSection) {
	sort.SliceStable(sections, func(i, j int) bool {
		ri, oi, ii := SectionSortKey(sections[i])
		rj, oj, ij := SectionSortKey(sections[j])
		if ri != rj {
			return ri < rj
		}
		if oi != oj {
			return oi < oj
		}
		return ii < ij
	})
}

// ContextProvider 上下文节的提供方（各域实现：伏笔、角色、记忆、摘要…）。
//
// 依赖倒置的目的：新增一个上下文来源只需实现本接口并注册，不必改组装器。
type ContextProvider interface {
	// Name 提供方标识（用于去重与调试）。
	Name() string
	// Provide 返回本提供方在当前章号下要注入的节（可为空切片）。
	Provide(chapterNum int) []ContextSection
}

// ContextBuilder 把多个提供方的节组装成一段受预算约束的文本。
type ContextBuilder interface {
	Register(p ContextProvider)
	Build(chapterNum int) []ContextSection
}

// ContextAssembler 组装并渲染上下文文本。
//
// ★ gaea 的既有优势必须保留（handoff §4-1）：`novelcontext.Render(maxRunes)` 的
// **全局字符/token 预算**是 gaea 独有能力（MuMu 全后端零预算）。本接口只是把
// 「预算」提到契约层，供各域复用——不得实现成无预算的直连。
// 既有实现 novelcontext.SceneBible.Render(maxRunes) 天生满足本接口，无需改动。
type ContextAssembler interface {
	Render(maxRunes int) string
}

// DefaultContextMaxRunes 默认全局预算，与 novelcontext.DefaultMaxRunes 保持一致。
//
// 这里再声明一份是为了让契约层可独立使用；**两处数值必须相等**，
// 各域不得再定义第三套阈值（同 handoff §4-1 的「不另立阈值」精神）。
const DefaultContextMaxRunes = 2000

// ── ③ 伏笔 Lint 接口（t3/t4）────────────────────────────────

// LintCode 伏笔体检项代码（前 5 个为 gaea 既有能力，必须保留；后 3 个为本域新增）。
type LintCode string

const (
	// ── gaea 现有 5 类（升级是扩展，不是替换；handoff §4-6）──
	LintOrdering       LintCode = "ordering"        // 回收章早于埋设章
	LintStatusMismatch LintCode = "status-mismatch" // 状态与回收章不一致
	LintDangling       LintCode = "dangling"        // 指向不存在的章节
	LintStale          LintCode = "stale"           // 悬置超过 foreshadowStaleAfterChapters
	LintDuplicate      LintCode = "duplicate"       // 归一描述重复
	// ── 本域新增 ──
	LintOverduePredicted LintCode = "overdue-predicted" // 计划回收章 vs 实际进度，超期预测
	LintAmbiguousMatch   LintCode = "ambiguous-match"   // 引用失效但内容匹配到多个候选
	LintUnresolvable     LintCode = "unresolvable"      // 引用失效且禁止回落 → 无法回收
)

// LintSeverity 体检项严重度。
type LintSeverity string

const (
	SeverityHigh   LintSeverity = "high"
	SeverityMedium LintSeverity = "medium"
	SeverityLow    LintSeverity = "low"
)

// ForeshadowLintFinding 一条体检发现。
//
// JSON tag 使用与既有 `internal/app.ForeshadowLintFinding` **完全一致**的
// camelCase 命名（foreshadowId/itemDesc），保证前端不必改动即可消费扩展后的报告。
type ForeshadowLintFinding struct {
	Code         LintCode     `json:"code"`
	Severity     LintSeverity `json:"severity"`
	ForeshadowID string       `json:"foreshadowId"`
	ItemDesc     string       `json:"itemDesc"`
	Message      string       `json:"message"`
	Chapter      string       `json:"chapter,omitempty"`
	// Kind 与 Severity 同义，供按「问题种类」分组的消费方使用（新增，omitempty）。
	Kind string `json:"kind,omitempty"`
}

// ForeshadowLintReport 体检报告。
type ForeshadowLintReport struct {
	TotalChapters int                     `json:"totalChapters"`
	Items         int                     `json:"items"`
	Planted       int                     `json:"planted"`
	Hinted        int                     `json:"hinted"`
	Revealed      int                     `json:"revealed"`
	LongTerm      int                     `json:"longTerm"`
	Findings      []ForeshadowLintFinding `json:"findings"`
	// PredictedOverdue 超期预测计数（新增能力，omitempty 保持旧前端兼容）。
	PredictedOverdue int `json:"predictedOverdue,omitempty"`
}

// ForeshadowLintOptions Lint 入参。
type ForeshadowLintOptions struct {
	// TotalChapters 已写章节总数（调用方用 countWrittenChapters 口径提供）。
	TotalChapters int
}

// ForeshadowLinter 伏笔体检契约。
//
// 实现方：`internal/app`（在既有 LintForeshadows 基础上扩展，不是另起一套）。
// 之所以不把 `*project.Manager` 写进签名：types 必须保持叶子包，
// 数据读取由实现方自行完成，本接口只约定「输入条目 → 输出发现」。
type ForeshadowLinter interface {
	LintForeshadowItems(items []Foreshadow, opts ForeshadowLintOptions) ForeshadowLintReport
}

// ForeshadowIndexPath / ForeshadowRefPhrase 之类展示用常量放在实现域，不在此处。

// ── ④ 章节文件名 ↔ 章号（跨域共用口径）──────────────────────

// ChapterNumOf 从章节文件名派生章号："001.md"→1、"001a.md"→1；空/无数字→0。
//
// 权威口径（handoff §2-2）：**不设章号冗余字段**，一律由文件名派生。
// 这是 internal/app.chapterNumOf 的契约层镜像（同一算法的单一来源声明），
// 各域不得再写第三份实现。
func ChapterNumOf(filename string) int {
	n := strings.IndexFunc(filename, func(r rune) bool { return r < '0' || r > '9' })
	digits := filename
	if n >= 0 {
		digits = filename[:n]
	}
	if digits == "" {
		return 0
	}
	v, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return v
}
