package schedule

// blocks_test.go — 斑马复刻四块承载位（reviews / aoa / networkStyle / views）的
// 落盘往返与 fail-closed 校验用例。
//
// 目标：证明"能存能读不丢"（往返后逐字段一致）+ "坏形状不许落盘"。
// 与前端 blocks.test.ts 互为镜像（同一组非法样例）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func intp(v int) *int         { return &v }
func boolp(v bool) *bool      { return &v }
func f64p(v float64) *float64 { return &v }

// blocksProject 带全部四个区块的工程（合法样例，供往返与"接受"用例共用）。
func blocksProject() Project {
	p := sampleProject()
	p.Reviews = []Review{
		{At: 0, Date: "2026-09-01", Actuals: map[string]ReviewActual{
			"A": {Start: intp(0), Percent: intp(100)},
		}},
		{At: 5, Date: "2026-09-08", Frozen: true, Note: "首次拉直", Actuals: map[string]ReviewActual{
			"A": {Start: intp(0), Finish: intp(2), Percent: intp(100)},
			"B": {Start: intp(3), Percent: intp(40)},
		}},
	}
	p.Aoa = &AoaGraph{
		Dots: []AoaDot{
			{ID: "S", Num: 1, IsBegin: true, Layer: intp(0)},
			{ID: "e1", Num: 2, Layer: intp(1)},
			{ID: "T", Num: 3, IsEnd: true, Layer: intp(3)},
		},
		Arrows: []AoaArrow{
			{ID: "a1", From: "S", To: "e1", Kind: "task", TaskID: "A"},
			{ID: "a2", From: "e1", To: "T", Kind: "task", TaskID: "B", Lag: 1},
			{ID: "a3", From: "S", To: "T", Kind: "dummy", Lag: -1},
		},
		BeginDotID: "S",
	}
	p.NetworkStyle = &NetworkStyle{
		Mode: "timescaled", DotStyle: "circle", ArrowStyle: "hollow", NamePos: "auto",
		NodeRadius: intp(16), HeadHeight: intp(8), LayerHeight1: intp(112), LayerHeight2: intp(60),
		Stagger: boolp(true), HideUnimportantLink: boolp(false),
		TickUnit: "monthDay", TickLines: "small", Density: "compact",
		AspectMax: f64p(7), MaxPages: intp(200),
	}
	p.Views = []View{
		{ID: "v1", Name: "计划编制", Kind: "gantt", Hide: []string{"code"}, GroupBy: []string{"level"}},
		{ID: "v2", Name: "网络图检视", Kind: "aoa", RenderMode: "logic",
			Sort: &ViewSort{Field: "start", Dir: "asc"}, Filter: "critical"},
	}
	return p
}

// TestBlocksRoundTripSaved 走真实 Save/Load（含 Validate 与原子写盘），断言四块逐字段不丢。
func TestBlocksRoundTripSaved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blocks.gsched.json")
	in := blocksProject()
	if err := Save(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out.Reviews) != 2 || out.Reviews[1].At != 5 || !out.Reviews[1].Frozen ||
		out.Reviews[1].Note != "首次拉直" {
		t.Fatalf("reviews 未保真：%+v", out.Reviews)
	}
	if got := out.Reviews[1].Actuals["B"]; got.Percent == nil || *got.Percent != 40 || got.Finish != nil {
		t.Fatalf("review actuals 未保真：%+v", got)
	}
	if out.Aoa == nil || len(out.Aoa.Dots) != 3 || len(out.Aoa.Arrows) != 3 || out.Aoa.BeginDotID != "S" {
		t.Fatalf("aoa 未保真：%+v", out.Aoa)
	}
	if out.Aoa.Arrows[1].Lag != 1 || out.Aoa.Arrows[2].Lag != -1 || out.Aoa.Arrows[2].Kind != "dummy" {
		t.Fatalf("aoa arrows 未保真：%+v", out.Aoa.Arrows)
	}
	if out.NetworkStyle == nil || out.NetworkStyle.Stagger == nil || !*out.NetworkStyle.Stagger ||
		out.NetworkStyle.HideUnimportantLink == nil || *out.NetworkStyle.HideUnimportantLink ||
		out.NetworkStyle.AspectMax == nil || *out.NetworkStyle.AspectMax != 7 {
		t.Fatalf("networkStyle 未保真：%+v", out.NetworkStyle)
	}
	if len(out.Views) != 2 || out.Views[1].RenderMode != "logic" ||
		out.Views[1].Sort == nil || out.Views[1].Sort.Dir != "asc" || out.Views[1].Filter != "critical" {
		t.Fatalf("views 未保真：%+v", out.Views)
	}
}

