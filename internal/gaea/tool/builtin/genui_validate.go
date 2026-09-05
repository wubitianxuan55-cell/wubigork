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
  "spec":{"type":"string","description":"The JSON text inside the genui fence (without the fence markers)."}
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
	var p struct {
		Spec string `json:"spec"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("genui_validate 参数无效：%v", err)
	}
	if strings.TrimSpace(p.Spec) == "" {
		return "", fmt.Errorf("genui_validate 需要 spec 参数（围栏内 JSON 文本）")
	}
	r := genui.ValidateSpec(p.Spec)
	if r.OK {
		return fmt.Sprintf("✅ OK（%d 节点）", r.Nodes), nil
	}
	return "❌ " + strings.Join(r.Errors, "；"), nil
}
