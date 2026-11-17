package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// D5 容量口径自检（dsh 蒸馏计划 01 §4·D5）：上游 workflow 的
// maxConcurrentAgents（协作排队池）与 subagent 硬限（maxActiveSubagents=8，
// 超限 fatal）是两个性质不同的池，16 核机器第 9 个并发即炸；gaea 的对应面
// 是单闸不变量——所有「每实例一条 LLM 循环」的子代理负载（task 派发 /
// run_skill / RunNew 宿主 DAG 节点 / runFollowUp 追问）都过同一扇
// acquireSubSlot 门，超限一律排队（queued 侧车可见）而非拒绝或绕行。
// 本例锁宿主路径：RunNew 槽满必排队，槽释放后起跑完成。
func TestRunNewRespectsParallelGate(t *testing.T) {
	setSlotsForTest(t, 1)
	dir := t.TempDir()
	store := NewSubagentStore(dir)
	gate := make(chan struct{})
	gated := &gateProvider{name: "gated", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "dag-node"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(gated, nil, parentReg, 10, 0, 0.0, "", "sys", nil).WithTranscripts(store)

	// 第一路 RunNew 起跑并持住唯一槽位（gated provider 挂起 LLM 采样）。
	firstDone := make(chan error, 1)
	go func() {
		_, _, err := task.RunNew(context.Background(), "first", nil)
		firstDone <- err
	}()
	waitFor(t, 2*time.Second, "first RunNew to hold the slot", func() bool {
		return SubagentSlotsBusy() == 1
	})

	// 第二路 RunNew 必须排队：不持槽、不完成（绕闸回归在此暴露）。
	secondDone := make(chan error, 1)
	go func() {
		_, _, err := task.RunNew(context.Background(), "second", nil)
		secondDone <- err
	}()
	time.Sleep(100 * time.Millisecond)
	if busy := SubagentSlotsBusy(); busy != 1 {
		t.Fatalf("second RunNew must queue while the only slot is busy, busy=%d", busy)
	}
	select {
	case err := <-secondDone:
		t.Fatalf("second RunNew must not finish while queued, err=%v", err)
	default:
	}
	// 排队诚实化：等待期的 DAG 节点有 queued 侧车（分工面板可见）。
	queuedDeadline := time.Now().Add(2 * time.Second)
	queuedOK := false
	for time.Now().Before(queuedDeadline) {
		if _, queued := countMetas(t, dir, "queued"); queued == 1 {
			queuedOK = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !queuedOK {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if b, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
				t.Logf("meta %s: %s", e.Name(), b)
			}
		}
		total, _ := countMetas(t, dir, "queued")
		t.Fatalf("queued meta missing (total=%d)", total)
	}

	// 放行第一路：槽释放 → 第二路起跑并完成。
	close(gate)
	if err := <-firstDone; err != nil {
		t.Fatalf("first RunNew: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second RunNew after release: %v", err)
	}
	if busy := SubagentSlotsBusy(); busy != 0 {
		t.Fatalf("slots must drain to zero, busy=%d", busy)
	}
}
