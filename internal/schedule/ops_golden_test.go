package schedule

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// diff 确认闭环刀D（v4.149）：Go/TS 对拍 golden fixture。
//
// 本测试是 ops 通道口径的**唯一权威源**：以内部用例序列跑 Go ApplyOps，把
// after 状态（或错误）写入 frontend/src/schedule/ops_golden.fixture.json；
// vitest（opsSim.golden.test.ts）读同一份文件跑 TS simulateOps 逐例比对
// 「是否失败 + after 状态」。任何一侧语义漂移都会炸掉对端测试——这就是
// 「批的是 A，落的是 B」风险的对冲主阵地（设计 §5）。
//
// 重新生成：go test ./internal/schedule/ -run TestApplyOpsGolden -update-golden
// （改了 Go ops 语义后必须重生成并提交 fixture，让 TS 例试图对齐失败显式化）。

var updateGolden = flag.Bool("update-golden", false, "重新生成 ops_golden.fixture.json")

type goldenCase struct {
	Name  string   `json:"name"`
	Base  Project  `json:"base"`
	Ops   []Op     `json:"ops"`
	Err   *string  `json:"error,omitempty"`
	After *Project `json:"after,omitempty"`
}

type goldenFile struct {
	Cases []goldenCase `json:"cases"`
}

const goldenRelPath = "../../frontend/src/schedule/ops_golden.fixture.json"

func cloneProject(p Project) Project {
	raw, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	var out Project
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func gStrPtr(s string) *string   { return &s }
func gIntPtr(i int) *int         { return &i }
func gF64Ptr(f float64) *float64 { return &f }
func gBoolPtr(b bool) *bool      { return &b }

func goldenBase() Project {
	return Project{
		Name:      "对拍工程",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "A", Name: "挖土", Duration: 2, Level: 1, Progress: 0},
			{ID: "B", Name: "垫层", Duration: 3, Level: 1, Progress: 40},
		},
		Links: []Link{{From: "A", To: "B", Type: FS, Lag: 0}},
	}
}

func goldenBaseGrouped() Project {
	p := goldenBase()
	p.Tasks = []Task{
		{ID: "G", Name: "基础工程", Duration: 0, Level: 0, Progress: 0},
		{ID: "A", Name: "挖土", Duration: 2, Level: 1, Progress: 0},
		{ID: "B", Name: "垫层", Duration: 3, Level: 1, Progress: 40},
	}
	p.Links = []Link{{From: "A", To: "B", Type: FS, Lag: 0}}
	return p
}

func goldenBaseResources() Project {
	p := goldenBase()
	p.Resources = []Resource{
		{ID: "r1", Name: "挖机", Type: ResWork, StandardRate: 800, CostPerUse: 200, MaxUnits: 2},
		{ID: "r2", Name: "混凝土", Type: ResMaterial, Unit: "m³", StandardRate: 450},
	}
	p.Assignments = []Assignment{
		{TaskID: "A", ResourceID: "r1", Units: gF64Ptr(1.5)},
		{TaskID: "B", ResourceID: "r2", Quantity: 30},
	}
	return p
}

func goldenWithBaseline() Project {
	p := goldenBase()
	b, err := SnapshotBaseline(&p, "2026-09-07 08:00", "开工版")
	if err != nil {
		panic(err)
	}
	p.Baseline = b
	return p
}

