package app

// 阶段 5 T5-1 任务调度器 App 层测试：绑定方法（List/Cancel/Retry）、
// 价格抓取任务化（单源/全部/定时）、索引任务化已在 gaea_file_index_test.go 覆盖。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
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

// TestGaeaTaskListSpacePassthrough S1 双空间透传回归：绑定入口 GaeaTaskList
// 零参数 = 不过滤（旧行为零变化）；未导出 taskListInSpace 按空间过滤。
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

	// 绑定入口（零参数）= 跨空间全量
	if l := a.GaeaTaskList(); len(l) != 2 {
		t.Fatalf("GaeaTaskList() = %d 条, want 2（不过滤）", len(l))
	}
	// 透传按空间过滤
	if l := a.taskListInSpace("work"); len(l) != 1 || l[0].ID != tkWork.ID {
		t.Fatalf("taskListInSpace(work) = %+v, want [tkWork]", l)
	}
	if l := a.taskListInSpace("play"); len(l) != 1 || l[0].ID != tkPlay.ID {
		t.Fatalf("taskListInSpace(play) = %+v, want [tkPlay]", l)
	}
	// 空 space = 不过滤
	if l := a.taskListInSpace(""); len(l) != 2 {
		t.Fatalf("taskListInSpace(\"\") = %d 条, want 2", len(l))
	}
	// 未知空间 = 空集
	if l := a.taskListInSpace("none"); len(l) != 0 {
		t.Fatalf("taskListInSpace(none) = %d 条, want 0", len(l))
	}
}
