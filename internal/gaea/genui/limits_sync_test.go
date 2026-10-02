package genui

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// limits_sync_test.go —— X1-07 的机器守卫 + GA2-05 的「常量真的被执行」守卫。
//
// 1) TestGenuiLimitsSync：Go 侧常量集（limits.go）↔ frontend/src/genui/limits.ts
//    逐字节相同。手改 TS、或改 Go 后忘记重新生成，这里都会红——前端 spec.ts 的
//    GENUI_LIMITS / GENUI_NODE_TYPES 正是从这个生成物构造的。
//    重新生成： GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync ./internal/gaea/genui/
//
// 2) TestEveryLimitIsEnforced：为每个上限常量构造"刚刚超限"的规格，断言
//    ValidateSpec 判非法。GA2-05 的根因是常量声明后零引用（Go 不报未使用常量、
//    lint 兜不住），本测试让"新增常量但忘了接校验"当场失败。

// limitsTSPath 是前端生成物的仓内路径（相对本包目录）。
const limitsTSPath = "../../../frontend/src/genui/limits.ts"

// renderLimitsTS 渲染生成物内容（生成与校验共用同一函数，故两侧不可能漂移）。
func renderLimitsTS() string {
	var b strings.Builder
	b.WriteString("// Code generated from internal/gaea/genui/limits.go — DO NOT EDIT MANUALLY.\n")
	b.WriteString("// 单一真相源在 Go 侧（审计 2026-10-02 X1-07）；重新生成：\n")
	b.WriteString("//   GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync ./internal/gaea/genui/\n")
	b.WriteString("// 由 limits_sync_test.go 的 TestGenuiLimitsSync 逐字节校验（手改必红）。\n\n")
	b.WriteString("/** Go 侧 genui 上限常量（唯一真相源；spec.ts 在此之上补渲染器专属上限）。 */\n")
	b.WriteString("export const GENUI_GO_LIMITS = {\n")
	for _, e := range Limits {
		fmt.Fprintf(&b, "  %s: %d,\n", e.Name, e.Value)
	}
	b.WriteString("} as const;\n\n")
	b.WriteString("/** Go 侧合法组件 type 白名单（唯一真相源；spec.ts 用它构造 GENUI_NODE_TYPES）。 */\n")
	b.WriteString("export const GENUI_GO_NODE_TYPES: readonly string[] = [\n")
	for _, t := range NodeTypeList() {
		fmt.Fprintf(&b, "  %q,\n", t)
	}
	b.WriteString("];\n")
	return b.String()
}

// TestGenuiLimitsSync 双向断言：Go 常量集 == 前端生成物（逐字节）。
func TestGenuiLimitsSync(t *testing.T) {
	want := renderLimitsTS()
	if os.Getenv("GAEA_GENUI_LIMITS_UPDATE") == "1" {
		if err := os.WriteFile(limitsTSPath, []byte(want), 0o644); err != nil {
			t.Fatalf("重新生成 %s 失败: %v", limitsTSPath, err)
		}
		t.Logf("已从 Go 常量重新生成 %s", limitsTSPath)
		return
	}
	got, err := os.ReadFile(limitsTSPath)
	if err != nil {
		t.Fatalf("读前端生成物失败（%s）: %v；先跑 GAEA_GENUI_LIMITS_UPDATE=1 生成", limitsTSPath, err)
	}
	if string(got) != want {
		t.Fatalf("Go 常量与前端生成物已漂移（%s）：\n--- got ---\n%s\n--- want ---\n%s\n"+
			"修法：GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync ./internal/gaea/genui/",
			limitsTSPath, got, want)
	}
	if len(Limits) == 0 || len(NodeTypeList()) != 34 {
		t.Fatalf("常量表异常：限值 %d 条，type %d 个（应为 34）", len(Limits), len(NodeTypeList()))
	}
}

