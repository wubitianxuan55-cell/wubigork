package app

import (
	"context"
	"testing"
	"time"

	ai "github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// 2026-11-16 图文切书/热重载整板块冻结（点击无响应）死锁回归锁：旧代码
// wenxueReloadIfInit/WenxueReload 持 wx.mu 跨 wenxueBuildController，而
// boot.Build 同步回调 OnFollowUpReady（回调写锁）——同 goroutine 重入
// sync.Mutex 永久挂死。修复后：构建不持 wx.mu、回调走独立 followUpMu。
// 本测试直打 reload 路径；旧代码在 watchdog 到点时判死（死锁 goroutine
// 泄漏但套件继续），新代码必须秒级完成。
func TestWenxueReloadIfInitCompletesUnderFollowUpCallback(t *testing.T) {
	// 图文脑目录隔离（seam；防测试写真实 %AppData% 图文脑）。覆盖缝直接
	// 顶替 wenxueBrainDir() 返回值（不拼 wenxue-brain 子目录）。
	prevBrain := wenxueBrainDirOverride
	wenxueBrainDirOverride = t.TempDir()
	brainDir := wenxueBrainDirOverride
	t.Cleanup(func() {
		wenxueBrainDirOverride = prevBrain
		// 图文脑 DB 句柄不关会让 TempDir 清理失败（unlinkat 占用）。
		knowledge.ResetForTest()
		db.CloseDatabase(brainDir)
	})

	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{Name: "gaea", Kind: "wubigrok", Model: "", ContextWindow: 1_000_000}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	t.Cleanup(func() { gaeaConfig.SetLoader(nil) })
	bridge.SetClient(ai.NewClient(config.Load()))
	t.Cleanup(func() { bridge.SetClient(nil) })

	a := &App{core: &core{cfg: config.Load()}}
	a.ctx = context.Background()

	// 首建（wx.mu 未持——本就无死锁；顺带验证 Build→OnFollowUpReady 回调链）。
	if err := a.WenxueEngineInit(); err != nil {
		t.Fatalf("图文引擎首建失败: %v", err)
	}
	t.Cleanup(func() {
		wx.mu.Lock()
		old := wx.ctrl
		wx.ctrl = nil
		wx.mu.Unlock()
		if old != nil {
			old.Close()
		}
	})

	// reload 路径带 watchdog：旧代码此处永久挂死。
	done := make(chan struct{})
	go func() {
		a.wenxueReloadIfInit()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("wenxueReloadIfInit 30s 未完成——重建自死锁回归（OnFollowUpReady 回锁与重建互斥）")
	}

	// 重建后 followUp 必须重新接线（新控制器回调已写 followUpMu 家族）。
	wx.followUpMu.Lock()
	wired := wx.followUp != nil
	wx.followUpMu.Unlock()
	if !wired {
		t.Fatal("重建后子代理追问执行器未重新接线")
	}

	// 再来一次热重载（WenxueReload 同型点）。
	if _, err := a.WenxueReload(); err != nil {
		t.Fatalf("图文热重载失败: %v", err)
	}
}
