package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestRunPersistedSubAgentReceiptCarriesRef：技能子代理（runAs=subagent 的
// run_skill 派生路径）成功回执必须附 "Subagent reference: sa_…" 尾巴——
// 2026-11-20 委派家族收窄后，前端 isDelegationCall 靠这行把 run_skill 调用
// 升级成委派卡（inline/pipeline 无此行按普通工具渲染）；缺行=技能子代理卡
// 既无徽标也不可点。
func TestRunPersistedSubAgentReceiptCarriesRef(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	prov := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "技能子代理的最终答复"},
		{Type: provider.ChunkDone},
	}}
	reg := tool.NewRegistry()

	out, err := RunPersistedSubAgent(context.Background(), prov, reg,
		"sys-prompt", "task body", "任务标题", Options{}, event.Discard, store, nil)
	if err != nil {
		t.Fatalf("RunPersistedSubAgent: %v", err)
	}
	if !strings.HasPrefix(out, "技能子代理的最终答复") {
		t.Errorf("回执应原样携带最终答复在前，got %q", out)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("应恰好一条运行记录，got %d", len(runs))
	}
	if want := "Subagent reference: " + runs[0].Ref; !strings.Contains(out, want) {
		t.Errorf("回执缺子代理引用行 %q，got %q", want, out)
	}
}
