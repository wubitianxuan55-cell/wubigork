package tasks

// S1.4 任务按空间分账调度测试（docs/gaea-space-assembly-design.md §3）：
// PerSpace 双空间额度互不挤占、Priority 优先级出队、缺省回退（空 map = 旧
// 全局单队列 FIFO）、HasActiveInSpace 按空间去重。

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPerSpaceSchedulingNoCrossBlocking 双空间额度互不挤占：work 额度（1）被
// 长任务占满时，play 任务仍能立即出队执行；work 的第二个任务在本空间额度
// 腾出前保持 queued，w1 完成释放额度后经 re-signal 立即被调度（防饥饿）。
func TestPerSpaceSchedulingNoCrossBlocking(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{
		BackoffBase: 10 * time.Millisecond,
		PerSpace:    map[string]int{defaultTaskSpace: 1, "play": 1},
	})
	workStarted := make(chan struct{})
	releaseWork := make(chan struct{})
	playStarted := make(chan struct{})
	releasePlay := make(chan struct{})
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		switch tk.Label {
		case "w1":
			close(workStarted)
			<-releaseWork
		case "p1":
			close(playStarted)
			<-releasePlay
		}
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	w1, err := m.SubmitSpace(KindFileIndex, "w1", nil, defaultTaskSpace)
	if err != nil {
		t.Fatalf("submit w1: %v", err)
	}
	select {
	case <-workStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("w1 未运行")
	}

	// work 额度已满：play 任务必须仍能出队（不被 work 阻塞）
	p1, err := m.SubmitSpace(KindFileIndex, "p1", nil, "play")
	if err != nil {
		t.Fatalf("submit p1: %v", err)
	}
	select {
	case <-playStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("work 额度占满时 play 任务未能出队执行（跨空间挤占）")
	}

	// work 第二个任务：本空间额度=1 已满，应保持 queued（额度不超发）
	w2, err := m.SubmitSpace(KindFileIndex, "w2", nil, defaultTaskSpace)
	if err != nil {
		t.Fatalf("submit w2: %v", err)
	}
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cur, _ := m.Get(w2.ID); cur.Status != string(StatusQueued) {
			t.Fatalf("work 额度已满，w2 不应出队，实际 %s", cur.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// w1 完成 → 释放 work 额度 → re-signal → w2 立即被调度执行
	close(releaseWork)
	waitTerminal(t, m, w1.ID, 3*time.Second)
	waitTerminal(t, m, w2.ID, 3*time.Second)
	close(releasePlay)
	waitTerminal(t, m, p1.ID, 3*time.Second)
}

// TestPriorityPicksHigherFirst 优先级排序：高优先级 kind 先出队（创建更晚也
// 超前）；同优先级保持 created_at 升序（FIFO）。
func TestPriorityPicksHigherFirst(t *testing.T) {
	db := openTestDB(t)
	kindNormal, kindUrgent := Kind("normal_kind"), Kind("urgent_kind")
	m := New(db, nil, Options{
		MaxConcurrent: 1,
		BackoffBase:   10 * time.Millisecond,
		Priority:      map[string]int{string(kindUrgent): 10},
	})
	block := make(chan struct{})
	var mu sync.Mutex
	var order []string
	record := func(tk *Task) {
		mu.Lock()
		order = append(order, tk.Kind)
		mu.Unlock()
	}
	m.Register(kindNormal, func(ctx context.Context, tk *Task, p *Progress) error {
		record(tk)
		<-block
		return nil
	})
	m.Register(kindUrgent, func(ctx context.Context, tk *Task, p *Progress) error {
		record(tk)
		<-block
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	// 占住唯一 worker
	first, err := m.Submit(kindNormal, "先到", nil)
	if err != nil {
		t.Fatalf("submit first: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(order)
		mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("首个任务未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 排队：normal 先提交（创建更早、优先级 0），urgent 后提交（优先级 10）
	low, err := m.Submit(kindNormal, "低优", nil)
	if err != nil {
		t.Fatalf("submit normal: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // 错开 created_at，保证同优先级 FIFO 断言稳定
	high, err := m.Submit(kindUrgent, "高优", nil)
	if err != nil {
		t.Fatalf("submit urgent: %v", err)
	}

	close(block) // 放行全部：出队顺序应为 urgent（高优）→ normal（低优）
	waitTerminal(t, m, first.ID, 3*time.Second)
	waitTerminal(t, m, high.ID, 3*time.Second)
	waitTerminal(t, m, low.ID, 3*time.Second)

	mu.Lock()
	defer mu.Unlock()
	want := []string{string(kindNormal), string(kindUrgent), string(kindNormal)}
	if len(order) != len(want) {
		t.Fatalf("执行顺序 %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("执行顺序 %v, want %v（高优先级应先出队）", order, want)
		}
	}
}

// TestDefaultOptionsLegacyFIFO 缺省回退（铁律）：PerSpace/Priority 为空 =
// 旧全局单队列——跨空间任务也严格按 created_at 串行 FIFO，play 任务不因
// work 占用而被豁免并发（与改造前行为逐字节等价）。
func TestDefaultOptionsLegacyFIFO(t *testing.T) {
	db := openTestDB(t)
	m := New(db, nil, Options{BackoffBase: 10 * time.Millisecond})
	block := make(chan struct{})
	var mu sync.Mutex
	var order []string
	m.Register(KindFileIndex, func(ctx context.Context, tk *Task, p *Progress) error {
		mu.Lock()
		order = append(order, tk.Label)
		mu.Unlock()
		<-block
		return nil
	})
	if _, err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer m.Close()

	t1, err := m.Submit(KindFileIndex, "w-first", nil) // 缺省 work
	if err != nil {
		t.Fatalf("submit t1: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	t2, err := m.SubmitSpace(KindFileIndex, "w-second", nil, defaultTaskSpace)
	if err != nil {
		t.Fatalf("submit t2: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	t3, err := m.SubmitSpace(KindFileIndex, "p-third", nil, "play")
	if err != nil {
		t.Fatalf("submit t3: %v", err)
	}

	// 等首个任务运行（阻塞中）
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(order)
		mu.Unlock()
		if n == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("首个任务未运行")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 单队列：占住 worker 期间，后续任务（含 play）不得并发执行
	deadline = time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(order)
		mu.Unlock()
		if n != 1 {
			t.Fatalf("缺省应为全局单队列串行，实际已并发执行 %d 个: %v", n, order)
		}
		time.Sleep(20 * time.Millisecond)
	}

	close(block)
	waitTerminal(t, m, t1.ID, 3*time.Second)
	waitTerminal(t, m, t2.ID, 3*time.Second)
	waitTerminal(t, m, t3.ID, 3*time.Second)

	mu.Lock()
	defer mu.Unlock()
	want := []string{"w-first", "w-second", "p-third"}
	if len(order) != len(want) {
		t.Fatalf("执行顺序 %v, want %v（缺省应 FIFO）", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("执行顺序 %v, want %v", order, want)
		}
	}
}

// TestHasActiveInSpaceDedup 按空间去重：同 kind 跨空间互不挡；空 space =
// 跨空间全量（等价 HasActive）；全局 HasActive 兼容语义不变。
func TestHasActiveInSpaceDedup(t *testing.T) {
	gdb := openTestDB(t)
	m := New(gdb, nil, Options{})
	// 不 Start：任务停在 queued（queued 同样计入 active）
	if _, err := m.SubmitSpace(KindFileIndex, "work 任务", nil, defaultTaskSpace); err != nil {
		t.Fatalf("submit work: %v", err)
	}
	if !m.HasActiveInSpace(KindFileIndex, defaultTaskSpace) {
		t.Fatal("work 空间的 file_index 应为 active")
	}
	if m.HasActiveInSpace(KindFileIndex, "play") {
		t.Fatal("play 空间无任务，不应为 active（同 kind 跨空间不互挡）")
	}
	if _, err := m.SubmitSpace(KindFileIndex, "play 任务", nil, "play"); err != nil {
		t.Fatalf("submit play: %v", err)
	}
	if !m.HasActiveInSpace(KindFileIndex, "play") {
		t.Fatal("play 空间的 file_index 应为 active")
	}
	if m.HasActiveInSpace(KindPriceFetch, defaultTaskSpace) {
		t.Fatal("不同 kind 不应为 active")
	}
	// 兼容：空 space = 跨空间全量；全局 HasActive 语义不变
	if !m.HasActiveInSpace(KindFileIndex, "") {
		t.Fatal("空 space 应跨空间命中")
	}
	if !m.HasActive(KindFileIndex) {
		t.Fatal("HasActive 兼容语义应保持跨空间全量")
	}
}
