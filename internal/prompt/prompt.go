package prompt

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template RTCO Prompt 模板
type Template struct {
	Name        string              `json:"name"`
	System      string              `json:"system"`
	Task        string              `json:"task"`
	Inputs      map[string]InputDef `json:"input_sections"`
	Output      OutputDef           `json:"output"`
	Constraints ConstraintDef       `json:"constraints"`
	// ── t6 提示词工坊新增元数据（规格 进度计划/gaea-prompt-workshop-t6-20260916.md
	// §3.1）：全部 omitempty——旧模板 JSON 原样可读，零迁移脚本；与
	// internal/types.PromptTemplate 同名同 JSON tag（共享契约形状先例）。
	Version     string   `json:"version,omitempty"`     // 版本记录，磁盘模板 "1" 起步（Q5 裁决：只落库不做合并交互）
	Category    string   `json:"category,omitempty"`    // 工坊面板分组（worldview/character/outline/… 取值域见规格 §3.3）
	Description string   `json:"description,omitempty"` // 一句话中文说明（列表态展示，列表接口不回传正文的前提）
	Parameters  []string `json:"parameters,omitempty"`  // 声明 {{name}} 占位符名（promptstore.Validate 的校验契约）
}

// InputDef 输入节定义
type InputDef struct {
	Priority string `json:"priority"` // P0 / P1 / P2
	Label    string `json:"label"`
	// Order 同优先级内的稳定顺序（升序；未写为 0，同值按 key 字典序）。
	//
	// 规格来源：docs/distill/02-book-import.md §8.2 「RTCO 引擎的一个真实约束」——
	// 原实现按 Go map 遍历输入节，**输出顺序随机**，导致同模板两次渲染字节不同
	// （prompt 缓存无法命中，且长文本无法稳定置尾）。t2 反推/上下文编译都依赖确定性。
	Order int `json:"order,omitempty"`
}

// OutputDef 输出节定义
type OutputDef struct {
	Format      string `json:"format"`
	Description string `json:"description"`
}

// ConstraintDef 约束定义
type ConstraintDef struct {
	Must      []string `json:"must"`
	Forbidden []string `json:"forbidden"`
	Style     []string `json:"style,omitempty"`
}

// Engine RTCO 模板引擎
type Engine struct {
	templates map[string]*Template
	dir       string
	embedded  fs.FS // 可选：内置模板（单文件 exe 分发兜底）
	// override 全局覆盖解析钩子（t6 三级解析第一级：覆盖 → 磁盘 → embed，
	// 规格 §3.1/§4.3）。nil = 未启用，既有两级解析行为不变。
	//
	// 并发契约：本字段启动装配期写一次、之后只读——SetOverride 必须在任何
	// 并发 Get 之前调用（线 B app.go 两处装配点统一走 applyPromptOverrides），
	// 引擎侧不加锁；覆盖内容的失效/重载由线 B 缓存层在闭包内自理。
	override func(name string) *Template
}

// NewEngine 创建模板引擎，从默认目录加载
func NewEngine(dir string) *Engine {
	e := &Engine{
		templates: make(map[string]*Template),
		dir:       dir,
	}
	e.loadDir()
	return e
}

// NewEngineWithEmbedded 创建模板引擎：先加载内置模板（go:embed 兜底，单文件
// exe 部署时没有磁盘 prompts/ 也能工作），再叠加磁盘目录——同名磁盘模板优先，
// 方便开发期直接改 prompts/*.json 生效。
func NewEngineWithEmbedded(dir string, embedded fs.FS) *Engine {
	e := &Engine{
		templates: make(map[string]*Template),
		dir:       dir,
		embedded:  embedded,
	}
	if embedded != nil {
		e.loadEmbedded()
	}
	e.loadDir()
	return e
}

func (e *Engine) loadEmbedded() {
	if e.embedded == nil {
		return
	}
	files, err := fs.Glob(e.embedded, "prompts/*.json")
	if err != nil {
		slog.Warn("prompt: 枚举内置模板失败", "error", err)
		return
	}
	for _, f := range files {
		data, err := fs.ReadFile(e.embedded, f)
		if err != nil {
			slog.Warn("prompt: 读取内置模板失败", "file", f, "error", err)
			continue
		}
		t, err := parseTemplate(data, f)
		if err != nil {
			slog.Warn("prompt: 内置模板解析失败", "file", f, "error", err)
			continue
		}
		e.templates[t.Name] = t
	}
}

