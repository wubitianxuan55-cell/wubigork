package weixin

import (
	"sync/atomic"
	"testing"
	"time"
)

// newTestServer 构造不联网的测试 Server：notify 通知全部注入替换为空实现。
// getUpdates 由各测试单独注入（避免真实 HTTP）。
func newTestServer(t *testing.T, botToken string) *Server {
	t.Helper()
	srv := New(Config{BotToken: botToken, AssistantID: "t"}, nil)
	srv.notifyStartFn = func() {}
	srv.notifyStopFn = func() {}
	return srv
}

// TestStop_IdempotentNoPanic 二次 Stop（以及从未 Start 直接 Stop）都不应 panic、无副作用。
func TestStop_IdempotentNoPanic(t *testing.T) {
	srv := newTestServer(t, "tok")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	srv.Stop()
	srv.Stop() // 二次 Stop：close of closed channel 的旧缺陷在此修复
	if srv.IsRunning() {
		t.Error("Stop 后 IsRunning 应为 false")
	}

	// 从未 Start 直接 Stop 也应安全
	srv2 := newTestServer(t, "tok")
	srv2.Stop()
	srv2.Stop()
}

// TestStartAfterStop_RestartsPolling Stop→Start 后轮询真正恢复（不是空转）：
// 用注入 getUpdatesFn 的调用计数（atomic）验证重启后轮询继续增长。
func TestStartAfterStop_RestartsPolling(t *testing.T) {
	srv := newTestServer(t, "tok")
	var polls atomic.Int64
	srv.getUpdatesFn = func(req *pollReq, timeout time.Duration) (*pollResp, error) {
		polls.Add(1)
		time.Sleep(2 * time.Millisecond) // 模拟长轮询节奏，避免空转计数
		return &pollResp{}, nil
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if first := polls.Load(); first == 0 {
		t.Fatal("Start 后应发生轮询")
	}
	if srv.SessionExpired() {
		t.Error("启动初期 sessionExpired 应为 false")
	}

	srv.Stop()
	time.Sleep(15 * time.Millisecond) // 等旧 pollLoop 退出
	before := polls.Load()

	if err := srv.Start(); err != nil { // 重启（stopCh 已关闭，应重建）
		t.Fatalf("重启 Start: %v", err)
	}
	if srv.SessionExpired() {
		t.Error("重启后 sessionExpired 应被重置为 false")
	}
	time.Sleep(60 * time.Millisecond)
	if after := polls.Load(); after <= before {
		t.Fatalf("Stop→Start 后轮询未恢复: before=%d after=%d", before, after)
	}
	srv.Stop()
}

// TestSessionExpired_TriggersCallback getUpdates 返回 errcode=-14 sessExp 时：
// 回调被触发一次、sessionExpired=true、轮询退出（不再继续调用 getUpdates）。
func TestSessionExpired_TriggersCallback(t *testing.T) {
	srv := newTestServer(t, "tok")
	expiredCh := make(chan struct{})
	srv.OnSessionExpired = func() { close(expiredCh) }
	var calls atomic.Int64
	srv.getUpdatesFn = func(req *pollReq, timeout time.Duration) (*pollResp, error) {
		calls.Add(1)
		return &pollResp{ErrCode: sessExp}, nil
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-expiredCh:
	case <-time.After(2 * time.Second):
		t.Fatal("会话过期回调未被触发")
	}
	if !srv.SessionExpired() {
		t.Error("sessionExpired 应为 true")
	}

	// 回调后轮询应停止（不再 5 分钟空转）：短暂等待后 getUpdates 调用数不得再增长
	time.Sleep(40 * time.Millisecond)
	if n := calls.Load(); n != 1 {
		t.Errorf("会话过期后轮询应停止, getUpdates 调用 %d 次（期望 1）", n)
	}
	srv.Stop()
}
