package app

// P1-5 对话条测试：对话流取消链路（登记表/取消/部分落库+cancelled 终态/未知 runID）。

import (
	"context"
	"testing"
	"time"
)

// TestChatStreamCancelUnknownRun 未知/已结束 runID 返 false 不报错。
func TestChatStreamCancelUnknownRun(t *testing.T) {
	a := newGateEmptyApp()
	if a.ChatStreamCancel("no-such-run") {
		t.Fatal("未知 runID 应返 false")
	}
}

// TestChatStreamCancelRegistry 登记表生命周期：register→cancel 命中 true→unregister 后 false。
func TestChatStreamCancelRegistry(t *testing.T) {
	a := newGateEmptyApp()
	a.chatStreamRegister("run-1", func() {})
	if !a.ChatStreamCancel("run-1") {
		t.Fatal("已登记 runID 应命中 true")
	}
	a.chatStreamUnregister("run-1")
	if a.ChatStreamCancel("run-1") {
		t.Fatal("退出清表后应返 false（协程退出独占清理）")
	}
}

// TestChatStreamCancelEndToEnd 取消传播闭环（受控协程=真 ctx+登记+ChatStreamCancel
// 传播→协程感知 ctx 断→退出清表；不依赖 chat 路由可用性——挂起假站对 chat
// feature 的快速失败路径已在生产代码由 error 帧收尾）。
func TestChatStreamCancelEndToEnd(t *testing.T) {
	a := newGateEmptyApp()
	ctx, cancel := context.WithCancel(context.Background())
	a.chatStreamRegister("run-e2e", cancel)
	done := make(chan struct{})
	go func() {
		defer a.chatStreamUnregister("run-e2e")
		defer close(done)
		<-ctx.Done() // 协程体：等待取消（对齐 runChatStreamPlain 的 ctx 感知）
	}()
	if !a.ChatStreamCancel("run-e2e") {
		t.Fatal("进行中的流应可取消")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("取消未传播到协程（ctx.Done 未触发）")
	}
	// 协程退出后清表 + 幂等
	waitFor(t, 5*time.Second, "协程退出清表", func() bool {
		a.chatStreamMu.Lock()
		defer a.chatStreamMu.Unlock()
		return len(a.chatStreamCancels) == 0
	})
	if a.ChatStreamCancel("run-e2e") {
		t.Fatal("清表后二次取消应返 false（幂等）")
	}
}