func (e *Engine) loadDir() {
	if e.dir == "" {
		return
	}
	files, err := filepath.Glob(filepath.Join(e.dir, "*.json"))
	if err != nil {
		slog.Warn("prompt: 枚举磁盘模板失败", "dir", e.dir, "error", err)
		return
	}
	for _, f := range files {
		t, err := loadTemplate(f)
		if err != nil {
			slog.Warn("prompt: 磁盘模板加载失败", "file", f, "error", err)
			continue
		}
		e.templates[t.Name] = t
	}
	if len(e.templates) > 0 {
		slog.Info("prompt: 模板引擎就绪", "dir", e.dir, "count", len(e.templates))
	}
}

// Get 获取指定名称的模板。t6 起为三级解析（规格 §1/§3.1）：
//  1. override 回调非 nil 且返回非 nil → 返回覆盖模板（promptstore 覆盖层，
//     isActive=false 的行不会到这一步——线 B 装配闭包只回传激活覆盖）；
//  2. 回调为 nil / 回调返回 nil → 回落内置 templates map（磁盘 JSON 同名
//     优先于 embed，加载期已在 NewEngine* 内叠加完成，本方法不重复判级）；
//  3. 都没有 → nil（调用方既有 nil 判定口径不变）。
func (e *Engine) Get(name string) *Template {
	if e.override != nil {
		if t := e.override(name); t != nil {
			return t
		}
	}
	return e.templates[name]
}

// SetOverride 设置全局覆盖解析钩子（规格 §3.1；线 B 启动装配调用）。
// 传 nil 清除覆盖层（回落内置表）。覆盖层不新增模板键（V1 只做既有键的
// 整模板覆盖，自建新键在观察池——规格 §8），故 Names() 仍是全量可选键集。
//
// 并发纪律：须在引擎进入并发服务（任何并发 Get）之前调用——启动装配期
// 一次性设置，之后 override 字段只读；运行中换钩子属未定义行为。
func (e *Engine) SetOverride(fn func(name string) *Template) {
	e.override = fn
}

