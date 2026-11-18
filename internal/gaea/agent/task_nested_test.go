package agent

// 嵌套委派测试（2026-11-18，dsh 完整智能体形态蒸馏：子智能体可再派孙代）：
// 三层血缘链+深度记账、深度闸边界、嵌套容量快拒（防自锁）、协调元工具的
// 「自己的子树」守卫。座子复用既有惯例（scriptedProvider/gateProvider/
// setSlotsForTest/fakeTool）。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// nestedFixture 组装带嵌套深度的 task 工具 + 全套协调元工具注册表。
func nestedFixture(t *testing.T, prov provider.LLMProvider, maxDepth int, store *SubagentStore) (*TaskTool, *tool.Registry) {
	t.Helper()
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	task := NewTaskTool(prov, nil, reg, 12, 0, 0.0, "", "sys", nil)
	task.SetSubagentMaxDepth(maxDepth)
	if store != nil {
		task.WithTranscripts(store)
	}
	reg.Add(task)
	reg.Add(NewSubagentListTool(store))
	reg.Add(NewInterruptTool(store))
	reg.Add(NewReadSubagentResultTool(store))
	return task, reg
}

// TestNestedDelegationThreeLevels 三层端到端：L1 派 L2、L2 派 L3，各自收尾
// 归因；meta 血缘链（parentRef/Depth）与 AncestryContains 全链核验。
func TestNestedDelegationThreeLevels(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	// 轮次脚本：L1→(派L2)→L2→(派L3)→L3(答)→L2(答)→L1(答)。
	prov := &scriptedProvider{name: "nested", turns: [][]provider.Chunk{
		{toolCallChunk("c2", "task", `{"prompt":"level2 please"}`)},
		{toolCallChunk("c3", "task", `{"prompt":"level3 please"}`)},
		{{Type: provider.ChunkText, Text: "L3-final"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "L2-final"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "L1-final"}, {Type: provider.ChunkDone}},
	}}
	task, _ := nestedFixture(t, prov, 3, store)

	out, err := task.Execute(context.Background(), []byte(`{"prompt":"top"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "L1-final") {
		t.Fatalf("顶层应拿一代的最终回答，got %q", out)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("应恰好三条运行（一代/二代/三代），got %d", len(runs))
	}
	byDepth := map[int]SubagentRunInfo{}
	for _, r := range runs {
		byDepth[r.Depth] = r
	}
	l1, ok1 := byDepth[1]
	l2, ok2 := byDepth[2]
	l3, ok3 := byDepth[3]
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("深度记账缺级: L1=%v L2=%v L3=%v", ok1, ok2, ok3)
	}
	if l1.ParentRef != "" {
		t.Fatalf("一代 parentRef 应空（主会话直派），got %q", l1.ParentRef)
	}
	if l2.ParentRef != l1.Ref || l3.ParentRef != l2.Ref {
		t.Fatalf("血缘链断裂: L2.parent=%q L3.parent=%q", l2.ParentRef, l3.ParentRef)
	}
	if !store.AncestryContains(l1.Ref, l2.Ref) || !store.AncestryContains(l1.Ref, l3.Ref) {
		t.Fatalf("L1 应是 L2/L3 的祖先")
	}
	if store.AncestryContains(l2.Ref, l1.Ref) {
		t.Fatalf("祖先关系不可反向")
	}
}

// TestNestedDepthCapContract 深度闸契约：未开（maxDepth=1）全量排除；开启
// 后协调四件对未到底级别可见、非委派面恒排除；到底级别回到全量排除。
func TestNestedDepthCapContract(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	reg.Add(fakeTool{name: "run_skill", readOnly: false})
	// 协调四件入父注册表（嵌套开放的前提是它们存在于父面）。
	reg.Add(NewInterruptTool(nil))
	reg.Add(NewSubagentListTool(NewSubagentStore(t.TempDir())))
	reg.Add(NewReadSubagentResultTool(NewSubagentStore(t.TempDir())))
	prov := &mockProvider{name: "cap", chunks: []provider.Chunk{{Type: provider.ChunkDone}}}

	// 关闭（未 Set = 零值 / 显式 1）：level1 就没有 task——现行为逐字节。
	closed := NewTaskTool(prov, nil, reg, 12, 0, 0.0, "", "sys", nil)
	closed.SetSubagentMaxDepth(1)
	reg.Add(closed)
	if reg2, err := closed.buildSubRegAt(nil, false, 1); err != nil {
		t.Fatalf("buildSubRegAt: %v", err)
	} else if _, ok := reg2.Get("task"); ok {
		t.Fatalf("maxDepth=1 时 level1 不应见 task")
	}

	// 开启（maxDepth=3）：level1/2 可见协调四件；run_skill 恒排除；level3 到底全排除。
	open := NewTaskTool(prov, nil, reg, 12, 0, 0.0, "", "sys", nil)
	open.SetSubagentMaxDepth(3)
	reg.Add(open)
	for _, level := range []int{1, 2} {
		regN, err := open.buildSubRegAt(nil, false, level)
		if err != nil {
			t.Fatalf("buildSubRegAt(level=%d): %v", level, err)
		}
		for _, want := range []string{"task", "interrupt_agent", "subagent_list", "read_subagent_result", "read_file"} {
			if _, ok := regN.Get(want); !ok {
				t.Fatalf("level%d 应见 %q", level, want)
			}
		}
		for _, banned := range []string{"run_skill", "install_skill", "exit_plan_mode"} {
			if _, ok := regN.Get(banned); ok {
				t.Fatalf("level%d 不应见 %q", level, banned)
			}
		}
	}
	reg3, err := open.buildSubRegAt(nil, false, 3)
	if err != nil {
		t.Fatalf("buildSubRegAt(level=3): %v", err)
	}
	if _, ok := reg3.Get("task"); ok {
		t.Fatalf("到底级别（level>=maxDepth）不应见 task")
	}

	// ExecuteWithContext 防御核验：在超限级别直接调 task → fail-loud。
	deepCtx := withSubagentDepth(context.Background(), 3)
	if _, err := open.Execute(deepCtx, []byte(`{"prompt":"x"}`)); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("超限派发应报 depth limit，got %v", err)
	}
}

// TestNestedCapacityFailFast 嵌套派发槽满快拒不排队（防「持槽等后代」自锁）；
// 顶层派发维持排队语义（ctx 取消收口 queued→aborted）。
func TestNestedCapacityFailFast(t *testing.T) {
	setSlotsForTest(t, 1)
	store := NewSubagentStore(t.TempDir())
	gate := make(chan struct{})
	prov := &gateProvider{name: "gate", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone},
	}}
	task, _ := nestedFixture(t, prov, 3, store)

	// 占槽者：顶层派发，provider 阻塞在 gate 上（持槽在跑）。
	occupyDone := make(chan error, 1)
	go func() {
		_, err := task.Execute(context.Background(), []byte(`{"prompt":"occupy"}`))
		occupyDone <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for SubagentSlotsBusy() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if SubagentSlotsBusy() != 1 {
		t.Fatalf("占槽者应已持槽在跑，busy=%d", SubagentSlotsBusy())
	}

	// 嵌套派发（caller 级别 1）：槽满必须**立即**失败——不排队。用短超时
	// 证明没阻塞：若排队会吃满超时并返回 ctx.Err 而非容量文案。
	nestedCtx, cancel := context.WithTimeout(withSubagentDepth(context.Background(), 1), 800*time.Millisecond)
	defer cancel()
	_, err := task.Execute(nestedCtx, []byte(`{"prompt":"nested"}`))
	if err == nil || !strings.Contains(err.Error(), "nested delegation at capacity") {
		t.Fatalf("嵌套槽满应快拒报容量，got %v", err)
	}

	// 顶层派发走排队：同槽满状态下阻塞等槽（ctx 超时取消 → queued 语义收口）。
	topCtx, cancelTop := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancelTop()
	_, err = task.Execute(topCtx, []byte(`{"prompt":"top-queued"}`))
	if err == nil || !strings.Contains(err.Error(), "sub-agent queued") {
		t.Fatalf("顶层槽满应排队（ctx 取消收口），got %v", err)
	}

	close(gate)
	if err := <-occupyDone; err != nil {
		t.Fatalf("占槽者收尾: %v", err)
	}
}

// TestCoordinationToolsScopedToSubtree 协调元工具的「自己的子树」守卫：
// 子代理调用只见/可停/可读自己的后代，兄弟与主会话其它派生一律拒。
func TestCoordinationToolsScopedToSubtree(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	mk := func(parentRef, answer string) string {
		run, err := store.PrepareFresh("sys", "")
		if err != nil {
			t.Fatalf("PrepareFresh: %v", err)
		}
		run.ParentRef = parentRef
		run.Depth = 1
		if answer != "" {
			run.Session.Add(provider.Message{Role: provider.RoleAssistant, Content: answer})
		}
		if err := store.MarkRunning(run); err != nil {
			t.Fatalf("MarkRunning: %v", err)
		}
		if err := store.SaveCompleted(run); err != nil {
			t.Fatalf("SaveCompleted: %v", err)
		}
		return run.Ref
	}
	l1 := mk("", "")
	l2 := mk(l1, "L2-secret")
	sibling := mk("", "sibling-secret")

	// subagent_list：以 L1 身份调用只见 L2。
	list := NewSubagentListTool(store)
	out, err := list.Execute(WithSubagentRef(context.Background(), l1), []byte(`{}`))
	if err != nil {
		t.Fatalf("list Execute: %v", err)
	}
	if !strings.Contains(out, l2) || strings.Contains(out, sibling) {
		t.Fatalf("子代理清单应收窄到自己的子树\n%s", out)
	}
	// 主会话身份（无盖）：全量可见（现行为）。
	outMain, err := list.Execute(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("list main Execute: %v", err)
	}
	if !strings.Contains(outMain, l2) || !strings.Contains(outMain, sibling) {
		t.Fatalf("主会话清单应全量\n%s", outMain)
	}

	// interrupt_agent：L1 不能停兄弟，能停自己的后代（后代不在跑 → 落到
	// 「不在跑」错误，证明守卫已通过）。
	it := NewInterruptTool(store)
	if _, err := it.Execute(WithSubagentRef(context.Background(), l1), mustJSON(map[string]string{"ref": sibling})); err == nil || !strings.Contains(err.Error(), "not a descendant") {
		t.Fatalf("兄弟中断应被子树守卫拒绝，got %v", err)
	}
	if _, err := it.Execute(WithSubagentRef(context.Background(), l1), mustJSON(map[string]string{"ref": l2})); err == nil || strings.Contains(err.Error(), "not a descendant") {
		t.Fatalf("后代中断应过守卫（落「不在跑」）, got %v", err)
	}
	// job 面（非 sa_ 引用）对子代理整个不开放。
	if _, err := it.Execute(WithSubagentRef(context.Background(), l1), mustJSON(map[string]string{"ref": "job_1"})); err == nil || !strings.Contains(err.Error(), "main session") {
		t.Fatalf("子代理杀 job 应拒绝，got %v", err)
	}

	// read_subagent_result：L1 可读后代、读不到兄弟。
	rt := NewReadSubagentResultTool(store)
	outRead, err := rt.Execute(WithSubagentRef(context.Background(), l1), mustJSON(map[string]string{"ref": l2}))
	if err != nil {
		t.Fatalf("read Execute: %v", err)
	}
	if !strings.Contains(outRead, "L2-secret") {
		t.Fatalf("后代结果应可读: %s", outRead)
	}
	outSib, err := rt.Execute(WithSubagentRef(context.Background(), l1), mustJSON(map[string]string{"ref": sibling}))
	if err != nil {
		t.Fatalf("read sibling Execute: %v", err)
	}
	if !strings.Contains(outSib, "outside your own sub-agent tree") {
		t.Fatalf("兄弟结果应拒读: %s", outSib)
	}
}

func mustJSON(v map[string]string) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
