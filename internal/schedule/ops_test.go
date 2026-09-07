// schedule/ops_test.go — 资源/分配增量操作用例（v4.122.0 资源成本刀2）。
//
// 与 ops.go 扩枚举同批：upsert_resource / patch_resource（指针三态）/
// remove_resource（级联删分配）/ set_assignments（整体替换）/ patch_task.fixedCost；
// fail-closed 口径与 Validate 同律（分组行禁挂/悬空引用/负值拒绝）。
package schedule

import (
	"math"
	"strings"
	"testing"
)

func resOpsProject() Project {
	return Project{
		Name:      "资源Ops",
		StartDate: "2026-01-05",
		Tasks: []Task{
			{ID: "G", Name: "分组", Level: 0},
			{ID: "A", Name: "A", Duration: 3, Level: 1},
			{ID: "B", Name: "B", Duration: 2, Level: 1},
		},
		Links: []Link{lnk("A", "B", FS, 0)},
	}
}

func TestOpsUpsertResource(t *testing.T) {
	p := resOpsProject()
	sums, err := ApplyOps(&p, []Op{
		{Type: "upsert_resource", Resource: &Resource{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 300}},
		{Type: "upsert_resource", Resource: &Resource{ID: "m1", Name: "钢材", Type: ResMaterial, StandardRate: 55, Unit: "t"}},
		// 同 id 再 upsert = 整量替换
		{Type: "upsert_resource", Resource: &Resource{ID: "r1", Name: "普工", Type: ResWork, StandardRate: 200, CostPerUse: 100}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(sums) != 3 || !strings.Contains(sums[2], "更新资源") {
		t.Fatalf("替换摘要 = %v", sums)
	}
	if len(p.Resources) != 2 || p.Resources[0].Name != "普工" || p.Resources[0].CostPerUse != 100 {
		t.Fatalf("按 id 替换失败：%+v", p.Resources)
	}
	if p.Resources[0].Unit != "" || p.Resources[1].Unit != "t" {
		t.Fatalf("unit 往返不一致：%+v", p.Resources)
	}

	// fail-closed：缺载荷/空 id/非法类型/负费率
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_resource"}}); err == nil {
		t.Fatal("缺 resource 应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_resource", Resource: &Resource{ID: " ", Name: "x", Type: ResWork}}}); err == nil {
		t.Fatal("空 id 应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_resource", Resource: &Resource{ID: "r9", Name: "x", Type: ResourceType("weird")}}}); err == nil {
		t.Fatal("非法类型应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_resource", Resource: &Resource{ID: "r9", Name: "x", Type: ResWork, StandardRate: -1}}}); err == nil {
		t.Fatal("负费率应拒绝")
	}
}

