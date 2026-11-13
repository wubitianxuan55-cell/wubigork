package comhelper

// client.go — helper 进程客户端：spawn + stdio JSON 行协议（与上游
// helper-client.ts 逐字节对齐的线格式：请求 `{…args, id, cmd}`，回复
// `{id, ok, result|error, code}`，启动首行 `{"event":"ready"}`）。
//
// 语义差异（有据）：Go 侧调用互斥串行化（COM 单命令串行的保守化——上游靠
// id 匹配允许并发，helper 依旧逐条处理；串行化避免对同一 Office 应用叠加
// 并发 COM 调用）。超时=杀进程重启（上游同语义：单命令卡死会阻塞后续全部
// 命令，杀掉重启是唯一出路）；ctx 取消=只放弃本次调用不杀进程（helper 里
// 的 COM 调用继续跑完，迟到的回复按未知 id 丢弃——中途硬杀可能把文档留在
// 半改状态，比超时更伤）。

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// Logger 客户端诊断日志钩子（stderr 行/非 JSON 行），app 层接 slog；nil=丢弃。
var Logger func(message string)

func logf(format string, args ...any) {
	if Logger != nil {
		Logger(fmt.Sprintf(format, args...))
	}
}

// ComError helper 结构化错误（code: BUSY / OFFICE_ERROR / BAD_ARGS / NOT_FOUND /
// ANCHOR_MISMATCH …——上游 Describe 的 Fail.Code 与 COMException 归类）。
type ComError struct {
	Code    string
	Message string
}

func (e *ComError) Error() string { return e.Message }

// Code 报告 helper 错误码（非 ComError 的错误返回空串）。
func Code(err error) string {
	var ce *ComError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

// IsBusy 上游口径的「应用忙」（用户在单元格里输入/开着模态对话框）——
// 模型看到该错误应稍候重试。
func IsBusy(err error) bool { return Code(err) == "BUSY" }

type lineReply struct {
	ok     bool
	result json.RawMessage
	err    error
}

// Client 一条 helper 进程的生命周期管理。零值不可用，经 NewClient 创建。
// 并发安全：Call 内部串行，Dispose 幂等。
type Client struct {
	mu       sync.Mutex // 串行化 Call；Start/Dispose 与 Call 互斥
	callMu   sync.Mutex
	exePath  string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	pending  map[int64]chan lineReply
	nextID   int64
	disposed bool
	// testHookSpawn 测试注入口：绕过 EnsureHelperExe+exec.Command 的真进程
	// 路径（协议测试用假 helper 进程）。
	testHookSpawn func(exe string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error)
}

// NewClient 创建客户端（惰性 spawn：首次 Call 才编译并拉起 helper）。
func NewClient() *Client {
	return &Client{pending: map[int64]chan lineReply{}}
}

// Call 发送一条命令并等回复。args 允许 nil。timeout<=0 时取 30s 缺省。
func (c *Client) Call(ctx context.Context, cmd string, args map[string]any, timeout time.Duration) (json.RawMessage, error) {
	c.callMu.Lock()
	defer c.callMu.Unlock()
	return c.call(ctx, cmd, args, timeout)
}

func (c *Client) call(ctx context.Context, cmd string, args map[string]any, timeout time.Duration) (json.RawMessage, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	c.mu.Lock()
	if c.disposed {
		c.mu.Unlock()
		return nil, errors.New("office helper 已关闭")
	}
	if err := c.ensureLocked(ctx); err != nil {
		c.mu.Unlock()
		return nil, err
	}
	c.nextID++
	id := c.nextID
	ch := make(chan lineReply, 1)
	c.pending[id] = ch
	line := make(map[string]any, len(args)+2)
	for k, v := range args {
		line[k] = v
	}
	line["id"] = id
	line["cmd"] = cmd
	payload, err := json.Marshal(line)
	if err != nil {
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("office helper 命令序列化失败: %w", err)
	}
	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		delete(c.pending, id)
		c.killLocked()
		c.mu.Unlock()
		return nil, fmt.Errorf("office helper 通道写入失败: %w", err)
	}
	c.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case reply := <-ch:
		if reply.err != nil {
			return nil, reply.err
		}
		return reply.result, nil
	case <-timer.C:
		// 上游语义：卡死的单命令阻塞全部后续命令，超时即杀重启。
		c.mu.Lock()
		delete(c.pending, id)
		c.killLocked()
		c.mu.Unlock()
		return nil, fmt.Errorf("office helper 在 %s 上超时（%.0fs），进程已重启待命", cmd, timeout.Seconds())
	case <-ctx.Done():
		// 取消不杀进程：让在途 COM 调用跑完（迟到回复按未知 id 丢弃）。
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

// ensureLocked 命中活进程或重新拉起（c.mu 必须已持有）。
func (c *Client) ensureLocked(ctx context.Context) error {
	if c.cmd != nil && c.cmd.ProcessState == nil {
		return nil
	}
	exe := c.exePath
	if exe == "" {
		var err error
		exe, err = EnsureHelperExe()
		if err != nil {
			return err
		}
		c.exePath = exe
	}
	var (
		cmd      *exec.Cmd
		stdin    io.WriteCloser
		stdout   io.ReadCloser
		stderr   io.ReadCloser
		spawnErr error
	)
	if c.testHookSpawn != nil {
		// hook 只装配（不 Start）：Start 由下方统一执行。
		cmd, stdin, stdout, stderr, spawnErr = c.testHookSpawn(exe)
	} else {
		cmd = exec.Command(exe)
		proc.HideWindow(cmd)
		in, inErr := cmd.StdinPipe()
		out, outErr := cmd.StdoutPipe()
		errPipe, errErr := cmd.StderrPipe()
		spawnErr = errors.Join(inErr, outErr, errErr)
		stdin, stdout, stderr = in, out, errPipe
	}
	if spawnErr != nil {
		return fmt.Errorf("office helper 管道建立失败: %w", spawnErr)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("office helper 启动失败: %w", err)
	}
	c.cmd = cmd
	c.stdin = stdin
	c.pending = map[int64]chan lineReply{}

	ready := make(chan struct{})
	go c.readLoop(stdout, ready)
	go func() {
		data, _ := io.ReadAll(stderr)
		if len(data) > 0 {
			logf("office helper stderr: %s", string(data))
		}
	}()
	select {
	case <-ready:
		return nil
	case <-time.After(15 * time.Second):
		c.killLocked()
		return errors.New("office helper 启动后未就绪（15s 无 ready）")
	case <-ctx.Done():
		c.killLocked()
		return ctx.Err()
	}
}

// readLoop 逐行读 helper 输出：ready 事件放行启动等待，其余按 id 分发。
func (c *Client) readLoop(r io.Reader, ready chan<- struct{}) {
	reader := bufio.NewReader(r) // 结果 JSON 可达 MB 级，用 Reader 不用 Scanner
	signaled := false
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			c.dispatch(line, ready, &signaled)
		}
		if err != nil {
			c.mu.Lock()
			pending := c.pending
			c.pending = map[int64]chan lineReply{}
			exited := c.cmd
			c.cmd = nil
			c.mu.Unlock()
			if exited != nil {
				go func() { _ = exited.Wait() }() // 读侧已到 EOF，Wait 合法且防句柄泄漏
			}
			for id, ch := range pending {
				ch <- lineReply{err: fmt.Errorf("office helper 进程退出（第 %d 条在途调用），请重试", id)}
			}
			return
		}
	}
}

