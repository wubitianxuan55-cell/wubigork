package schedule

// duration_negative_paths_test.go — 批次十三 round15：负工期的「对话写入路径」
// 与「文件保存路径」必须给出同一合法性结论。
//
// 背景（批次十二 D5 实测真缺口）：ops upsert_task 完全没有负工期闸 ⇒
// ApplyOps(upsert_task{Duration:-5}) 返回 err=nil 且把 -5 落进计划；同一计划走
// Save→Validate 才报「任务 N 工期为负」。判定与文案已收敛到 project.go 的
// validateDuration（唯一实现），本测把三条路径钉在同一判据上，并覆盖
// 「被拒不落库/不留半改状态」「合法态（0/正数/缺省）不误伤」。

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// negBase 负工期样本计划（叶任务 A 工期 2；与 duration_unit_paths_test 同形）。
func negBase() Project {
	return Project{
		Name:      "负工期样本",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "G", Name: "分组", Level: 0},
			{ID: "A", Name: "挖土", Duration: 2, Level: 1},
			{ID: "B", Name: "垫层", Duration: 3, Level: 1},
		},
		Links: []Link{{From: "A", To: "B", Type: FS, Lag: 0}},
	}
}

// TestNegativeDurationThreePathsAgree 三路径同结论矩阵：同一条「工期为负」事实
// 经 Validate（文件闸）/ upsert_task（整任务写入）/ patch_task（部分更新）
// 必须命中同一判据、同一规则文本（patch 通道仅允许「任务 <id> 」前缀差）。
func TestNegativeDurationThreePathsAgree(t *testing.T) {
	// ── 路径 1：Validate + Save（文件路径，fail-closed 且不落盘） ──
	bad := negBase()
	bad.Tasks[1].Duration = -5
	valErr := Validate(&bad)
	if valErr == nil {
		t.Fatal("Validate 应拒绝负工期")
	}
	if got := valErr.Error(); got != "任务 A 工期为负" {
		t.Fatalf("Validate 文案漂移：%q", got)
	}
	path := filepath.Join(t.TempDir(), "plan.gsched.json")
	saveErr := Save(path, bad)
	if saveErr == nil {
		t.Fatal("Save 应拒绝负工期")
	}
	if saveErr.Error() != valErr.Error() {
		t.Fatalf("Save 与 Validate 结论不一致：\n save=%q\n val =%q", saveErr, valErr)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("校验失败仍落盘：stat err=%v", statErr)
	}

	// ── 路径 2：对话 upsert_task（新增整任务，idInMsg=true 同 Validate 原文） ──
	p := negBase()
	_, upErr := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "负工期", Duration: -5, Level: 1}}})
	if upErr == nil {
		t.Fatal("upsert_task{Duration:-5} 应被拒绝（旧实现 err=nil 且把 -5 落进计划）")
	}
	if !strings.Contains(upErr.Error(), "任务 C 工期为负") {
		t.Fatalf("upsert 错误文案未与 Validate 同源：%v", upErr)
	}
	if got, want := ruleText(upErr, "C"), ruleText(valErr, "A"); got != want {
		t.Fatalf("upsert 与文件路径规则文本不一致：\n upsert=%q\n save  =%q", got, want)
	}

	// ── 路径 3：对话 patch_task（部分更新，历史文案不带 id 前缀） ──
	r := negBase()
	_, paErr := ApplyOps(&r, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(-5)}}})
	if paErr == nil {
		t.Fatal("patch_task{Duration:-5} 应被拒绝")
	}
	if got, want := ruleText(paErr, "A"), ruleText(valErr, "A"); got != want {
		t.Fatalf("patch 与文件路径规则文本不一致（前缀之外）：\n patch=%q\n save =%q", got, want)
	}
}

