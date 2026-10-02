package app

// gaea_file_index_space_test.go — 审计 AP4-07：文件语义索引任务的入队去重判据
// 与落库空间必须同源、且按**调用方所在空间**判定（work/play 各按自身空间，
// 不是一律 work）。
//
// 收敛前的两套口径：gaea_file_index.go 的 manual/cron 入口用 HasActive（全局、
// 跨空间）+ Submit（space 落缺省 work）；gaea_tasks.go 的 watch 入口用
// HasActiveInSpace(work) + SubmitSpace(work)——于是 play 空间的在途索引任务对
// watch 判据不可见，两者可并发扫同一工作区，Stale 阶段互相覆盖 keep 集合
// （把对方正在索引的文件判为删除）。本用例钉死「跨空间互不吞并 + 同空间必去重」。

import (
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// setFileIndexTestSpace 把「当前生效空间」置为 space（""=space.mode=off 平铺形态），
// 测试结束恢复。fileIndexSpace()/gaeaEffectiveSpace() 读的就是 ga.cfg。
func setFileIndexTestSpace(t *testing.T, space string) {
	t.Helper()
	ga.mu.Lock()
	old := ga.cfg
	cfg := &gaeaConfig.Config{}
	if space != "" {
		cfg.Session.Space = space
	}
	ga.cfg = cfg
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.cfg = old
		ga.mu.Unlock()
	})
}

// fileIndexTasksBySpace 数出各空间的 file_index 任务条数（不 Start 调度器，
// 任务停在 queued，空间归属可从库里读）。
func fileIndexTasksBySpace(t *testing.T, a *App) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, tk := range a.GaeaTaskList() {
		if tk.Kind == string(tasks.KindFileIndex) {
			out[tk.Space]++
		}
	}
	return out
}

// submitPlayIndexInFlight 造一个 play 空间在途索引任务（play 会话入口口径：
// SubmitSpaceSession(space=play)，对齐 novel_import_task.go 的会话提交形态）。
func submitPlayIndexInFlight(t *testing.T, a *App, label string) {
	t.Helper()
	if _, err := a.officeState.tasks.SubmitSpaceSession(
		tasks.KindFileIndex, label, map[string]any{"reason": "play-manual"}, spaces.SpacePlay, "sess-play"); err != nil {
		t.Fatalf("提交 play 在途索引任务失败: %v", err)
	}
}

// TestFileIndexDedupPerSpaceNotCrossSpace play 空间已有在途索引任务时，work 生效
// 的入口必须照常入队（不被跨空间吞掉），且同空间重复入队必须被去重。
func TestFileIndexDedupPerSpaceNotCrossSpace(t *testing.T) {
	a := newTestTaskApp(t)
	setFileIndexTestSpace(t, spaces.SpaceWork)
	submitPlayIndexInFlight(t, a, "play 在途索引")

	// ① 跨空间互不吞并：play 在途不得挡住 work 生效时的 manual 入队
	tk, err := a.GaeaFileIndexRebuild()
	if err != nil {
		t.Fatalf("play 在途不得吞掉 work 入队（跨空间互不挡）: %v", err)
	}
	if tk.Space != spaces.SpaceWork {
		t.Fatalf("work 生效时落库空间 = %q, want work", tk.Space)
	}
	// ② 同空间不重复并发：work 再来一次必须命中去重
	if _, err := a.GaeaFileIndexRebuild(); err == nil || !strings.Contains(err.Error(), "已在队列中") {
		t.Fatalf("work 同空间重复入队应被去重（不能并发扫同一工作区），实际 err=%v", err)
	}
	// ③ play 侧判据仍认得自己的在途任务（切生效空间 → play）：去重命中而不是吞掉
	setFileIndexTestSpace(t, spaces.SpacePlay)
	if _, err := a.GaeaFileIndexRebuild(); err == nil || !strings.Contains(err.Error(), "已在队列中") {
		t.Fatalf("play 生效时 play 在途应挡 play 入队，实际 err=%v", err)
	}
	// 空间分布：两个空间各一条在途（没有任何一方被吞并/重复）
	if got := fileIndexTasksBySpace(t, a); got[spaces.SpacePlay] != 1 || got[spaces.SpaceWork] != 1 {
		t.Fatalf("任务空间分布 = %v, want play:1 work:1", got)
	}
}

// TestFileIndexSubmitFollowsCallerSpace 空间语义按调用方所在空间：play 生效时
// manual 与后台入口都落 play（不是一律 work），且三者共用同一去重判据。
func TestFileIndexSubmitFollowsCallerSpace(t *testing.T) {
	a := newTestTaskApp(t)
	setFileIndexTestSpace(t, spaces.SpacePlay)
	// work 空间在途（反向：work 在途不得吞掉 play 入队）
	if _, err := a.officeState.tasks.SubmitSpaceSession(
		tasks.KindFileIndex, "work 在途索引", nil, spaces.SpaceWork, ""); err != nil {
		t.Fatalf("提交 work 在途索引任务失败: %v", err)
	}

	tk, err := a.GaeaFileIndexRebuild()
	if err != nil {
		t.Fatalf("work 在途不得吞掉 play 入队: %v", err)
	}
	if tk.Space != spaces.SpacePlay {
		t.Fatalf("play 生效时落库空间 = %q, want play（不是一律 work）", tk.Space)
	}
	// 后台入口（cron 轮询兜底 / watch 兜底）与 manual 同一判据空间：play 已有在途
	// → 两条后台路径都被去重，不新增任何任务
	a.submitFileIndexTask("工作区语义索引（轮询兜底）", "cron")
	a.submitFileIndexTask("工作区语义索引", "watch-full")
	if got := fileIndexTasksBySpace(t, a); got[spaces.SpacePlay] != 1 || got[spaces.SpaceWork] != 1 {
		t.Fatalf("后台入口未与 manual 同点去重（判据空间同源）：%v", got)
	}
}

// TestFileIndexSpaceModeOffFallsBackWork space.mode=off（生效空间为空）时三入口
// 一律回退 work：play 的在途任务不影响 work 入队，落库空间为 work（现状兼容）。
func TestFileIndexSpaceModeOffFallsBackWork(t *testing.T) {
	a := newTestTaskApp(t)
	setFileIndexTestSpace(t, "")
	submitPlayIndexInFlight(t, a, "play 在途索引")

	tk, err := a.GaeaFileIndexRebuild()
	if err != nil {
		t.Fatalf("mode=off 回退 work，不应被 play 在途吞掉: %v", err)
	}
	if tk.Space != spaces.SpaceWork {
		t.Fatalf("mode=off 落库空间 = %q, want work（整体回退）", tk.Space)
	}
}
