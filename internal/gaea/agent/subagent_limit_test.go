package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// setSlotsForTest 设定并发上限并在测试结束后恢复默认（包级信号量是共享状态，
// 任何测试不得把非默认容量泄漏给同包其它用例）。
func setSlotsForTest(t *testing.T, n int) {
	t.Helper()
	prev := SubagentMaxParallel()
	SetSubagentMaxParallel(n)
	t.Cleanup(func() { SetSubagentMaxParallel(prev) })
}

// waitFor 轮询谓词直到成立或超时（并发登记是异步落盘的，固定 sleep 既脆又慢）。
func waitFor(t *testing.T, d time.Duration, what string, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// gateProvider 在首次 Stream 时阻塞在 gate 上（模拟子代理 LLM 循环在跑），
// 放行后回放预置 chunks。用于占住并发槽位的集成测试。
type gateProvider struct {
	name   string
	gate   chan struct{}
	chunks []provider.Chunk
}

func (g *gateProvider) Name() string { return g.name }

func (g *gateProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, len(g.chunks))
	go func() {
		defer close(ch)
		select {
		case <-g.gate:
		case <-ctx.Done():
			return
		}
		for _, c := range g.chunks {
			ch <- c
		}
	}()
	return ch, nil
}

func (g *gateProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, g, req)
}

func TestSetSubagentMaxParallelClamps(t *testing.T) {
	setSlotsForTest(t, 3) // 先固定，验收后逐档验证钳制口径
	for _, tc := range []struct {
		in, want int
	}{
		{0, DefaultSubagentMaxParallel},
		{-5, DefaultSubagentMaxParallel},
		{99, MaxSubagentMaxParallel},
		{1, 1},
		{5, 5},
	} {
		SetSubagentMaxParallel(tc.in)
		if got := SubagentMaxParallel(); got != tc.want {
			t.Fatalf("SetSubagentMaxParallel(%d) = cap %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestSubagentSlotCapEnforced(t *testing.T) {
	setSlotsForTest(t, 2)
	const n = 6
	var concurrent, maxConcurrent atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := acquireSubSlot(context.Background()); err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			defer releaseSubSlot()
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
		}()
	}
	wg.Wait()
	if got := maxConcurrent.Load(); got > 2 {
		t.Fatalf("并发峰值 %d 超过上限 2", got)
	}
	if got := maxConcurrent.Load(); got < 2 {
		t.Fatalf("并发峰值 %d < 2：闸没有允许达到上限的并行度", got)
	}
	if busy := SubagentSlotsBusy(); busy != 0 {
		t.Fatalf("全部释放后 busy=%d，want 0", busy)
	}
}

func TestAcquireSubSlotCtxCancel(t *testing.T) {
	setSlotsForTest(t, 1)
	if err := acquireSubSlot(context.Background()); err != nil {
		t.Fatalf("首次获取应成功: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- acquireSubSlot(ctx) }()
	waitFor(t, time.Second, "second acquire to block", func() bool { return SubagentSlotsBusy() == 1 })
	cancel()
	if err := <-errCh; err == nil {
		t.Fatal("槽位占满时取消的等待应返回 ctx.Err()")
	}
	releaseSubSlot()
}

// countMetas 扫 store 目录返回 (总数, 指定状态数)。
func countMetas(t *testing.T, dir, status string) (int, int) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read store dir: %v", err)
	}
	total, matched := 0, 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		total++
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var m struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(b, &m) == nil && m.Status == status {
			matched++
		}
	}
	return total, matched
}

// TestTaskToolQueuedUnderBoundedSlots 集成：上限 1 时第二路前台 task 调用
// 先落 queued 侧车排队（而非假 running / 被拒），槽位释放后自动起跑并完成。
func TestTaskToolQueuedUnderBoundedSlots(t *testing.T) {
	setSlotsForTest(t, 1)
	dir := t.TempDir()
	store := NewSubagentStore(dir)
	gate := make(chan struct{})
	gated := &gateProvider{name: "gated", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "first done"},
		{Type: provider.ChunkDone},
	}}
	fast := &mockProvider{name: "fast", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "second done"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(gated, nil, parentReg, 10, 0, 0.0, "", "sys", nil).WithTranscripts(store)

	type call struct {
		out string
		err error
	}
	first := make(chan call, 1)
	go func() {
		out, err := task.Execute(context.Background(), []byte(`{"prompt":"first task"}`))
		first <- call{out, err}
	}()
	waitFor(t, 2*time.Second, "first call to hold the slot", func() bool { return SubagentSlotsBusy() == 1 })

	second := make(chan call, 1)
	// 第二路换 provider：TaskTool 的 prov 在构造时定死，这里直接换字段
	// （同包测试白盒；真机多路派生共享同一 TaskTool，provider 相同）。
	task.prov = fast
	go func() {
		out, err := task.Execute(context.Background(), []byte(`{"prompt":"second task"}`))
		second <- call{out, err}
	}()
	waitFor(t, 2*time.Second, "second call to register queued", func() bool {
		_, queued := countMetas(t, dir, "queued")
		return queued == 1
	})
	if busy := SubagentSlotsBusy(); busy != 1 {
		t.Fatalf("排队期间在跑子代理数=%d，want 1（排队者未起跑）", busy)
	}

	close(gate)
	fc := <-first
	if fc.err != nil {
		t.Fatalf("first call: %v", fc.err)
	}
	sc := <-second
	if sc.err != nil {
		t.Fatalf("second call: %v", sc.err)
	}
	if !strings.Contains(sc.out, "second done") {
		t.Fatalf("second call output missing result: %q", sc.out)
	}
	// 排队者起跑后终态落 completed（queued → running → completed 全链）。
	waitFor(t, 2*time.Second, "second run to complete", func() bool {
		_, done := countMetas(t, dir, "completed")
		return done == 2
	})
}