// TestUpsertNegativeDurationRejectedAndNotPersisted upsert 负工期必须报错且
// **不落库**（新增态/同 id 覆盖态/多 op 序列三种进入方式）。
func TestUpsertNegativeDurationRejectedAndNotPersisted(t *testing.T) {
	// ① 新增态：任务表逐字不动（旧实现会追加一行 Duration=-5）。
	p := negBase()
	before := cloneProject(p)
	_, err := ApplyOps(&p, []Op{{Type: "upsert_task", Task: &Task{ID: "C", Name: "负工期", Duration: -5, Level: 1}}})
	if err == nil {
		t.Fatalf("upsert_task{Duration:-5} 应被拒绝；实际 err=nil，任务表=%+v", p.Tasks)
	}
	if !reflect.DeepEqual(p.Tasks, before.Tasks) {
		t.Fatalf("被拒的 upsert 仍改动任务表：\n before=%+v\n after =%+v", before.Tasks, p.Tasks)
	}

	// ② 同 id 覆盖态：不得把 A 的 2 改成 -5。
	p2 := negBase()
	before2 := cloneProject(p2)
	if _, err := ApplyOps(&p2, []Op{{Type: "upsert_task", Task: &Task{ID: "A", Name: "挖土", Duration: -5, Level: 1}}}); err == nil {
		t.Fatal("upsert_task 同 id 替换为负工期应被拒绝")
	}
	if !reflect.DeepEqual(p2.Tasks, before2.Tasks) {
		t.Fatalf("被拒的同 id upsert 仍改动任务表：\n before=%+v\n after =%+v", before2.Tasks, p2.Tasks)
	}

	// ③ fail-closed：负工期 op 之后的 op 不得生效（单条失败即中止）。
	p3 := negBase()
	if _, err := ApplyOps(&p3, []Op{
		{Type: "upsert_task", Task: &Task{ID: "C", Name: "负工期", Duration: -5, Level: 1}},
		{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(9)}},
	}); err == nil {
		t.Fatal("含负工期 op 的序列应被拒绝")
	}
	if got := p3.Tasks[1].Duration; got != 2 {
		t.Fatalf("负工期失败后后续 op 仍生效：A.Duration=%d（应保持 2）", got)
	}
}

// TestPatchNegativeDurationRejectedAndNotPersisted patch 同语义：报错、不留半改
// 状态（含「单位+负工期同一条 op」的生效态路径），文案保持 golden 冻结原文。
func TestPatchNegativeDurationRejectedAndNotPersisted(t *testing.T) {
	// ① 纯负工期：文案不带 id 前缀（被 golden fixture「patch 负工期」冻结）。
	p := negBase()
	before := cloneProject(p)
	_, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(-5)}}})
	if err == nil {
		t.Fatal("patch_task{Duration:-5} 应被拒绝")
	}
	if !strings.Contains(err.Error(), "工期为负") {
		t.Fatalf("patch 错误未命中「工期为负」：%v", err)
	}
	if strings.Contains(err.Error(), "任务 A 工期为负") {
		t.Fatalf("patch 通道文案被改动（golden fixture 冻结）：%v", err)
	}
	if !reflect.DeepEqual(p.Tasks, before.Tasks) {
		t.Fatalf("被拒的 patch 仍改动任务表：\n before=%+v\n after =%+v", before.Tasks, p.Tasks)
	}

	// ② 单位+负工期同一条 op：生效态校验先于任何赋值 → 不得留下半改
	// DurationUnit（旧实现先写 durationUnit 再查工期，失败留半改状态）。
	p2 := negBase()
	before2 := cloneProject(p2)
	cd := UnitCd
	if _, err := ApplyOps(&p2, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{DurationUnit: &cd, Duration: gIntPtr(-1)}}}); err == nil {
		t.Fatal("单位+负工期 patch 应被拒绝")
	}
	if !reflect.DeepEqual(p2.Tasks, before2.Tasks) {
		t.Fatalf("失败 patch 留下半改状态：\n before=%+v\n after =%+v", before2.Tasks, p2.Tasks)
	}
}

// TestPatchTaskUntouchedNegativeDurationNotRejected 闸宽边界（批次十三 round15
// 主代理预审补）：patch_task 的负工期闸只在本 op **触及 duration** 时生效，宽度
// 与 D5 缺口一致、不外溢。计划里既有的 Duration<0（只能直接构造/来自旧数据——
// Load/Save 都会拒）不因「只改 level/isMilestone」的无关补丁被拒：那不是本次
// 改动造成的非法态，按 ops.go 头注契约（「应用后由调用方统一校验 + CPM
// fail-closed」）由调用方 Validate/Save 兜底。三件事一起钉住：
//
//	① 无关补丁放行且真的生效（不得被无关字段绊住）；
//	② 不变量没丢——同一条计划照旧过不了 Validate/Save（同一原文）；
//	③ 触及 duration 的补丁（含修复）仍按统一判据判决。
func TestPatchTaskUntouchedNegativeDurationNotRejected(t *testing.T) {
	broken := func() Project {
		p := negBase()
		p.Tasks[1].Duration = -5 // A：既有负工期
		return p
	}

	// ① 只改 level：放行，且改动落地、工期不被无关补丁改动。
	p := broken()
	sums, err := ApplyOps(&p, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Level: gIntPtr(0)}}})
	if err != nil {
		t.Fatalf("只改 level 的补丁不应被既有负工期绊住：%v", err)
	}
	if !strings.Contains(sums[0], "层级→0") || p.Tasks[1].Level != 0 {
		t.Fatalf("补丁未生效：summary=%q task=%+v", sums, p.Tasks[1])
	}
	if p.Tasks[1].Duration != -5 {
		t.Fatalf("无关补丁改动了工期：%d", p.Tasks[1].Duration)
	}

	// ①b 只改 isMilestone：同样放行。
	p1b := broken()
	if _, err := ApplyOps(&p1b, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{IsMilestone: gBoolPtr(true)}}}); err != nil {
		t.Fatalf("只改里程碑的补丁不应被既有负工期绊住：%v", err)
	}

	// ② 不变量没丢：这条计划照旧在文件闸被拒（同一原文，带 id 定位真凶）。
	if err := Validate(&p); err == nil || err.Error() != "任务 A 工期为负" {
		t.Fatalf("既有负工期应仍在 Validate 被拒：%v", err)
	}
	if err := Save(filepath.Join(t.TempDir(), "plan.gsched.json"), p); err == nil {
		t.Fatal("既有负工期应仍在 Save 被拒")
	}

	// ③ 触及 duration：修复放行、仍负拒绝。
	p3 := broken()
	if _, err := ApplyOps(&p3, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(5)}}}); err != nil {
		t.Fatalf("改回正工期（修复）应放行：%v", err)
	}
	if err := Validate(&p3); err != nil {
		t.Fatalf("修复后应合法：%v", err)
	}
	p4 := broken()
	if _, err := ApplyOps(&p4, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: gIntPtr(-1)}}}); err == nil {
		t.Fatal("仍为负的工期补丁应被拒绝")
	}
}

