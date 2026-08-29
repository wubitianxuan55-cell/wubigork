// Package tasks 通用任务调度器 + 批处理队列（阶段 5 T5-1）。
//
// 长任务（价格抓取、文件索引重建、批量导入/OCR 等）统一走持久化任务表
// （Hephaestus.db SchemaV8）：状态机 queued → running → succeeded|failed|cancelled，
// 进度经回调报告（持久化 + 事件推送），支持取消（context 传播）、自动重试
// （指数退避）、手动重试、以及「重启续跑」（Startup 把上次 running 的任务
// 恢复为 queued 重新排队）。任务记录落库，进程崩溃/重启不丢。
package tasks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Kind 任务类型（注册 handler 的键）。
type Kind string

// 内置任务类型：价格抓取（单源/全部+定时）、工作区语义索引重建。
const (
	KindPriceFetch    Kind = "price_fetch"
	KindPriceFetchAll Kind = "price_fetch_all"
	KindFileIndex     Kind = "file_index"
)

// Status 任务生命周期状态。
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusStopping  Status = "stopping" // 用户已请求取消，等待 handler 退出（C1）
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Task 是一条持久化任务（DB 行 ↔ JSON 视图同构）。
type Task struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Progress   int    `json:"progress"` // 0-100
	Message    string `json:"message"`
	Error      string `json:"error"`
	RetryCount int    `json:"retryCount"`
	MaxRetries int    `json:"maxRetries"`
	Payload    string `json:"payload"`   // 不透明 JSON（提交方定义）
	Result     string `json:"result"`    // 不透明 JSON（handler 产出）
	CreatedAt  int64  `json:"createdAt"` // unix 毫秒
	StartedAt  int64  `json:"startedAt"`
	FinishedAt int64  `json:"finishedAt"`

	// 空间归属（S1 双空间列落库，SchemaV14 space_id 列）：零值/空串按 "work"
	// 落库（提交入口 SubmitSpace 归一化）；读取端 Get/List 原样回带。
	Space string `json:"spaceId,omitempty"`

	// 以下为事件视图字段（不落库）：appendOutput / markTerminal 触发的
	// gaea-task 事件携带输出尾部整尾回放（C9 事件驱动 + 轮询兜底）；
	// Get/List 查询与进度事件返回时为空（omitempty），载荷有界（环形缓冲上限）。
	OutputTail      string `json:"outputTail,omitempty"`
	OutputTruncated bool   `json:"outputTruncated,omitempty"`
}

// Progress 是 handler 的进度报告器：Report 更新进度（持久化 + 节流事件），
// Result 写入任务结果 JSON，Output 追加一行实时输出（C1：任务输出面板回放）。
type Progress struct {
	mu      sync.Mutex
	manager *Manager
	id      string
}

// Report 报告进度（percent 0-100）与当前说明。
func (p *Progress) Report(percent int, message string) {
	if p == nil || p.manager == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.manager.updateProgress(p.id, percent, message)
}

// Result 写入任务结果（不透明 JSON 字符串，前端按 kind 解析）。
func (p *Progress) Result(result string) {
	if p == nil || p.manager == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.manager.updateResult(p.id, result)
}

// Output 追加一行实时输出（尾部回放，GaeaTaskOutput 读取；环形缓冲截断）。
func (p *Progress) Output(line string) {
	if p == nil || p.manager == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.manager.appendOutput(p.id, line)
}

// Handler 执行一个任务。ctx 在用户取消 / Manager.Close 时 Done；错误返回后
// 由调度器按重试策略处理（ctx.Err()==context.Canceled 视为用户取消）。
type Handler func(ctx context.Context, t *Task, p *Progress) error

