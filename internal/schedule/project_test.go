package schedule

// project_test.go / ops_test.go — 落盘、校验与增量操作用例（v4.113.0 刀4）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleProject() Project {
	return Project{
		Name:      "测试工程",
		StartDate: "2026-09-01",
		Calendar:  nil,
		Tasks:     []Task{tsk("A", 3, nil), tsk("B", 2, nil)},
		Links:     []Link{lnk("A", "B", FS, 0)},
	}
}

func TestProjectRoundTripAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.gsched.json")
	if err := Save(path, sampleProject()); err != nil {
		t.Fatalf("save: %v", err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.Name != "测试工程" || len(p.Tasks) != 2 || p.Calendar == nil || len(p.Calendar.Workweek) != 5 {
		t.Fatalf("roundtrip 不一致：%+v", p)
	}
}

func TestSaveRejectsCycleAndBadLinks(t *testing.T) {
	dir := t.TempDir()
	p := sampleProject()
	p.Links = append(p.Links, lnk("B", "A", FS, 0))
	if err := Save(filepath.Join(dir, "cycle.gsched.json"), p); err == nil || !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("环应拒绝：%v", err)
	}
	p2 := sampleProject()
	p2.Links = append(p2.Links, lnk("A", "X", FS, 0))
	if err := Save(filepath.Join(dir, "dangle.gsched.json"), p2); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("悬空应拒绝：%v", err)
	}
}

func TestValidateFieldChecks(t *testing.T) {
	p := sampleProject()
	p.Tasks[0].Progress = 120
	if err := Validate(&p); err == nil || !strings.Contains(err.Error(), "进度") {
		t.Fatalf("进度越界应拒绝：%v", err)
	}
	p.Tasks[0].Progress = 0
	p.Tasks[0].Level = 2
	if err := Validate(&p); err == nil || !strings.Contains(err.Error(), "层级") {
		t.Fatalf("层级非法应拒绝：%v", err)
	}
	p.Tasks[0].Level = 1
	p.Tasks[1].ID = "A"
	if err := Validate(&p); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复 id 应拒绝：%v", err)
	}
}

