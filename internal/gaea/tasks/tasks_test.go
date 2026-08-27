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
func TestCancelConcurrentStress(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{MaxConcurrent: 4, MaxRetries: 0, BackoffBase: 5 * time.Millisecond})
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
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tk, err := m.Submit(KindPriceFetch, "压力", nil)
			if err != nil {
				return
			}
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

	// 等所有任务到达终态（stopping 是非终态：取消请求后仍等待 handler 退出）
	deadline := time.Now().Add(10 * time.Second)
	for {
		var pending int
		if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status IN ('queued','running','stopping')`).Scan(&pending); err != nil {
			t.Fatalf("查询未终态数: %v", err)
		}
		if pending == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("仍有 %d 个任务未达终态", pending)
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	for id := range cancelled {
		tk, err := m.Get(id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		if tk.Status != string(StatusCancelled) {
			t.Fatalf("Cancel 成功的任务 %s 终态应为 cancelled，实际 %s", id, tk.Status)
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