// Options 调度器配置。
type Options struct {
	// MaxConcurrent 同时执行的任务数（默认 1，对齐 herdsman local_concurrency=1）。
	MaxConcurrent int
	// PerSpace 每空间并发上限（S1.4 任务按空间分账）：键=space_id，值=该空间
	// 同时 running 的任务数上限；未列入的空间不设专项额度（仅受 worker 总数
	// 兜底）。空 map = 不启用空间分账：全局 MaxConcurrent 语义，旧行为零变化。
	// 启用时 worker 数自动取 max(MaxConcurrent, Σ PerSpace)，保证各空间额度
	// 可真正并行（某空间额度占满不阻塞其他空间出队）。
	PerSpace map[string]int
	// Priority 按 kind 的出队优先级（S1.4）：值越大越先出队，缺省 0；同优先级
	// 按 created_at 升序（FIFO）。空 map = 纯 FIFO（旧行为零变化）。
	Priority map[string]int
	// MaxRetries 自动重试上限（默认 2）。
	MaxRetries int
	// BackoffBase 重试退避基数（默认 2s，按 2^retry 递增，封顶 60s）。
	BackoffBase time.Duration
	// EmitThrottle 进度事件节流间隔（默认 400ms）。
	EmitThrottle time.Duration
}

// Manager 任务调度器：Submit 入队 → 工作协程执行 → 状态/进度落库并事件推送。
type Manager struct {
	db *sql.DB
	// emit 每次状态变化/节流进度时收到最新任务视图（nil 安全）。
	emit func(Task)

	mu        sync.Mutex
	handlers  map[string]Handler
	cancels   map[string]context.CancelFunc
	cancelReq map[string]bool // 用户显式取消（区别于 Close 中断）
	lastEmit  map[string]time.Time

	// lastOutputEmit 是输出事件的独立节流时间戳（C9）：与进度事件分开计时，
	// 防止 Output→Report 紧邻调用时进度更新被输出事件挤掉节流窗口。
	lastOutputEmit map[string]time.Time
	outputs        map[string]*taskOutput // 任务实时输出环形缓冲（C1）
	// outSeq 是输出缓冲的 LRU 写入时钟（m.mu 保护，单调递增）：appendOutput
	// 每次写入刷新对应 taskOutput.lastWrite，缓冲表超限时淘汰 lastWrite 最旧者。
	outSeq int64

	opts   Options
	sem    chan struct{}
	wake   chan struct{}
	closed chan struct{}
	wg     sync.WaitGroup
	once   sync.Once

	// spaceRunning 是各空间当前占用的并发额度（S1.4 按空间分账）：与
	// opts.PerSpace 组成按空间信号量（m.mu 保护；仅配置了额度的空间被跟踪，
	// pickNext 预留 / releaseSpace 释放严格配对，额度永不超发）。
	spaceRunning map[string]int
}

// taskOutput 单任务实时输出（环形缓冲：上限行数 + 字节数，超出截断标注）。
type taskOutput struct {
	lines     []string
	bytes     int
	trunc     bool
	lastSeq   int   // 预留：将来可按 seq 增量拉取（当前整尾回放）
	lastWrite int64 // 最近一次写入的全局序号（LRU 时钟：缓冲表超限时淘汰最旧者）
}

// 输出缓冲上限：200 行 / 64KB（对齐插件 jobs.output 的字节 cap + truncated 语义）；
// 缓冲表最多保留 100 个任务的输出（LRU：写入时钟最旧优先淘汰）。
const (
	outputMaxLines = 200
	outputMaxBytes = 64 * 1024
	outputMaxTasks = 100
)

