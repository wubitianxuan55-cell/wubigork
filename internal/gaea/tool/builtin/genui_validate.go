package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/genui"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// genuiValidate 是 ```genui 围栏的结构校验工具（只读、无副作用）。
// 只做结构检查（语法/白名单/预算/缺字段）；渲染器仍是最终权威，
// 与 frontend/src/genui/guard.ts 的行为差异以提示词声明为准。
type genuiValidate struct{}

func init() {
	tool.RegisterBuiltin(genuiValidate{})
}

func (genuiValidate) Name() string { return "genui_validate" }

func (genuiValidate) ReadOnly() bool { return true }

func (genuiValidate) Description() string {
	return "Validate a genui fence JSON spec before emitting it. Returns ✅ OK（N 节点）or ❌ with paths/fields to fix. Use for any complex spec (≥3 components or a table)."
}

func (genuiValidate) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "spec":{"description":"The genui fence JSON — either the JSON text (string) or the parsed object/array itself. Both forms are accepted."}
},
"required":["spec"]
}`)
}

func (genuiValidate) CompactDescription() string {
	return "Validate a genui fence JSON spec before emitting it."
}

func (genuiValidate) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"spec":{"type":"string"}},"required":["spec"]}`)
}

func (genuiValidate) Execute(_ context.Context, args json.RawMessage) (string, error) {
	// 2026-11-21 造价门实测 4.3：手册说「spec=围栏内 JSON」、旧实现只收字符串
	// 形态——传 {"spec":{...}} 直接 unmarshal 炸「of type string」。双形态兼容：
	// 字符串按原文、对象/数组按原样 JSON 文本送校验，两种手册口径都通。
	var p struct {
		Spec json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("genui_validate 参数无效：%v", err)
	}
	raw := strings.TrimSpace(string(p.Spec))
	if raw == "" {
		return "", fmt.Errorf("genui_validate 需要 spec 参数（围栏内 JSON，字符串或对象形态均可）")
	}
	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(p.Spec, &s); err != nil {
			return "", fmt.Errorf("genui_validate spec 字符串形态解析失败: %v", err)
		}
		raw = s
	}
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("genui_validate 需要 spec 参数（围栏内 JSON，字符串或对象形态均可）")
	}
	r := genui.ValidateSpec(raw)
	if r.OK {
		return fmt.Sprintf("✅ OK（%d 节点）", r.Nodes), nil
	}
	return "❌ " + strings.Join(r.Errors, "；"), nil
}