func TestOpsPatchResourceThreeState(t *testing.T) {
	p := resOpsProject()
	p.Resources = []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}}
	// 指针三态：只动给出的字段
	sums, err := ApplyOps(&p, []Op{
		{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{
			Name:         ptrStr("塔吊"),
			StandardRate: floatPtr(888.5),
		}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(sums[0], "888.5") || !strings.Contains(sums[0], "元/工日") {
		t.Fatalf("费率摘要缺单位标注：%q", sums[0])
	}
	r := p.Resources[0]
	if r.Name != "塔吊" || r.StandardRate != 888.5 || r.Type != ResWork || r.CostPerUse != 0 || r.MaxUnits != 0 {
		t.Fatalf("三态 patch 失败（未给出字段应不动）：%+v", r)
	}
	// 类型切到材料后费率单位标注跟随（元/单位）；空串清除 Unit
	p.Resources[0].Unit = "t"
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{
		Type: resTypePtr(ResMaterial), Unit: ptrStr(""),
	}}}); err != nil {
		t.Fatalf("patch type/unit: %v", err)
	}
	if p.Resources[0].Type != ResMaterial || p.Resources[0].Unit != "" {
		t.Fatalf("类型/清单位失败：%+v", p.Resources[0])
	}
	// fail-closed
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "zz", ResourcePatch: &patchResource{}}}); err == nil {
		t.Fatal("不存在资源应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "r1"}}); err == nil {
		t.Fatal("未提供任何字段应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{StandardRate: floatPtr(-0.5)}}}); err == nil {
		t.Fatal("负费率应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{CostPerUse: floatPtr(math.NaN())}}}); err == nil {
		t.Fatal("NaN 每次使用应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_resource", ID: "r1", ResourcePatch: &patchResource{MaxUnits: floatPtr(math.Inf(1))}}}); err == nil {
		t.Fatal("Inf 上限应拒绝")
	}
}

func TestOpsRemoveResourceCascadesAssignments(t *testing.T) {
	p := resOpsProject()
	p.Resources = []Resource{
		{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100},
		{ID: "r2", Name: "钢材", Type: ResMaterial, StandardRate: 55},
	}
	p.Assignments = []Assignment{
		{TaskID: "A", ResourceID: "r1"},
		{TaskID: "B", ResourceID: "r1"},
		{TaskID: "B", ResourceID: "r2", Quantity: 10},
	}
	sums, err := ApplyOps(&p, []Op{{Type: "remove_resource", ID: "r1"}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(sums[0], "级联") && !strings.Contains(sums[0], "2 条分配") {
		t.Fatalf("级联摘要 = %q", sums[0])
	}
	if len(p.Resources) != 1 || p.Resources[0].ID != "r2" {
		t.Fatalf("资源未删对：%+v", p.Resources)
	}
	if len(p.Assignments) != 1 || p.Assignments[0].ResourceID != "r2" {
		t.Fatalf("级联删分配失败：%+v", p.Assignments)
	}
	if err := Validate(&p); err != nil {
		t.Fatalf("删除后应过校验：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "remove_resource", ID: "r1"}}); err == nil {
		t.Fatal("不存在资源应拒绝")
	}
}

func TestOpsSetAssignmentsReplaces(t *testing.T) {
	p := resOpsProject()
	p.Resources = []Resource{
		{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100},
		{ID: "r2", Name: "钢材", Type: ResMaterial, StandardRate: 55},
		{ID: "r3", Name: "差旅", Type: ResCost},
	}
	p.Assignments = []Assignment{
		{TaskID: "A", ResourceID: "r1"},
		{TaskID: "B", ResourceID: "r1", Quantity: 3},
	}
	// 整体替换 A 的分配集：r1 → {r2, r3}；B 的分配原样保留
	_, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{
		{ResourceID: "r2", Quantity: 10},
		{ResourceID: "r3", Amount: 300},
	}}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	aAssign := []Assignment{}
	for _, a := range p.Assignments {
		if a.TaskID == "A" {
			aAssign = append(aAssign, a)
		}
	}
	if len(aAssign) != 2 || aAssign[0].ResourceID != "r2" || aAssign[0].Quantity != 10 || aAssign[1].ResourceID != "r3" || aAssign[1].Amount != 300 {
		t.Fatalf("A 分配集未整体替换：%+v", aAssign)
	}
	bKept := 0
	for _, a := range p.Assignments {
		if a.TaskID == "B" && a.ResourceID == "r1" && a.Quantity == 3 {
			bKept++
		}
	}
	if bKept != 1 {
		t.Fatalf("他任务分配应原样保留：%+v", p.Assignments)
	}
	if err := Validate(&p); err != nil {
		t.Fatalf("替换后应过校验：%v", err)
	}
	// 空集 = 清空该任务分配
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{}}}); err != nil {
		t.Fatalf("清空: %v", err)
	}
	for _, a := range p.Assignments {
		if a.TaskID == "A" {
			t.Fatalf("清空后不应残留 A 的分配：%+v", p.Assignments)
		}
	}
	// 载荷里 taskId 缺省归一到目标任务；显式他任务 id 拒绝
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "r1", TaskID: "B"}}}}); err == nil {
		t.Fatal("混入他任务分配应拒绝")
	}
}