// appendOutput 追加一行输出；超限丢弃最旧行并置截断标记。
// C9：输出变更随 gaea-task 事件推送整尾回放（节流与进度事件独立计时，避免
// 紧邻的 Report 被挤掉节流窗口；被节流跳过的行由下一次事件或轮询兜底回放）。
func (m *Manager) appendOutput(id, line string) {
	if m == nil {
		return
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	m.mu.Lock()
	o := m.outputs[id]
	if o == nil {
		o = &taskOutput{}
		if m.outputs == nil {
			m.outputs = map[string]*taskOutput{}
		}
		m.outputs[id] = o
	}
	m.outSeq++
	o.lastWrite = m.outSeq
	o.lines = append(o.lines, line)
	o.bytes += len(line)
	for len(o.lines) > outputMaxLines || o.bytes > outputMaxBytes {
		drop := o.lines[0]
		o.lines = o.lines[1:]
		o.bytes -= len(drop)
		o.trunc = true
	}
	// 防止长时间运行任务累积过多缓冲（保留最近写入的 outputMaxTasks 个任务的
	// 缓冲）：按 LRU 淘汰写入时钟最旧的整个缓冲，而非随机 map 顺序——频繁
	// 写入的任务时钟持续刷新不会被过早丢弃，冷任务缓冲最先让位。当前任务刚
	// 写入时钟必为最新，绝不自淘汰；输出尾部语义（整尾回放 + 截断标注）不变。
	for len(m.outputs) > outputMaxTasks {
		victim, victimWrite := "", int64(0)
		for k, v := range m.outputs {
			if k == id {
				continue
			}
			if victim == "" || v.lastWrite < victimWrite {
				victim, victimWrite = k, v.lastWrite
			}
		}
		if victim == "" {
			break // 只剩当前任务缓冲（防御：正常不会发生）
		}
		delete(m.outputs, victim)
	}
	tail := strings.Join(o.lines, "\n")
	trunc := o.trunc
	if m.lastOutputEmit == nil { // 零值 Manager（测试构造）惰性初始化
		m.lastOutputEmit = map[string]time.Time{}
	}
	now := time.Now()
	if last, ok := m.lastOutputEmit[id]; ok && now.Sub(last) < m.opts.EmitThrottle {
		m.mu.Unlock()
		return
	}
	m.lastOutputEmit[id] = now
	m.mu.Unlock()

	t, err := m.Get(id)
	if err != nil {
		return // 任务行不存在（已清理）：缓冲仍供轮询回放，事件静默跳过
	}
	t.OutputTail = tail
	t.OutputTruncated = trunc
	m.emitView(t)
}

// Output 返回任务实时输出尾部（整尾回放，不消费游标——前端按需轮询）。
// 不存在/无输出的任务返回空串、truncated=false。
func (m *Manager) Output(id string) (string, bool) {
	if m == nil {
		return "", false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o := m.outputs[id]
	if o == nil {
		return "", false
	}
	return strings.Join(o.lines, "\n"), o.trunc
}

// New 创建调度器（未启动；调用 Start 后开始执行）。
func New(db *sql.DB, emit func(Task), opts Options) *Manager {
	if db == nil {
		return &Manager{handlers: map[string]Handler{}}
	}
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 1
	}
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 2
	}
	if opts.BackoffBase <= 0 {
		opts.BackoffBase = 2 * time.Second
	}
	if opts.EmitThrottle <= 0 {
		opts.EmitThrottle = 400 * time.Millisecond
	}
	if len(opts.PerSpace) > 0 {
		ps := make(map[string]int, len(opts.PerSpace))
		for sp, n := range opts.PerSpace {
			if n > 0 { // 非正值视为未配置：该空间不设专项额度
				ps[sp] = n
			}
		}
		opts.PerSpace = ps
	}
	return &Manager{
		db:             db,
		emit:           emit,
		handlers:       map[string]Handler{},
		cancels:        map[string]context.CancelFunc{},
		cancelReq:      map[string]bool{},
		lastEmit:       map[string]time.Time{},
		lastOutputEmit: map[string]time.Time{},
		outputs:        map[string]*taskOutput{},
		opts:           opts,
		sem:            make(chan struct{}, opts.MaxConcurrent),
		spaceRunning:   map[string]int{},
		wake:           make(chan struct{}, 1),
		closed:         make(chan struct{}),
	}
}

// Available 报告调度器是否可用（db 就绪）。
func (m *Manager) Available() bool { return m != nil && m.db != nil }

// Register 注册某类任务的执行函数（Start 前调用）。
func (m *Manager) Register(kind Kind, h Handler) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[string(kind)] = h
}

// Start 启动调度器：先做重启续跑（上次 running → queued），再拉起工作协程。
// 返回续跑任务数。
func (m *Manager) Start() (int, error) {
	if m == nil || m.db == nil {
		return 0, nil
	}
	n, err := m.resumeInterrupted()
	if err != nil {
		slog.Warn("tasks: 重启续跑失败", "error", err)
	}
	workers := m.opts.MaxConcurrent
	if len(m.opts.PerSpace) > 0 {
		total := 0
		for _, n := range m.opts.PerSpace {
			total += n
		}
		if total > workers {
			workers = total // 各空间额度可并行，worker 数须覆盖总额度（S1.4）
		}
	}
	for i := 0; i < workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
	if n > 0 {
		slog.Info("tasks: 重启续跑", "count", n)
	}
	m.signal()
	return n, nil
}