// TestTaskToolQueuedCancelMarksFailed：排队等待期间 ctx 取消，queued 侧车
// 如实收口为 failed，不留永久排队假态。
func TestTaskToolQueuedCancelMarksFailed(t *testing.T) {
	setSlotsForTest(t, 1)
	dir := t.TempDir()
	store := NewSubagentStore(dir)
	gate := make(chan struct{})
	gated := &gateProvider{name: "gated", gate: gate, chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "first done"},
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
		_, err := task.Execute(context.Background(), []byte(`{"prompt":"first task"}`))
		firstDone <- err
	}()
	waitFor(t, 2*time.Second, "first call to hold the slot", func() bool { return SubagentSlotsBusy() == 1 })

	task.prov = fast
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := task.Execute(ctx, []byte(`{"prompt":"second task"}`))
		errCh <- err
	}()
	waitFor(t, 2*time.Second, "second call to register queued", func() bool {
		_, queued := countMetas(t, dir, "queued")
		return queued == 1
	})
	cancel()
	if err := <-errCh; err == nil {
		t.Fatal("排队中取消应返回错误")
	}
	waitFor(t, 2*time.Second, "queued meta to settle failed", func() bool {
		_, failed := countMetas(t, dir, "failed")
		return failed == 1
	})
	// 测试结束前放行并等第一路收尾：TempDir 清理不得与子代理快照写盘竞速。
	close(gate)
	if err := <-firstDone; err != nil {
		t.Fatalf("first call: %v", err)
	}
}

func TestMarkQueuedSemantics(t *testing.T) {
	store := NewSubagentStore(t.TempDir())

	// 新鲜 run：queued → running 覆写链。
	fresh, err := store.PrepareFresh("sys", "")
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	defer fresh.Release()
	if err := store.MarkQueued(fresh); err != nil {
		t.Fatalf("MarkQueued: %v", err)
	}
	m, err := store.loadMeta(fresh.Ref)
	if err != nil || m.Status != SubagentQueued {
		t.Fatalf("fresh run meta status = %q (err=%v), want queued", m.Status, err)
	}
	if err := store.MarkRunning(fresh); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	if m, _ = store.loadMeta(fresh.Ref); m.Status != SubagentRunning {
		t.Fatalf("after MarkRunning status = %q, want running", m.Status)
	}

	// 已有终态 meta 的 run（continue_from 装载形态）：MarkQueued 不覆盖。
	done, err := store.PrepareFresh("sys", "")
	if err != nil {
		t.Fatalf("PrepareFresh#2: %v", err)
	}
	defer done.Release()
	if err := store.MarkRunning(done); err != nil {
		t.Fatalf("MarkRunning#2: %v", err)
	}
	if err := store.SaveCompleted(done); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}
	if err := store.MarkQueued(done); err != nil {
		t.Fatalf("MarkQueued on completed: %v", err)
	}
	if m, _ = store.loadMeta(done.Ref); m.Status != SubagentCompleted {
		t.Fatalf("terminal meta overwritten to %q, want completed", m.Status)
	}
}

func TestPrepareContinueRefusesQueued(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	run, err := store.PrepareFresh("sys", "")
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	defer run.Release()
	if err := store.MarkQueued(run); err != nil {
		t.Fatalf("MarkQueued: %v", err)
	}
	if _, err := store.PrepareContinue(run.Ref, ""); err == nil {
		t.Fatal("queued ref 应拒绝续跑")
	} else if !strings.Contains(err.Error(), "queued") {
		t.Fatalf("拒绝原因应点明排队态，got: %v", err)
	}
}

func TestCleanupStaleRunningIncludesQueued(t *testing.T) {
	store := NewSubagentStore(t.TempDir())
	run, err := store.PrepareFresh("sys", "")
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	defer run.Release()
	if err := store.MarkQueued(run); err != nil {
		t.Fatalf("MarkQueued: %v", err)
	}
	n, err := store.CleanupStaleRunning()
	if err != nil {
		t.Fatalf("CleanupStaleRunning: %v", err)
	}
	if n != 1 {
		t.Fatalf("清理数 = %d, want 1（queued 与 running 同判陈旧）", n)
	}
	if m, _ := store.loadMeta(run.Ref); m.Status != SubagentFailed {
		t.Fatalf("清理后 status = %q, want failed", m.Status)
	}
}

func TestTaskDescriptionMentionsSlotLimit(t *testing.T) {
	setSlotsForTest(t, 4)
	parentReg := tool.NewRegistry()
	task := NewTaskTool(&mockProvider{name: "p"}, nil, parentReg, 10, 0, 0.0, "", "sys", nil)
	if !strings.Contains(task.Description(), "slot limit (4)") {
		t.Fatalf("Description 应随配置注入并发上限（slot limit (4)）: %s", task.Description())
	}
}