func manyStrs(n int, mk func(i int) string) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = mk(i)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// TestEveryLimitIsEnforced 是 GA2-05 的守卫：每个上限常量都必须真的被执行。
func TestEveryLimitIsEnforced(t *testing.T) {
	long := strings.Repeat("x", 70000)
	deep := strings.Repeat(`{"type":"card","items":[`, 12) + `{"type":"divider"}` + strings.Repeat(`]}`, 12)
	cases := []struct {
		limit string
		raw   string
	}{
		{"maxString", `{"type":"text","content":"` + strings.Repeat("x", maxString+1) + `"}`},
		{"maxCode", `{"type":"code","code":"` + strings.Repeat("x", maxCode+1) + `"}`},
		{"maxTableRows", `{"type":"table","columns":["a"],"rows":` +
			manyStrs(maxTableRows+1, func(int) string { return `["1"]` }) + `}`},
		{"maxTableCols", `{"type":"table","columns":` +
			manyStrs(maxTableCols+1, func(i int) string { return fmt.Sprintf(`"c%d"`, i) }) + `,"rows":[]}`},
		{"maxOptions", `{"type":"select","options":` +
			manyStrs(maxOptions+1, func(i int) string { return fmt.Sprintf(`"o%d"`, i) }) + `}`},
		{"maxChartPoints", `{"type":"chart","data":` +
			manyStrs(maxChartPoints+1, func(i int) string { return fmt.Sprintf(`{"label":"%d","value":1}`, i) }) + `}`},
		{"maxGridCols", `{"type":"grid","cols":13,"items":[]}`},
		{"maxNodes", `{"items":` + manyStrs(maxNodes+1, func(int) string { return `{"type":"divider"}` }) + `}`},
		{"maxDepth", deep},
		{"maxFenceBody", `{"type":"text","content":"` + long + `"}`},
	}
	for _, c := range cases {
		if r := ValidateSpec(c.raw); r.OK {
			t.Errorf("%s 未被任何校验路径执行：超限规格被判合法（GA2-05 复发）", c.limit)
		}
	}
}

// TestLimitBoundariesAccepted 边界口径：恰好等于上限必须合法（不多不少）。
func TestLimitBoundariesAccepted(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"text 恰为 maxString", `{"type":"text","content":"` + strings.Repeat("x", maxString) + `"}`},
		{"code 恰为 maxCode", `{"type":"code","code":"` + strings.Repeat("x", maxCode) + `"}`},
		{"table 恰为 maxTableRows", `{"type":"table","columns":["a"],"rows":` +
			manyStrs(maxTableRows, func(int) string { return `["1"]` }) + `}`},
		{"table 恰为 maxTableCols", `{"type":"table","columns":` +
			manyStrs(maxTableCols, func(i int) string { return fmt.Sprintf(`"c%d"`, i) }) + `,"rows":[]}`},
		{"select 恰为 maxOptions", `{"type":"select","options":` +
			manyStrs(maxOptions, func(i int) string { return fmt.Sprintf(`"o%d"`, i) }) + `}`},
		{"chart 恰为 maxChartPoints", `{"type":"chart","data":` +
			manyStrs(maxChartPoints, func(i int) string { return fmt.Sprintf(`{"label":"%d","value":1}`, i) }) + `}`},
		{"grid 恰为 maxGridCols", `{"type":"grid","cols":12,"items":[]}`},
	}
	for _, c := range cases {
		if r := ValidateSpec(c.raw); !r.OK {
			t.Errorf("%s：边界值应合法，却报 %v", c.name, r.Errors)
		}
	}
}

// TestStringLimitUsesUTF16Units 钉死字符串口径 = UTF-16 码元（JS String.length）：
// maxString 个 emoji（每个 2 码元）必须超限，maxString/2 个必须合法；若改用
// rune 计数，两个断言都会翻面——那正是「后端放行、前端截断」的来源。
func TestStringLimitUsesUTF16Units(t *testing.T) {
	over := `{"type":"text","content":"` + strings.Repeat("🎉", maxString) + `"}`
	if r := ValidateSpec(over); r.OK {
		t.Error("maxString 个 emoji（= 2*maxString UTF-16 码元）应超限")
	} else if !strings.Contains(strings.Join(r.Errors, "\n"), "UTF-16") {
		t.Errorf("错误信息应说明 UTF-16 口径: %v", r.Errors)
	}
	ok := `{"type":"text","content":"` + strings.Repeat("🎉", maxString/2) + `"}`
	if r := ValidateSpec(ok); !r.OK {
		t.Errorf("maxString/2 个 emoji（= maxString 码元）应合法: %v", r.Errors)
	}
	// 中文是 1 码元/字：maxString 个汉字恰好合法，maxString+1 超限。
	if r := ValidateSpec(`{"type":"text","content":"` + strings.Repeat("中", maxString) + `"}`); !r.OK {
		t.Errorf("maxString 个汉字应合法: %v", r.Errors)
	}
}

// TestChartSeriesPointsAlsoLimited series[].data 与 data 同一上限。
func TestChartSeriesPointsAlsoLimited(t *testing.T) {
	pts := manyStrs(maxChartPoints+1, func(i int) string { return fmt.Sprintf(`{"label":"%d","value":1}`, i) })
	raw := `{"type":"chart","data":[{"label":"a","value":1}],"series":[{"label":"s","data":` + pts + `}]}`
	if r := ValidateSpec(raw); r.OK {
		t.Fatal("series[].data 超限应判非法")
	}
}