// Close 停止调度器：中断全部运行中任务（不落 cancelled，留待下次启动续跑），
// 等待工作协程退出。
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		close(m.closed)
		m.mu.Lock()
		for _, cancel := range m.cancels {
			cancel()
		}
		m.mu.Unlock()
		m.wg.Wait()
	})
}

// defaultTaskSpace 是任务的空间缺省值（S1 双空间：旧调用零值 = work，
// space.mode 开关由 S2 接线）。
const defaultTaskSpace = "work"

// Submit 提交一个新任务（queued 入队），返回任务视图。空间归属缺省 "work"
// （等价 SubmitSpace(kind, label, payload, "")，既有调用零变化）。
func (m *Manager) Submit(kind Kind, label string, payload map[string]any) (*Task, error) {
	return m.SubmitSpace(kind, label, payload, "")
}

// SubmitSpace 提交一个带空间归属的新任务（S1 双空间）：space 为空按 "work"
// 落库，非空原样写 space_id（work/play 合法值由 S2 spaces 包校验）。
func (m *Manager) SubmitSpace(kind Kind, label string, payload map[string]any, space string) (*Task, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("任务调度器不可用")
	}
	pb, _ := json.Marshal(payload)
	if pb == nil {
		pb = []byte("{}")
	}
	if space == "" {
		space = defaultTaskSpace
	}
	t := &Task{
		ID:         newID(),
		Kind:       string(kind),
		Label:      label,
		Status:     string(StatusQueued),
		Payload:    string(pb),
		MaxRetries: m.opts.MaxRetries,
		CreatedAt:  nowMillis(),
		Space:      space,
	}
	if _, err := m.db.Exec(`INSERT INTO tasks(id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id)
VALUES(?,?,?,?,0,'','',0,?,?,'',?,0,0,?)`,
		t.ID, t.Kind, t.Label, t.Status, t.MaxRetries, t.Payload, t.CreatedAt, t.Space); err != nil {
		return nil, fmt.Errorf("任务入队失败: %w", err)
	}
	m.emitView(t)
	m.signal()
	return t, nil
}

// Get 返回单个任务。
func (m *Manager) Get(id string) (*Task, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("任务调度器不可用")
	}
	row := m.db.QueryRow(`SELECT id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id FROM tasks WHERE id=?`, id)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("任务不存在: %s", id)
	}
	return t, err
}

// List 返回最近 limit 条任务（新→旧，跨空间全量；等价 ListInSpace(limit, "")）。
func (m *Manager) List(limit int) ([]*Task, error) {
	return m.ListInSpace(limit, "")
}

// ListInSpace 返回最近 limit 条任务（新→旧），按空间过滤（S1 双空间谓词）：
// space 为空不过滤（旧行为恒真，既有调用零变化）；非空时 WHERE space_id=?。
func (m *Manager) ListInSpace(limit int, space string) ([]*Task, error) {
	if m == nil || m.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id FROM tasks`
	args := []any{}
	if space != "" {
		query += ` WHERE space_id=?`
		args = append(args, space)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// HasActive 报告某类任务是否已有 queued/running（去重提交用）。
func (m *Manager) HasActive(kind Kind) bool {
	if m == nil || m.db == nil {
		return false
	}
	var n int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE kind=? AND status IN ('queued','running')`, string(kind)).Scan(&n)
	return err == nil && n > 0
}

// HasActiveInSpace 报告某空间内某类任务是否已有 queued/running（S1.4 按空间
// 去重）：同一 kind 跨空间互不挡（play 的抓取不阻止 work 的抓取）。space 为空
// = 跨空间不过滤（等价 HasActive，旧行为）。
func (m *Manager) HasActiveInSpace(kind Kind, space string) bool {
	if m == nil || m.db == nil {
		return false
	}
	query := `SELECT COUNT(*) FROM tasks WHERE kind=? AND status IN ('queued','running')`
	args := []any{string(kind)}
	if space != "" {
		query += ` AND space_id=?`
		args = append(args, space)
	}
	var n int
	err := m.db.QueryRow(query, args...).Scan(&n)
	return err == nil && n > 0
}

