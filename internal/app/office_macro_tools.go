package app

// office_macro_tools.go — 受控宏逃生舱（office_macro）。
//
// 定位（docs/gaea-lingxi-optimize-plan-2026-11.md P0-2，A 档）：
// 灵犀的 `wps_*_script` 让模型在高层工具覆盖不到时写脚本兜底，代价是任意代码
// 执行。本工具取其「兜底」语义但**不给任意代码**：宏是一张注册表，每个宏把
// 少量参数确定性地展开成既有 office_edit 的 ops，再走同一条 COM 执行路径。
//
// 收益：
//   - 模型不必自己拼多字段 op（拼错是当前最常见的失败模式）；
//   - 宏集合即能力边界，模型可 list 自查，不会以为「做不到」而放弃；
//   - 零新增执行面：不引入脚本解释器、不碰 VBA 工程信任。
//
// 扩宏只需在 officeMacros 里加一条，Build 返回既有 op 即可（禁止新语义）。

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// officeMacroSpec 是一条受控宏：把参数确定性展开成 office_edit 的 ops。
type officeMacroSpec struct {
	Name  string
	App   string // word | excel | ppt | any
	Desc  string
	Args  string // 参数说明（给模型看）
	Build func(args map[string]any) ([]map[string]any, error)
}

// officeMacros 是宏注册表（有序输出按名排序，保证提示词/工具输出稳定）。
var officeMacros = map[string]officeMacroSpec{
	"word_replace_all": {
		Name: "word_replace_all",
		App:  "word",
		Desc: "把 Word 全文里的某段文字全部替换为另一段（等价一次 replace_text，避免手拼 ops）",
		Args: `{"find":"被替换的原文","replace":"替换后的文字"}`,
		Build: func(args map[string]any) ([]map[string]any, error) {
			find := macroStr(args, "find")
			if strings.TrimSpace(find) == "" {
				return nil, fmt.Errorf(`缺少参数 find（要替换的原文）`)
			}
			replace, ok := args["replace"]
			if !ok {
				return nil, fmt.Errorf(`缺少参数 replace（替换后的文字；要删除就传空串 ""）`)
			}
			text, isStr := replace.(string)
			if !isStr {
				return nil, fmt.Errorf("参数 replace 必须是字符串")
			}
			return []map[string]any{{"op": "replace_text", "find": find, "replace": text}}, nil
		},
	},
	"sheet_replace_all": {
		Name: "sheet_replace_all",
		App:  "excel",
		Desc: "在 Excel 指定区域里把某段文字全部替换（等价一次 replace，区域必填以免误伤整表）",
		Args: `{"range":"A1:F200","find":"原文","replace":"新文"}`,
		Build: func(args map[string]any) ([]map[string]any, error) {
			rng := macroStr(args, "range")
			if strings.TrimSpace(rng) == "" {
				return nil, fmt.Errorf(`缺少参数 range（如 "A1:F200"）；整表替换请显式给已用区域`)
			}
			find := macroStr(args, "find")
			if strings.TrimSpace(find) == "" {
				return nil, fmt.Errorf(`缺少参数 find（要替换的原文）`)
			}
			replace := macroStr(args, "replace")
			return []map[string]any{{"op": "replace", "range": rng, "find": find, "replace": replace}}, nil
		},
	},
}