func goldenCases() []goldenCase {
	return []goldenCase{
		// ── upsert_task ──
		{Name: "upsert 追加表尾", Base: goldenBase(), Ops: []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "基础", Duration: 2, Level: 1}}}},
		{"upsert afterId 插入", goldenBase(), []Op{{Type: "upsert_task", AfterID: "A", Task: &Task{ID: "C", Name: "基础", Duration: 2, Level: 1}}}, nil, nil},
		{"upsert 同 id 整量替换", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "A", Name: "挖土方", Duration: 5, Level: 1, Progress: 20}}}, nil, nil},
		{"upsert 零值语义（缺字段=0/分组）", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "Z"}}}, nil, nil},
		{"upsert afterId 不存在", goldenBase(), []Op{{Type: "upsert_task", AfterID: "NOPE", Task: &Task{ID: "C", Name: "x", Duration: 1, Level: 1}}}, nil, nil},
		{"upsert 层级非法", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "x", Duration: 1, Level: 2}}}, nil, nil},
		{"upsert 分组行禁固定成本", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "G", Name: "分组", Level: 0, FixedCost: 100}}}, nil, nil},
		{"upsert 负固定成本", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "x", Duration: 1, Level: 1, FixedCost: -5}}}, nil, nil},
		// ── patch_task ──
		{"patch 多字段", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Name: gStrPtr("挖土方"), Duration: gIntPtr(4), Progress: gIntPtr(50)}}}, nil, nil},
		{"patch 模式手动+锁定开始", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Mode: (*TaskMode)(gStrPtr("manual")), ManualStart: gIntPtr(2)}}}, nil, nil},
		{"patch 里程碑", goldenBase(), []Op{{Type: "patch_task", ID: "B", Patch: &patchTask{IsMilestone: gBoolPtr(true), Duration: gIntPtr(0)}}}, nil, nil},
		{"patch 固定成本", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: gF64Ptr(120.5)}}}, nil, nil},
		{"patch 无字段", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{}}}, nil, nil},
		{"patch 任务不存在", goldenBase(), []Op{{Type: "patch_task", ID: "NOPE", Patch: &patchTask{Duration: gIntPtr(1)}}}, nil, nil},
		{"patch 分组行禁固定成本", goldenBaseGrouped(), []Op{{Type: "patch_task", ID: "G", Patch: &patchTask{FixedCost: gF64Ptr(9)}}}, nil, nil},
		{"patch 负工期", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(-1)}}}, nil, nil},
		// ── 双工期（v4.151 刀2）：durationUnit ──
		{"patch 单位→cd+工期", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd")), Duration: gIntPtr(28)}}}, nil, nil},
		{"patch 单位→wd 显式", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "养护", Duration: 28, Level: 1, DurationUnit: UnitCd}}, {Type: "patch_task", ID: "H", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("wd"))}}}, nil, nil},
		{"patch 单位非法", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("week"))}}}, nil, nil},
		{"patch 单位空串=no-op", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr(""))}}}, nil, nil},
		{"patch cd 分组行拒绝", goldenBaseGrouped(), []Op{{Type: "patch_task", ID: "G", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd"))}}}, nil, nil},
		{"patch cd 里程碑拒绝", goldenBase(), []Op{{Type: "patch_task", ID: "B", Patch: &patchTask{IsMilestone: gBoolPtr(true)}}, {Type: "patch_task", ID: "B", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd"))}}}, nil, nil},
		{"patch 单位cd+工期超上限", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd")), Duration: gIntPtr(3651)}}}, nil, nil},
		{"upsert cd 任务", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "养护", Duration: 28, Level: 1, DurationUnit: UnitCd}}}, nil, nil},
		{"upsert cd 里程碑拒绝", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "M", Name: "m", Level: 1, IsMilestone: true, DurationUnit: UnitCd}}}, nil, nil},
		{"upsert cd 分组行拒绝", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "G2", Name: "g", Level: 0, DurationUnit: UnitCd}}}, nil, nil},
		{"upsert cd 超上限", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "x", Duration: 4000, Level: 1, DurationUnit: UnitCd}}}, nil, nil},
		{"upsert 单位非法", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "x", Duration: 1, Level: 1, DurationUnit: "week"}}}, nil, nil},
		{"cd 全链 upsert+FS+基线", goldenBase(), []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "养护", Duration: 28, Level: 1, DurationUnit: UnitCd}}, {Type: "set_links", ToID: "H", Links: []opLink{{From: "B"}}}, {Type: "set_baseline", SavedAt: "2026-09-07 08:00", BaselineName: "含养护"}}, nil, nil},
		// ── remove_task ──
		{"remove 分组级联子孙与搭接", goldenBaseGrouped(), []Op{{Type: "remove_task", ID: "G"}}, nil, nil},
		{"remove 叶任务清相关搭接", goldenBase(), []Op{{Type: "remove_task", ID: "A"}}, nil, nil},
		{"remove 不存在", goldenBase(), []Op{{Type: "remove_task", ID: "NOPE"}}, nil, nil},
		// ── set_links ──
		{"set_links 整体替换+缺省 FS", goldenBase(), []Op{{Type: "set_links", ToID: "B", Links: []opLink{{From: "A", Lag: 1}, {From: "B", Type: SS, Lag: 0}}}}, nil, nil},
		{"set_links 自前置", goldenBase(), []Op{{Type: "set_links", ToID: "B", Links: []opLink{{From: "B"}}}}, nil, nil},
		{"set_links 前置不存在", goldenBase(), []Op{{Type: "set_links", ToID: "B", Links: []opLink{{From: "NOPE"}}}}, nil, nil},
		// ── set_meta ──
		{"set_meta 全字段含日历归一", goldenBase(), []Op{{Type: "set_meta", Name: "新名", StartDate: "2026-10-01", Calendar: &Calendar{Workweek: []int{1, 2, 3, 4, 5, 6}}, Deadline: gStrPtr("2026-10-31")}}, nil, nil},
		{"set_meta 空串清竣工", goldenBase(), []Op{{Type: "set_meta", Deadline: gStrPtr("")}}, nil, nil},
		{"set_meta null 竣工=不动", goldenBase(), []Op{{Type: "set_meta", Name: "只改名", Deadline: nil}}, nil, nil},
		{"set_meta 日期口径错", goldenBase(), []Op{{Type: "set_meta", StartDate: "2026-9-1"}}, nil, nil},
		{"set_meta 无字段", goldenBase(), []Op{{Type: "set_meta"}}, nil, nil},
		// ── auto_chain ──
		{"auto_chain 三叶补二", goldenBaseGrouped(), []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "钢筋", Duration: 2, Level: 1}}, {Type: "auto_chain"}}, nil, nil},
		{"auto_chain 跳过手动", goldenBaseGrouped(), []Op{{Type: "patch_task", ID: "B", Patch: &patchTask{Mode: (*TaskMode)(gStrPtr("manual"))}}, {Type: "auto_chain"}}, nil, nil},
		{"auto_chain 无可补", goldenBase(), []Op{{Type: "auto_chain"}}, nil, nil},
		// ── set_baseline / clear_baseline ──
		{"set_baseline 固化", goldenBase(), []Op{{Type: "set_baseline", SavedAt: "2026-09-07 08:00", BaselineName: "对拍版"}}, nil, nil},
		{"set_baseline 缺 savedAt", goldenBase(), []Op{{Type: "set_baseline", BaselineName: "x"}}, nil, nil},
		{"set_baseline CPM 拒绝", Project{Name: "环", StartDate: "2026-09-07", Tasks: []Task{{ID: "A", Name: "a", Duration: 1, Level: 1}, {ID: "B", Name: "b", Duration: 1, Level: 1}}, Links: []Link{{From: "A", To: "B", Type: FS}, {From: "B", To: "A", Type: FS}}}, []Op{{Type: "set_baseline", SavedAt: "2026-09-07 08:00"}}, nil, nil},
		{"clear_baseline 清除", goldenWithBaseline(), []Op{{Type: "clear_baseline"}}, nil, nil},
		{"clear_baseline 无基线", goldenBase(), []Op{{Type: "clear_baseline"}}, nil, nil},
		// ── 资源与分配 ──
		{"upsert_resource 新增", goldenBaseResources(), []Op{{Type: "upsert_resource", Resource: &Resource{ID: "r3", Name: "吊车", Type: ResCost, CostPerUse: 500}}}, nil, nil},
		{"upsert_resource 整量替换", goldenBaseResources(), []Op{{Type: "upsert_resource", Resource: &Resource{ID: "r1", Name: "挖机2", Type: ResWork, StandardRate: 900}}}, nil, nil},
		{"upsert_resource 类型非法", goldenBaseResources(), []Op{{Type: "upsert_resource", Resource: &Resource{ID: "r9", Name: "x", Type: "nope"}}}, nil, nil},
		{"patch_resource 多字段含清单位", goldenBaseResources(), []Op{{Type: "patch_resource", ID: "r2", ResourcePatch: &patchResource{Unit: gStrPtr("t"), StandardRate: gF64Ptr(500)}}}, nil, nil},
		{"patch_resource 无字段", goldenBaseResources(), []Op{{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{}}}, nil, nil},
		{"patch_resource 不存在", goldenBaseResources(), []Op{{Type: "patch_resource", ID: "NOPE", ResourcePatch: &patchResource{Name: gStrPtr("x")}}}, nil, nil},
		{"remove_resource 级联分配", goldenBaseResources(), []Op{{Type: "remove_resource", ID: "r1"}}, nil, nil},
		{"set_assignments 整体替换", goldenBaseResources(), []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "r2", Quantity: 10}, {ResourceID: "r1", Units: gF64Ptr(2)}}}}, nil, nil},
		{"set_assignments 分组行拒绝", goldenBaseGrouped(), []Op{{Type: "set_assignments", TaskID: "G", Assignments: []Assignment{{ResourceID: "r1"}}}}, nil, nil},
		{"set_assignments 资源不存在", goldenBaseResources(), []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "NOPE"}}}}, nil, nil},
		{"set_assignments 重复对", goldenBaseResources(), []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "r1"}, {ResourceID: "r1", Units: gF64Ptr(2)}}}}, nil, nil},
		{"set_assignments 外来 taskId", goldenBaseResources(), []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{TaskID: "B", ResourceID: "r1"}}}}, nil, nil},
		// ── 组合与fail-closed ──
		{"未知操作类型", goldenBase(), []Op{{Type: "nope"}}, nil, nil},
		{"多 op 中途失败（半途状态不外泄）", goldenBase(), []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(9)}}, {Type: "patch_task", ID: "NOPE", Patch: &patchTask{Duration: gIntPtr(1)}}}, nil, nil},
	}
}

