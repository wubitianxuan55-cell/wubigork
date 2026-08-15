package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

type planStubProvider struct {
	err bool
}

func (planStubProvider) Name() string { return "stub" }

func (p planStubProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	if p.err {
		return nil, errors.New("boom")
	}
	ch := make(chan provider.Chunk, 3)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: "1. 读取资料\n"}
	ch <- provider.Chunk{Type: provider.ChunkText, Text: "2. 生成表格\n"}
	close(ch)
	return ch, nil
}

// Chat 满足 LLM seam 接口：聚合本 stub 的 Stream。
func (p planStubProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, p, req)
}

func TestAgentRunnerPlan(t *testing.T) {
	r := New(planStubProvider{}, nil, nil, Options{}, nil)
	got, err := r.Plan(context.Background(), "你是 gaea", "帮我做成本测算")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if got != "1. 读取资料\n2. 生成表格" {
		t.Fatalf("Plan = %q", got)
	}
}

func TestAgentRunnerPlanError(t *testing.T) {
	r := New(planStubProvider{err: true}, nil, nil, Options{}, nil)
	if _, err := r.Plan(context.Background(), "", "x"); err == nil {
		t.Fatal("expected error")
	}
}
