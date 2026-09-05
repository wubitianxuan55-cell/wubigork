package genui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// 上限常量与 frontend/src/genui/spec.ts GENUI_LIMITS 同源，改动须同步。
const (
	maxDepth      = 8
	maxNodes      = 200
	maxString     = 2000
	maxCode       = 12000
	maxFenceBody  = 65536
	maxGridCols   = 12
	maxTableRows  = 50
	maxTableCols  = 12
	maxOptions    = 50
	maxChartPoint = 60
)

// nodeTypes 与 TS GENUI_NODE_TYPES 同源（未知 type 一律报错）。
var nodeTypes = map[string]bool{
	"text": true, "row": true, "col": true, "grid": true, "card": true,
	"divider": true, "spacer": true,
	"stat": true, "badge": true, "progress": true, "keyvalue": true,
	"list": true, "table": true, "timeline": true, "callout": true,
	"steps": true, "avatar": true, "copy": true,
	"chart": true, "code": true, "json": true, "diff": true,
	"button": true, "input": true, "select": true, "checkbox": true,
	"switch": true, "radio": true, "slider": true, "textarea": true,
	"submit": true, "tabs": true, "accordion": true, "quiz": true,
}

// Validation 是结构校验结果（只做结构检查；渲染器仍是最终权威）。
type Validation struct {
	OK     bool
	Nodes  int
	Errors []string
}

// ValidateSpec 校验 ```genui 围栏体 JSON：语法、根形状、type 白名单、
// 节点/深度预算与明显缺字段。错误带路径便于模型修正。
func ValidateSpec(raw string) Validation {
	text := strings.TrimSpace(raw)
	if text == "" {
		return Validation{Errors: []string{"规格为空"}}
	}
	if len(text) > maxFenceBody {
		return Validation{Errors: []string{fmt.Sprintf("围栏体超过 %d 字节上限", maxFenceBody)}}
	}
	var root any
	if err := json.Unmarshal([]byte(text), &root); err != nil {
		return Validation{Errors: []string{fmt.Sprintf("JSON 语法错误：%v", err)}}
	}
	v := &validator{}
	rootObj, ok := root.(map[string]any)
	if !ok {
		return Validation{Errors: []string{"规格必须是 JSON 对象"}}
	}
	if _, hasType := rootObj["type"]; hasType {
		v.walk("root", rootObj, 0)
	} else if items, hasItems := rootObj["items"].([]any); hasItems {
		for i, child := range items {
			v.walk(fmt.Sprintf("items[%d]", i), child, 0)
		}
		if len(items) == 0 {
			v.errors = append(v.errors, "items 不能为空")
		}
	} else {
		v.errors = append(v.errors, "根必须是 {items:[…]} 或单个组件对象")
	}
	return Validation{OK: len(v.errors) == 0, Nodes: v.nodes, Errors: v.errors}
}

type validator struct {
	nodes  int
	errors []string
}

func (v *validator) add(path, msg string) {
	if len(v.errors) < 30 {
		v.errors = append(v.errors, fmt.Sprintf("%s: %s", path, msg))
	}
}

func (v *validator) walk(path string, value any, depth int) {
	obj, ok := value.(map[string]any)
	if !ok {
		v.add(path, "应为 JSON 对象")
		return
	}
	if depth > maxDepth {
		v.add(path, "嵌套深度超限")
		return
	}
	if v.nodes >= maxNodes {
		v.add(path, "节点预算已耗尽")
		return
	}
	typ, _ := obj["type"].(string)
	if typ == "" {
		v.add(path, "缺少 type 字段")
		return
	}
	if !nodeTypes[typ] {
		v.add(path, fmt.Sprintf("未知组件 type %q", typ))
		return
	}
	v.nodes++
	v.checkRequired(path, typ, obj)
	for _, key := range []string{"items", "tabs", "steps", "options", "diffs", "series"} {
		if arr, ok := obj[key].([]any); ok {
			for i, child := range arr {
				if len(v.errors) >= 30 {
					return
				}
				v.walk(fmt.Sprintf("%s.%s[%d]", path, key, i), child, depth+1)
			}
		}
	}
}

func (v *validator) checkRequired(path, typ string, obj map[string]any) {
	need := func(key string) {
		if _, ok := obj[key]; !ok {
			v.add(path, fmt.Sprintf("%s 缺少必填字段 %s", typ, key))
		}
	}
	switch typ {
	case "text":
		need("content")
	case "row", "col", "grid", "card", "tabs", "accordion":
		need("items")
	case "stat":
		need("label")
		need("value")
	case "badge":
		need("label")
	case "progress":
		need("value")
	case "keyvalue":
		need("pairs")
	case "table":
		need("columns")
		need("rows")
	case "timeline":
		need("items")
	case "callout":
		need("content")
	case "steps":
		need("steps")
	case "avatar":
		need("name")
	case "copy":
		need("text")
	case "chart":
		need("data")
	case "code":
		need("code")
	case "json":
		need("value")
	case "diff":
		need("diffs")
	case "button", "checkbox", "switch":
		need("label")
	case "select", "radio":
		need("options")
	case "quiz":
		need("question")
		need("options")
	}
	if cols, ok := obj["cols"]; ok && typ == "grid" {
		if n, ok2 := cols.(float64); !ok2 || n < 1 || n > maxGridCols {
			v.add(path, "grid.cols 应为 1–12 整数")
		}
	}
}
