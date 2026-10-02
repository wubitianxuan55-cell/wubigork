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

// maxErrors 是单次校验的错误条数上限（防刷屏）。TS 侧 validateGenuiSpec 用 50，
// 属已知口径分叉；工具契约以本值为准。
const maxErrors = 30

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
	if !v.full() {
		v.errors = append(v.errors, fmt.Sprintf("%s: %s", path, msg))
	}
}

// full 报告错误预算是否已耗尽。
func (v *validator) full() bool { return len(v.errors) >= maxErrors }

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
	v.checkData(path, typ, obj)
	v.walkChildren(path, typ, obj, depth)
}

// walkChildren 只在真正的子节点容器里递归（对照 spec.ts / guard.ts）：
// row/col/grid/card/list 的 items、tabs 行的 items、accordion 行的 items。
// 数据数组（options/diffs/series/data/timeline.items/steps 等）绝不在此递归——
// 它们没有 type 字段，按节点走会逐行误报「缺少 type 字段」并吃掉错误预算。
func (v *validator) walkChildren(path, typ string, obj map[string]any, depth int) {
	switch typ {
	case "row", "col", "grid", "card":
		v.walkNodes(path+".items", obj["items"], depth, false)
	case "list":
		// list.items 元素可为字符串或节点（spec.ts GenuiList）。
		v.walkNodes(path+".items", obj["items"], depth, true)
	case "tabs":
		// tabs 行是数据行 {label,items}：行本身不是节点，行内 items 才是。
		for i, raw := range v.arrayAt(path+".tabs", obj["tabs"]) {
			p := fmt.Sprintf("%s.tabs[%d]", path, i)
			row, ok := raw.(map[string]any)
			if !ok {
				v.add(p, "应为 JSON 对象")
				continue
			}
			v.checkStringField(p, typ, "行", "label", row)
			v.walkNodes(p+".items", row["items"], depth, false)
		}
	case "accordion":
		// accordion.items 行是数据行 {title,items}：行本身不是节点，行内 items 才是。
		for i, raw := range v.arrayAt(path+".items", obj["items"]) {
			p := fmt.Sprintf("%s.items[%d]", path, i)
			row, ok := raw.(map[string]any)
			if !ok {
				v.add(p, "应为 JSON 对象")
				continue
			}
			v.checkStringField(p, typ, "行", "title", row)
			v.walkNodes(p+".items", row["items"], depth, false)
		}
	}
}

// walkNodes 递归一列子节点；allowString=true 时字符串元素合法（list.items）。
func (v *validator) walkNodes(path string, value any, depth int, allowString bool) {
	arr, ok := value.([]any)
	if !ok {
		return // 字段缺失/类型错由 checkRequired 与 checkData 报告，此处不重复
	}
	for i, child := range arr {
		if v.full() {
			return
		}
		if allowString {
			if _, isStr := child.(string); isStr {
				continue
			}
		}
		v.walk(fmt.Sprintf("%s[%d]", path, i), child, depth+1)
	}
}

// arrayAt 取数据数组：字段缺失返回 nil（缺字段由 checkRequired 报告），
// 存在但不是数组时报一次错。
func (v *validator) arrayAt(path string, value any) []any {
	if value == nil {
		return nil
	}
	arr, ok := value.([]any)
	if !ok {
		v.add(path, "应为数组")
		return nil
	}
	return arr
}

// checkData 只做数据数组的字段/形状校验（与 guard.ts 消费的字段对齐）。
func (v *validator) checkData(path, typ string, obj map[string]any) {
	switch typ {
	case "select", "radio":
		for i, raw := range v.arrayAt(path+".options", obj["options"]) {
			if v.full() {
				return
			}
			if _, ok := raw.(string); !ok {
				v.add(fmt.Sprintf("%s.options[%d]", path, i), fmt.Sprintf("%s 选项应为字符串", typ))
			}
		}
	case "quiz":
		v.checkDataRows(path, typ, "options", obj["options"], "label")
	case "timeline":
		v.checkDataRows(path, typ, "items", obj["items"], "title")
	case "steps":
		v.checkDataRows(path, typ, "steps", obj["steps"], "title")
	case "chart":
		v.checkChartPoints(path+".data", typ, obj["data"])
		for i, raw := range v.arrayAt(path+".series", obj["series"]) {
			p := fmt.Sprintf("%s.series[%d]", path, i)
			s, ok := raw.(map[string]any)
			if !ok {
				v.add(p, "应为 JSON 对象")
				continue
			}
			if _, ok := s["label"].(string); !ok {
				v.add(p, fmt.Sprintf("%s 数据序列缺少必填字段 label", typ))
			}
			val, has := s["data"]
			if !has {
				v.add(p, fmt.Sprintf("%s 数据序列缺少必填字段 data", typ))
				continue
			}
			arr, isArr := val.([]any)
			if !isArr {
				v.add(p, fmt.Sprintf("%s 数据序列必填字段 data 应为数组", typ))
				continue
			}
			v.checkChartPoints(p+".data", typ, arr)
		}
	case "diff":
		// {path,oldText?,newText}；oldText 可为字符串或 null（省略即新增行）。
		v.checkDataRows(path, typ, "diffs", obj["diffs"], "path", "newText")
	}
}

// checkDataRows 校验数据行数组：每行必须是对象且含给定必填字符串字段。
func (v *validator) checkDataRows(path, typ, key string, value any, fields ...string) {
	for i, raw := range v.arrayAt(path+"."+key, value) {
		if v.full() {
			return
		}
		p := fmt.Sprintf("%s.%s[%d]", path, key, i)
		row, ok := raw.(map[string]any)
		if !ok {
			v.add(p, "应为 JSON 对象")
			continue
		}
		for _, f := range fields {
			v.checkStringField(p, typ, "行", f, row)
		}
	}
}

// checkStringField 校验数据行/数据点的必填字符串字段（缺字段与类型错分开报）。
func (v *validator) checkStringField(path, typ, kind, field string, row map[string]any) {
	val, ok := row[field]
	if !ok {
		v.add(path, fmt.Sprintf("%s 数据%s缺少必填字段 %s", typ, kind, field))
		return
	}
	if _, ok := val.(string); !ok {
		v.add(path, fmt.Sprintf("%s 数据%s必填字段 %s 应为字符串", typ, kind, field))
	}
}

// checkChartPoints 校验 chart.data 与 chart.series[].data 的数据点：
// 必须是对象且含字符串 label 与数字 value（guard.ts chartPoint 同口径）。
func (v *validator) checkChartPoints(base, typ string, value any) {
	for i, raw := range v.arrayAt(base, value) {
		if v.full() {
			return
		}
		p := fmt.Sprintf("%s[%d]", base, i)
		pt, ok := raw.(map[string]any)
		if !ok {
			v.add(p, "应为 JSON 对象")
			continue
		}
		v.checkStringField(p, typ, "点", "label", pt)
		val, has := pt["value"]
		if !has {
			v.add(p, fmt.Sprintf("%s 数据点缺少必填字段 value", typ))
		} else if _, ok := val.(float64); !ok {
			v.add(p, fmt.Sprintf("%s 数据点必填字段 value 应为数字", typ))
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
	case "row", "col", "grid", "card", "accordion":
		need("items")
	case "tabs":
		// spec.ts/guard.ts：tabs 由 tabs:[{label,items}] 承载，顶层没有 items。
		need("tabs")
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
