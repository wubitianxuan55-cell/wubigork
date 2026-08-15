package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// ── LLM seam 常量 ──────────────────────────────────────────────

func TestLLMKindConstants(t *testing.T) {
	if LLMKindWubigrok != "wubigrok" {
		t.Errorf("LLMKindWubigrok = %q, want %q", LLMKindWubigrok, "wubigrok")
	}
	if DefaultLLMKind != LLMKindWubigrok {
		t.Errorf("DefaultLLMKind = %q, want LLMKindWubigrok (%q)", DefaultLLMKind, LLMKindWubigrok)
	}
}

// ── LLM seam 三纪律：提供者互斥注册（重复 panic） ──────────────

func TestNewLLM_DuplicateKindPanics(t *testing.T) {
	kind := "llm-test-dup-" + t.Name()
	Register(kind, func(cfg Config) (Provider, error) { return &mockProvider{}, nil })
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("duplicate Register must panic (seam 三纪律：提供者互斥注册)")
		}
	}()
	// 第二次注册同一 kind → 编译期接线错误，必须 panic。
	Register(kind, func(cfg Config) (Provider, error) { return &mockProvider{}, nil })
}

// ── LLM seam 三纪律：未知 kind fail-closed ─────────────────────

func TestNewLLM_UnknownKindFailsClosed(t *testing.T) {
	_, err := NewLLM("llm-test-unknown-xyzzy", Config{})
	if err == nil {
		t.Fatal("unknown kind must fail closed (不可用即 fail-closed，绝不静默降级)")
	}
	if !strings.Contains(err.Error(), "unknown kind") {
		t.Errorf("error should name the unknown kind: %v", err)
	}
}

// ── LLM seam：按 config 的 kind 选择（切换后端只改配置） ───────

// namedProvider 是一个最小 Provider（无原生 Chat），Name 可配置。
type namedProvider struct{ name string }

func (p *namedProvider) Name() string { return p.name }
func (p *namedProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	ch := make(chan Chunk, 1)
	ch <- Chunk{Type: ChunkDone}
	close(ch)
	return ch, nil
}

func TestNewLLM_SelectsByKind(t *testing.T) {
	kindA := "llm-test-a-" + t.Name()
	kindB := "llm-test-b-" + t.Name()
	Register(kindA, func(cfg Config) (Provider, error) { return &namedProvider{name: "A"}, nil })
	Register(kindB, func(cfg Config) (Provider, error) { return &namedProvider{name: "B"}, nil })

	pa, err := NewLLM(kindA, Config{})
	if err != nil {
		t.Fatalf("NewLLM(%s): %v", kindA, err)
	}
	if pa.Name() != "A" {
		t.Errorf("kindA provider Name = %q, want A", pa.Name())
	}
	pb, err := NewLLM(kindB, Config{})
	if err != nil {
		t.Fatalf("NewLLM(%s): %v", kindB, err)
	}
	if pb.Name() != "B" {
		t.Errorf("kindB provider Name = %q, want B", pb.Name())
	}
}

// ── LLM seam：未原生实现 Chat 的提供者自动适配（注册即 LLM 能力） ──

func TestNewLLM_AutoWrapsProviderWithoutChat(t *testing.T) {
	kind := "llm-test-wrap-" + t.Name()
	// mockProvider（provider_test.go）只实现 Name/Stream，无 Chat。
	Register(kind, func(cfg Config) (Provider, error) { return &mockProvider{}, nil })

	p, err := NewLLM(kind, Config{})
	if err != nil {
		t.Fatalf("NewLLM: %v", err)
	}
	if p.Name() != "mock" {
		t.Errorf("Name = %q, want mock", p.Name())
	}
	// 自动包一层 Stream 聚合适配器后，Chat 可用且结果与 Stream 一致。
	c, err := p.Chat(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat (auto-wrapped): %v", err)
	}
	if c.Content != "" || len(c.ToolCalls) != 0 {
		t.Errorf("auto-wrapped Chat = %+v, want empty completion", c)
	}
}

// ── LLM seam：原生实现 Chat 的提供者不被包装 ───────────────────

