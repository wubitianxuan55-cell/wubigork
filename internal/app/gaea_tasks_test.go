package app

// 阶段 5 T5-1 任务调度器 App 层测试：绑定方法（List/Cancel/Retry）、
// 价格抓取任务化（单源/全部/定时）、索引任务化已在 gaea_file_index_test.go 覆盖。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

const priceFixtureHTML = `<!DOCTYPE html><html><body>
<table id="tbPrice">
  <thead><tr class="tc fb bggray"><td>名称</td><td>规格</td><td>单位</td><td>是否含税</td><td>成都市区</td></tr></thead>
  <tr><td>热轧光圆钢筋</td><td>HPB300 Φ12</td><td>t</td><td>不含税</td><td><a class="orange">￥3181.00</a></td></tr>
</table></body></html>`

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

// mockPriceServer 返回一个返回造价信息页的 mock 服务。
func mockPriceServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(priceFixtureHTML))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func seedPriceSource(t *testing.T, a *App, name, url string, freq int) string {
	t.Helper()
	store := a.hubPriceStore()
	src := pricefeed.Source{
		ID: "src-" + name, Name: name, URL: url, Parser: "sc_table",
		FrequencyHours: freq, Area: "成都市区", Enabled: true,
	}
	if err := store.SaveSource(src); err != nil {
		t.Fatalf("save source: %v", err)
	}
	return src.ID
}

func TestGaeaPriceFetchTaskFlow(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	srv := mockPriceServer(t)
	a.officeState.tasks.Register(tasks.KindPriceFetch, a.priceFetchTaskHandler)
	id := seedPriceSource(t, a, "a", srv.URL, 0)

	// 单源抓取：提交任务 → 等待终态 → pending 记录出现
	tk, err := a.GaeaPriceFetch(id)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if tk.Status != "queued" {
		t.Fatalf("期望 queued，实际 %s", tk.Status)
	}
	done, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 10*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != "succeeded" {
		t.Fatalf("抓取失败: %s (%s)", done.Status, done.Error)
	}
	var res struct {
		FetchID string `json:"fetchId"`
		Count   int    `json:"count"`
	}
	if err := json.Unmarshal([]byte(done.Result), &res); err != nil || res.FetchID == "" || res.Count == 0 {
		t.Fatalf("结果异常: %s err=%v", done.Result, err)
	}
	// pending 记录落库
	recs := a.GaeaPriceFetches()
	found := false
	for _, f := range recs {
		if f.ID == res.FetchID && f.Status == "pending" {
			found = true
		}
	}
	if !found {
		t.Fatalf("pending 记录未落库: %+v", recs)
	}
}

