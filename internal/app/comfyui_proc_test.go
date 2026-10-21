package app

import (
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// refSnapshot 在锁内读取当前登记的引用对，供测试断言。
func (a *mediaState) refSnapshot() (context.CancelFunc, *exec.Cmd) {
	a.comfyProcMu.Lock()
	defer a.comfyProcMu.Unlock()
	return a.comfyUICancel, a.comfyUICmd
}

// TestComfyUIProcRefClearIfCurrent 验证 ABA 防误清：引用被新一轮覆盖后，
// 上一轮回收 goroutine 的晚到清理不得生效；匹配本轮时才清。
func TestComfyUIProcRefClearIfCurrent(t *testing.T) {
	a := &mediaState{}
	_, oldCancel := context.WithCancel(context.Background())
	_, newCancel := context.WithCancel(context.Background())
	defer oldCancel()
	defer newCancel()
	oldCmd, newCmd := &exec.Cmd{}, &exec.Cmd{}

	a.comfyProcRefSet(oldCancel, oldCmd)
	// 登记被新一轮覆盖后，上一轮回收 goroutine 的晚到清理不得生效。
	a.comfyProcRefSet(newCancel, newCmd)
	a.comfyProcRefClearIfCurrent(oldCancel, oldCmd)
	if c, cmd := a.refSnapshot(); c == nil || cmd != newCmd {
		t.Fatalf("上一轮晚到清理误删了新一轮登记: cancel=%v cmd=%v", c, cmd)
	}
	// 匹配本轮：清空生效。
	a.comfyProcRefClearIfCurrent(newCancel, newCmd)
	if c, cmd := a.refSnapshot(); c != nil || cmd != nil {
		t.Fatalf("本轮清理未生效: cancel=%v cmd=%v", c, cmd)
	}
	// cmd 为 nil（仅登记了 cancel 的窗口）时是 no-op，不误伤。
	a.comfyProcRefSet(newCancel, nil)
	a.comfyProcRefClearIfCurrent(newCancel, nil)
	if c, _ := a.refSnapshot(); c == nil {
		t.Fatalf("nil cmd 的晚到清理误删了登记")
	}
	a.comfyProcRefClear()
	newCancel()
}

// TestComfyUIProcRefStopKillsAndCancels 验证 refStop 真杀子进程 + 触发 cancel + 清引用。
func TestComfyUIProcRefStopKillsAndCancels(t *testing.T) {
	a := &mediaState{}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.Command("cmd", "/c", "ping -n 30 127.0.0.1 > nul")
	if err := cmd.Start(); err != nil {
		t.Skipf("无法启动子进程: %v", err)
	}
	a.comfyProcRefSet(cancel, cmd)

	a.comfyProcRefStop()

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("refStop 未触发 cancel")
	}
	if c, cmdRef := a.refSnapshot(); c != nil || cmdRef != nil {
		t.Fatalf("refStop 后引用未清空: cancel=%v cmd=%v", c, cmdRef)
	}
	// 子进程确实被杀：Wait 应立即返回而非等满 30s。
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("子进程未被杀停")
	}
}

// TestComfyUIProcRefRegistered 空闲自动退出的「gaea 本会话拉起」闸（与天璇/
// Strata 的 localEngineProc.registered 同语义）：登记即 true，退出清理/停止
// 后 false——外部手动启动的 ComfyUI（gaea 无引用）不被代管。
func TestComfyUIProcRefRegistered(t *testing.T) {
	a := &mediaState{}
	if a.comfyProcRefRegistered() {
		t.Fatal("零值状态不应视为 gaea 拉起")
	}
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.comfyProcRefSet(cancel, &exec.Cmd{})
	if !a.comfyProcRefRegistered() {
		t.Fatal("登记后应视为 gaea 拉起")
	}
	a.comfyProcRefStop()
	if a.comfyProcRefRegistered() {
		t.Fatal("停止清引用后不应再视为 gaea 拉起")
	}
	// 进程自然退出路径（回收 goroutine 的 ClearIfCurrent）同样解除登记。
	a.comfyProcRefSet(cancel, &exec.Cmd{})
	a.comfyProcRefClear()
	if a.comfyProcRefRegistered() {
		t.Fatal("引用清理后不应再视为 gaea 拉起")
	}
}

// TestComfyUIProcRefConcurrent 多 goroutine 压榨 refSet/ClearIfCurrent/Clear/Stop，
// 供 -race 检测数据竞争（本机无 gcc 时仅验证无死锁）。
func TestComfyUIProcRefConcurrent(t *testing.T) {
	a := &mediaState{}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				ctx, cancel := context.WithCancel(context.Background())
				cmd := &exec.Cmd{}
				a.comfyProcRefSet(cancel, cmd)
				switch j % 3 {
				case 0:
					a.comfyProcRefClearIfCurrent(cancel, cmd)
				case 1:
					a.comfyProcRefStop()
				default:
					a.comfyProcRefClear()
				}
				cancel()
				_ = ctx
			}
		}()
	}
	wg.Wait()
	if c, cmd := a.refSnapshot(); c != nil || cmd != nil {
		t.Fatalf("并发收尾后引用残留: cancel=%v cmd=%v", c, cmd)
	}
}
