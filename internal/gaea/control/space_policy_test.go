package control

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// spaceStubTool 最小 tool.Tool（空间策略测试用）。
type spaceStubTool struct{ name string }

func (s spaceStubTool) Name() string                                             { return s.name }
func (s spaceStubTool) Description() string                                      { return "" }
func (s spaceStubTool) Schema() json.RawMessage                                  { return json.RawMessage(`{"type":"object"}`) }
func (s spaceStubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }
func (s spaceStubTool) ReadOnly() bool                                           { return false }

// TestHardAskSetParameterized hardAsk 参数化（S1.5-A）：Options.HardAskTools
// 非 nil 快照生效（含空集），nil 回退包级默认集（现状）。
func TestHardAskSetParameterized(t *testing.T) {
	c := New(Options{HardAskTools: map[string]bool{}})
	if got := c.hardAskSet(); got == nil || len(got) != 0 {
		t.Fatalf("空集应生效: %v", got)
	}
	custom := New(Options{HardAskTools: map[string]bool{"cost_save": true}})
	if got := custom.hardAskSet(); len(got) != 1 || !got["cost_save"] {
		t.Fatalf("自定义集应生效: %v", got)
	}
	def := New(Options{})
	if got := def.hardAskSet(); len(got) != len(hardAskTools) {
		t.Fatalf("nil 应回退包级默认集: got %v want %v", got, hardAskTools)
	}
}

// TestHardAskEmptySetPersistsRule play 语义（hard_ask 空集）：cost_save 不再
// 强制逐条确认——persist_allow 正常回写规则（对照默认集的完全降级不回写）。
func TestHardAskEmptySetPersistsRule(t *testing.T) {
	c, ids, _ := approvalIDs()
	c.hardAskTools = map[string]bool{} // 等价 boot 按 play 策略注入空集（Options.HardAskTools）
	rules := make(chan string, 8)
	c.persistAllowRule = func(rule string) error { rules <- rule; return nil }

	go func() { c.Approve(<-ids, DecisionPersistAllow) }()
	allow, remember, err := gateApprover{c}.Approve(context.Background(), "cost_save", "", json.RawMessage(`{"title":"测试"}`))
	if err != nil || !allow || remember {
		t.Fatalf("空集下 persist_allow = (%v,%v,%v), want allow", allow, remember, err)
	}
	select {
	case r := <-rules:
		if r != "cost_save" {
			t.Fatalf("回写规则 = %q, want cost_save", r)
		}
	case <-time.After(time.Second):
		t.Fatal("空集下 persist_allow 应回写策略文件（不再按 hardAsk 完全降级）")
	}
}

// TestPlayPolicyNoApprovalCard play 装配语义（产品确认）：mode=allow +
// hard_ask 空集 → 记忆/知识库等写入不再弹审批卡；deny 硬拒绝规则仍生效。
// gate 按 EnableInteractiveApproval/SetPermLevel 的挂点方式组装。
func TestPlayPolicyNoApprovalCard(t *testing.T) {
	requests := 0
	c := New(Options{
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.ApprovalRequest {
				requests++
			}
		}),
		Policy:       permission.New("allow", nil, nil, []string{"bash(rm*)"}),
		HardAskTools: map[string]bool{}, // play 产品默认空集
	})
	g := permission.NewGate(c.policy, gateApprover{c})
	g.AlwaysAsk = c.hardAskSet() // 与 controller.go 三个挂点一致

	// 写入类（remember 非只读）：mode=allow 直接放行，不弹卡。
	allow, reason, err := g.Check(context.Background(), "remember", json.RawMessage(`{"name":"偏好","title":"t"}`), false)
	if err != nil || !allow || reason != "" {
		t.Fatalf("play remember Check = (%v,%q,%v), want 放行", allow, reason, err)
	}
	// AlwaysAsk 空集：cost_save 也不经 hardAsk 强制确认（策略 allow 放行）。
	allow, _, err = g.Check(context.Background(), "cost_save", json.RawMessage(`{"title":"x"}`), false)
	if err != nil || !allow {
		t.Fatalf("play cost_save Check = (%v,%v), want 放行", allow, err)
	}
	if requests != 0 {
		t.Fatalf("play 不应弹审批卡，实际发出 %d 张", requests)
	}

	// 硬拒绝规则仍生效：deny > allow。
	allow, reason, err = g.Check(context.Background(), "bash", json.RawMessage(`{"command":"rm -rf /"}`), false)
	if err != nil || allow || reason == "" {
		t.Fatalf("play deny 规则应硬拒绝: (%v,%q,%v)", allow, reason, err)
	}
}

// TestSpaceAllowedTools MCP spec 层过滤（S1.3-B）：shared 保留、work/play 按
// 空间保留、mode=off（空空间）全保留。
func TestSpaceAllowedTools(t *testing.T) {
	tools := []tool.Tool{
		spaceStubTool{name: "edit_file"},   // work（名字表）
		spaceStubTool{name: "image_gen"},   // play（名字表）
		spaceStubTool{name: "memory_search"}, // shared（名字表）
		spaceStubTool{name: "mcp__gh__search"}, // MCP 动态名缺省 shared
	}
	work := spaceAllowedTools(tools, "work")
	if len(work) != 3 || work[0].Name() != "edit_file" || work[1].Name() != "memory_search" || work[2].Name() != "mcp__gh__search" {
		t.Fatalf("work 过滤结果 = %v", names(work))
	}
	play := spaceAllowedTools(tools, "play")
	if len(play) != 3 || play[0].Name() != "image_gen" {
		t.Fatalf("play 过滤结果 = %v", names(play))
	}
	off := spaceAllowedTools(tools, "")
	if len(off) != 4 {
		t.Fatalf("mode=off 应全保留: %v", names(off))
	}
}

func names(tools []tool.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name())
	}
	return out
}