// TestNegativeDurationLegalStatesNotHurt 合法态不误伤：0/正数/缺省（零值语义）
// 在三路径上一律放行。
func TestNegativeDurationLegalStatesNotHurt(t *testing.T) {
	// Validate / Save：工期 0 与正数放行。
	for _, d := range []int{0, 1, 3, 3650} {
		p := negBase()
		p.Tasks[1].Duration = d
		if err := Validate(&p); err != nil {
			t.Fatalf("Validate 误伤工期 %d：%v", d, err)
		}
		if err := Save(filepath.Join(t.TempDir(), "plan.gsched.json"), p); err != nil {
			t.Fatalf("Save 误伤工期 %d：%v", d, err)
		}
	}
	// upsert：显式 0 / 正数 / 缺 duration 放行。缺省路径尤其要钉住：Go 的
	// Task.Duration 无 omitempty，JSON 缺 key 即零值 0，必须 ≥0 不误伤
	//（TS materializeTask `t.duration ?? 0` 同义；golden fixture 的
	// 「upsert 零值语义（缺字段=0/分组）」是同一路径的跨语言对拍）。
	for _, tc := range []struct {
		name string
		task Task
		want int
	}{
		{"显式 0", Task{ID: "C", Name: "零工期", Duration: 0, Level: 1}, 0},
		{"正数", Task{ID: "C", Name: "正工期", Duration: 5, Level: 1}, 5},
		{"缺 duration（零值语义）", Task{ID: "C", Name: "缺省"}, 0},
	} {
		q := negBase()
		sums, err := ApplyOps(&q, []Op{{Type: "upsert_task", Task: &tc.task}})
		if err != nil {
			t.Fatalf("upsert 误伤「%s」：%v", tc.name, err)
		}
		got := q.Tasks[len(q.Tasks)-1]
		if got.ID != "C" || got.Duration != tc.want {
			t.Fatalf("「%s」落地态漂移：%+v（want duration=%d）", tc.name, got, tc.want)
		}
		if !strings.Contains(sums[0], fmt.Sprintf("工期 %d 工作日", tc.want)) {
			t.Fatalf("「%s」回执未按 %d 工期呈现：%q", tc.name, tc.want, sums[0])
		}
	}
	// patch：改 0 与改正数放行，且摘要含改期回执。
	q := negBase()
	zero := 0
	sums, err := ApplyOps(&q, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: &zero}}})
	if err != nil {
		t.Fatalf("patch 改 0 应放行：%v", err)
	}
	if !strings.Contains(sums[0], "工期 2→0 工作日") {
		t.Fatalf("patch 改 0 回执异常：%q", sums[0])
	}
	five := 5
	if _, err := ApplyOps(&q, []Op{{Type: "patch_task", ID: "A", Patch: &patchTask{Duration: &five}}}); err != nil {
		t.Fatalf("patch 改正数应放行：%v", err)
	}
	// 单点实现：validateDuration 与 Validate 对同一事实同结论。
	if err := validateDuration(Task{ID: "X", Duration: 0}, true); err != nil {
		t.Fatalf("validateDuration 误伤 0：%v", err)
	}
	if got := validateDuration(Task{ID: "X", Duration: -1}, true); got == nil || got.Error() != "任务 X 工期为负" {
		t.Fatalf("validateDuration 负值原文漂移：%v", got)
	}
	if got := validateDuration(Task{ID: "X", Duration: -1}, false); got == nil || got.Error() != "工期为负" {
		t.Fatalf("validateDuration patch 口径原文漂移：%v", got)
	}
}
