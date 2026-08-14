package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openTestDB 临时 SQLite（含 tasks 表结构）。
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"-"+fmt.Sprint(time.Now().UnixNano())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY, kind TEXT NOT NULL DEFAULT '', label TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'queued', progress INTEGER NOT NULL DEFAULT 0,
  message TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '',
  retry_count INTEGER NOT NULL DEFAULT 0, max_retries INTEGER NOT NULL DEFAULT 2,
  payload TEXT NOT NULL DEFAULT '{}', result TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL DEFAULT 0, started_at INTEGER NOT NULL DEFAULT 0,
  finished_at INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

type eventCollector struct {
	mu  sync.Mutex
	evs []Task
}

func (c *eventCollector) add(t Task) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evs = append(c.evs, t)
}

func (c *eventCollector) snapshot() []Task {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Task, len(c.evs))
	copy(out, c.evs)
	return out
}

func (c *eventCollector) last() (Task, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.evs) == 0 {
		return Task{}, false
	}
	return c.evs[len(c.evs)-1], true
}

func (c *eventCollector) countStatus(st string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, e := range c.evs {
		if e.Status == st {
			n++
		}
	}
	return n
}

// waitTerminal 等待任务到达终态。
func waitTerminal(t *testing.T, m *Manager, id string, timeout time.Duration) *Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		tk, err := m.Get(id)
		if err != nil {
			t.Fatalf("get task: %v", err)
		}
		if isTerminal(tk.Status) {
			return tk
		}
		if time.Now().After(deadline) {
			t.Fatalf("任务 %s 未在 %v 内到达终态（当前 %s）", id, timeout, tk.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSubmitRunSucceed(t *testing.T) {
	db := openTestDB(t)
	col := &eventCollector{}
	m := New(db, col.add, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Report(50, "扫描中")
		p.Report(100, "完成")
		p.Result(`{"total":10}`)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, err := m.Submit(KindFileIndex, "重建索引", map[string]any{"scope": "workspace"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("期望 succeeded，实际 %s（error=%s）", done.Status, done.Error)
	}
	if done.Progress != 100 {
		t.Fatalf("期望进度 100，实际 %d", done.Progress)
	}
	if !strings.Contains(done.Result, `"total":10`) {
		t.Fatalf("结果未写入: %s", done.Result)
	}
	if col.countStatus(string(StatusSucceeded)) == 0 {
		t.Fatal("缺少 succeeded 事件")
	}
	row := db.QueryRow(`SELECT status FROM tasks WHERE id=?`, tk.ID)
	var st string
	if err := row.Scan(&st); err != nil || st != "succeeded" {
		t.Fatalf("持久化失败: %v st=%s", err, st)
	}
}

func TestSubmitUnregisteredKindFails(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, err := m.Submit("nope_kind", "未知任务", nil)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusFailed) || !strings.Contains(done.Error, "未注册") {
		t.Fatalf("期望未注册失败，实际 %s/%s", done.Status, done.Error)
	}
}

func TestCancelRunningTask(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "抓取价格", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	if err := m.Cancel(tk.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
}

func TestCancelQueuedTask(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxConcurrent: 1, BackoffBase: 10 * time.Millisecond})
	block := make(chan struct{})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		<-block
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	first, _ := m.Submit(KindFileIndex, "任务一", nil)
	second, _ := m.Submit(KindFileIndex, "任务二", nil)
	deadline := time.Now().Add(2 * time.Second)
	for {
		f, _ := m.Get(first.ID)
		if f.Status == string(StatusRunning) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("任务一未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := m.Cancel(second.ID); err != nil {
		t.Fatalf("cancel queued: %v", err)
	}
	s, _ := m.Get(second.ID)
	if s.Status != string(StatusCancelled) {
		t.Fatalf("期望 queued 任务直接 cancelled，实际 %s", s.Status)
	}
	close(block)
	waitTerminal(t, m, first.ID, 3*time.Second)
}

func TestRetryFailedTask(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxRetries: 0, BackoffBase: 10 * time.Millisecond})
	var attempts int
	var mu sync.Mutex
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n == 1 {
			return errors.New("首次失败")
		}
		p.Result(`{"ok":true}`)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "重试任务", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusFailed) {
		t.Fatalf("第一次应失败，实际 %s", done.Status)
	}
	if err := m.Retry(tk.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	done2 := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done2.Status != string(StatusSucceeded) {
		t.Fatalf("重试后应 succeeded，实际 %s（%s）", done2.Status, done2.Error)
	}
	if done2.RetryCount != 0 {
		t.Fatalf("手动重试应清零 retry_count，实际 %d", done2.RetryCount)
	}
}

func TestAutoRetryWithBackoff(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxRetries: 2, BackoffBase: 20 * time.Millisecond})
	var attempts int
	var mu sync.Mutex
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		mu.Lock()
		attempts++
		mu.Unlock()
		return errors.New("网络错误")
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "自动重试", nil)
	done := waitTerminal(t, m, tk.ID, 5*time.Second)
	if done.Status != string(StatusFailed) {
		t.Fatalf("最终应失败，实际 %s", done.Status)
	}
	mu.Lock()
	n := attempts
	mu.Unlock()
	if n != 3 { // 首次 + 2 次重试
		t.Fatalf("期望 3 次尝试（含重试），实际 %d", n)
	}
	if done.RetryCount != 2 {
		t.Fatalf("期望 retry_count=2，实际 %d", done.RetryCount)
	}
}

