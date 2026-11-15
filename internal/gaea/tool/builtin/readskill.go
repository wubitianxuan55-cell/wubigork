package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// readSkill lets the agent dynamically read a skill's frontmatter and body at
// runtime. This decouples the skill index (which lists available skills in the
// system prompt) from the full skill content, saving prompt tokens while still
// letting the agent load any skill on demand.
// Skill resolution is injected via SetResolver to avoid circular imports.
type readSkill struct {
	resolve func(name string) (string, error)
}

func (readSkill) Name() string { return "read_skill" }
func (readSkill) Description() string {
	return "读取指定技能(skill)的完整内容(前置元数据+正文)"
}
func (readSkill) ReadOnly() bool { return true }

func (readSkill) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "name": {"type": "string", "description": "技能名称(如 'explore', 'tdd', 'review')"}
  },
  "required": ["name"]
}`)
}

func (r readSkill) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("read_skill 参数解析失败: %w", err)
	}
	if in.Name == "" {
		return "", fmt.Errorf("read_skill requires a 'name' parameter")
	}
	if r.resolve == nil {
		return "", fmt.Errorf("read_skill resolver not configured")
	}
	content, err := r.resolve(in.Name)
	if err != nil {
		return "", fmt.Errorf("skill not found: %s", in.Name)
	}
	return content, nil
}

func (readSkill) CompactDescription() string     { return compactDesc["read_skill"] }
func (readSkill) CompactSchema() json.RawMessage { return compactSchema["read_skill"] }

func init() {
	tool.RegisterBuiltin(&readSkill{})
}

// WireReadSkillResolver injects the skill resolution callback. Call from boot.
//
// Deprecated（2026-11-15 刀2）：全局注入在双引擎（办公+图文）下后建者覆盖全局
// 实例，跨引擎串仓。boot 已改为经 NewReadSkillWithResolver 注册实例化同名件
// （Registry.Add 覆盖内置）；本函数保留给 CLI 旧调用面，boot 链路不再使用。
func WireReadSkillResolver(resolve func(name string) (string, error)) {
	// Update all registered readSkill instances.
	for _, t := range tool.Builtins() {
		if rs, ok := t.(*readSkill); ok {
			rs.resolve = resolve
		}
	}
}

// NewReadSkillWithResolver 构造自持 resolver 的 read_skill 实例（刀2）：boot
// 每次构建用它注册同名件，替换内置全局实例——每引擎技能面独立（filter 收口
// 在 resolver 闭包内完成）。
func NewReadSkillWithResolver(resolve func(name string) (string, error)) tool.Tool {
	return &readSkill{resolve: resolve}
}
