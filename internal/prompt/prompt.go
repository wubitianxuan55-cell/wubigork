package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
}

// NewEngine 创建模板引擎，从默认目录加载
func NewEngine(dir string) *Engine {
	e := &Engine{
		templates: make(map[string]*Template),
		dir:       dir,
	}
	e.loadAll()
	return e
}

func (e *Engine) loadAll() {
	files, err := filepath.Glob(filepath.Join(e.dir, "*.json"))
	if err != nil {
		return
	}
	for _, f := range files {
		t, err := loadTemplate(f)
		if err != nil {
			continue
		}
		e.templates[t.Name] = t
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
	var sb strings.Builder
	for key, def := range inputs {
		if def.Priority != priority {
			continue
		}
		if content, ok := contexts[key]; ok && content != "" {
			sb.WriteString(fmt.Sprintf("## %s\n", def.Label))
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
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("解析模板 %s 失败: %w", path, err)
	}
	return &t, nil
}