func TestRestartResume(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO tasks(id,kind,label,status,retry_count,max_retries,payload,created_at) VALUES('tsk_old','price_fetch','上次任务','running',0,2,'{}',1)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	ran := make(chan string, 1)
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		ran <- tk.ID
		return nil
	})
	n, err := m.Start()
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	if n != 1 {
		t.Fatalf("期望续跑 1 个任务，实际 %d", n)
	}
	select {
	case id := <-ran:
		if id != "tsk_old" {
			t.Fatalf("续跑的任务 id 不符: %s", id)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("续跑任务未执行")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		tk, _ := m.Get("tsk_old")
		if isTerminal(tk.Status) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("续跑任务未完成: %s", tk.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestCloseInterruptRequeues(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	tk, _ := m.Submit(KindPriceFetch, "中断任务", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	m.Close() // 模拟进程关闭：running 应被重新排队（不是 cancelled）
	tk2, err := m.Get(tk.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if tk2.Status != string(StatusQueued) {
		t.Fatalf("Close 后应重新排队（供重启续跑），实际 %s", tk2.Status)
	}
}

func TestWaitReturnsOnTerminal(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "等待任务", nil)
	done, err := m.Wait(context.Background(), tk.ID, 3*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("期望 succeeded，实际 %s", done.Status)
	}
}

func TestListAndHasActive(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	t1, _ := m.Submit(KindFileIndex, "任务一", nil)
	t2, _ := m.Submit(KindFileIndex, "任务二", nil)
	if !m.HasActive(KindFileIndex) {
		t.Fatal("HasActive 应为 true")
	}
	waitTerminal(t, m, t1.ID, 3*time.Second)
	waitTerminal(t, m, t2.ID, 3*time.Second)
	list, err := m.List(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("期望至少 2 条，实际 %d", len(list))
	}
	if list[0].CreatedAt < list[1].CreatedAt {
		t.Fatal("列表应按新→旧排序")
	}
	if m.HasActive(KindFileIndex) {
		t.Fatal("全部结束后 HasActive 应为 false")
	}
}

func TestPayloadRoundTrip(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	var got map[string]any
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		_ = json.Unmarshal([]byte(tk.Payload), &got)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "载荷", map[string]any{"sourceId": "src-1", "cron": true})
	waitTerminal(t, m, tk.ID, 3*time.Second)
	if got["sourceId"] != "src-1" || got["cron"] != true {
		t.Fatalf("payload 往返失败: %v", got)
	}
}

func TestProgressThrottleEmits(t *testing.T) {
	db := openTestDB(t)
	col := &eventCollector{}
	m := New(db, col.add, Options{BackoffBase: 10 * time.Millisecond, EmitThrottle: 100 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		for i := 1; i <= 10; i++ {
			p.Report(i*10, fmt.Sprintf("第 %d 步", i))
		}
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "节流", nil)
	waitTerminal(t, m, tk.ID, 3*time.Second)
	evs := col.snapshot()
	if len(evs) < 2 {
		t.Fatalf("节流后仍应有至少 2 个事件（首+终态），实际 %d", len(evs))
	}
	last, _ := col.last()
	if last.Status != string(StatusSucceeded) || last.Progress != 100 {
		t.Fatalf("终态事件缺失或进度错误: %+v", last)
	}
}