// Cancel 取消一个任务：queued 用带 status 条件的原子 UPDATE 直接置 cancelled
// （与 worker 出队竞态：同一时刻只有一方能成功）；running 则中断（context 传播）。
// 重复取消幂等；对已结束/不存在的任务返回错误。
func (m *Manager) Cancel(id string) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务调度器不可用")
	}
	// ① 原子取消 queued 任务：与 worker 出队（claimQueued）共用 status 条件互斥，
	// 避免「Cancel 读到 queued 后 worker 已出队、终态 UPDATE 覆盖 running」的竞态。
	res, err := m.db.Exec(`UPDATE tasks SET status=?, message=?, finished_at=? WHERE id=? AND status=?`,
		string(StatusCancelled), "已取消", nowMillis(), id, string(StatusQueued))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		t, err := m.Get(id)
		if err != nil {
			return err
		}
		m.emitView(t)
		return nil
	}
	// ② 未命中 queued（已被出队或已结束）：尝试中断 running 任务。
	m.mu.Lock()
	cancel, running := m.cancels[id]
	if running {
		m.cancelReq[id] = true
	}
	m.mu.Unlock()
	if running {
		// C1 结束态细分：先条件置 stopping（仅当仍为 running，避免覆盖已终态），
		// 再传播取消；handler 退出后由 runNext 收尾为 cancelled。
		_ = m.setStoppingIfRunning(id)
		cancel()
		return nil
	}
	return fmt.Errorf("任务不存在或已结束")
}

// setStoppingIfRunning 把仍处于 running 的任务置为 stopping（C1 结束态细分）。
// 条件 UPDATE 防竞态：若任务已先到达终态（cancelled/succeeded/failed），不覆盖。
func (m *Manager) setStoppingIfRunning(id string) error {
	res, err := m.db.Exec(`UPDATE tasks SET status=?, message=? WHERE id=? AND status=?`,
		string(StatusStopping), "正在停止…", id, string(StatusRunning))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil // 已到达终态，无需改写
	}
	t, err := m.Get(id)
	if err != nil {
		return err
	}
	m.emitView(t)
	return nil
}

// Retry 重试一个已结束（failed/cancelled）的任务：重置状态重新排队。
func (m *Manager) Retry(id string) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("任务调度器不可用")
	}
	res, err := m.db.Exec(`UPDATE tasks SET status=?, progress=0, message='', error='', retry_count=0, finished_at=0 WHERE id=? AND status IN ('failed','cancelled')`,
		string(StatusQueued), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("只有失败/已取消的任务可以重试")
	}
	t, err := m.Get(id)
	if err != nil {
		return err
	}
	m.emitView(t)
	m.signal()
	return nil
}

// Wait 阻塞等待任务到达终态（succeeded/failed/cancelled），超时返回最新视图。
// 轮询本地 DB，无额外依赖。
func (m *Manager) Wait(ctx context.Context, id string, timeout time.Duration) (*Task, error) {
	deadline := time.Now().Add(timeout)
	for {
		t, err := m.Get(id)
		if err != nil {
			return nil, err
		}
		if isTerminal(t.Status) {
			return t, nil
		}
		select {
		case <-ctx.Done():
			return t, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return t, fmt.Errorf("等待任务超时（%v）", timeout)
		}
	}
}

// ─── 内部实现 ─────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (*Task, error) {
	var t Task
	err := s.Scan(&t.ID, &t.Kind, &t.Label, &t.Status, &t.Progress, &t.Message, &t.Error,
		&t.RetryCount, &t.MaxRetries, &t.Payload, &t.Result, &t.CreatedAt, &t.StartedAt, &t.FinishedAt, &t.Space)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (m *Manager) resumeInterrupted() (int, error) {
	// running 与 stopping（取消途中崩溃）都视为中断，恢复为 queued 重新排队。
	res, err := m.db.Exec(`UPDATE tasks SET status=?, message='上次运行中断，已重新排队', error='' WHERE status IN (?, ?)`,
		string(StatusQueued), string(StatusRunning), string(StatusStopping))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		rows, err := m.db.Query(`SELECT id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id FROM tasks WHERE status=? ORDER BY created_at`, string(StatusQueued))
		if err == nil {
			for rows.Next() {
				if t, err := scanTask(rows); err == nil {
					m.emitView(t)
				}
			}
			rows.Close()
		}
	}
	return int(n), nil
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		// 先查关闭再消费唤醒，避免 Close 竞态下重跑已重新排队的任务
		select {
		case <-m.closed:
			return
		default:
		}
		select {
		case <-m.closed:
			return
		case <-m.wake:
		}
		m.runNext()
	}
}