func TestApplyOpsGoldenFixture(t *testing.T) {
	cases := goldenCases()
	file := goldenFile{Cases: make([]goldenCase, 0, len(cases))}
	for _, tc := range cases {
		p := cloneProject(tc.Base)
		_, err := ApplyOps(&p, tc.Ops)
		gc := goldenCase{Name: tc.Name, Base: tc.Base, Ops: tc.Ops}
		if err != nil {
			msg := err.Error()
			gc.Err = &msg
		} else {
			after := cloneProject(p)
			gc.After = &after
		}
		file.Cases = append(file.Cases, gc)
	}

	path := filepath.Join("..", "..", "frontend", "src", "schedule", "ops_golden.fixture.json")
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')

	if *updateGolden {
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("golden fixture 已重生成：%s", path)
		return
	}

	existing, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 golden fixture 失败（先跑 -update-golden 生成）：%v", err)
	}
	var want goldenFile
	if err := json.Unmarshal(existing, &want); err != nil {
		t.Fatalf("golden fixture 解析失败：%v", err)
	}
	if len(want.Cases) != len(file.Cases) {
		t.Fatalf("用例数漂移：fixture=%d computed=%d（-update-golden 重生成）", len(want.Cases), len(file.Cases))
	}
	for i, got := range file.Cases {
		w := want.Cases[i]
		if w.Name != got.Name {
			t.Errorf("case %d 名称漂移：%q vs %q", i, w.Name, got.Name)
			continue
		}
		if !reflect.DeepEqual(w.Err, got.Err) {
			t.Errorf("case %q 错误口径漂移：\n fixture=%v\n computed=%v", got.Name, w.Err, got.Err)
		}
		if !reflect.DeepEqual(w.After, got.After) {
			wb, _ := json.Marshal(w.After)
			gb, _ := json.Marshal(got.After)
			t.Errorf("case %q after 状态漂移：\n fixture=%s\n computed=%s", got.Name, wb, gb)
		}
	}
}
