package genui

import "sort"

// limits.go —— GenUI 上限常量与合法 type 白名单的**单一真相源**
// （审计 2026-10-02 X1-07 + GA2-05 + FE6-03）。
//
// 这些常量既被本包 ValidateSpec 用于结构校验，又由测试
// （limits_sync_test.go 的 TestGenuiLimitsSync）生成并校验
// frontend/src/genui/limits.ts；前端 spec.ts 再从生成物构造 GENUI_LIMITS 与
// GENUI_NODE_TYPES。任何一侧被手改、或 Go 侧改了值而忘记重新生成，该测试都会
// 失败——这就是「后端放行、前端截断」这类静默分叉的机器守卫。
//
// 命名与 TS 侧逐字一致（注意 maxChartPoints 带复数 s，收敛前的
// maxChartPoint 单数命名已废弃）：改名只改这里 + 重新生成。
const (
	maxDepth = 8
	maxNodes = 200
	// maxString 是单个字符串字段（text/callout.content 等）的长度上限，口径为
	// **UTF-16 码元数**，与前端 JS String.length（guard.ts 的 str(v, cap) 截断点）
	// 逐值一致；用 rune 数会让 emoji/增补平面字符的判定与前端分叉。
	maxString = 2000
	// maxCode 是 code.code（以及 json.value 序列化后）的长度上限，口径同 maxString。
	maxCode = 12000
	// maxFenceBody 是围栏体的字节上限（Go 按字节；前端 parse.ts 按 body.length
	// 即 UTF-16 码元，两者属不同口径，仅数值同源，见 validate.go 的注释）。
	maxFenceBody   = 65536
	maxGridCols    = 12
	maxTableRows   = 50
	maxTableCols   = 12
	maxOptions     = 50
	maxChartPoints = 60
)

// maxErrors 是单次校验的错误条数上限（防刷屏）。前端曾有 validateGenuiSpec
// 用 50，该通道已按 FE6-04 判定为零消费死代码并删除，故现存唯一口径是本值。
const maxErrors = 30

// nodeTypes 与 TS GENUI_NODE_TYPES 同源（未知 type 一律报错）；TS 侧由
// limits.ts 的 GENUI_GO_NODE_TYPES 生成，权威清单是本表。
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

// LimitEntry 是上限常量的机器可读条目：Value 直接引用上面的常量，改名或改值
// 漏一处就编译不过，因此这张表不可能与常量漂移。
type LimitEntry struct {
	Name  string
	Value int
}

// Limits 是全部上限常量的机器可读清单——生成 frontend/src/genui/limits.ts
// 的唯一输入（顺序即生成顺序）。
var Limits = []LimitEntry{
	{"maxDepth", maxDepth},
	{"maxNodes", maxNodes},
	{"maxString", maxString},
	{"maxCode", maxCode},
	{"maxFenceBody", maxFenceBody},
	{"maxGridCols", maxGridCols},
	{"maxTableRows", maxTableRows},
	{"maxTableCols", maxTableCols},
	{"maxOptions", maxOptions},
	{"maxChartPoints", maxChartPoints},
}

// NodeTypeList 返回白名单 type 的排序清单：nodeTypes 的稳定导出，供 TS 生成
// （GENUI_GO_NODE_TYPES）与双向断言使用。
func NodeTypeList() []string {
	out := make([]string, 0, len(nodeTypes))
	for t := range nodeTypes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
