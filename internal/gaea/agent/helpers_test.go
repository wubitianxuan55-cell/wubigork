package agent

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// fakeTool is a no-op tool whose read-only flag, delay, error, and call counter
// the test controls.  Used across most agent test files.
type fakeTool struct {
	name         string
	readOnly     bool
	persistWrite bool // T6-2.5: 标记为持久化写入工具（子代理禁写集合据此推导）
	delay        time.Duration
	err          error
	calls        *int32 // shared counter to assert all dispatched
}

func (f fakeTool) Name() string            { return f.name }
func (f fakeTool) Description() string     { return "" }
func (f fakeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (f fakeTool) ReadOnly() bool          { return f.readOnly }
func (f fakeTool) PersistWrite() bool      { return f.persistWrite }
func (f fakeTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	if f.calls != nil {
		atomic.AddInt32(f.calls, 1)
	}
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if f.err != nil {
		return "", f.err
	}
	return f.name + " done", nil
}

// mockProvider replays preset chunks and records the last request it received.
// Used across agent tests (dispatch_test, guards_test, planmode_test, task_test).
type mockProvider struct {
	name    string
	chunks  []provider.Chunk
	lastReq provider.Request
	// streamFn 非 nil 时每次 Stream 调用它取当次 chunks（第 call 次调用，
	// 从 1 起）——瞬态序列（如空响应后退避恢复）用，与固定 chunks 互斥。
	streamFn func(call int) []provider.Chunk
	calls    int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	m.lastReq = req
	chunks := m.chunks
	if m.streamFn != nil {
		m.calls++
		chunks = m.streamFn(m.calls)
	}
	ch := make(chan provider.Chunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

// Chat 满足 LLM seam 接口：聚合本 mock 的 Stream（与真实后端同一默认实现）。
func (m *mockProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, m, req)
}

// lastUser returns the content of the last user-role message in a request.
func lastUser(req provider.Request) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == provider.RoleUser {
			return req.Messages[i].Content
		}
	}
	return ""
}