// TestBlocksAbsentKeepsLegacyShape 缺省区块不写键（旧文件零迁移的正面证据）。
func TestBlocksAbsentKeepsLegacyShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.gsched.json")
	if err := Save(path, sampleProject()); err != nil {
		t.Fatalf("save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, key := range []string{`"reviews"`, `"aoa"`, `"networkStyle"`, `"views"`} {
		if strings.Contains(string(raw), key) {
			t.Fatalf("缺省区块不应写键 %s：%s", key, raw)
		}
	}
}

// TestValidateAcceptsAndRejectsBlocks 逐条非法样例必须被 Validate 拒绝（fail-closed）。
func TestValidateAcceptsAndRejectsBlocks(t *testing.T) {
	ok := blocksProject()
	if err := Validate(&ok); err != nil {
		t.Fatalf("合法样例应通过：%v", err)
	}

	cases := []struct {
		name string
		mut  func(p *Project)
		want string
	}{
		{"检视重复时间", func(p *Project) {
			p.Reviews = append(p.Reviews, Review{At: 0, Date: "2026-09-01", Actuals: map[string]ReviewActual{}})
		}, "检视时间重复"},
		{"检视日期形状错", func(p *Project) { p.Reviews[0].Date = "2026/09/01" }, "日期口径应为 YYYY-MM-DD"},
		{"检视时间为负", func(p *Project) { p.Reviews[0].At = -1 }, "检视时间非法"},
		{"实际完成早于开始", func(p *Project) {
			p.Reviews[1].Actuals["A"] = ReviewActual{Start: intp(3), Finish: intp(1)}
		}, "实际完成早于实际开始"},
		{"完成百分比越界", func(p *Project) { p.Reviews[1].Actuals["B"] = ReviewActual{Percent: intp(120)} }, "完成百分比越界"},
		{"点编号重复", func(p *Project) { p.Aoa.Dots[1].Num = 1 }, "点编号重复"},
		{"箭指向不存在的点", func(p *Project) { p.Aoa.Arrows[0].To = "nope" }, "头事件不存在"},
		{"箭自环", func(p *Project) { p.Aoa.Arrows[0].To = "S" }, "自环"},
		{"箭类型非法", func(p *Project) { p.Aoa.Arrows[0].Kind = "flow" }, "类型非法"},
		{"时距越界", func(p *Project) { p.Aoa.Arrows[0].Lag = 10000 }, "时距越界"},
		{"只有点没有箭", func(p *Project) { p.Aoa.Arrows = nil }, "须同时有点和箭"},
		{"起点 id 不在点集", func(p *Project) { p.Aoa.BeginDotID = "X" }, "起点事件 id 不在点集中"},
		{"图面模式非法", func(p *Project) { p.NetworkStyle.Mode = "fancy" }, "图面参数 mode 非法"},
		{"节点半径越界", func(p *Project) { p.NetworkStyle.NodeRadius = intp(999) }, "nodeRadius 越界"},
		{"纵横比越界", func(p *Project) { p.NetworkStyle.AspectMax = f64p(0) }, "aspectMax 越界"},
		{"视图 id 重复", func(p *Project) {
			p.Views = append(p.Views, View{ID: "v1", Name: "重复", Kind: "gantt"})
		}, "视图 id 重复"},
		{"视图类型非法", func(p *Project) { p.Views[0].Kind = "kanban" }, "类型非法"},
		{"视图排序方向非法", func(p *Project) { p.Views[1].Sort.Dir = "up" }, "排序方向非法"},
		{"视图显示模式非法", func(p *Project) { p.Views[1].RenderMode = "wireframe" }, "显示模式非法"},
	}
	for _, c := range cases {
		p := blocksProject()
		c.mut(&p)
		err := Validate(&p)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s：期望包含 %q，实际 %v", c.name, c.want, err)
		}
	}
}

// TestBlocksJSONShapeGuardsGoMirror 保证 Go 侧字段名与前端 types.ts 完全一致
// （手写契约镜像的护栏：改名会让前端写的键在 Go 里静默丢失）。
func TestBlocksJSONShapeGuardsGoMirror(t *testing.T) {
	raw, err := json.Marshal(blocksProject())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{
		`"reviews"`, `"at"`, `"date"`, `"frozen"`, `"actuals"`, `"note"`,
		`"aoa"`, `"dots"`, `"arrows"`, `"beginDotId"`, `"isBegin"`, `"isEnd"`, `"taskId"`, `"lag"`,
		`"networkStyle"`, `"dotStyle"`, `"arrowStyle"`, `"namePos"`, `"nodeRadius"`, `"headHeight"`,
		`"layerHeight1"`, `"layerHeight2"`, `"stagger"`, `"hideUnimportantLink"`,
		`"tickUnit"`, `"tickLines"`, `"density"`, `"aspectMax"`, `"maxPages"`,
		`"views"`, `"kind"`, `"groupBy"`, `"filter"`, `"sort"`, `"renderMode"`,
	} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("Go 侧缺键 %s（前端 types.ts 已定义）：%s", key, raw)
		}
	}
}
