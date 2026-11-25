package app

// office_cost_library_contract.go — cost_library 回执携带面的渲染（2026-11-25 办公
// 工具梳理批）。
//
// import_add 三组行、classify 四组的字段契约不常驻 schema，改在流程必经的节点
// 随回执发放（import_open → 行契约；dims → classify 契约；action=help 随取）。
// 文本从 office_cost_library_schema.go 的同一份源常量渲染——字段名、必填说明、
// 取值枚举都不会和 schema 各写各的。渲染成缩进清单而不是 JSON Schema：省掉
// {"type":…,"description":…} 的包装，同样的信息约占一半字符。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// clNode 是渲染用到的 JSON Schema 字段子集。
type clNode struct {
	Type       string          `json:"type"`
	Desc       string          `json:"description"`
	Enum       []string        `json:"enum"`
	Items      json.RawMessage `json:"items"`
	Properties json.RawMessage `json:"properties"`
}

// clOrderedProps 按出现顺序解出一个 properties 对象的键与值（Go map 丢序，
// 契约要按说明的书写顺序呈现）。
func clOrderedProps(raw json.RawMessage) ([]clProp, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil { // '{'
		return nil, err
	}
	var props []clProp
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		props = append(props, clProp{Name: fmt.Sprint(keyTok), Schema: v})
	}
	return props, nil
}

type clProp struct {
	Name   string
	Schema json.RawMessage
}

// clAllScalar 报告一组属性是否全是标量或标量数组（可压成一行）。
func clAllScalar(props []clProp) bool {
	for _, p := range props {
		var n clNode
		if err := json.Unmarshal(p.Schema, &n); err != nil {
			return false
		}
		if len(n.Properties) > 0 {
			return false
		}
		if n.Type == "array" && len(n.Items) > 0 {
			var it clNode
			if json.Unmarshal(n.Items, &it) == nil && it.Type == "object" {
				return false
			}
		}
	}
	return true
}

// clTypeLabel 渲染类型标签：枚举给 a|b|c，其余给类型名。
func clTypeLabel(n clNode) string {
	if len(n.Enum) > 0 {
		return strings.Join(n.Enum, "|")
	}
	return n.Type
}

// clRenderProps 把 properties 渲染成缩进清单。
func clRenderProps(b *strings.Builder, indent string, props []clProp) {
	for _, p := range props {
		var n clNode
		if err := json.Unmarshal(p.Schema, &n); err != nil {
			continue
		}
		switch {
		case n.Type == "array" && len(n.Items) > 0:
			var it clNode
			_ = json.Unmarshal(n.Items, &it)
			if it.Type == "object" {
				fmt.Fprintf(b, "%s%s[] — %s\n", indent, p.Name, n.Desc)
				clRenderObject(b, indent+"  ", it)
				continue
			}
			fmt.Fprintf(b, "%s%s %s[] — %s\n", indent, p.Name, clTypeLabel(it), n.Desc)
		case n.Type == "object" && len(n.Properties) > 0:
			subs, err := clOrderedProps(n.Properties)
			if err == nil && clAllScalar(subs) {
				parts := make([]string, 0, len(subs))
				for _, sp := range subs {
					var sn clNode
					_ = json.Unmarshal(sp.Schema, &sn)
					if sn.Type == "array" {
						parts = append(parts, sp.Name+" "+clTypeLabel(sn)+"[]")
						continue
					}
					parts = append(parts, sp.Name+" "+clTypeLabel(sn))
				}
				fmt.Fprintf(b, "%s%s{%s} — %s\n", indent, p.Name, strings.Join(parts, ", "), n.Desc)
				continue
			}
			fmt.Fprintf(b, "%s%s{} — %s\n", indent, p.Name, n.Desc)
			clRenderObject(b, indent+"  ", n)
		default:
			fmt.Fprintf(b, "%s%s %s — %s\n", indent, p.Name, clTypeLabel(n), n.Desc)
		}
	}
}

func clRenderObject(b *strings.Builder, indent string, n clNode) {
	subs, err := clOrderedProps(n.Properties)
	if err != nil {
		return
	}
	clRenderProps(b, indent, subs)
}

// clMustContract 把一段 properties 片段（office_cost_library_schema.go 的源常量）
// 渲染成契约文本。片段解析失败=源常量写坏了，测试必红，启动即败比模型收到
// 半截契约强。
func clMustContract(title, fragment string) string {
	body := `{"properties":{` + strings.TrimRight(strings.TrimSpace(fragment), ",") + `}}`
	var wrap struct {
		Properties json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(body), &wrap); err != nil {
		panic("cost_library: 契约源常量不是合法 JSON 片段（" + title + "）: " + err.Error())
	}
	props, err := clOrderedProps(wrap.Properties)
	if err != nil {
		panic("cost_library: 契约源常量解析失败（" + title + "）: " + err.Error())
	}
	var b strings.Builder
	b.WriteString("📋 " + title + "\n")
	clRenderProps(&b, "  ", props)
	return strings.TrimRight(b.String(), "\n")
}

var (
	// clImportRowsContract import_add 三组行的字段契约（随 import_open 回执与 help 发放）。
	clImportRowsContract = clMustContract(
		"import_add 行字段（顶层参数 resources / quotas / bills，三组可同批也可分批；先工料机、再定额、后清单）",
		clImportRowsSchema)
	// clClassifyContract classify 四组的字段契约（随 dims 回执与 help 发放）。
	clClassifyContract = clMustContract("classify 字段（顶层参数；每组 ≤500，没给的字段不动）", clClassifySchema)
)

// clActionFlow 定义动作在 help 里的呈现顺序（录入 → 定额工料机价格 → 删除 → 归类
// → 字典），不在表里的动作排在后面。
var clActionFlow = []string{
	"import_open", "import_add", "import_project",
	"save_quota", "save_resource", "set_price", "borrow_quota",
	"delete_quota", "delete_resource", "delete_items", "delete_project", "dedupe",
	"classify", "dict", "dict_add", "dict_update", "dict_delete",
}

// clHelpReceipt action=help：各动作参数骨架 + 两段不常驻的字段契约。
func (t costLibraryTool) clHelpReceipt() string {
	skel := clActionSkeletons()
	names := make([]string, 0, len(skel))
	for name := range skel {
		names = append(names, name)
	}
	rank := map[string]int{}
	for i, n := range clActionFlow {
		rank[n] = i
	}
	sort.SliceStable(names, func(a, b int) bool {
		ra, rb := rank[names[a]], rank[names[b]]
		if ra != rb {
			return ra < rb
		}
		return names[a] < names[b]
	})
	var b strings.Builder
	b.WriteString("## cost_library 参数契约（只读，不写库）\n\n**各动作参数骨架**（照抄改值；批量与删除先 dry_run=true 演练）:\n")
	for _, name := range names {
		fmt.Fprintf(&b, "- %s: %s\n", name, skel[name])
	}
	b.WriteString("- 其余动作（import_check / import_commit / import_discard / health / seed / template）参数见常驻说明。\n\n")
	b.WriteString(clImportRowsContract + "\n\n" + clClassifyContract)
	return b.String()
}
