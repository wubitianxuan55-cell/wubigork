package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── P0-B：白名单未知名 fail-loud（dsh toolFilter loud unknown-name 纪律）──

func TestUnknownToolNames(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file"})
	reg.Add(fakeTool{name: "grep"})
	got := UnknownToolNames(reg, []string{"read_file", "nope", "grep", "also_missing"})
	if len(got) != 2 || got[0] != "nope" || got[1] != "also_missing" {
		t.Fatalf("UnknownToolNames = %v, want [nope also_missing]", got)
	}
	// 排除清单（元工具按设计静默）不算未知
	if got := UnknownToolNames(reg, []string{"task", "read_file"}, "task"); len(got) != 0 {
		t.Fatalf("排除清单内的名字不应算未知: %v", got)
	}
	// 空白名单（继承父全量）恒无未知
	if got := UnknownToolNames(reg, nil); got != nil {
		t.Fatalf("空清单应返回 nil, got %v", got)
	}
}

func TestBuildSubRegFailsLoudOnUnknownNames(t *testing.T) {
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file"})
	task := NewTaskTool(&mockProvider{name: "p"}, nil, parentReg, 10, 0, 0.0, "", "sys", nil)

	_, err := task.Execute(context.Background(), []byte(`{"prompt":"x","tools":["read_file","grep_typo"]}`))
	if err == nil || !strings.Contains(err.Error(), "grep_typo") {
		t.Fatalf("未知白名单名应报错且点名错名: %v", err)
	}
	// 元工具在清单内不炸（按设计静默剔除）——直接测 buildSubReg（Execute
	// 会继续走 LLM 循环，mock 无 chunks 的报错与白名单无关）。
	if _, err := task.buildSubReg([]string{"read_file", "task"}); err != nil {
		t.Fatalf("元工具白名单不应报错: %v", err)
	}
}

// ── P0-A：interrupt_agent 中断面 ─────────────────────────────────────

func findLiveRunnerRef(t *testing.T, wantRunning bool) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var found string
		subRuns.Range(func(key, value any) bool {
			h := value.(*subRunHandle)
			if wantRunning && h.runner != nil {
				found = key.(string)
				return false
			}
			if !wantRunning && h.runner == nil {
				found = key.(string)
				return false
			}
			return true
		})
		if found != "" {
			return found
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for a subRuns entry (running=%v)", wantRunning)
	return ""
}

// 运行中中断：gated provider 占住 LLM 采样，interrupt 后 task 结果如实报错
// 收尾，侧车落 failed。
func TestInterruptRunningSubagent(t *testing.T) {
	setSlotsForTest(t, 3)
	dir := t.TempDir()
	store := NewSubagentStore(dir)
	gate := make(chan struct{})
	gated := &gateProvider{name: "gated", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "unused"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(gated, nil, parentReg, 10, 0, 0.0, "", "sys", nil).WithTranscripts(store)

	done := make(chan error, 1)
	go func() {
		_, err := task.Execute(context.Background(), []byte(`{"prompt":"long research"}`))
		done <- err
	}()
	ref := findLiveRunnerRef(t, true)

	if err := InterruptSubagent(ref); err != nil {
		t.Fatalf("InterruptSubagent: %v", err)
	}
	if err := <-done; err == nil {
		t.Fatal("被中断的 task 应返回错误（ctx 取消）")
	}
	waitFor(t, 2*time.Second, "sidecar to settle failed", func() bool {
		_, failed := countMetas(t, dir, "failed")
		return failed == 1
	})
}

// 排队中中断：并发槽位占满时排队调用可被取消，queued 侧车收口 failed。
func TestInterruptQueuedSubagent(t *testing.T) {
	setSlotsForTest(t, 1)
	dir := t.TempDir()
	store := NewSubagentStore(dir)
	gate := make(chan struct{})
	gated := &gateProvider{name: "gated", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "first"},
		{Type: provider.ChunkDone},
	}}
	fast := &mockProvider{name: "fast", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "unused"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(gated, nil, parentReg, 10, 0, 0.0, "", "sys", nil).WithTranscripts(store)

	firstDone := make(chan error, 1)
	go func() {
		_, err := task.Execute(context.Background(), []byte(`{"prompt":"first"}`))
		firstDone <- err
	}()
	waitFor(t, 2*time.Second, "first to hold the slot", func() bool { return SubagentSlotsBusy() == 1 })
	// 等第一路的 runner 注册进句柄（runForeground 先挂 {cancel}、起跑后由
	// runSubAgentInternal 覆写 {runner,cancel}）——此后 runner==nil 的条目
	// 才确定是第二路的排队句柄。
	var runningRef string
	waitFor(t, 2*time.Second, "first runner to register", func() bool {
		subRuns.Range(func(key, value any) bool {
			if value.(*subRunHandle).runner != nil {
				runningRef = key.(string)
				return false
			}
			return true
		})
		return runningRef != ""
	})

	task.prov = fast
	secondDone := make(chan error, 1)
	go func() {
		_, err := task.Execute(context.Background(), []byte(`{"prompt":"second"}`))
		secondDone <- err
	}()
	waitFor(t, 2*time.Second, "second to register queued", func() bool {
		_, queued := countMetas(t, dir, "queued")
		return queued == 1
	})
	var queuedRef string
	subRuns.Range(func(key, value any) bool {
		if key.(string) != runningRef && value.(*subRunHandle).runner == nil {
			queuedRef = key.(string)
			return false
		}
		return true
	})
	if queuedRef == "" {
		t.Fatal("第二路的排队句柄未登记")
	}
	if err := InterruptSubagent(queuedRef); err != nil {
		t.Fatalf("InterruptSubagent(queued): %v", err)
	}
	if err := <-secondDone; err == nil {
		t.Fatal("排队中被中断的第二路应返回错误")
	}
	waitFor(t, 2*time.Second, "queued meta to settle failed", func() bool {
		_, failed := countMetas(t, dir, "failed")
		return failed == 1
	})
	// 收尾：放行并等第一路正常结束（TempDir 清理不得与快照写盘竞速）。
	close(gate)
	if err := <-firstDone; err != nil {
		t.Fatalf("first call: %v", err)
	}
}

func TestInterruptSubagentErrors(t *testing.T) {
	if err := InterruptSubagent(""); err == nil {
		t.Fatal("空 ref 应拒绝")
	}
	if err := InterruptSubagent("sa_never_existed"); err == nil {
		t.Fatal("未知 ref 应报错")
	}
}

// interrupt_agent 工具面：未知目标报错；接口完整性（CompactDescriptor）
// 由编译器保证，这里钉模型面文案带关键语义。
func TestInterruptToolExecute(t *testing.T) {
	it := NewInterruptTool()
	if _, err := it.Execute(context.Background(), []byte(`{"ref":"sa_ghost"}`)); err == nil {
		t.Fatal("未知 ref 应报错")
	}
	if _, err := it.Execute(context.Background(), []byte(`{}`)); err == nil {
		t.Fatal("缺 ref 应报错")
	}
	if !strings.Contains(it.CompactDescription(), "排队中") || !strings.Contains(it.CompactDescription(), "sa_") {
		t.Fatalf("CompactDescription 应携带 fire-and-return 语义与目标形态: %s", it.CompactDescription())
	}
}
