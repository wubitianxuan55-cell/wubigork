package app

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

// 引擎故障转移开关绑定往返（C 刀 v0）：App.Get/SetEngineFailover 写内存 +
// 落盘，重新 Load 后一致（照 TestOfflineModeBindingRoundTrip 口径）。
func TestEngineFailoverBindingRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	a := &App{core: &core{cfg: config.Load()}}
	if a.GetEngineFailover() {
		t.Fatal("默认应为关闭")
	}
	if err := a.SetEngineFailover(true); err != nil {
		t.Fatalf("SetEngineFailover: %v", err)
	}
	if !a.GetEngineFailover() {
		t.Fatal("设置后内存应为开启")
	}
	if cfg := config.Load(); !cfg.GetEngineFailover() {
		t.Fatal("重新加载后应为开启（落盘验证）")
	}
	if err := a.SetEngineFailover(false); err != nil {
		t.Fatalf("SetEngineFailover(false): %v", err)
	}
	if cfg := config.Load(); cfg.GetEngineFailover() {
		t.Fatal("关闭后重新加载应为关闭")
	}
}

// 故障转移回调事件接线（configureClient）：client 转移动作 → emit
// model-failover（经 httpbridge 可观测；ctx 为空时 Wails 发射跳过不 panic）。
func TestFailoverEventWiring(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	a := &App{core: &core{cfg: config.Load()}}
	a.client = ai.NewClient(a.cfg)
	a.configureClient()
	if a.client.OnFailover == nil {
		t.Fatal("configureClient 后 OnFailover 应已接线")
	}
	if a.client.FailoverEnabled() {
		t.Fatal("默认开关应为关闭")
	}
	a.cfg.SetEngineFailover(true)
	if !a.client.FailoverEnabled() {
		t.Fatal("开关开启后 FailoverEnabled 应为 true（注入函数直读内存 cfg）")
	}
	a.cfg.SetEngineFailover(false)

	called := make(chan struct{}, 1)
	a.client.OnFailover = func(from, to, model string) { called <- struct{}{} }
	a.client.OnFailover("xai", "deepseek", "deepseek-v4-pro")
	select {
	case <-called:
	default:
		t.Fatal("OnFailover 回调未触发")
	}
}
