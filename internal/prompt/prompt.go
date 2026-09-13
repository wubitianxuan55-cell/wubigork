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

// Get 获取指定名称的模板
func (e *Engine) Get(name string) *Template {
	return e.templates[name]
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
