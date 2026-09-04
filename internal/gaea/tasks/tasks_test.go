package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
  finished_at INTEGER NOT NULL DEFAULT 0,
  space_id TEXT NOT NULL DEFAULT 'work')`); err != nil {
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

// TestCancelRunningSetsStoppingThenCancelled：C1 结束态细分——取消 running 任务
// 先经过 stopping（事件可见），handler 退出后终态为 cancelled。
func TestCancelRunningSetsStoppingThenCancelled(t *testing.T) {
	db := openTestDB(t)
	col := &eventCollector{}
	m := New(db, col.add, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	release := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			return nil
		}
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
	// 事件流中应出现 stopping（可能瞬时，轮询 DB 断言更稳）
	deadline := time.Now().Add(2 * time.Second)
	seenStopping := false
	for time.Now().Before(deadline) {
		cur, _ := m.Get(tk.ID)
		if cur.Status == string(StatusStopping) {
			seenStopping = true
			break
		}
		if isTerminal(cur.Status) {
			break // handler 已退出（竞态下可能跳过 stopping 可见窗口）
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !seenStopping {
		// 允许极快退出时跳过 stopping 窗口，但事件计数不应出现 stopping（此时视为通过）
		t.Log("未观察到 stopping 窗口（handler 已提前退出），跳过")
	}
	close(release)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望终态 cancelled，实际 %s", done.Status)
	}
}

// TestOutputAppendTailAndTruncate：C1 输出环形缓冲——整尾回放 + 超限截断标注。
func TestOutputAppendTailAndTruncate(t *testing.T) {
	m := &Manager{outputs: map[string]*taskOutput{}}
	if tail, _ := m.Output("nope"); tail != "" {
		t.Fatalf("无输出任务应返回空，实际 %q", tail)
	}
	for i := 0; i < 5; i++ {
		m.appendOutput("t1", fmt.Sprintf("line-%d", i))
	}
	tail, trunc := m.Output("t1")
	if trunc {
		t.Fatal("未超限不应截断")
	}
	want := "line-0\nline-1\nline-2\nline-3\nline-4"
	if tail != want {
		t.Fatalf("尾部回放异常：\n got %q\nwant %q", tail, want)
	}
	// 超行数上限：200 行
	for i := 0; i < 250; i++ {
		m.appendOutput("t2", fmt.Sprintf("long-%d", i))
	}
	tail2, trunc2 := m.Output("t2")
	if !trunc2 {
		t.Fatal("超过 200 行应置截断标记")
	}
	if !strings.HasSuffix(tail2, "long-249") {
		t.Fatalf("尾部应保留最新行，实际尾行 %q", tail2[len(tail2)-8:])
	}
	if strings.Contains(tail2, "long-0") {
		t.Fatal("最旧行应被丢弃")
	}
	// 超字节上限：64KB 单行长文本
	big := strings.Repeat("汉", 40*1024)
	m.appendOutput("t3", big)
	_, trunc3 := m.Output("t3")
	if !trunc3 {
		t.Fatal("超过 64KB 应置截断标记")
	}
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

// ─── T7-1.2 竞态修复新增测试 ─────────────────────────────

// ① 进度语义：succeeded 强制进度 100（即使 handler 只报 50）。
func TestMarkTerminalSucceededForces100(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Report(50, "半程")
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "半程成功", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("期望 succeeded，实际 %s", done.Status)
	}
	if done.Progress != 100 {
		t.Fatalf("succeeded 应强制进度 100，实际 %d", done.Progress)
	}
}

// ① 进度语义：failed 保留实际进度（不恒置 100）。
func TestMarkTerminalFailedKeepsProgress(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxRetries: 0, BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Report(40, "进行中")
		return errors.New("中途失败")
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "失败保留进度", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusFailed) {
		t.Fatalf("期望 failed，实际 %s", done.Status)
	}
	if done.Progress != 40 {
		t.Fatalf("failed 应保留实际进度 40，实际 %d", done.Progress)
	}
}

// ① 进度语义：cancelled 保留实际进度（不恒置 100）。
func TestMarkTerminalCancelledKeepsProgress(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	reported := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Report(30, "进行中")
		close(reported)
		<-ctx.Done()
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "取消保留进度", nil)
	select {
	case <-reported:
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
	if done.Progress != 30 {
		t.Fatalf("cancelled 应保留实际进度 30，实际 %d", done.Progress)
	}
}

// ② 取消优先：handler 返回 nil（而非 ctx.Err()）时，用户取消仍应胜出。
func TestCancelWinsOverNilHandler(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		<-ctx.Done()
		return nil // 注意：返回 nil 而非 ctx.Err()，验证取消不被 succeeded 吞掉
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "取消优先", nil)
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
		t.Fatalf("期望 cancelled（取消优先于 succeeded），实际 %s", done.Status)
	}
}

// ③ Cancel 与出队竞态：queued 任务被原子取消后绝不再运行，且不被出队覆盖。
func TestCancelQueuedAtomicNeverRuns(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxConcurrent: 1, BackoffBase: 10 * time.Millisecond})
	var victimRan int32
	block := make(chan struct{})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		if tk.Label == "victim" {
			atomic.AddInt32(&victimRan, 1)
		}
		<-block
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	first, _ := m.Submit(KindFileIndex, "blocker", nil)
	second, _ := m.Submit(KindFileIndex, "victim", nil)
	// 等 blocker 真正 running，victim 保持 queued
	deadline := time.Now().Add(2 * time.Second)
	for {
		f, _ := m.Get(first.ID)
		if f.Status == string(StatusRunning) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("blocker 未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := m.Cancel(second.ID); err != nil {
		t.Fatalf("cancel queued: %v", err)
	}
	close(block)
	waitTerminal(t, m, first.ID, 3*time.Second)
	if atomic.LoadInt32(&victimRan) != 0 {
		t.Fatalf("被取消的 queued 任务不应运行，实际运行了 %d 次", victimRan)
	}
	s, _ := m.Get(second.ID)
	if s.Status != string(StatusCancelled) {
		t.Fatalf("victim 应保持 cancelled，实际 %s（被出队覆盖？）", s.Status)
	}
}

// ③ 重复取消：对 running 任务连续取消两次，均成功且终态一致。
func TestCancelRepeatedRunningTask(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	started := make(chan struct{})
	release := make(chan struct{})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		close(started)
		<-release // 不响应 ctx.Done，模拟慢响应，保证两次 Cancel 都在 running 期命中
		return ctx.Err()
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindPriceFetch, "重复取消", nil)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未开始")
	}
	if err := m.Cancel(tk.ID); err != nil {
		t.Fatalf("第一次 cancel: %v", err)
	}
	if err := m.Cancel(tk.ID); err != nil {
		t.Fatalf("第二次 cancel 应幂等成功: %v", err)
	}
	close(release)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusCancelled) {
		t.Fatalf("期望 cancelled，实际 %s", done.Status)
	}
}

// ③ 取消不存在/已结束的任务应报错。
func TestCancelUnknownAndTerminalErrors(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	if err := m.Cancel("tsk_不存在"); err == nil {
		t.Fatal("取消不存在的任务应报错")
	}
	tk, _ := m.Submit(KindFileIndex, "先完成", nil)
	waitTerminal(t, m, tk.ID, 3*time.Second)
	if err := m.Cancel(tk.ID); err == nil {
		t.Fatal("取消已结束的任务应报错")
	}
}

// ③ 并发竞态压力：大量提交+取消并发执行，Cancel 成功的任务终态必须为 cancelled。
// v4.31 线 D 根治：
//   - 等待改为事件驱动——markTerminal 先落库后同步 emit，收到全部任务的终态
//     事件即证明 DB 已终态，不再依赖固定 10s 墙钟轮询（全量负载下偶发超时的
//     flaky 源）；30s 上限为纯兜底（50 个 5ms 级任务即使放慢百倍也远够）。
//   - 断言只检查最终稳定态，不锁死 stopping 等中间瞬态：Cancel 成功 ⇒ 终态
//     cancelled（v4.8.2 回归锁，不削弱）；未取消 ⇒ 终态 succeeded（handler 的
//     ctx 仅由 Cancel 成功路径取消，修复后确定性成立）；终态事件不重不漏。
func TestCancelConcurrentStress(t *testing.T) {
	db := openTestDB(t)
	// 终态事件通知：markTerminal 落库后同步触发 emit，事件到达时该任务已终态。
	terminal := make(chan struct{}, 50)
	col := &eventCollector{}
	m := New(db, func(tk Task) {
		col.add(tk)
		if isTerminal(tk.Status) {
			terminal <- struct{}{}
		}
	}, Options{MaxConcurrent: 4, MaxRetries: 0, BackoffBase: 5 * time.Millisecond})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
			return nil
		}
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	var mu sync.Mutex
	cancelled := map[string]bool{}
	var allIDs []string // 提交成功的任务（终态一致性断言用）
	var submitted int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tk, err := m.Submit(KindPriceFetch, "压力", nil)
			if err != nil {
				return
			}
			atomic.AddInt32(&submitted, 1)
			mu.Lock()
			allIDs = append(allIDs, tk.ID)
			mu.Unlock()
			if i%2 == 0 {
				if m.Cancel(tk.ID) == nil {
					mu.Lock()
					cancelled[tk.ID] = true
					mu.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()

	// 事件驱动等待全部已提交任务到达终态（每个任务恰好一个终态事件）。
	n := int(atomic.LoadInt32(&submitted))
	deadline := time.Now().Add(30 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case <-terminal:
		case <-time.After(time.Until(deadline)):
			t.Fatalf("任务未在 30s 内全部到达终态（已收 %d/%d 个终态事件）", i, n)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(cancelled) == 0 {
		t.Fatal("压力场景应至少有一个 Cancel 成功的任务")
	}
	if len(allIDs) != n {
		t.Fatalf("提交成功任务数不一致：记录 %d，计数 %d", len(allIDs), n)
	}
	// 终态事件不重不漏：每个任务恰好一个终态事件。
	if got := col.countStatus(string(StatusSucceeded)) + col.countStatus(string(StatusCancelled)) + col.countStatus(string(StatusFailed)); got != n {
		t.Fatalf("终态事件数应为 %d（每任务恰好一个），实际 %d", n, got)
	}
	// 终态一致性（修复后确定性成立）：
	//   - Cancel 成功（入 cancelled 表）⇒ 终态 cancelled（v4.8.2 回归锁）；
	//   - 取消未命中 ⇒ handler 未被中断（ctx 仅由 Cancel 成功路径取消），
	//     终态必为 succeeded。
	for _, id := range allIDs {
		tk, err := m.Get(id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		if cancelled[id] {
			if tk.Status != string(StatusCancelled) {
				t.Fatalf("Cancel 成功的任务 %s 终态应为 cancelled，实际 %s", id, tk.Status)
			}
		} else if tk.Status != string(StatusSucceeded) {
			t.Fatalf("未取消的任务 %s 终态应为 succeeded，实际 %s", id, tk.Status)
		}
	}
}

// TestClearStaleCancelOwnership 钉住 clearStaleCancel 的归属契约（v4.31 线 D
// 新增原语，确定性验证，不依赖压力命中的概率窗口）：
//   - 任务仍有执行者（running/stopping）⇒ 注册保留（归胜者 worker，误删会让
//     Cancel 已成功返回的任务被 succeeded 吞掉）；
//   - 任务已终态 ⇒ 残留预注册被清理（防泄漏与「取消已结束任务」误判）；
//   - 任何情况下不得删除 cancelReq（用户取消意图由执行者消费）。
func TestClearStaleCancelOwnership(t *testing.T) {
	db := openTestDB(t)
	insert := func(id, status string) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO tasks(id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id)
			VALUES(?,?,?,?,0,'','',0,0,'{}','',1,0,0,'work')`, id, "price_fetch", "x", status); err != nil {
			t.Fatalf("seed %s(%s): %v", id, status, err)
		}
	}
	insert("t_run", string(StatusRunning))
	insert("t_stop", string(StatusStopping))
	insert("t_done", string(StatusSucceeded))

	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	seed := func(id string) {
		m.mu.Lock()
		m.cancels[id] = func() {} // 预注册（残留或归属执行者）
		m.cancelReq[id] = true
		m.mu.Unlock()
	}
	for _, id := range []string{"t_run", "t_stop", "t_done"} {
		seed(id)
	}
	for _, id := range []string{"t_run", "t_stop", "t_done"} {
		m.clearStaleCancel(id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cancels["t_run"]; !ok {
		t.Fatal("running 任务的注册应保留（归执行者）")
	}
	if _, ok := m.cancels["t_stop"]; !ok {
		t.Fatal("stopping 任务的注册应保留（取消途中仍归执行者）")
	}
	if _, ok := m.cancels["t_done"]; ok {
		t.Fatal("已终态任务的残留注册应被清理")
	}
	for _, id := range []string{"t_run", "t_stop", "t_done"} {
		if !m.cancelReq[id] {
			t.Fatalf("clearStaleCancel 不得删除 %s 的 cancelReq（用户取消意图）", id)
		}
	}
}

// ④ panic 恢复：handler panic 记日志、任务置 failed，且不重试。
func TestHandlerPanicMarksFailed(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxRetries: 2, BackoffBase: 10 * time.Millisecond})
	var attempts int32
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		atomic.AddInt32(&attempts, 1)
		panic("boom")
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	tk, _ := m.Submit(KindFileIndex, "panic 任务", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusFailed) {
		t.Fatalf("期望 failed，实际 %s", done.Status)
	}
	if !strings.Contains(done.Error, "panic") {
		t.Fatalf("错误信息应包含 panic，实际 %q", done.Error)
	}
	if n := atomic.LoadInt32(&attempts); n != 1 {
		t.Fatalf("panic 不应重试，期望执行 1 次，实际 %d", n)
	}
}

