package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(saveTemplate{})
	tool.RegisterBuiltin(runTemplate{})
}

// --- save_template ---

type saveTemplate struct{}

func (saveTemplate) Name() string { return "save_template" }

func (saveTemplate) Description() string {
	return "保存多步骤工具链模板。将当前工作流步骤保存为 JSON 模板文件，后续可复用。模板包含步骤名称、工具调用和参数定义。"
}

func (saveTemplate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"模板名称（用于标识和加载）"},
  "description":{"type":"string","description":"模板描述"},
  "steps":{"type":"array","items":{"type":"object","properties":{"tool":{"type":"string","description":"工具名称"},"args":{"type":"object","description":"工具参数（支持 {{.param}} 模板变量）"}},"required":["tool"]},"description":"步骤列表"}
},
"required":["name","steps"]
}`)
}

func (saveTemplate) ReadOnly() bool { return false }

func (saveTemplate) CompactDescription() string     { return compactDesc["save_template"] }
func (saveTemplate) CompactSchema() json.RawMessage { return compactSchema["save_template"] }

func (saveTemplate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Steps       []struct {
			Tool string                 `json:"tool"`
			Args map[string]interface{} `json:"args,omitempty"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Name == "" || len(p.Steps) == 0 {
		return "", fmt.Errorf("name 和 steps 不能为空")
	}

	tplDir, err := getTemplateDir()
	if err != nil {
		return "", fmt.Errorf("获取模板目录失败: %w", err)
	}
	os.MkdirAll(tplDir, 0755)

	tplPath := filepath.Join(tplDir, sanitizeName(p.Name)+".json")
	data, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(tplPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入模板文件失败: %w", err)
	}

	return tool.WrapText(fmt.Sprintf("✅ 模板已保存: %s（%s，%d 个步骤）", tplPath, p.Description, len(p.Steps))), nil
}

// --- run_template ---

type runTemplate struct{}

func (runTemplate) Name() string { return "run_template" }

func (runTemplate) Description() string {
	return "加载并运行多步骤工具链模板。加载预定义的模板，替换参数 {{.param}} 后生成执行计划。模板由 save_template 保存。"
}

func (runTemplate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"模板名称"},
  "params":{"type":"object","description":"参数映射，替换模板中的 {{.param}} 变量","additionalProperties":{"type":"string"}}
},
"required":["name"]
}`)
}

func (runTemplate) ReadOnly() bool { return true }

func (runTemplate) CompactDescription() string     { return compactDesc["run_template"] }
func (runTemplate) CompactSchema() json.RawMessage { return compactSchema["run_template"] }

func (runTemplate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name   string            `json:"name"`
		Params map[string]string `json:"params,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.Name == "" {
		return "", fmt.Errorf("name 不能为空")
	}

	tplDir, err := getTemplateDir()
	if err != nil {
		return "", fmt.Errorf("获取模板目录失败: %w", err)
	}

	tplPath := filepath.Join(tplDir, sanitizeName(p.Name)+".json")
	data, err := os.ReadFile(tplPath)
	if err != nil {
		return "", fmt.Errorf("模板「%s」未找到（路径: %s）: %w", p.Name, tplPath, err)
	}

	var tpl struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Steps       []struct {
			Tool string                 `json:"tool"`
			Args map[string]interface{} `json:"args,omitempty"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &tpl); err != nil {
		return "", fmt.Errorf("模板文件格式无效: %w", err)
	}

	// 替换参数
	sessionDir := "{{.sessionDir}}" // 占位
	if p.Params == nil {
		p.Params = make(map[string]string)
	}
	p.Params["sessionDir"] = sessionDir

	var b strings.Builder
	fmt.Fprintf(&b, "📋 模板: %s\n", tpl.Name)
	if tpl.Description != "" {
		fmt.Fprintf(&b, "描述: %s\n", tpl.Description)
	}
	fmt.Fprintf(&b, "步骤数: %d\n\n", len(tpl.Steps))
	fmt.Fprintf(&b, "## 执行计划\n\n")
	fmt.Fprintf(&b, "| 步骤 | 工具 | 参数 |\n|------|------|------|\n")

	for i, step := range tpl.Steps {
		argsJSON, _ := json.Marshal(step.Args)
		argsStr := string(argsJSON)

		// 模板替换
		tmpl, err := template.New("arg").Parse(argsStr)
		if err == nil {
			var buf strings.Builder
			if tmpl.Execute(&buf, p.Params) == nil {
				argsStr = buf.String()
			}
		}

		// 截断过长的参数显示
		if len(argsStr) > 100 {
			argsStr = argsStr[:100] + "..."
		}
		fmt.Fprintf(&b, "| %d | `%s` | `%s` |\n", i+1, step.Tool, argsStr)
	}

	fmt.Fprintf(&b, "\n---\n*使用 task 工具逐步骤执行，或手动逐一调用。*\n")

	return tool.WrapText(b.String()), nil
}

func getTemplateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gaea", "templates"), nil
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}
