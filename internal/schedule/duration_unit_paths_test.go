package schedule

// duration_unit_paths_test.go — IN3-01：工期口径规则的「对话写入路径」与
// 「文件保存路径」必须给出同一合法性结论。
//
// 同一张非法样本表同时喂给三条路径：
//  1. Validate（Save/Load 的 fail-closed 闸，文件路径）；
//  2. ops upsert_task（对话整任务写入）；
//  3. ops patch_task（对话部分更新，含「生效态」投射）。
//
// 判定规则实现已收敛到 validateDurationUnit；本测把「同一结论 + 规则文本
// 逐字一致（patch 通道仅允许「任务 <id> 」前缀差，golden 冻结文案）」钉死。

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func duBase() Project {
	return Project{
		Name:      "口径样本",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "G", Name: "分组", Level: 0},
			{ID: "A", Name: "挖土", Duration: 2, Level: 1},
			{ID: "B", Name: "养护", Duration: 28, Level: 1, DurationUnit: UnitCd},
			{ID: "M", Name: "里程碑", Level: 1, IsMilestone: true},
		},
	}
}

// duCase 一行非法样本：upsert 载荷与 patch 载荷各自把同一规则打成非法态。
type duCase struct {
	name   string
	id     string // patch 目标 id
	upsert Task
	patch  patchTask
	rule   string // 期望命中的规则原文关键词
}

func duCases() []duCase {
	cd := func() *DurationUnit { u := UnitCd; return &u }
	bad := func() *DurationUnit { u := DurationUnit("week"); return &u }
	zero := func() *int { i := 0; return &i }
	big := func() *int { i := 3651; return &i }
	yes := func() *bool { b := true; return &b }
	return []duCase{
		{
			name:   "非法单位枚举",
			id:     "A",
			upsert: Task{ID: "A", Name: "挖土", Duration: 2, Level: 1, DurationUnit: "week"},
			patch:  patchTask{DurationUnit: bad()},
			rule:   "工期单位非法（wd|cd）：week",
		},
		{
			name:   "分组行禁 cd",
			id:     "G",
			upsert: Task{ID: "G", Name: "分组", Level: 0, DurationUnit: UnitCd},
			patch:  patchTask{DurationUnit: cd()},
			rule:   "禁止日历天（cd）工期",
		},
		{
			name:   "里程碑禁 cd",
			id:     "M",
			upsert: Task{ID: "M", Name: "里程碑", Level: 1, IsMilestone: true, DurationUnit: UnitCd},
			patch:  patchTask{DurationUnit: cd()},
			rule:   "禁止日历天（cd）工期",
		},
		{
			name:   "cd 工期超上限（patch 改工期）",
			id:     "B",
			upsert: Task{ID: "B", Name: "养护", Duration: 3651, Level: 1, DurationUnit: UnitCd},
			patch:  patchTask{Duration: big()},
			rule:   "日历天工期超上限（3650）：3651",
		},
		{
			// 旧实现：patch 只改单位时用「现值工期」查上限，先赋值后查
			//（半改状态）；生效态校验后与 Validate 同结论。
			name:   "cd 单位 + 超上限工期（同一条 op）",
			id:     "A",
			upsert: Task{ID: "A", Name: "挖土", Duration: 3651, Level: 1, DurationUnit: UnitCd},
			patch:  patchTask{DurationUnit: cd(), Duration: big()},
			rule:   "日历天工期超上限（3650）：3651",
		},
		{
			// IN3-01 的真实语义分叉：旧 patch 先查 cd 约束、后改 level，
			// 单条 op 把 cd 叶任务降成分组行 → 对话路径放行、文件路径拒绝。
			name:   "降为分组行 + 保留 cd（单条 patch）",
			id:     "B",
			upsert: Task{ID: "B", Name: "养护", Duration: 28, Level: 0, DurationUnit: UnitCd},
			patch:  patchTask{Level: zero()},
			rule:   "禁止日历天（cd）工期",
		},
		{
			// 同上：先设 cd 后设里程碑（同一 op 内顺序）旧实现会放行。
			name:   "设为里程碑 + 保留 cd（单条 patch）",
			id:     "B",
			upsert: Task{ID: "B", Name: "养护", Duration: 28, Level: 1, IsMilestone: true, DurationUnit: UnitCd},
			patch:  patchTask{IsMilestone: yes()},
			rule:   "禁止日历天（cd）工期",
		},
	}
}

// withUpsert 按 upsert 语义把任务放进计划（同 id 即整量替换），避免撞上
// 「任务 id 重复」这条无关规则而掩盖工期口径规则。
func withUpsert(base Project, t Task) Project {
	for i := range base.Tasks {
		if base.Tasks[i].ID == t.ID {
			base.Tasks[i] = t
			return base
		}
	}
	base.Tasks = append(base.Tasks, t)
	return base
}

// ruleText 去掉 ApplyOps 的条序号包装与「任务 <id> 」前缀，得到规则原文。
// patch 通道的历史文案不带 id 前缀（golden fixture 冻结），此规范化后三条
// 路径的规则文本必须逐字一致。
func ruleText(err error, id string) string {
	s := err.Error()
	if i := strings.Index(s, "条操作失败："); i >= 0 {
		s = s[i+len("条操作失败："):]
	}
	s = strings.TrimPrefix(s, "任务 "+id+" ")
	return s
}