func (m *Manager) isClosed() bool {
	select {
	case <-m.closed:
		return true
	default:
		return false
	}
}

// runNext 取出下一条要执行的 queued 任务并执行一条（无则等待下一次唤醒）。
// S1.4 调度语义：
//   - PerSpace/Priority 均未配置 → 取全局最早 queued（GetFirstQueued，旧行为）；
//   - 否则在「其空间并发额度有余量」的 queued 任务中按 优先级高→创建早 选择，
//     空间额度占满的任务暂不取（不阻塞其他空间出队）；执行收尾先释放额度再
//     re-signal 唤醒——本空间完成腾出额度时，等待中的同空间任务立即被重新
//     调度（防饥饿）。
func (m *Manager) runNext() {
	t := m.pickNext()
	if t == nil {
		return
	}
	m.execute(t)
	m.releaseSpace(t.Space)
	if !m.isClosed() {
		m.signal()
	}
}

// pickNext 选择下一条要执行的 queued 任务；选中即预留其空间并发额度（m.mu 下
// 原子完成「查空间余量 → 择优 → 额度 +1」，多 worker 并发取任务不超发）。
// 返回 nil 表示当前无可执行任务（队列空或各空间额度均已占满）。
func (m *Manager) pickNext() *Task {
	if len(m.opts.PerSpace) == 0 && len(m.opts.Priority) == 0 {
		t, err := m.GetFirstQueued()
		if err != nil {
			return nil
		}
		return t
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, err := m.db.Query(`SELECT id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id FROM tasks WHERE status=? ORDER BY created_at ASC`, string(StatusQueued))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var best *Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			break
		}
		if !m.spaceHasRoomLocked(t.Space) {
			continue // 该空间额度已满：暂不取（不阻塞其他空间出队）
		}
		if best == nil || m.opts.Priority[t.Kind] > m.opts.Priority[best.Kind] {
			best = t // 同优先级保持扫描序（created_at 升序）= FIFO
		}
	}
	if best != nil {
		if _, ok := m.opts.PerSpace[best.Space]; ok {
			m.spaceRunning[best.Space]++ // 预留空间额度（releaseSpace 释放）
		}
	}
	return best
}

// spaceHasRoomLocked 报告某空间是否还有并发额度余量（m.mu 持有下调用）：
// 未配置额度的空间不受限（全局 worker 总数兜底）。
func (m *Manager) spaceHasRoomLocked(space string) bool {
	limit, ok := m.opts.PerSpace[space]
	if !ok {
		return true
	}
	return m.spaceRunning[space] < limit
}

// releaseSpace 释放一个空间并发额度（与 pickNext 的预留严格配对；未配置额度
// 的空间为 no-op）。必须在 re-signal 之前调用，保证被额度挡住的同空间任务
// 在本次唤醒内即可被重新选中。
func (m *Manager) releaseSpace(space string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.opts.PerSpace[space]; !ok {
		return
	}
	if n := m.spaceRunning[space]; n > 1 {
		m.spaceRunning[space] = n - 1
	} else {
		delete(m.spaceRunning, space)
	}
}