func TestOpsUpsertPatchRemoveLinksMeta(t *testing.T) {
	p := sampleProject()
	sums, err := ApplyOps(&p, []Op{
		{Type: "upsert_task", Task: &Task{ID: "C", Name: "装修", Level: 1, Duration: 5}},
		{Type: "upsert_task", Task: &Task{ID: "M", Name: "竣工里程碑", Level: 1, IsMilestone: true}, AfterID: "C"},
		{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: ptrInt(6)}},
		{Type: "set_links", ToID: "C", Links: []opLink{{From: "B"}}},
		{Type: "set_meta", StartDate: "2026-09-15"},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(sums) != 5 {
		t.Fatalf("sums = %v", sums)
	}
	if idx := indexOfTask(p.Tasks, "M"); idx != 3 || p.Tasks[idx].Duration != 0 || !p.Tasks[idx].IsMilestone {
		t.Fatalf("AfterID 应插在 C 之后（下标 3）：%+v", p.Tasks)
	}
	if p.Tasks[0].Duration != 6 {
		t.Fatalf("patch 失败：%+v", p.Tasks[0])
	}
	if len(p.Links) != 2 || p.Links[1].Type != FS {
		t.Fatalf("set_links 失败：%+v", p.Links)
	}
	if p.StartDate != "2026-09-15" {
		t.Fatalf("set_meta 失败：%s", p.StartDate)
	}
	// 应用后整体应过校验+CPM
	if err := Validate(&p); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if c := ComputeCpm(p.Tasks, p.Links); !c.OK || c.Duration != 13 {
		t.Fatalf("cpm: %+v", c)
	}
}

func TestOpsFailures(t *testing.T) {
	p := sampleProject()
	if _, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: " ", Name: "x", Level: 1}}}); err == nil {
		t.Fatal("空 id 应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "ZZ", Patch: &patchTask{}}}); err == nil {
		t.Fatal("不存在任务应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "set_links", ToID: "A", Links: []opLink{{From: "A"}}}}); err == nil {
		t.Fatal("自环前置应拒绝")
	}
	if _, err := ApplyOps(&p, []Op{{Type: "weird"}}); err == nil {
		t.Fatal("未知类型应拒绝")
	}
	// remove_task 连子孙与搭接一起清
	g := Project{Tasks: []Task{
		{ID: "g", Name: "分组", Level: 0},
		{ID: "c1", Name: "子1", Level: 1},
		{ID: "c2", Name: "子2", Level: 1},
	}, Links: []Link{lnk("c1", "c2", FS, 0), lnk("g", "c1", FS, 0)}}
	if _, err := ApplyOps(&g, []Op{{Type: "remove_task", ID: "g"}}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(g.Tasks) != 0 || len(g.Links) != 0 {
		t.Fatalf("子孙应级联清除：%+v %+v", g.Tasks, g.Links)
	}
}

func TestOpsAutoChain(t *testing.T) {
	// 结构：g(分组) a b(无前置) | g2 c(有前置) d(无前置) m(手动) M(里程碑,无前置)
	p := Project{Tasks: []Task{
		{ID: "g", Name: "分组1", Level: 0},
		tsk("a", 2, nil),
		tsk("b", 3, nil),
		{ID: "g2", Name: "分组2", Level: 0},
		tsk("c", 1, nil),
		tsk("d", 4, nil),
		tsk("m", 2, func(x *Task) { x.Mode = ModeManual }),
		tsk("M", 0, func(x *Task) { x.IsMilestone = true }),
	}, Links: []Link{lnk("a", "c", FS, 0)}}
	sums, err := ApplyOps(&p, []Op{{Type: "auto_chain"}})
	if err != nil {
		t.Fatalf("auto_chain: %v", err)
	}
	// 期望补：b←a（组内顺序）；d←c（同组上一叶）；c 有前置不动；
	// m 手动跳过（不补、不改链源）；M 里程碑无前置←d（最后一个可排程叶，
	// 不挂在手动任务上——manual 忽略入边，链它无意义）。
	if len(p.Links) != 4 {
		t.Fatalf("links = %+v", p.Links)
	}
	expect := map[string]string{"b": "a", "d": "c", "M": "d"}
	for _, l := range p.Links {
		if want, ok := expect[l.To]; ok && l.From != want {
			t.Fatalf("%s 前置 = %s, want %s", l.To, l.From, want)
		}
	}
	if !strings.Contains(sums[0], "3 条") {
		t.Fatalf("summary = %q", sums[0])
	}
	// 已全部有前置 → 拒绝（无事可做）
	p2 := Project{Tasks: []Task{tsk("a", 1, nil), tsk("b", 1, nil)}, Links: []Link{lnk("a", "b", FS, 0)}}
	if _, err := ApplyOps(&p2, []Op{{Type: "auto_chain"}}); err == nil {
		t.Fatal("无可补应拒绝")
	}
	// 应用后 CPM 仍 OK
	if c := ComputeCpm(p.Tasks, p.Links); !c.OK {
		t.Fatalf("cpm: %+v", c)
	}
}

func TestAnalyzeNarrative(t *testing.T) {
	p := Project{
		Name:      "分析",
		StartDate: "2026-09-01",
		Tasks: []Task{
			tsk("A", 10, nil), tsk("B", 2, nil), tsk("M", 0, func(x *Task) { x.IsMilestone = true }),
		},
		Links: []Link{lnk("A", "M", FS, 0), lnk("B", "M", FS, 0)},
	}
	cpm, a := p.Analyze()
	if !cpm.OK || cpm.Duration != 10 {
		t.Fatalf("cpm: %+v", cpm)
	}
	if len(a.Critical) != 2 || a.Critical[0] != "A" || a.Critical[1] != "M" {
		t.Fatalf("critical = %v（里程碑参与关键标记，与前端口径一致）", a.Critical)
	}
	// B 总时差 8 > 2 不进近关键；A 关键不进
	if len(a.NearCritical) != 0 {
		t.Fatalf("near = %+v", a.NearCritical)
	}
	if len(a.Milestones) != 1 || a.Milestones[0].Workday != 10 {
		t.Fatalf("milestones = %+v", a.Milestones)
	}
	if a.LeafCount != 3 {
		t.Fatalf("leaf = %d", a.LeafCount)
	}
}

func TestQualityChecksInTools(t *testing.T) {
	// qualityChecks 在 builtin 包，这里验证 Analyze 的 LeafCount 分组不计
	p := Project{Tasks: []Task{
		{ID: "g", Name: "分组", Level: 0},
		tsk("A", 2, nil),
	}}
	_, a := p.Analyze()
	if a.LeafCount != 1 {
		t.Fatalf("leaf = %d", a.LeafCount)
	}
}

func ptrInt(v int) *int { return &v }

func TestSaveAtomicNoTempLeft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.gsched.json")
	if err := Save(path, sampleProject()); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, sampleProject()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("残留临时文件：%s", e.Name())
		}
	}
}
