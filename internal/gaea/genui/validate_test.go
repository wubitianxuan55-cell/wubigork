package genui

import (
	"strings"
	"testing"
)

func TestValidateSpec_OK(t *testing.T) {
	raw := `{"title":"看板","items":[
		{"type":"stat","label":"营收","value":"¥128k"},
		{"type":"chart","kind":"bars","data":[{"label":"1月","value":98}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK, got %v", r.Errors)
	}
	if r.Nodes != 2 {
		t.Fatalf("nodes = %d, want 2", r.Nodes)
	}
}

func TestValidateSpec_SingleComponentRoot(t *testing.T) {
	r := ValidateSpec(`{"type":"callout","content":"hi","tone":"info"}`)
	if !r.OK {
		t.Fatalf("want OK for single component root, got %v", r.Errors)
	}
}

func TestValidateSpec_UnknownTypeAndMissingField(t *testing.T) {
	r := ValidateSpec(`{"items":[
		{"type":"evil"},
		{"type":"stat","label":"x"}
	]}`)
	if r.OK {
		t.Fatal("want not OK")
	}
	joined := strings.Join(r.Errors, "\n")
	if !strings.Contains(joined, `未知组件 type "evil"`) {
		t.Fatalf("missing unknown-type error: %s", joined)
	}
	if !strings.Contains(joined, "stat 缺少必填字段 value") {
		t.Fatalf("missing required-field error: %s", joined)
	}
}

func TestValidateSpec_SyntaxError(t *testing.T) {
	r := ValidateSpec(`{oops`)
	if r.OK || len(r.Errors) == 0 {
		t.Fatalf("want syntax error, got %+v", r)
	}
}

func TestValidateSpec_DepthBudget(t *testing.T) {
	var raw strings.Builder
	raw.WriteString(`{"items":[]}`)
	r := ValidateSpec(raw.String())
	if r.OK {
		t.Fatal("empty items should fail")
	}
}

// P0-18：diff.diffs 是数据行数组（{path,oldText?,newText}），不是子节点数组。
// 修复前会被当成节点递归，逐行报「缺少 type 字段」。
func TestValidateSpec_DiffRowsAreDataNotNodes(t *testing.T) {
	raw := `{"items":[
		{"type":"diff","diffs":[
			{"path":"a.go","oldText":"x","newText":"y"},
			{"path":"b.go","newText":"z"}
		]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK for diff with data rows, got %v", r.Errors)
	}
	if r.Nodes != 1 {
		t.Fatalf("nodes = %d, want 1（diffs 数据行不占节点预算）", r.Nodes)
	}
}

// P0-18：chart.series 是数据序列（{label,data}），不是子节点数组。
func TestValidateSpec_ChartSeriesAreDataNotNodes(t *testing.T) {
	raw := `{"items":[
		{"type":"chart","kind":"bars",
		 "data":[{"label":"1月","value":98}],
		 "series":[{"label":"今年","data":[{"label":"1月","value":98}]}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK for chart with series, got %v", r.Errors)
	}
	if r.Nodes != 1 {
		t.Fatalf("nodes = %d, want 1（series 数据行不占节点预算）", r.Nodes)
	}
}

// 仅有 data 的 chart（kind 变体）同样必须合法。
func TestValidateSpec_ChartDataOnly(t *testing.T) {
	raw := `{"title":"占比","items":[
		{"type":"chart","kind":"donut","data":[{"label":"A","value":1},{"label":"B","value":2}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK for chart data only, got %v", r.Errors)
	}
	if r.Nodes != 1 {
		t.Fatalf("nodes = %d, want 1", r.Nodes)
	}
}

// P0-18：其余数据数组（select/radio.options 字符串、quiz.options 行对象、
// timeline.items 行对象、steps.steps 行对象）同样不是子节点数组。
func TestValidateSpec_OtherDataArraysAreNotNodes(t *testing.T) {
	raw := `{"items":[
		{"type":"select","label":"范围","options":["本周","本月"],"action":"range"},
		{"type":"radio","options":["A","B"],"group":"q1","answer":"A"},
		{"type":"timeline","items":[{"title":"立项","time":"1月"},{"title":"上线","desc":"ok"}]},
		{"type":"steps","steps":[{"title":"一","desc":"a"},{"title":"二"}]},
		{"type":"quiz","question":"?","options":[{"label":"A","correct":true},{"label":"B"}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK for data arrays, got %v", r.Errors)
	}
	if r.Nodes != 5 {
		t.Fatalf("nodes = %d, want 5", r.Nodes)
	}
}

// tabs/accordion 的行对象是数据行，只有行内的 items 才是节点：嵌套节点仍须校验。
func TestValidateSpec_TabsAndAccordionRows(t *testing.T) {
	raw := `{"items":[
		{"type":"tabs","tabs":[{"label":"A","items":[{"type":"text","content":"x"}]}]},
		{"type":"accordion","items":[{"title":"T","items":[{"type":"badge","label":"b"}]}]}
	]}`
	r := ValidateSpec(raw)
	if !r.OK {
		t.Fatalf("want OK for tabs/accordion rows, got %v", r.Errors)
	}
	if r.Nodes != 4 {
		t.Fatalf("nodes = %d, want 4", r.Nodes)
	}
}

// 反向用例：真正的节点数组里塞了缺 type 的元素，必须仍然报错（证明校验没被关掉）。
func TestValidateSpec_NodeArrayStillRequiresType(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"row.items", `{"type":"row","items":[{"content":"x"}]}`},
		{"card.items", `{"items":[{"type":"card","items":[{"content":"x"}]}]}`},
		{"tabs 行 items", `{"type":"tabs","tabs":[{"label":"A","items":[{"content":"x"}]}]}`},
		{"accordion 行 items", `{"type":"accordion","items":[{"title":"T","items":[{"content":"x"}]}]}`},
		{"list 对象元素", `{"type":"list","items":[{"content":"x"}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := ValidateSpec(c.raw)
			if r.OK {
				t.Fatalf("want error for node without type: %s", c.raw)
			}
			if !strings.Contains(strings.Join(r.Errors, "\n"), "缺少 type 字段") {
				t.Fatalf("want 缺少 type 字段, got %v", r.Errors)
			}
		})
	}
}

// 反向用例：数据数组改做形状校验，形状错误（而非被无视）必须报出来。
func TestValidateSpec_DataShapeErrorsReported(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"series 缺 data", `{"type":"chart","data":[],"series":[{"label":"今年"}]}`},
		{"series 缺 label", `{"type":"chart","data":[],"series":[{"data":[{"label":"A","value":1}]}]}`},
		{"chart 点 value 非数字", `{"type":"chart","data":[{"label":"A","value":"1"}]}`},
		{"diff 缺 path", `{"type":"diff","diffs":[{"newText":"y"}]}`},
		{"diff 缺 newText", `{"type":"diff","diffs":[{"path":"a.go"}]}`},
		{"timeline 行缺 title", `{"type":"timeline","items":[{"desc":"x"}]}`},
		{"steps 行缺 title", `{"type":"steps","steps":[{"desc":"x"}]}`},
		{"quiz 行缺 label", `{"type":"quiz","question":"?","options":[{"label":"A"},{"correct":true}]}`},
		{"select 选项非字符串", `{"type":"select","options":["A",1]}`},
		{"tabs 行缺 label", `{"type":"tabs","tabs":[{"items":[]}]}`},
		{"accordion 行缺 title", `{"type":"accordion","items":[{"items":[]}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := ValidateSpec(c.raw)
			if r.OK {
				t.Fatalf("want error for malformed data row: %s", c.raw)
			}
			if len(r.Errors) == 0 {
				t.Fatal("want non-empty errors")
			}
		})
	}
}

// 错误条数上限仍是 30（每节点 2 条缺失错误，40 个节点只报 30 条）。
func TestValidateSpec_ErrorCapIs30(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; i < 40; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"type":"stat"}`)
	}
	b.WriteString(`]}`)
	r := ValidateSpec(b.String())
	if len(r.Errors) != 30 {
		t.Fatalf("errors = %d, want 30", len(r.Errors))
	}
}