// v4.5.1a 红线补课：taskSchedulerOptions 把 [tasks] 配置解析为按空间分账的
// 调度器选项——缺省启用 {work=1, play=1} + 价格抓取优先；显式 per_space={}
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
	if opts.Priority[string(tasks.KindPriceFetch)] <= opts.Priority[string(tasks.KindFileIndex)] {
		t.Fatalf("缺省优先级 price_fetch=%d 应高于 file_index=%d",
			opts.Priority[string(tasks.KindPriceFetch)], opts.Priority[string(tasks.KindFileIndex)])
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

func TestGaeaPriceFetchDedup(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	srv := mockPriceServer(t)
	a.officeState.tasks.Register(tasks.KindPriceFetch, func(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
		// 慢任务：确保第二次提交发生在任务仍在运行时
		<-ctx.Done()
		return ctx.Err()
	})
	id := seedPriceSource(t, a, "b", srv.URL, 0)
	tk1, err := a.GaeaPriceFetch(id)
	if err != nil {
		t.Fatalf("submit1: %v", err)
	}
	// 等任务进入 running
	deadline := time.Now().Add(3 * time.Second)
	for {
		cur, _ := a.officeState.tasks.Get(tk1.ID)
		if cur != nil && cur.Status == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("任务未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}
	tk2, err := a.GaeaPriceFetch(id)
	if err != nil {
		t.Fatalf("submit2: %v", err)
	}
	if tk2.ID != tk1.ID {
		t.Fatalf("重复提交应返回同一任务: %s vs %s", tk1.ID, tk2.ID)
	}
}

func TestGaeaPriceFetchAllTask(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	srv := mockPriceServer(t)
	a.officeState.tasks.Register(tasks.KindPriceFetchAll, a.priceFetchAllTaskHandler)
	seedPriceSource(t, a, "c1", srv.URL, 0)
	seedPriceSource(t, a, "c2", srv.URL, 0)

	tk, err := a.GaeaPriceFetchAll()
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	done, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 10*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != "succeeded" {
		t.Fatalf("一键抓取失败: %s (%s)", done.Status, done.Error)
	}
	var res struct {
		Fetched int `json:"fetched"`
		Failed  int `json:"failed"`
	}
	if err := json.Unmarshal([]byte(done.Result), &res); err != nil || res.Fetched != 2 {
		t.Fatalf("结果异常: %s err=%v", done.Result, err)
	}
}

func TestGaeaTaskListCancelRetry(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	// 用真实 handler：不存在的源会快速失败（便于验证 Cancel/Retry 分支）
	a.officeState.tasks.Register(tasks.KindPriceFetch, a.priceFetchTaskHandler)

	// List：空
	if l := a.GaeaTaskList(); len(l) != 0 {
		t.Fatalf("期望空列表，实际 %d", len(l))
	}
	// 提交一个不存在的源（会失败但先验证任务注册表）
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
		t.Fatalf("期望 failed（价格源不存在），实际 %s", done.Status)
	}
	// Cancel 已结束任务 → 报错
	if err := a.GaeaTaskCancel(tk.ID); err == nil {
		t.Fatal("取消已结束任务应报错")
	}
	// Retry：重新排队 → 再次失败（源仍不存在）
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

func TestPriceFetchAllCronSkipsWhenNotDue(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	srv := mockPriceServer(t)
	a.officeState.tasks.Register(tasks.KindPriceFetchAll, a.priceFetchAllTaskHandler)
	// 频率 24h + lastFetchAt=刚刚 → cron 应跳过（SaveSource 冲突更新不含
	// last_fetch_at，用 TouchSource 设置）
	id := seedPriceSource(t, a, "d", srv.URL, 24)
	store := a.hubPriceStore()
	if err := store.TouchSource(id, time.Now().Add(-time.Minute).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	tk, err := a.officeState.tasks.Submit(tasks.KindPriceFetchAll, "定时抓取", map[string]any{"cron": true})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	done, err := a.officeState.tasks.Wait(context.Background(), tk.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != "succeeded" || !json.Valid([]byte(done.Result)) {
		t.Fatalf("定时抓取应成功且无到期源: %s/%s", done.Status, done.Result)
	}
	var res struct {
		Fetched int `json:"fetched"`
	}
	_ = json.Unmarshal([]byte(done.Result), &res)
	if res.Fetched != 0 {
		t.Fatalf("到期源应被跳过，实际抓取 %d", res.Fetched)
	}
}

// TestTickPriceCronDedup tickPriceCron 在已有活动任务时不应重复提交。
func TestTickPriceCronDedup(t *testing.T) {
	a := newTestTaskApp(t)
	startTasks(t, a)
	a.officeState.tasks.Register(tasks.KindPriceFetchAll, func(ctx context.Context, tk *tasks.Task, p *tasks.Progress) error {
		<-ctx.Done()
		return ctx.Err()
	})
	// 先提交一个活动任务（长跑）
	tk1, err := a.officeState.tasks.Submit(tasks.KindPriceFetchAll, "活动任务", map[string]any{"cron": true})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		cur, _ := a.officeState.tasks.Get(tk1.ID)
		if cur != nil && cur.Status == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("任务未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// tickPriceCron 应跳过（已有活动抓取任务）
	a.tickPriceCron()
	// 不再新增任务
	rows, err := a.officeState.tasks.List(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("cron 去重失败：期望 1 个任务，实际 %d", len(rows))
	}
}

var _ = os.Getenv // keep os import if unused in future edits

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

// TestTaskSubmissionsTargetWorkSpace S1.4 提交点空间断言：现有 kind
// （price_fetch/price_fetch_all/file_index）全部显式落 work——尤其 cron 后台
// 提交点（tickPriceCron）无会话空间，必须显式 work（设计 §3 铁律）。
func TestTaskSubmissionsTargetWorkSpace(t *testing.T) {
	a := newTestTaskApp(t)
	// 不 Start：任务停在 queued，空间归属可断言

	// ① cron 后台提交点：定时价格抓取（无会话空间 → 显式 work）
	a.tickPriceCron()
	// ② 绑定入口：一键抓取全部
	tkAll, err := a.GaeaPriceFetchAll()
	if err != nil {
		t.Fatalf("GaeaPriceFetchAll: %v", err)
	}
	if tkAll.Space != spaces.SpaceWork {
		t.Fatalf("GaeaPriceFetchAll Space = %q, want work", tkAll.Space)
	}
	// ③ 绑定入口：单源抓取
	srv := mockPriceServer(t)
	id := seedPriceSource(t, a, "space", srv.URL, 0)
	tkOne, err := a.GaeaPriceFetch(id)
	if err != nil {
		t.Fatalf("GaeaPriceFetch: %v", err)
	}
	if tkOne.Space != spaces.SpaceWork {
		t.Fatalf("GaeaPriceFetch Space = %q, want work", tkOne.Space)
	}
	// ④ 后台维护：工作区语义索引
	a.submitFileIndexTask("test")

	list := a.GaeaTaskList()
	kinds := map[string]int{}
	for _, task := range list {
		if task.Space != spaces.SpaceWork {
			t.Fatalf("任务 %s(%s) Space = %q, want work", task.Kind, task.ID, task.Space)
		}
		kinds[task.Kind]++
	}
	if kinds[string(tasks.KindPriceFetchAll)] != 2 { // cron + 手动一键
		t.Fatalf("price_fetch_all 应有 2 条（cron+手动），实际 %v", kinds)
	}
	if kinds[string(tasks.KindPriceFetch)] != 1 || kinds[string(tasks.KindFileIndex)] != 1 {
		t.Fatalf("提交点 kind 覆盖不全: %v", kinds)
	}
}