func TestOpsSetAssignmentsRejects(t *testing.T) {
	p := resOpsProject()
	p.Resources = []Resource{{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100}}
	// 分组行禁挂分配
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "G", Assignments: []Assignment{{ResourceID: "r1"}}}}); err == nil || !strings.Contains(err.Error(), "禁止挂分配") {
		t.Fatalf("分组行应拒绝：%v", err)
	}
	// 悬空资源引用
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "ghost"}}}}); err == nil || !strings.Contains(err.Error(), "资源不存在") {
		t.Fatalf("悬空资源应拒绝：%v", err)
	}
	// 任务不存在
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "zz"}}); err == nil {
		t.Fatal("不存在任务应拒绝")
	}
	// 集内 (taskId,resourceId) 重复
	if _, err := ApplyOps(&p, []Op{{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{
		{ResourceID: "r1"}, {ResourceID: "r1"},
	}}}); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复分配应拒绝：%v", err)
	}
}

func TestOpsPatchTaskFixedCost(t *testing.T) {
	p := resOpsProject()
	sums, err := ApplyOps(&p, []Op{
		{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: floatPtr(120.5)}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(sums[0], "固定成本") || !strings.Contains(sums[0], "120.5") {
		t.Fatalf("固定成本摘要 = %q", sums[0])
	}
	if p.Tasks[1].FixedCost != 120.5 {
		t.Fatalf("fixedCost 未写入：%+v", p.Tasks[1])
	}
	// 清零（指针三态允许显式 0）
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: floatPtr(0)}}}); err != nil {
		t.Fatalf("清零: %v", err)
	}
	if p.Tasks[1].FixedCost != 0 {
		t.Fatalf("清零失败：%+v", p.Tasks[1])
	}
	// fail-closed：负值/NaN/Inf/分组行
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: floatPtr(-1)}}}); err == nil {
		t.Fatal("负固定成本应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: floatPtr(math.NaN())}}}); err == nil {
		t.Fatal("NaN 固定成本应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "G", Patch: &patchTask{FixedCost: floatPtr(50)}}}); err == nil || !strings.Contains(err.Error(), "禁止固定成本") {
		t.Fatalf("分组行固定成本应拒绝：%v", err)
	}
}

func TestOpsUpsertTaskRejectsGroupFixedCost(t *testing.T) {
	p := resOpsProject()
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "G2", Name: "分组2", Level: 0, FixedCost: 100}}}); err == nil || !strings.Contains(err.Error(), "禁止固定成本") {
		t.Fatalf("分组行带固定成本应拒绝：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "X", Name: "X", Level: 1, FixedCost: math.Inf(-1)}}}); err == nil {
		t.Fatal("Inf 固定成本应拒绝")
	}
	// 合法叶任务带固定成本照常
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "C", Level: 1, Duration: 1, FixedCost: 88}}}); err != nil {
		t.Fatalf("叶任务固定成本不应拒绝：%v", err)
	}
}

