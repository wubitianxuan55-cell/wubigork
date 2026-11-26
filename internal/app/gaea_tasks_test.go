package app

// 阶段 5 T5-1 任务调度器 App 层测试：绑定方法（List/Cancel/Retry/Kill）与调度选项；
// 索引任务化在 gaea_file_index_test.go 覆盖。（价格抓取任务化的用例随旧成本库退役
// 拆除，2026-11-26；KindPriceFetch 只作为任意任务种类的样例名留在通用用例里。）

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// newTestTaskApp 装配带任务调度器的裸 App（临时 HOME → 临时 Hephaestus.db）。
func newTestTaskApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	gdb := db.GetDatabase(config.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(config.MemoryUserDir()) })
	a := &App{officeState: &officeState{core: &core{}}}
	m := tasks.New(gdb, nil, tasks.Options{BackoffBase: 10 * time.Millisecond})
	a.officeState.tasks = m
	t.Cleanup(m.Close)
	return a
}

func startTasks(t *testing.T, a *App) {
	t.Helper()
	if _, err := a.officeState.tasks.Start(); err != nil {
		t.Fatalf("task start: %v", err)
	}
}

// v4.5.1a 红线补课：taskSchedulerOptions 把 [tasks] 配置解析为按空间分账的
// 调度器选项——缺省启用 {work=1, play=1} + 办公批处理优先；显式 per_space={}
// 关闭分账回退全局 sem（旧行为）；显式值原样保留。
func TestTaskSchedulerOptionsDefaults(t *testing.T) {
	// 缺省（加载失败/零值配置）：max_concurrent=1 + 空间分账 + 默认优先级
	opts := taskSchedulerOptions(nil, nil)
	if opts.MaxConcurrent != 1 {
		t.Fatalf("缺省 max_concurrent = %d, want 1", opts.MaxConcurrent)
	}
	if opts.PerSpace[spaces.SpaceWork] != 1 || opts.PerSpace[spaces.SpacePlay] != 1 {
		t.Fatalf("缺省 per_space = %#v, want {work:1, play:1}", opts.PerSpace)
	}
	if opts.Priority[string(tasks.KindOfficeBatch)] <= opts.Priority[string(tasks.KindFileIndex)] {
		t.Fatalf("缺省优先级 office_batch=%d 应高于 file_index=%d",
			opts.Priority[string(tasks.KindOfficeBatch)], opts.Priority[string(tasks.KindFileIndex)])
	}

	// 显式关闭分账：per_space = {}（空 map 非 nil）→ 保持空 = 全局 sem
	empty := map[string]int{}
	opts = taskSchedulerOptions(&config.Config{Tasks: config.TasksConfig{
		PerSpace: empty,
	}}, nil)
	if opts.PerSpace == nil || len(opts.PerSpace) != 0 {
		t.Fatalf("显式空 per_space 应保持空, got %#v", opts.PerSpace)
	}

	// 显式值原样保留
	opts = taskSchedulerOptions(&config.Config{Tasks: config.TasksConfig{
		MaxConcurrent: 3,
		PerSpace:      map[string]int{spaces.SpaceWork: 2},
		Priority:      map[string]int{string(tasks.KindFileIndex): 99},
	}}, nil)
	if opts.MaxConcurrent != 3 || opts.PerSpace[spaces.SpaceWork] != 2 || opts.Priority[string(tasks.KindFileIndex)] != 99 {
		t.Fatalf("显式配置未保留: %+v", opts)
	}

	// max_concurrent=0 → 回退 1
	opts = taskSchedulerOptions(&config.Config{}, nil)
	if opts.MaxConcurrent != 1 {
		t.Fatalf("max_concurrent=0 回退 = %d, want 1", opts.MaxConcurrent)
	}
}