// execute 运行已选中的任务一条：认领 → 执行 handler → 终态收尾（原 runNext
// 主体）。空间额度由 pickNext 预留、runNext 收尾释放，本方法内所有提前返回
// 路径都处于「已预留」状态，由调用方统一释放。
func (m *Manager) execute(t *Task) {
	// 出队（状态置 running）之前先注册 cancel，消除「已出队但尚未注册」的窗口，
	// 保证 Cancel 对刚出队、正在跑的任务也能命中并中断。
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancels[t.ID] = cancel
	m.mu.Unlock()

	claimed, err := m.claimQueued(t.ID)
	if err != nil {
		m.unregisterCancel(t.ID)
		cancel()
		slog.Warn("tasks: 领取任务失败", "id", t.ID, "error", err)
		return
	}
	if !claimed {
		// 已被并发 Cancel（或其它 worker）消费
		m.unregisterCancel(t.ID)
		cancel()
		return
	}
	t.Status = string(StatusRunning)
	t.StartedAt = nowMillis()
	m.emitView(t)

	m.mu.Lock()
	h := m.handlers[t.Kind]
	m.mu.Unlock()
	if h == nil {
		m.unregisterCancel(t.ID)
		cancel()
		_ = m.markTerminal(t.ID, StatusFailed, "", "未注册的任务类型: "+t.Kind)
		return
	}

	progress := &Progress{manager: m, id: t.ID}
	handlerErr := m.callHandler(h, ctx, t, progress)

	m.mu.Lock()
	delete(m.cancels, t.ID)
	userCancel := m.cancelReq[t.ID]
	delete(m.cancelReq, t.ID)
	m.mu.Unlock()
	// 关键：必须在 cancel() 之前判定中断（handler 若因 ctx.Done 返回
	// ctx.Err()，此时 ctx.Err() 非空；普通业务错误则为空）。
	interrupted := ctx.Err() != nil
	cancel()

	switch {
	// 取消优先：用户取消必须胜过 succeeded（handler 即便返回 nil 也不吞掉取消）
	case interrupted && userCancel:
		_ = m.markTerminal(t.ID, StatusCancelled, "", "已取消")
	case handlerErr == nil:
		_ = m.markTerminal(t.ID, StatusSucceeded, "", "")
	case interrupted:
		// Close 中断：重新排队（下次 Start 续跑），不记失败
		_ = m.requeue(t.ID, "运行中断，等待恢复")
	default:
		var pe *handlerPanicError
		if errors.As(handlerErr, &pe) {
			// handler panic：不重试（退避救不了编程错误），直接置 failed
			_ = m.markTerminal(t.ID, StatusFailed, "", handlerErr.Error())
		} else {
			cur := m.mustGet(t.ID)
			if cur != nil && cur.RetryCount < cur.MaxRetries {
				_ = m.requeueWithBackoff(t.ID, cur.RetryCount+1, handlerErr)
			} else {
				_ = m.markTerminal(t.ID, StatusFailed, "", handlerErr.Error())
			}
		}
	}
}

// claimQueued 用带 status 条件的原子 UPDATE 把 queued 任务置 running，
// 返回是否成功认领（false 表示该任务已被并发取消/消费）。
func (m *Manager) claimQueued(id string) (bool, error) {
	res, err := m.db.Exec(`UPDATE tasks SET status=?, started_at=? WHERE id=? AND status=?`,
		string(StatusRunning), nowMillis(), id, string(StatusQueued))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// unregisterCancel 清除某任务的取消登记（用于未真正运行/未进入终态收尾的路径）。
func (m *Manager) unregisterCancel(id string) {
	m.mu.Lock()
	delete(m.cancels, id)
	delete(m.cancelReq, id)
	m.mu.Unlock()
}

// handlerPanicError 标记 handler 内部 panic（区别于普通业务错误，不参与退避重试）。
type handlerPanicError struct {
	value any
}

func (e *handlerPanicError) Error() string {
	return fmt.Sprintf("任务 handler panic: %v", e.value)
}

// callHandler 调用 handler，并在其 panic 时恢复：记日志、转成错误返回，
// 使 worker 循环存活（不能死一个任务卡死整个调度器）。
func (m *Manager) callHandler(h Handler, ctx context.Context, t *Task, p *Progress) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("tasks: 任务 handler panic", "id", t.ID, "kind", t.Kind, "panic", r)
			err = &handlerPanicError{value: r}
		}
	}()
	return h(ctx, t, p)
}