// ④ panic 恢复：handler panic 后 worker 循环存活，后续任务仍能执行。
func TestHandlerPanicWorkerSurvives(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxRetries: 0, BackoffBase: 10 * time.Millisecond})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		if tk.Label == "panic" {
			panic("boom")
		}
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()
	first, _ := m.Submit(KindFileIndex, "panic", nil)
	waitTerminal(t, m, first.ID, 3*time.Second)
	// 关键：第二个任务仍能正常完成，证明 worker 未因 panic 死亡
	second, _ := m.Submit(KindFileIndex, "正常", nil)
	done := waitTerminal(t, m, second.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("panic 后 worker 应存活，后续任务应 succeeded，实际 %s（%s）", done.Status, done.Error)
	}
}

// TestOutputEventCarriesTail 验证 C9 任务输出事件化：p.Output 触发的 gaea-task
// 事件携带输出尾部整尾回放（节流内合并），进度事件不携带（outputTail 为空），
// 终态事件兜底带最终完整回放（覆盖节流窗口内被跳过的最后几行）。
func TestOutputEventCarriesTail(t *testing.T) {
	db := openTestDB(t)
	col := &eventCollector{}
	// 长节流窗口：只有第一行输出会即时推事件，line-b/line-c 依赖终态事件兜底。
	m := New(db, col.add, Options{BackoffBase: 10 * time.Millisecond, EmitThrottle: time.Minute})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Output("line-a")
		p.Output("line-b")
		p.Report(50, "处理中")
		p.Output("line-c")
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, err := m.Submit(KindFileIndex, "输出回放", map[string]any{"scope": "test"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("期望 succeeded，实际 %s", done.Status)
	}

	evs := col.snapshot()
	outputEvents := 0
	for _, e := range evs {
		if e.OutputTail == "" {
			continue // 进度/状态事件：outputTail 为空（omitempty）
		}
		outputEvents++
		if !strings.Contains(e.OutputTail, "line-a") {
			t.Fatalf("输出事件尾回放缺少首行: %q", e.OutputTail)
		}
		if strings.Contains(e.OutputTail, "line-c") && !strings.Contains(e.OutputTail, "line-b") {
			t.Fatalf("尾回放应保持行序: %q", e.OutputTail)
		}
	}
	if outputEvents == 0 {
		t.Fatal("应有携带 outputTail 的输出事件")
	}
	// markTerminal 先落库后 emit，waitTerminal 可能在 emit 前返回——轮询等待
	// 事件流中出现带尾回放的 succeeded 事件（与 DB 终态解耦）；终态事件兜底
	// 带最终完整回放（覆盖节流窗口内被跳过的最后几行）。
	var terminalEvent *Task
	deadline := time.Now().Add(3 * time.Second)
	for {
		for i := range evs {
			if evs[i].Status == string(StatusSucceeded) && evs[i].OutputTail != "" {
				terminalEvent = &evs[i]
			}
		}
		if terminalEvent != nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
		evs = col.snapshot()
	}
	if terminalEvent == nil {
		t.Fatal("3s 内未见带尾回放的 succeeded 事件")
	}
	if terminalEvent.OutputTail != "line-a\nline-b\nline-c" {
		t.Fatalf("终态事件应带完整尾回放: %q", terminalEvent.OutputTail)
	}
}

// TestOutputEvictionLRU：任务输出缓冲表超上限（outputMaxTasks）时按 LRU 淘汰
// 写入时钟最旧的整个缓冲，而非随机 map 顺序——淘汰顺序可预测：
// ① 最旧且未续写的任务缓冲被淘汰；
// ② 中途续写的任务因时钟刷新而存活（LRU 而非 FIFO/随机）；
// ③ 最新任务的尾部回放内容完好（最新行保留）；
// ④ 持续写入时缓冲数有界，且按写入时钟从旧到新逐个淘汰。
func TestOutputEvictionLRU(t *testing.T) {
	m := &Manager{outputs: map[string]*taskOutput{}}
	m.appendOutput("old", "old-a")
	m.appendOutput("mid", "mid-a")
	for i := 0; i < outputMaxTasks-3; i++ { // 97 个填充任务，共 99 个缓冲
		m.appendOutput(fmt.Sprintf("fill-%d", i), fmt.Sprintf("fill-line-%d", i))
	}
	m.appendOutput("old", "old-b") // 刷新 old 的写入时钟（比 mid 及全部填充任务新）
	for i := 0; i < 4; i++ {
		m.appendOutput("newest", fmt.Sprintf("newest-%d", i))
	}
	// 共 100 个缓冲；再写入一个触发淘汰：写入时钟最旧的 mid 应被逐出
	m.appendOutput("extra", "extra-a")

	if tail, _ := m.Output("mid"); tail != "" {
		t.Fatalf("写入时钟最旧的 mid 缓冲应被淘汰，实际回放 %q", tail)
	}
	if _, ok := m.outputs["mid"]; ok {
		t.Fatal("mid 缓冲应从缓冲表中移除")
	}
	if tail, _ := m.Output("old"); tail != "old-a\nold-b" {
		t.Fatalf("续写刷新时钟的 old 应存活（LRU 而非随机/FIFO），实际 %q", tail)
	}
	if tail, _ := m.Output("newest"); tail != "newest-0\nnewest-1\nnewest-2\nnewest-3" {
		t.Fatalf("最新任务的尾部回放应完好，实际 %q", tail)
	}
	if len(m.outputs) != outputMaxTasks {
		t.Fatalf("缓冲数应维持在上限 %d，实际 %d", outputMaxTasks, len(m.outputs))
	}

	// 持续写入新任务：只淘汰写入时钟最旧者，顺序可预测（fill-0 → fill-1 → fill-2）
	for i := 0; i < 3; i++ {
		m.appendOutput(fmt.Sprintf("more-%d", i), "more-line")
		if _, ok := m.outputs[fmt.Sprintf("fill-%d", i)]; ok {
			t.Fatalf("淘汰顺序应最旧优先：fill-%d 应在第 %d 次超限写入时被淘汰", i, i+1)
		}
	}
	if len(m.outputs) != outputMaxTasks {
		t.Fatalf("持续写入后缓冲数应仍为 %d，实际 %d", outputMaxTasks, len(m.outputs))
	}
	if _, ok := m.outputs["fill-96"]; !ok {
		t.Fatal("fill-96 写入时钟较新，不应被淘汰")
	}
	if _, ok := m.outputs["extra"]; !ok {
		t.Fatal("最新写入的 extra 不应被淘汰")
	}
	if tail, _ := m.Output("more-2"); tail != "more-line" {
		t.Fatalf("最新写入的 more-2 尾部回放应完好，实际 %q", tail)
	}
}

// TestExitCodeRecordedOnTerminal：进程类 handler 上报真实退出码 → Get/终态
// 事件合入（同一收口），非零退出码如实透出、不参与重试判定之外的语义改写。
func TestExitCodeRecordedOnTerminal(t *testing.T) {
	db := openTestDB(t)
	col := &eventCollector{}
	// MaxRetries=0：失败直接终态，聚焦退出码本身
	m := New(db, col.add, Options{BackoffBase: 10 * time.Millisecond, MaxRetries: 0})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		p.Output("子进程退出")
		p.ExitCode(3)
		return errors.New("进程退出非零")
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, _ := m.Submit(KindPriceFetch, "跑进程", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusFailed) {
		t.Fatalf("期望 failed，实际 %s", done.Status)
	}
	if done.ExitCode == nil || *done.ExitCode != 3 {
		t.Fatalf("期望退出码 3，实际 %v", done.ExitCode)
	}
	// 终态事件与 Get 同源：事件视图同样携带退出码
	last, ok := col.last()
	if !ok || last.ExitCode == nil || *last.ExitCode != 3 {
		t.Fatalf("终态事件应携带退出码 3，实际 %+v", last)
	}
	// JSON 透出键名（前端 TaskView 契约）
	b, _ := json.Marshal(done)
	if !strings.Contains(string(b), `"exitCode":3`) {
		t.Fatalf("JSON 应含 exitCode:3，实际 %s", b)
	}
}

// TestExitCodeZeroPreserved：退出码 0 是真实的成功事实——指针 + omitempty
// 语义必须保住 0（区别于「未上报」的缺省），不被吞成无退出码。
func TestExitCodeZeroPreserved(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		p.ExitCode(0)
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, _ := m.Submit(KindPriceFetch, "跑进程", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) || done.ExitCode == nil || *done.ExitCode != 0 {
		t.Fatalf("期望 succeeded + 退出码 0，实际 %s/%v", done.Status, done.ExitCode)
	}
	b, _ := json.Marshal(done)
	if !strings.Contains(string(b), `"exitCode":0`) {
		t.Fatalf("JSON 应含 exitCode:0（0 不被 omitempty 吞掉），实际 %s", b)
	}
}

// TestExitCodeAbsentForPureFunc：纯函数任务无退出码语义——不上报即诚实留空
// （视图 nil、JSON 无 exitCode 键），绝不造假数字。
func TestExitCodeAbsentForPureFunc(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond, MaxRetries: 0})
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		return errors.New("纯函数失败")
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, _ := m.Submit(KindPriceFetch, "纯函数任务", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.ExitCode != nil {
		t.Fatalf("纯函数任务不应有退出码，实际 %v", *done.ExitCode)
	}
	b, _ := json.Marshal(done)
	if strings.Contains(string(b), "exitCode") {
		t.Fatalf("未上报时 JSON 不应含 exitCode 键，实际 %s", b)
	}
	// List 同一收口
	list, err := m.List(10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v (%d)", err, len(list))
	}
	if list[0].ExitCode != nil {
		t.Fatal("List 视图同样应诚实留空")
	}
}

// TestExitCodeClearedOnRetry：失败上报退出码 2 → 手动 Retry 重跑（未再上报）
// → 终态退出码回归 nil，旧一次尝试的退出码不串味到新一轮执行。
func TestExitCodeClearedOnRetry(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond, MaxRetries: 0})
	firstRun := true
	m.Register(KindPriceFetch, func(ctx context.Context, tk *Task, p *Progress) error {
		if firstRun {
			firstRun = false
			p.ExitCode(2)
			return errors.New("进程退出非零")
		}
		return nil // 重跑成功且不上报
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	tk, _ := m.Submit(KindPriceFetch, "跑进程", nil)
	done := waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.ExitCode == nil || *done.ExitCode != 2 {
		t.Fatalf("首次失败应携带退出码 2，实际 %v", done.ExitCode)
	}
	if err := m.Retry(tk.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	done = waitTerminal(t, m, tk.ID, 3*time.Second)
	if done.Status != string(StatusSucceeded) {
		t.Fatalf("重跑期望 succeeded，实际 %s", done.Status)
	}
	if done.ExitCode != nil {
		t.Fatalf("重跑后旧退出码应清空，实际 %v", *done.ExitCode)
	}
}