// Names 返回全部内置模板名（templates map 的键，字典序稳定）。
// 工坊面板列表用（规格 §3.1）：覆盖层不产生新键，这份名单即面板可见全集；
// 排序稳定保证列表与状态文件 diff 不随 map 遍历抖动。
func (e *Engine) Names() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// BuildSystemPrompt 构建 system prompt
// 对于 RTCO 模板，返回 <role> + <task> + <constraints>
func (t *Template) BuildSystemPrompt(skillMD string) string {
	var sb strings.Builder

	// Role
	sb.WriteString(t.System)
	sb.WriteString("\n\n")

	// Task
	sb.WriteString("## 任务\n")
	sb.WriteString(t.Task)
	sb.WriteString("\n\n")

	// Output format
	sb.WriteString("## 输出要求\n")
	sb.WriteString(fmt.Sprintf("格式: %s。%s\n\n", t.Output.Format, t.Output.Description))

	// Constraints
	sb.WriteString("## 创作约束\n")
	if len(t.Constraints.Must) > 0 {
		sb.WriteString("✅ 必须：\n")
		for _, m := range t.Constraints.Must {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
	}
	if len(t.Constraints.Forbidden) > 0 {
		sb.WriteString("❌ 禁止：\n")
		for _, f := range t.Constraints.Forbidden {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
	}
	if len(t.Constraints.Style) > 0 {
		sb.WriteString("✍️ 风格：\n")
		for _, s := range t.Constraints.Style {
			sb.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}

	// Skill 注入
	if skillMD != "" {
		sb.WriteString("\n\n---\n")
		sb.WriteString("## 额外写作指导\n")
		sb.WriteString(skillMD)
	}

	return sb.String()
}

// BuildUserPrompt 构建 user prompt
// 将 context 注入模板的 input_sections
func (t *Template) BuildUserPrompt(contexts map[string]string) string {
	var sb strings.Builder

	// P0 first
	sb.WriteString(buildSection(t.Inputs, contexts, "P0"))

	// P1
	sb.WriteString(buildSection(t.Inputs, contexts, "P1"))

	// P2 last — may be truncated
	sb.WriteString(buildSection(t.Inputs, contexts, "P2"))

	return sb.String()
}

func buildSection(inputs map[string]InputDef, contexts map[string]string, priority string) string {
	type sectionItem struct {
		key string
		def InputDef
	}
	items := make([]sectionItem, 0, len(inputs))
	for key, def := range inputs {
		if def.Priority == priority {
			items = append(items, sectionItem{key: key, def: def})
		}
	}
	// 稳定序：Order 升序 → key 字典序兜底（未标 order 的旧模板行为可预期而非随机）。
	sort.Slice(items, func(i, j int) bool {
		if items[i].def.Order != items[j].def.Order {
			return items[i].def.Order < items[j].def.Order
		}
		return items[i].key < items[j].key
	})
	var sb strings.Builder
	for _, it := range items {
		if content, ok := contexts[it.key]; ok && content != "" {
			sb.WriteString(fmt.Sprintf("## %s\n", it.def.Label))
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

func loadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseTemplate(data, path)
}

func parseTemplate(data []byte, source string) (*Template, error) {
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("解析模板 %s 失败: %w", source, err)
	}
	return &t, nil
}

// RenderPlaceholders 渲染 {{name}} 双层花括号占位符（t6 Q1 裁决的新语法，
// 规格 §3.1）。
//
// 为什么只认双层花括号：gaea 模板正文大量携带 JSON 字面量花括号（output 示例
// 里满是 {"nodes":[…]}），MuMu 沿用 Python str.format() 语义导致字面 `{`/`}`
// 必须双写转义、裸 format 直接 KeyError 炸链路（docs/distill/06-prompt-workshop.md
// §3.2/§9.3 花括号教训）。gaea 反向选择：{{name}} 才是占位符，单层 {x} 永远是
// 字面量——JSON 示例零转义可写，旧 {word_count} 语法（substituteWordCount
// 精确替换）也不被误伤。
//
// 行为契约（规格 §3.1）：
//   - vars 命中：替换为变量值；同一名字多次出现全部替换；替换值不再二次扫描
//     （变量值含花括号也原样落盘）；
//   - vars 缺失（含 vars 为 nil/空）：保留原文 {{name}} 不动，并把变量名收进
//     返回切片（按首次出现去重、保序）——不抛错，调用方作警告展示
//     （MuMu 缺参 KeyError→500 教训：渲染缺参只降级不崩）；
//   - 单层 {x}（JSON 字面量/旧语法）原样不动；
//   - 未闭合的 {{（其后无 }}）：整段照抄收尾、不计入未解析名（花括号平衡
//     由 promptstore.Validate 在保存时拦截，渲染侧不重复报错）。
//
// 返回切片恒非 nil（无未解析变量时为空切片，JSON 序列化为 [] 而非 null）。
func RenderPlaceholders(s string, vars map[string]string) (string, []string) {
	var sb strings.Builder
	missing := []string{}
	seen := make(map[string]bool)
	for i := 0; i < len(s); {
		next := strings.Index(s[i:], "{{")
		if next < 0 {
			sb.WriteString(s[i:]) // 尾段无占位符，原文照抄
			break
		}
		start := i + next
		sb.WriteString(s[i:start]) // 占位符前导原文（含单层花括号）照抄
		closeRel := strings.Index(s[start+2:], "}}")
		if closeRel < 0 {
			sb.WriteString(s[start:]) // 未闭合 {{：保守不吞正文，整段照抄收尾
			break
		}
		end := start + 2 + closeRel // name 结束位；}} 占 end..end+2
		name := s[start+2 : end]
		if v, ok := vars[name]; ok {
			sb.WriteString(v)
		} else {
			sb.WriteString(s[start : end+2]) // 缺失变量：保留原文 {{name}}
			if !seen[name] {
				seen[name] = true // 同名多次缺失只记一次（警告无噪音）
				missing = append(missing, name)
			}
		}
		i = end + 2
	}
	return sb.String(), missing
}