// GetFirstQueued 返回最早的 queued 任务（无则 nil）。
func (m *Manager) GetFirstQueued() (*Task, error) {
	row := m.db.QueryRow(`SELECT id,kind,label,status,progress,message,error,retry_count,max_retries,payload,result,created_at,started_at,finished_at,space_id FROM tasks WHERE status=? ORDER BY created_at ASC LIMIT 1`, string(StatusQueued))
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (m *Manager) updateProgress(id string, percent int, message string) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if _, err := m.db.Exec(`UPDATE tasks SET progress=?, message=? WHERE id=?`, percent, message, id); err != nil {
		slog.Warn("tasks: 进度落库失败", "id", id, "error", err)
		return
	}
	t, err := m.Get(id)
	if err != nil {
		return
	}
	m.mu.Lock()
	last := m.lastEmit[id]
	now := time.Now()
	throttled := now.Sub(last) < m.opts.EmitThrottle && !isTerminal(t.Status)
	if throttled {
		m.mu.Unlock()
		return
	}
	m.lastEmit[id] = now
	m.mu.Unlock()
	m.emitView(t)
}

func (m *Manager) updateResult(id, result string) {
	if _, err := m.db.Exec(`UPDATE tasks SET result=? WHERE id=?`, result, id); err != nil {
		slog.Warn("tasks: 结果落库失败", "id", id, "error", err)
	}
}

func (m *Manager) markTerminal(id string, status Status, message, errText string) error {
	// 只有 succeeded 才把进度置 100；fail/cancel 保留任务当前实际进度。
	// （修复 SQL 恒真：原 CASE WHEN ?=100 恒传 100，导致终态进度恒置 100。）
	set100 := 0
	if status == StatusSucceeded {
		set100 = 1
	}
	if _, err := m.db.Exec(`UPDATE tasks SET status=?, message=?, error=?, finished_at=?, progress=CASE WHEN ?=1 THEN 100 ELSE progress END WHERE id=?`,
		string(status), message, errText, nowMillis(), set100, id); err != nil {
		return err
	}
	t, err := m.Get(id)
	if err != nil {
		return err
	}
	// 终态事件携带输出尾部整尾回放（C9）：handler 最后几行输出可能落在输出
	// 事件节流窗口内被跳过，终态事件保证前端 dock 拿到完整最终回放。
	if tail, trunc := m.Output(id); tail != "" {
		t.OutputTail = tail
		t.OutputTruncated = trunc
	}
	m.emitView(t)
	return nil
}

func (m *Manager) requeue(id, message string) error {
	if _, err := m.db.Exec(`UPDATE tasks SET status=?, message=?, started_at=0, finished_at=0 WHERE id=?`,
		string(StatusQueued), message, id); err != nil {
		return err
	}
	t, err := m.Get(id)
	if err != nil {
		return err
	}
	m.emitView(t)
	return nil
}

// requeueWithBackoff 自动重试：retry_count+1 后按指数退避重新入队。
func (m *Manager) requeueWithBackoff(id string, retry int, cause error) error {
	delay := m.opts.BackoffBase * time.Duration(1<<min(retry, 5))
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}
	if _, err := m.db.Exec(`UPDATE tasks SET status=?, message=?, error='', retry_count=?, started_at=0, finished_at=0 WHERE id=?`,
		string(StatusQueued), fmt.Sprintf("第 %d 次重试（%v 后）", retry, delay), retry, id); err != nil {
		return err
	}
	slog.Warn("tasks: 任务失败将重试", "id", id, "retry", retry, "error", cause)
	time.AfterFunc(delay, m.signal)
	t, err := m.Get(id)
	if err != nil {
		return err
	}
	m.emitView(t)
	return nil
}

func (m *Manager) mustGet(id string) *Task {
	t, err := m.Get(id)
	if err != nil {
		return nil
	}
	return t
}

// signal 唤醒 worker（非阻塞）。
func (m *Manager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *Manager) emitView(t *Task) {
	if m == nil || m.emit == nil || t == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("tasks: 事件推送 panic 已恢复", "panic", r)
		}
	}()
	m.emit(*t)
}

func isTerminal(status string) bool {
	switch Status(status) {
	case StatusSucceeded, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("tsk_%d", time.Now().UnixNano())
	}
	return "tsk_" + hex.EncodeToString(b)
}

func nowMillis() int64 { return time.Now().UnixMilli() }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
