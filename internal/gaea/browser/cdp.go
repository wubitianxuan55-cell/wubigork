// Package browser 提供受控 Edge 浏览器自动化（MVP）：定位/拉起带 CDP 远程
// 调试端口的独立 Edge 实例（独立临时 profile，绝不触碰用户主 profile），并在
// 页面级 CDP WebSocket 会话上实现导航/读文本/交互元素清单/点击/输入/滚动。
// 供 tool/builtin 的 browser_* 工具使用；进程生命周期由 Job Object + Shutdown
// 收割，不随单次工具调用的 ctx 取消。
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// cdpWriteTimeout 单条消息写超时（对齐 realtime.writeTimeout 纪律）。
	cdpWriteTimeout = 3 * time.Second
	// cdpCallTimeout 单次 CDP 调用默认超时。
	cdpCallTimeout = 15 * time.Second
	// cdpEventBuffer 事件缓冲；满时丢弃（事件消费方只有导航等待，宁漏不堵）。
	cdpEventBuffer = 64
)

// errConnClosed 连接已关闭（读循环退出后所有 pending 的统一失败原因）。
var errConnClosed = errors.New("cdp: 连接已关闭")

// cdpError CDP 协议层错误（响应里的 error 对象）。
type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *cdpError) Error() string { return fmt.Sprintf("cdp error %d: %s", e.Code, e.Message) }

// cdpFrame 一次 CDP 响应（result/error 或传输层 err）。
type cdpFrame struct {
	Result json.RawMessage
	Error  *cdpError
	err    error
}

// Conn 页面级 CDP 会话：命令-响应按 id 匹配，事件旁路进带缓冲通道（满丢弃）。
// 写串行 + 写超时；Close 幂等；读循环退出时冲刷全部 pending。
type Conn struct {
	conn *websocket.Conn

	writeMu   sync.Mutex
	nextID    atomic.Int64
	closed    atomic.Bool
	closeOnce sync.Once

	pendingMu sync.Mutex
	pending   map[int64]chan cdpFrame

	events chan string // 仅事件 method 名；导航等待方按名过滤
	done   chan struct{}
}

// Dial 建立 CDP WebSocket 会话（ws://…，来自 /json/list 的 webSocketDebuggerUrl）。
func Dial(ctx context.Context, wsURL string) (*Conn, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cdp: dial %s: %w", wsURL, err)
	}
	c := &Conn{
		conn:    conn,
		pending: map[int64]chan cdpFrame{},
		events:  make(chan string, cdpEventBuffer),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// Call 发送一条 CDP 命令并等待同 id 响应（默认 15s 超时；out 为 nil 时忽略 result）。
func (c *Conn) Call(ctx context.Context, method string, params any, out any) error {
	return c.call(ctx, cdpCallTimeout, method, params, out)
}

// call 同 Call，但超时可调（测试需要短超时）。
func (c *Conn) call(ctx context.Context, timeout time.Duration, method string, params any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id := c.nextID.Add(1)
	ch := make(chan cdpFrame, 1)
	c.pendingMu.Lock()
	if c.closed.Load() {
		c.pendingMu.Unlock()
		return fmt.Errorf("cdp: %s: %w", method, errConnClosed)
	}
	c.pending[id] = ch
	c.pendingMu.Unlock()

	if err := c.writeJSON(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		c.dropPending(id)
		return fmt.Errorf("cdp: send %s: %w", method, err)
	}

	select {
	case f := <-ch:
		if f.err != nil {
			return fmt.Errorf("cdp: %s: %w", method, f.err)
		}
		if f.Error != nil {
			return fmt.Errorf("cdp: %s: %w", method, f.Error)
		}
		if out != nil && len(f.Result) > 0 {
			if err := json.Unmarshal(f.Result, out); err != nil {
				return fmt.Errorf("cdp: %s result 解析失败: %w", method, err)
			}
		}
		return nil
	case <-c.done:
		return fmt.Errorf("cdp: %s: %w", method, errConnClosed)
	case <-ctx.Done():
		c.dropPending(id)
		return fmt.Errorf("cdp: %s: %w", method, ctx.Err())
	}
}

// dropPending 注销一个未应答的 pending（响应迟到时通道缓冲 1 也不至于泄漏）。
func (c *Conn) dropPending(id int64) {
	c.pendingMu.Lock()
	delete(c.pending, id)
	c.pendingMu.Unlock()
}

// writeJSON 串行写一条 JSON 消息（带写超时）。
func (c *Conn) writeJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(cdpWriteTimeout))
	return c.conn.WriteJSON(v)
}

// readLoop 持续读取 CDP 消息：带 id 的响应投递给等待方；事件（无 id）只取
// method 名进缓冲通道，满则丢弃。退出时冲刷全部 pending 并 close(done)。
func (c *Conn) readLoop() {
	defer func() {
		c.pendingMu.Lock()
		for id, ch := range c.pending {
			ch <- cdpFrame{err: errConnClosed}
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
		close(c.done)
	}()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var head struct {
			ID     int64           `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Error  *cdpError       `json:"error"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			continue // 非 JSON 消息防御性忽略
		}
		if head.ID != 0 {
			c.pendingMu.Lock()
			ch := c.pending[head.ID]
			delete(c.pending, head.ID)
			c.pendingMu.Unlock()
			if ch != nil {
				ch <- cdpFrame{Result: head.Result, Error: head.Error} // 缓冲 1，必不阻塞
			}
		} else if head.Method != "" {
			select {
			case c.events <- head.Method:
			default: // 满：丢弃，绝不阻塞读循环
			}
		}
	}
}

// waitEvent 等待指定事件（最多 timeout）。连接关闭/ctx 取消/超时返回 false。
func (c *Conn) waitEvent(ctx context.Context, name string, timeout time.Duration) bool {
	t := time.NewTimer(timeout)
	defer t.Stop()
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				return false
			}
			if ev == name {
				return true
			}
		case <-c.done:
			return false
		case <-ctx.Done():
			return false
		case <-t.C:
			return false
		}
	}
}

// drainEvents 丢弃当前积压的全部事件（导航前清残留，防旧 loadEventFired 误配）。
func (c *Conn) drainEvents() {
	for {
		select {
		case _, ok := <-c.events:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

// healthy 探测会话可用性（轻量 evaluate）。
func (c *Conn) healthy(ctx context.Context) bool {
	return c.call(ctx, 3*time.Second, "Runtime.evaluate", map[string]any{"expression": "1"}, nil) == nil
}

// Close 关闭会话（幂等）。pending 由读循环退出时统一冲刷。
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		c.writeMu.Lock()
		_ = c.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(cdpWriteTimeout))
		c.writeMu.Unlock()
		err = c.conn.Close()
	})
	return err
}
