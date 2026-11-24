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
	// 报错分两层，开头写明是哪一层（2026-11-24 实弹：同一结构两次报不同错——一次是
	// 规格缺字段、一次是没传 spec——文案没区分，模型分不清该改调用还是改内容）：
	//   参数错误 = 调用方式不对（spec 没传 / 不是 JSON / 参数本身解析不了）；
	//   规格错误 = spec 内容不合规（按给出的路径与字段改 spec）。
	var p struct {
		Spec json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数错误（调用方式）：genui_validate 参数无效：%v", err)
	}
	raw := strings.TrimSpace(string(p.Spec))
	if raw == "" {
		return "", fmt.Errorf("参数错误（调用方式）：genui_validate 需要 spec 参数（围栏内 JSON，字符串或对象形态均可，如 {\"spec\":{…}} 或 {\"spec\":\"{…}\"}）")
	}
	if strings.HasPrefix(raw, `"`) {
		var s string
		if err := json.Unmarshal(p.Spec, &s); err != nil {
			return "", fmt.Errorf("参数错误（调用方式）：spec 字符串形态解析失败: %v", err)
		}
		raw = s
	}
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("参数错误（调用方式）：spec 是空串——把围栏内的 JSON 放进来")
	}
	r := genui.ValidateSpec(raw)
	if r.OK {
		return fmt.Sprintf("✅ OK（%d 节点）", r.Nodes), nil
	}
	return "❌ 规格错误（参数没问题，改 spec）：" + strings.Join(r.Errors, "；"), nil
}