func (c *Client) dispatch(line string, ready chan<- struct{}, signaled *bool) {
	var msg struct {
		ID     *int64          `json:"id"`
		Event  string          `json:"event"`
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
		Code   string          `json:"code"`
	}
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		logf("office helper 输出非 JSON 行: %.200s", line)
		return
	}
	if msg.Event != "" {
		if msg.Event == "ready" && !*signaled {
			*signaled = true
			close(ready)
		}
		return
	}
	if msg.ID == nil {
		return
	}
	c.mu.Lock()
	ch, ok := c.pending[*msg.ID]
	delete(c.pending, *msg.ID)
	c.mu.Unlock()
	if !ok {
		return // ctx 取消/超时已放弃的迟到回复
	}
	if msg.OK {
		ch <- lineReply{ok: true, result: msg.Result}
		return
	}
	code := msg.Code
	if code == "" {
		code = "ERROR"
	}
	ch <- lineReply{err: &ComError{Code: code, Message: msg.Error}}
}

// killLocked 关 stdin → 800ms 宽限 → 强杀（上游同节奏；helper 的 ReadLine
// 在父端关管道后自行退出）。c.mu 必须已持有。
func (c *Client) killLocked() {
	cmd := c.cmd
	if cmd == nil {
		return
	}
	c.cmd = nil
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if cmd.Process == nil {
		return
	}
	time.AfterFunc(800*time.Millisecond, func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // 防句柄泄漏（读侧已死或关闭，Wait 竞态安全：Wait 只在进程退出后回收）
	})
}

// Dispose 关闭客户端并杀掉 helper 进程（幂等；App 退出时调用，管道关闭本身
// 也会让 helper 自行退出，这里只是即时回收）。
func (c *Client) Dispose() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.disposed {
		return
	}
	c.disposed = true
	c.killLocked()
}

// Alive 进程是否仍在（诊断用）。
func (c *Client) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cmd != nil && c.cmd.ProcessState == nil
}

// Ping 轻量探活（helper Run 的 ping→pong）。
func (c *Client) Ping(ctx context.Context) error {
	raw, err := c.Call(ctx, "ping", nil, 15*time.Second)
	if err != nil {
		return err
	}
	var pong string
	if err := json.Unmarshal(raw, &pong); err != nil || pong != "pong" {
		return fmt.Errorf("office helper ping 异常回执: %s", strconv.Quote(string(raw)))
	}
	return nil
}