// TestOpsResourceOpsEndToEnd 通过纯 ops 建出含资源/分配/固定成本的计划，
// 落 Validate + ComputeCosts（刀1 闸不得被刀2 放松）。
func TestOpsResourceOpsEndToEnd(t *testing.T) {
	p := resOpsProject()
	_, err := ApplyOps(&p, []Op{
		{Type: "upsert_resource", Resource: &Resource{ID: "r1", Name: "人力", Type: ResWork, StandardRate: 100, CostPerUse: 200}},
		{Type: "upsert_resource", Resource: &Resource{ID: "m1", Name: "钢材", Type: ResMaterial, StandardRate: 55}},
		{Type: "patch_task", ID: "A", Patch: &patchTask{FixedCost: floatPtr(120.5)}},
		{Type: "set_assignments", TaskID: "A", Assignments: []Assignment{{ResourceID: "r1"}}},
		{Type: "set_assignments", TaskID: "B", Assignments: []Assignment{{ResourceID: "m1", Quantity: 10}}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if err := Validate(&p); err != nil {
		t.Fatalf("validate: %v", err)
	}
	r := ComputeCosts(p, ComputeCpm(p.Tasks, p.Links))
	// A = 120.5 + (3×1×100+200) = 620.5；B = 10×55 = 550；总 = 1170.5
	if !r.OK || r.Total != 1170.5 || r.Rows["A"].Total != 620.5 || r.Rows["B"].Total != 550 {
		t.Fatalf("成本 rollup = %+v, want A 620.5 / B 550 / total 1170.5", r)
	}
	// 成本叙事与 rollup 同源（Top 降序：A 620.5 > B 550）
	n := p.CostNarrative(ComputeCpm(p.Tasks, p.Links))
	if !n.OK || n.Total != 1170.5 || len(n.TopTasks) != 2 || n.TopTasks[0].ID != "A" || n.TopTasks[0].Total != 620.5 {
		t.Fatalf("叙事 = ok:%v total:%v top:%+v", n.OK, n.Total, n.TopTasks)
	}
	if len(n.Unpriced) != 0 {
		t.Fatalf("全部已定价不应有未定价：%+v", n.Unpriced)
	}
}

// ptrStr / resTypePtr ops_test 局部指针助手（floatPtr 复用 cost_test.go）。
func ptrStr(v string) *string                 { return &v }
func resTypePtr(t ResourceType) *ResourceType { return &t }

// TestOpsPatchTaskDurationUnit 双工期刀2（v4.151）：patch/upsert 的工期口径
// 三态、守卫与回执文案（语义对拍在 golden，本测钉回执口径词）。
func TestOpsPatchTaskDurationUnit(t *testing.T) {
	p := Project{
		Name:      "口径样板",
		StartDate: "2026-09-07",
		Tasks:     []Task{{ID: "A", Name: "挖土", Duration: 2, Level: 1}, {ID: "G", Name: "分组", Level: 0}},
	}
	// 单位→cd + 工期：回执带「日历天（自然日定时）」与「日历天」
	sums, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{
		DurationUnit: (*DurationUnit)(gStrPtr("cd")), Duration: gIntPtr(28),
	}}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(sums[0], "工期口径→日历天（自然日定时）") || !strings.Contains(sums[0], "28 日历天") {
		t.Fatalf("cd 摘要 = %q", sums[0])
	}
	if p.Tasks[0].DurationUnit != UnitCd || p.Tasks[0].Duration != 28 {
		t.Fatalf("未写入：%+v", p.Tasks[0])
	}
	// cd 任务再调工期：工期词=日历天
	sums, err = ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(30)}}})
	if err != nil || !strings.Contains(sums[0], "28→30 日历天") {
		t.Fatalf("cd 工期摘要 = %q err=%v", sums, err)
	}
	// 单位→wd：回执「工期口径→工作日」
	if sums, err = ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("wd"))}}}); err != nil || !strings.Contains(sums[0], "工期口径→工作日") {
		t.Fatalf("wd 摘要 = %q err=%v", sums, err)
	}
	if p.Tasks[0].DurationUnit != UnitWd {
		t.Fatalf("wd 未写入：%+v", p.Tasks[0])
	}
	// 守卫：非法枚举/空串 no-op/分组行 cd/cd 超上限
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("week"))}}}); err == nil || !strings.Contains(err.Error(), "工期单位非法") {
		t.Fatalf("非法单位应拒绝：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr(""))}}}); err == nil || !strings.Contains(err.Error(), "未提供任何字段") {
		t.Fatalf("空串应 no-op 报无字段：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "G", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd"))}}}); err == nil || !strings.Contains(err.Error(), "分组行") {
		t.Fatalf("分组行 cd 应拒绝：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: (*DurationUnit)(gStrPtr("cd")), Duration: gIntPtr(3651)}}}); err == nil || !strings.Contains(err.Error(), "超上限") {
		t.Fatalf("cd 超上限应拒绝：%v", err)
	}
	// upsert：cd 新增回执=日历天；里程碑/非法枚举拒绝
	sums, err = ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "H", Name: "养护", Duration: 28, Level: 1, DurationUnit: UnitCd}}})
	if err != nil || !strings.Contains(sums[0], "28 日历天") {
		t.Fatalf("upsert cd 摘要 = %q err=%v", sums, err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "M", Name: "m", Level: 1, IsMilestone: true, DurationUnit: UnitCd}}}); err == nil || !strings.Contains(err.Error(), "里程碑") {
		t.Fatalf("里程碑 cd 应拒绝：%v", err)
	}
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "X", Name: "x", Duration: 1, Level: 1, DurationUnit: "week"}}}); err == nil || !strings.Contains(err.Error(), "工期单位非法") {
		t.Fatalf("upsert 非法单位应拒绝：%v", err)
	}
}
