package app

// wenxue_chat_test.go — 图文会话态绑定最小测试面（2026-11-17 审查批4；此前
// wenxue_chat.go 747 行零测试）。复用死锁回归 harness（真实 Build，hermetic
// 脑目录+临时书根），覆盖：未选书拒新会话/建书后新会话/会话清单/历史空态/
// Rewind 桩诚实报错/命令清单非空/元数据形态。

import (
	"context"
	"strings"
	"testing"
	"time"

	ai "github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestWenxueChatSessionBindingSmoke(t *testing.T) {
	// 图文脑目录隔离（同 wenxue_reload_deadlock_test.go：句柄不关=TempDir 清理失败）。
	prevBrain := wenxueBrainDirOverride
	wenxueBrainDirOverride = t.TempDir()
	brainDir := wenxueBrainDirOverride
	t.Cleanup(func() {
		wenxueBrainDirOverride = prevBrain
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
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	t.Cleanup(func() {
		wx.mu.Lock()
		old := wx.ctrl
		wx.ctrl = nil
		wx.mu.Unlock()
		if old != nil {
			old.Close()
		}
	})

	if err := a.WenxueEngineInit(); err != nil {
		t.Fatalf("图文引擎首建失败: %v", err)
	}

	// 未选书：NewSession 诚实拒绝（wenxue_workspace.go 根守卫）。
	if err := a.WenxueNewSession(); err == nil || !strings.Contains(err.Error(), "小说") {
		t.Fatalf("未选书新建会话应被拒，got %v", err)
	}

	// 建书（自动切换工作空间）→ 新会话放行。
	if _, err := a.WenxueCreateNovel("围炉夜话", ""); err != nil {
		t.Fatalf("WenxueCreateNovel: %v", err)
	}
	if err := a.WenxueNewSession(); err != nil {
		t.Fatalf("选书后新建会话失败: %v", err)
	}

	// 会话清单含新建会话；历史只含引擎开场（无用户回合）。
	sessions := a.WenxueListSessions()
	if len(sessions) != 1 {
		t.Fatalf("会话清单 = %d 条, want 1", len(sessions))
	}
	for _, m := range a.WenxueHistory() {
		if m.Role == "user" {
			t.Fatalf("新会话不应有用户回合: %+v", m)
		}
	}

	// 运行态诚实 false；元数据工作区锚小说目录。
	if a.WenxueRunning() {
		t.Fatal("初始应非运行态")
	}
	if got := a.WenxuePendingDeniedActions(); len(got) != 0 {
		t.Fatalf("新图文引擎待追认账本应为空，got %+v", got)
	}
	a.WenxueApproveWithReason("missing", "deny", "测试理由")
	a.WenxueApproveDenied("missing")
	meta := a.WenxueMeta()
	if !strings.HasSuffix(strings.ReplaceAll(meta.Cwd, `\`, "/"), "/围炉夜话") {
		t.Fatalf("Meta.Cwd 应锚当前小说目录，got %q", meta.Cwd)
	}

	// Rewind scope 桩：诚实报错（错误文案指引替代路径）。
	done := make(chan error, 1)
	go func() { done <- a.WenxueRewind(1, "code") }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "暂未支持") {
			t.Fatalf("Rewind code 桩应给出诚实错误，got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("WenxueRewind 10s 未返回")
	}

	// 命令清单非空（任务模板/快捷命令在位）。
	if len(a.WenxueCommands()) == 0 {
		t.Fatal("WenxueCommands 不应为空")
	}
}