func TestGaeaTaskListCancelRetry(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	// 快速失败的 handler：便于验证 Cancel/Retry 分支
	a.officeState.tasks.Register(tasks.KindPriceFetch, func(context.Context, *tasks.Task, *tasks.Progress) error {
		return errors.New("任务参数缺失（sourceId）")
	})

	// List：空
	if l := a.GaeaTaskList(); len(l) != 0 {
		t.Fatalf("期望空列表，实际 %d", len(l))
	}
	// 提交一个会失败的任务（先验证任务注册表）
	tk, err := a.officeState.tasks.Submit(tasks.KindPriceFetch, "测试任务", map[string]any{"sourceId": "nope"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	// List 可见
	if l := a.GaeaTaskList(); len(l) != 1 || l[0].ID != tk.ID {
		t.Fatalf("List 异常: %+v", l)
	}
	// Cancel（任务很快失败，先等终态再验证 Cancel 错误分支）
	done, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != "failed" {
		t.Fatalf("期望 failed（handler 报错），实际 %s", done.Status)
	}
	// Cancel 已结束任务 → 报错
	if err := a.GaeaTaskCancel(tk.ID); err == nil {
		t.Fatal("取消已结束任务应报错")
	}
	// Retry：重新排队 → 再次失败（handler 仍报错）
	if err := a.GaeaTaskRetry(tk.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	done2, _ := a.officeState.tasks.Wait(context.Background(), tk.ID, 5*time.Second)
	if done2.Status != "failed" {
		t.Fatalf("重试后应再次失败，实际 %s", done2.Status)
	}
}

// TestGaeaTaskExitCodeSurfacesThroughList：进程类任务上报的真实退出码经绑定
// 面透出（字段级变更，GaeaTaskList 方法签名不变）；未上报的纯函数任务诚实
// 留空（nil，JSON omitempty 缺省）。
func TestGaeaTaskExitCodeSurfacesThroughList(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	a.officeState.tasks.Register(tasks.KindPriceFetch, func(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
		p.ExitCode(7)
		return errors.New("进程退出非零")
	})
	tk, err := a.officeState.tasks.Submit(tasks.KindPriceFetch, "进程任务", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 5*time.Second); err != nil {
		t.Fatalf("wait: %v", err)
	}
	l := a.GaeaTaskList()
	if len(l) != 1 || l[0].ID != tk.ID {
		t.Fatalf("List 异常: %+v", l)
	}
	if l[0].ExitCode == nil || *l[0].ExitCode != 7 {
		t.Fatalf("GaeaTaskList 应透出退出码 7，实际 %v", l[0].ExitCode)
	}
	// gaea-task 事件同源：JSON 视图携带 exitCode 键（emitTaskEvent 序列化路径）
	b, err := json.Marshal(l[0])
	if err != nil || !strings.Contains(string(b), `"exitCode":7`) {
		t.Fatalf("事件 JSON 应含 exitCode:7，实际 %s (err=%v)", b, err)
	}
}

// TestGaeaTaskListSpacePassthrough S1.4 变参透传回归（对齐 GaeaUnifiedSearch
// 变参先例）：绑定入口 GaeaTaskList 不传/空串 = 不过滤（旧行为零变化）；
// 非空 = 按空间过滤。
func TestGaeaTaskListSpacePassthrough(t *testing.T) {
	a := newTestTaskApp(t)
	// 不 Start：任务停在 queued，列表可读
	tkWork, err := a.officeState.tasks.Submit(tasks.KindPriceFetch, "work 任务", map[string]any{"sourceId": "nope"})
	if err != nil {
		t.Fatalf("submit work: %v", err)
	}
	tkPlay, err := a.officeState.tasks.SubmitSpace(tasks.KindPriceFetch, "play 任务", nil, "play")
	if err != nil {
		t.Fatalf("submit play: %v", err)
	}

	// 绑定入口不传参 = 跨空间全量
	if l := a.GaeaTaskList(); len(l) != 2 {
		t.Fatalf("GaeaTaskList() = %d 条, want 2（不过滤）", len(l))
	}
	// 变参按空间过滤
	if l := a.GaeaTaskList("work"); len(l) != 1 || l[0].ID != tkWork.ID {
		t.Fatalf("GaeaTaskList(\"work\") = %+v, want [tkWork]", l)
	}
	if l := a.GaeaTaskList("play"); len(l) != 1 || l[0].ID != tkPlay.ID {
		t.Fatalf("GaeaTaskList(\"play\") = %+v, want [tkPlay]", l)
	}
	// 空串 = 不过滤
	if l := a.GaeaTaskList(""); len(l) != 2 {
		t.Fatalf("GaeaTaskList(\"\") = %d 条, want 2", len(l))
	}
	// 未知空间 = 空集
	if l := a.GaeaTaskList("none"); len(l) != 0 {
		t.Fatalf("GaeaTaskList(\"none\") = %d 条, want 0", len(l))
	}
}

// TestTaskSubmissionsTargetWorkSpace S1.4 提交点空间断言：后台维护入口
// （工作区语义索引，无会话上下文）落 work（设计 §3 铁律）。价格抓取两个提交点
// 已随旧成本库退役拆除（2026-11-26）。
func TestTaskSubmissionsTargetWorkSpace(t *testing.T) {
	a := newTestTaskApp(t)
	// 不 Start：任务停在 queued，空间归属可断言

	// 后台维护：工作区语义索引
	a.submitFileIndexTask("工作区语义索引", "test")

	list := a.GaeaTaskList()
	if len(list) != 1 {
		t.Fatalf("应只提交 1 条索引任务，实际 %d: %+v", len(list), list)
	}
	if list[0].Kind != string(tasks.KindFileIndex) || list[0].Space != spaces.SpaceWork {
		t.Fatalf("任务 %s(%s) Space = %q, want work（kind=file_index）", list[0].Kind, list[0].ID, list[0].Space)
	}
}

// TestGaeaTaskKillForceTerminates：GaeaTaskKill 强杀运行中任务（queued 直接
// 取消、已结束报错）——透传 Manager.Kill 语义，强杀钩子由 tasks 包用例以
// 真实子进程覆盖，此处验证绑定面语义与终态判定。
func TestGaeaTaskKillForceTerminates(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	started := make(chan struct{})
	a.officeState.tasks.Register(tasks.KindPriceFetch, func(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	tk, err := a.officeState.tasks.Submit(tasks.KindPriceFetch, "测试强杀", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	if err := a.GaeaTaskKill(tk.ID); err != nil {
		t.Fatalf("kill: %v", err)
	}
	done, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != "cancelled" {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
	if err := a.GaeaTaskKill(tk.ID); err == nil {
		t.Fatal("强杀已结束任务应报错")
	}
}
