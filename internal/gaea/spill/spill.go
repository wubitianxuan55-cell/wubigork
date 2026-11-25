// Package spill implements the session-scoped tool-result spill store: when a
// tool result is too large to inline, the executor saves the FULL text here and
// the model sees a bounded preview plus a locator; the read_spill tool pages
// the text back. Distilled from dsh packages/spill/spill-policy (v4.379) —
// storage medium adapted to gaea: an in-memory store on the runner instead of
// session artifacts on disk (single runtime, no cross-process sharing; total
// budget + FIFO eviction bounds memory, restart clears it).
package spill

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxEntryBytes 是单条泄洪的字节上限：bash 前台输出被 boundedOutput 封顶在
// ~1MB，其余工具理论上可返回更大——超限拒收（best-effort，既有截断管线照旧），
// 不为一条结果兜住任意大的内存。
const MaxEntryBytes = 4 << 20

// TotalBudget 是泄洪库的总字节预算：超限按 FIFO 驱逐最旧条目。var 而非
// const——测试注入小预算验驱逐序，生产不动。
var TotalBudget = 64 << 20

// ReadDefaultBytes / ReadMaxBytes 是 read_spill 单次取回的默认/上限字节。
const (
	ReadDefaultBytes = 16 * 1024
	ReadMaxBytes     = 48 * 1024
)

// Entry 是一条泄洪的全文。
type Entry struct {
	ID      string
	Tool    string
	Data    string
	Bytes   int
	Created time.Time
}

// Store 是一个执行器（≈一个会话）的泄洪库。并发安全：executeOne 的并行
// goroutine 同时 Save，read_spill 工具随时 Retrieve。
type Store struct {
	mu      sync.Mutex
	seq     int
	total   int
	order   []string // FIFO 驱逐序
	byID    map[string]*Entry
	dropped int // 驱逐计数（诚实诊断用）
}

func NewStore() *Store {
	return &Store{byID: make(map[string]*Entry)}
}

// Save 存一条泄洪全文并返回 locator id。调用方（agent 侧策略）已把关下限；
// 这里把关单条上限。超限返回 ""（不泄洪只是退回旧行为，绝不影响工具成功）。
func (s *Store) Save(tool, data string) string {
	if s == nil || len(data) > MaxEntryBytes {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("sp-%06d", s.seq)
	e := &Entry{ID: id, Tool: tool, Data: data, Bytes: len(data), Created: time.Now()}
	s.byID[id] = e
	s.order = append(s.order, id)
	s.total += len(data)
	for s.total > TotalBudget && len(s.order) > 0 {
		oldest := s.order[0]
		s.order = s.order[1:]
		if oe, ok := s.byID[oldest]; ok {
			s.total -= oe.Bytes
			delete(s.byID, oldest)
			s.dropped++
		}
	}
	return id
}

// Retrieve 按 id 取回一段：offset 为字节偏移，maxBytes<=0 取默认档。
// 返回（chunk, 总字节, 剩余字节, error）。id 不存在/已被驱逐如实报错——
// locator 过期是真实形态（预算驱逐/重启清零），不静默给空串。
func (s *Store) Retrieve(id string, offset, maxBytes int) (string, int, int, error) {
	if s == nil {
		return "", 0, 0, fmt.Errorf("spill store unavailable in this context")
	}
	if maxBytes <= 0 {
		maxBytes = ReadDefaultBytes
	}
	if maxBytes > ReadMaxBytes {
		maxBytes = ReadMaxBytes
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return "", 0, 0, fmt.Errorf("spill %q not found (expired or never existed in this session)", id)
	}
	if offset < 0 || offset >= e.Bytes {
		return "", e.Bytes, 0, fmt.Errorf("spill %q offset %d out of range (total %d bytes)", id, offset, e.Bytes)
	}
	end := offset + maxBytes
	if end > e.Bytes {
		end = e.Bytes
	}
	return e.Data[offset:end], e.Bytes, e.Bytes - end, nil
}

// RetrieveLines 按行号区间取回（1-based，闭区间；startLine<=0 视为 1）。
// 泄洪的多是行式文本（文件正文、表格行），按字节分页会把中段切碎、要多轮拼装
// （2026-11-25 铜陵反馈 #12）；行区间一次直达。返回（区间文本[≤maxBytes 截断],
// 总行数, error）。
func (s *Store) RetrieveLines(id string, startLine, endLine, maxBytes int) (string, int, error) {
	if s == nil {
		return "", 0, fmt.Errorf("spill store unavailable in this context")
	}
	if maxBytes <= 0 {
		maxBytes = ReadDefaultBytes
	}
	if maxBytes > ReadMaxBytes {
		maxBytes = ReadMaxBytes
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return "", 0, fmt.Errorf("spill %q not found (expired or never existed in this session)", id)
	}
	lines := strings.Split(e.Data, "\n")
	total := len(lines)
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 || endLine > total {
		endLine = total
	}
	if startLine > total {
		return "", total, fmt.Errorf("spill %q start_line %d out of range (total %d lines)", id, startLine, total)
	}
	if endLine < startLine {
		endLine = startLine
	}
	chunk := strings.Join(lines[startLine-1:endLine], "\n")
	if len(chunk) > maxBytes {
		chunk = chunk[:maxBytes]
	}
	return chunk, total, nil
}

// storeKey 是泄洪库的 ctx 盖章键（read_spill 工具经此取库——工具是无状态
// builtin，库在 runner 侧，与 jobs/memory queue 同注入纪律）。
type storeKey struct{}

// WithStore 把泄洪库盖章进工具调用 ctx。nil 库不盖章（read_spill 报不可用）。
func WithStore(ctx context.Context, s *Store) context.Context {
	if s == nil {
		return ctx
	}
	return context.WithValue(ctx, storeKey{}, s)
}

// FromContext 读取执行器盖章的泄洪库。
func FromContext(ctx context.Context) (*Store, bool) {
	s, ok := ctx.Value(storeKey{}).(*Store)
	return s, ok && s != nil
}