// TestDurationUnitPathsAgree IN3-01 主守卫：非法工期口径在
// Validate（文件保存）/ upsert_task / patch_task 三条路径上同结论、同规则文本，
// 且校验失败时文件不落盘。
func TestDurationUnitPathsAgree(t *testing.T) {
	for _, c := range duCases() {
		t.Run(c.name, func(t *testing.T) {
			// ── 路径 1：文件保存（Save → Validate） ──
			p := withUpsert(duBase(), c.upsert)
			saveErr := Save(filepath.Join(t.TempDir(), "plan.gsched.json"), p)
			if saveErr == nil {
				t.Fatalf("文件保存路径应拒绝：%+v", p.Tasks)
			}
			if !strings.Contains(saveErr.Error(), c.rule) {
				t.Fatalf("文件路径错误未命中规则 %q：%v", c.rule, saveErr)
			}

			// 落盘 fail-closed：校验失败不得创建目标文件。
			path := filepath.Join(t.TempDir(), "plan.gsched.json")
			if err := Save(path, p); err == nil {
				t.Fatal("Save 应失败")
			}
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Fatalf("校验失败仍落盘：stat err=%v", statErr)
			}

			// ── 路径 2：对话 upsert_task ──
			q := duBase()
			_, upErr := ApplyOps(&q, []Op{{Type: "upsert_task", Task: &c.upsert}})
			if upErr == nil {
				t.Fatalf("upsert 路径应拒绝：%+v", c.upsert)
			}
			// upsert 与 Validate 共用同一文案口径（idInMsg=true）→ 逐字一致。
			if got, want := ruleText(upErr, c.upsert.ID), ruleText(saveErr, c.upsert.ID); got != want {
				t.Fatalf("upsert 与文件路径规则文本不一致：\n upsert=%q\n save  =%q", got, want)
			}

			// ── 路径 3：对话 patch_task ──
			r := duBase()
			_, paErr := ApplyOps(&r, []Op{{Type: "patch_task", ID: c.id, Patch: &c.patch}})
			if paErr == nil {
				t.Fatalf("patch 路径应拒绝：id=%s patch=%+v", c.id, c.patch)
			}
			// 仅允许「任务 <id> 」前缀差；规则文本必须一致。
			if got, want := ruleText(paErr, c.id), ruleText(saveErr, c.id); got != want {
				t.Fatalf("patch 与文件路径规则文本不一致（前缀之外）：\n patch=%q\n save =%q", got, want)
			}
			if !strings.Contains(paErr.Error(), c.rule) {
				t.Fatalf("patch 路径错误未命中规则 %q：%v", c.rule, paErr)
			}
		})
	}
}

// TestDurationUnitSingleImplementation 单点实现：每个非法样本经
// validateDurationUnit（唯一实现）与 Validate 得到同一错误原文。
func TestDurationUnitSingleImplementation(t *testing.T) {
	for _, c := range duCases() {
		t.Run(c.name, func(t *testing.T) {
			want := validateDurationUnit(c.upsert, true)
			if want == nil {
				t.Fatalf("validateDurationUnit 未拒绝：%+v", c.upsert)
			}
			p := withUpsert(duBase(), c.upsert)
			got := Validate(&p)
			if got == nil || got.Error() != want.Error() {
				t.Fatalf("Validate 与 validateDurationUnit 不一致：\n want=%v\n got =%v", want, got)
			}
		})
	}
	// 合法态不得误伤：''/wd/cd 三态与边界值 3650 一律放行。
	ok := []Task{
		{ID: "A", Duration: 3, Level: 1},
		{ID: "A", Duration: 3, Level: 1, DurationUnit: UnitWd},
		{ID: "A", Duration: 3650, Level: 1, DurationUnit: UnitCd},
	}
	for _, tk := range ok {
		if err := validateDurationUnit(tk, true); err != nil {
			t.Fatalf("合法态被拒：%+v → %v", tk, err)
		}
	}
}

// TestPatchDurationUnitNoPartialMutation 生效态校验先于赋值：被拒的 patch
// 不得在工程内存里留下半改状态（旧实现先写 DurationUnit 再查上限）。
func TestPatchDurationUnitNoPartialMutation(t *testing.T) {
	cd := UnitCd
	big := 3651
	p := duBase()
	before := append([]Task(nil), p.Tasks...)
	if _, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{
		DurationUnit: &cd, Duration: &big,
	}}}); err == nil {
		t.Fatal("超上限 patch 应被拒绝")
	}
	for i := range before {
		if !reflect.DeepEqual(p.Tasks[i], before[i]) {
			t.Fatalf("失败 patch 留下半改状态：\n before=%+v\n after =%+v", before[i], p.Tasks[i])
		}
	}
	// 降级为分组行的组合同样不得改动。
	p2 := duBase()
	before2 := append([]Task(nil), p2.Tasks...)
	zero := 0
	if _, err := ApplyOps(&p2, []Op{{Type: "patch_task", ID: "B", Patch: &patchTask{Level: &zero}}}); err == nil {
		t.Fatal("cd 任务降为分组行应被拒绝")
	}
	for i := range before2 {
		if !reflect.DeepEqual(p2.Tasks[i], before2[i]) {
			t.Fatalf("失败 patch 留下半改状态：\n before=%+v\n after =%+v", before2[i], p2.Tasks[i])
		}
	}
}