// officeMacroNames 返回按名排序的宏名（list 动作与报错提示共用）。
func officeMacroNames() []string {
	names := make([]string, 0, len(officeMacros))
	for n := range officeMacros {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// officeMacroCatalog 渲染成给模型看的清单文本。
func officeMacroCatalog() string {
	var b strings.Builder
	b.WriteString("可用的受控宏（宏名 + 适用应用 + 参数）：\n")
	for _, n := range officeMacroNames() {
		m := officeMacros[n]
		fmt.Fprintf(&b, "- %s [%s] %s；参数 %s\n", m.Name, m.App, m.Desc, m.Args)
	}
	b.WriteString("宏只能展开成既有 office_edit 的 ops，不支持任意脚本；需要的能力不在清单里时，直接用 office_edit 的 ops 或告知用户。")
	return b.String()
}

// macroStr 读字符串参数（缺省空串，非字符串按 fmt.Sprint 兜底）。
func macroStr(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	if s, isStr := v.(string); isStr {
		return s
	}
	return fmt.Sprint(v)
}

// officeMacroTool 是 office_macro 工具本体。
type officeMacroTool struct{}

func (officeMacroTool) Name() string { return "office_macro" }

// ReadOnly 恒 false：run 动作会写文档（list 只读，但工具整体按写工具对待）。
func (officeMacroTool) ReadOnly() bool { return false }

func (officeMacroTool) Description() string {
	return "受控宏逃生舱：当 office_edit 的 ops 组合容易拼错、或同一组改写反复出现时，用宏名 + 少量参数一次完成。" +
		"action=list 列出全部宏与参数；action=run 展开成既有 ops 后执行（与 office_edit 同一条 COM 路径，改动实时上屏）。" +
		"宏是白名单，不接受任意脚本。"
}

func (officeMacroTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "action":{"type":"string","enum":["list","run"],"description":"list=列出可用宏；run=执行指定宏"},
    "macro":{"type":"string","description":"宏名（action=run 必填），取值见 action=list 的返回"},
    "doc":{"type":"string","description":"文档路径或窗口名（与 office_edit 同语义）；缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在 doc 无扩展名时需要"},
    "args":{"type":"object","description":"宏参数，键名见 list 返回的参数说明；类型不匹配会被拒绝并提示具体字段"}
  },
  "required":["action"]
}`)
}

func (t officeMacroTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.ExecuteWithContext(ctx, tool.ToolContext{}, args)
}

func (t officeMacroTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	var in struct {
		Action string         `json:"action"`
		Macro  string         `json:"macro"`
		Doc    string         `json:"doc"`
		App    string         `json:"app"`
		Args   map[string]any `json:"args"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("office_macro 参数解析失败：%v", err)
	}
	action := strings.TrimSpace(strings.ToLower(in.Action))
	switch action {
	case "":
		return "", fmt.Errorf("缺少参数 action（list 或 run）")
	case "list":
		return officeMacroCatalog(), nil
	case "run":
		// 继续往下
	default:
		return "", fmt.Errorf("无效的 action %q，合法值: list, run", in.Action)
	}

	name := strings.TrimSpace(in.Macro)
	if name == "" {
		return "", fmt.Errorf("action=run 时必须给 macro，可用：%s", strings.Join(officeMacroNames(), ", "))
	}
	spec, ok := officeMacros[name]
	if !ok {
		return "", fmt.Errorf("未知宏 %q；可用：%s（用 action=list 看参数）", name, strings.Join(officeMacroNames(), ", "))
	}
	if in.App != "" && spec.App != "any" && in.App != spec.App {
		return "", fmt.Errorf("宏 %q 只适用于 %s，当前 app=%s", name, spec.App, in.App)
	}
	ops, err := spec.Build(in.Args)
	if err != nil {
		return "", fmt.Errorf("宏 %q 参数错误：%v；参考 %s", name, err, spec.Args)
	}
	if len(ops) == 0 {
		return "", fmt.Errorf("宏 %q 展开后没有任何操作（内部错误）", name)
	}

	// 展开成 office_edit 的入参，复用同一条 COM 执行路径与 Journal 证据。
	payload, err := json.Marshal(map[string]any{
		"doc": in.Doc,
		"app": in.App,
		"ops": ops,
	})
	if err != nil {
		return "", fmt.Errorf("宏 %q 展开失败：%v", name, err)
	}
	out, err := (officeComEditTool{}).ExecuteWithContext(ctx, tc, payload)
	if err != nil {
		return out, fmt.Errorf("宏 %q 执行失败：%w", name, err)
	}
	return fmt.Sprintf("宏 %s 已执行（展开 %d 个 op）\n%s", name, len(ops), out), nil
}
