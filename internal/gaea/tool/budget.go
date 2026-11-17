package tool

import (
	"fmt"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 工具 schema 字节预算（codex 蒸馏 T5，tools_budget 同构）：工具多时首轮
// schema 体积失控。共享预算内「名字优先全额保留，剩余在描述间按轮次均分」
// ——公平且确定，同输入同截断（可缓存、可比对）。默认关闭；关闭时调用方
// 根本不进来，行为与现状逐字节等价。

// defaultSchemaBudgetBytes 是开启预算但未配置字节数时的保守默认。取值依据：
// 一次首轮 schema 的描述文本主体通常在个位数 KiB，12KiB 足以保住全部工具名
// +紧凑描述，同时把失控的长描述压回轨道。
const defaultSchemaBudgetBytes = 12 * 1024

// defaultOmissionReserve 是「...还有 N 个工具」占位行的预留字节数。
const defaultOmissionReserve = 64

// omittedRow 是 TruncateRows 返回值里的「整行省略」哨兵（[]int 装不下 nil，
// 用 -1 表达同一语义；0 = 只留名字）。
const omittedRow = -1

// BudgetRow 是预算裁剪的输入行：工具名 + 描述文本。
type BudgetRow struct {
	Name string
	Desc string
}

// TruncateRows 在共享字节预算内裁描述。返回与 rows 同序的保留描述字节数：
// omittedRow(-1)=整行省略（名字也不保留），0=只留名字，>0=描述保留的字节上限
// （含截断尾标 "..." 的空间）。预算按 UTF-8 字节计；名字行固定成本
// len("- ")+len(name)+len("\n") 先全额保留；名字总和超预算时从尾部整行省略，
// 省略部分用一行「...还有 N 个工具」占位（吃 omissionReserve 预留）；剩余
// 预算在保留行的描述间按轮次均分（每轮每行 +1 字节直到用尽），保证公平且
// 同输入同结果。
func TruncateRows(rows []BudgetRow, byteBudget, omissionReserve int) []int {
	if byteBudget < 0 {
		byteBudget = 0
	}
	if omissionReserve < 0 {
		omissionReserve = 0
	}
	n := len(rows)
	out := make([]int, n)
	keep := make([]bool, n)
	nameCost := make([]int, n)
	total := 0
	for i, r := range rows {
		keep[i] = true
		nameCost[i] = len("- ") + len(r.Name) + len("\n")
		total += nameCost[i]
	}
	// 名字总额超预算（须给占位行留出 omission 预留）：从尾部整行省略。
	omitted := 0
	for total > byteBudget-omissionReserve {
		last := -1
		for i := n - 1; i >= 0; i-- {
			if keep[i] {
				last = i
				break
			}
		}
		if last < 0 {
			break // 全省略了，只剩占位行
		}
		keep[last] = false
		out[last] = omittedRow
		total -= nameCost[last]
		omitted++
	}
	// 剩余预算 = 总预算 - 名字行 - 占位行，在保留行间按轮次均分。
	rest := byteBudget - total
	if omitted > 0 {
		rest -= len(fmt.Sprintf("...还有 %d 个工具", omitted))
	}
	if rest < 0 {
		rest = 0
	}
	for i := 0; i < n; i++ {
		if keep[i] {
			out[i] = 0
		}
	}
	for rest > 0 {
		progress := false
		for i := 0; i < n && rest > 0; i++ {
			if keep[i] {
				out[i]++
				rest--
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	return out
}

// TruncateDescBytes 把描述截到 maxBytes 字节内：超长按 UTF-8 字符边界截断并
// 追加 "..."（尾标占预算）。maxBytes<=0 返回空串（只留名字）。
func TruncateDescBytes(desc string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(desc) <= maxBytes {
		return desc
	}
	keep := maxBytes - len("...")
	if keep <= 0 {
		return ""
	}
	// 回退到字符边界（不留半个 rune）。
	for keep > 0 && desc[keep]&0xC0 == 0x80 {
		keep--
	}
	return desc[:keep] + "..."
}

// ApplySchemaBudget 把字节预算套到一组工具 schema 上（FilteredSchemas 之后
// 的统一收口）：整行省略的工具从列表消失；0=只留名字；>0=描述截断。
// 参数 JSON 不参与预算（结构化部分另有紧凑档守卫 compact_drift）。
func ApplySchemaBudget(schemas []provider.ToolSchema, budget int) []provider.ToolSchema {
	if budget <= 0 {
		budget = defaultSchemaBudgetBytes
	}
	rows := make([]BudgetRow, len(schemas))
	for i, s := range schemas {
		rows[i] = BudgetRow{Name: s.Name, Desc: s.Description}
	}
	keep := TruncateRows(rows, budget, defaultOmissionReserve)
	out := make([]provider.ToolSchema, 0, len(schemas))
	for i, s := range schemas {
		switch keep[i] {
		case omittedRow:
			continue // 整行省略
		case 0:
			s.Description = "" // 只留名字
		default:
			s.Description = TruncateDescBytes(s.Description, keep[i])
		}
		out = append(out, s)
	}
	return out
}