// nativeChatProvider 原生实现 Chat（不经 Stream 聚合），用于验证 NewLLM
// 类型断言优先返回原生实现。
type nativeChatProvider struct{}

func (nativeChatProvider) Name() string { return "native" }
func (nativeChatProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	ch := make(chan Chunk, 2)
	ch <- Chunk{Type: ChunkText, Text: "via-stream"}
	ch <- Chunk{Type: ChunkDone}
	close(ch)
	return ch, nil
}
func (nativeChatProvider) Chat(ctx context.Context, req Request) (*Completion, error) {
	return &Completion{Content: "via-native-chat"}, nil
}

func TestNewLLM_NativeChatNotWrapped(t *testing.T) {
	kind := "llm-test-native-" + t.Name()
	Register(kind, func(cfg Config) (Provider, error) { return nativeChatProvider{}, nil })

	p, err := NewLLM(kind, Config{})
	if err != nil {
		t.Fatalf("NewLLM: %v", err)
	}
	c, err := p.Chat(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if c.Content != "via-native-chat" {
		t.Errorf("Chat = %q, want native implementation to win (via-native-chat)", c.Content)
	}
}

// ── ChatFromStream：聚合语义 ──────────────────────────────────

// scriptedProvider 按预设 chunk 序列回放（测试专用）。
type scriptedProvider struct {
	chunks []Chunk
}

func (scriptedProvider) Name() string { return "scripted" }
func (s *scriptedProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	ch := make(chan Chunk, len(s.chunks))
	for _, c := range s.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func TestChatFromStream_AggregatesCompletion(t *testing.T) {
	tc := &ToolCall{ID: "c1", Name: "ls", Arguments: "{}"}
	p := &scriptedProvider{chunks: []Chunk{
		{Type: ChunkReasoning, Text: "思考", Signature: "sig-1"},
		{Type: ChunkText, Text: "你好"},
		{Type: ChunkText, Text: "世界"},
		{Type: ChunkToolCall, ToolCall: tc},
		{Type: ChunkUsage, Usage: &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, FinishReason: "tool_calls"}},
		{Type: ChunkDone},
	}}
	c, err := ChatFromStream(context.Background(), p, Request{})
	if err != nil {
		t.Fatalf("ChatFromStream: %v", err)
	}
	if c.Content != "你好世界" {
		t.Errorf("Content = %q, want 你好世界", c.Content)
	}
	if c.ReasoningContent != "思考" || c.ReasoningSignature != "sig-1" {
		t.Errorf("reasoning = %q/%q", c.ReasoningContent, c.ReasoningSignature)
	}
	if len(c.ToolCalls) != 1 || c.ToolCalls[0].ID != "c1" {
		t.Errorf("tool calls = %+v, want c1", c.ToolCalls)
	}
	if c.Usage == nil || c.Usage.TotalTokens != 15 || c.FinishReason != "tool_calls" {
		t.Errorf("usage/finish = %+v / %q", c.Usage, c.FinishReason)
	}
}

func TestChatFromStream_StreamErrorPropagates(t *testing.T) {
	p := &scriptedProvider{chunks: []Chunk{
		{Type: ChunkText, Text: "partial"},
		{Type: ChunkError, Err: errors.New("boom")},
	}}
	if _, err := ChatFromStream(context.Background(), p, Request{}); err == nil || err.Error() != "boom" {
		t.Fatalf("ChatFromStream error = %v, want boom", err)
	}
}

// errorStartProvider 的 Stream 直接返回错误。
type errorStartProvider struct{}

func (errorStartProvider) Name() string { return "err" }
func (errorStartProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	return nil, errors.New("start-fail")
}

func TestChatFromStream_StartErrorPropagates(t *testing.T) {
	if _, err := ChatFromStream(context.Background(), &errorStartProvider{}, Request{}); err == nil || err.Error() != "start-fail" {
		t.Fatalf("ChatFromStream = %v, want start-fail", err)
	}
	if _, err := ChatFromStream(context.Background(), nil, Request{}); err == nil {
		t.Fatal("nil provider should error")
	}
}
