package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newWenxuePreflightTestApp 预检测试台：core + engineMgr（llama/xai 两引擎），
// 功能绑定按用例落。不建 wx.ctrl——预检面与控制器无关。
func newWenxuePreflightTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	c := &core{
		cfg:       &config.Config{Model: "grok-4.7"},
		engineMgr: modelengine.NewManager("", ""),
	}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "llama",
		Enabled:      true,
		DefaultModel: "qwen35",
		Models:       []modelengine.ModelInfo{{ID: "qwen35", Kind: "llm"}},
	}); err != nil {
		t.Fatalf("SaveEngine(llama): %v", err)
	}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:           "xai",
		Enabled:      true,
		DefaultModel: "grok-4.7",
		Models:       []modelengine.ModelInfo{{ID: "grok-4.7", Kind: "llm"}},
	}); err != nil {
		t.Fatalf("SaveEngine(xai): %v", err)
	}
	return &App{core: c}
}

// resetWenxueLeases 清租约表（包级状态，测试间隔离）。
func resetWenxueLeases(t *testing.T) {
	t.Helper()
	wenxueLeasesMu.Lock()
	defer wenxueLeasesMu.Unlock()
	wenxueLeases = map[string]time.Time{}
}

func TestWenxueResolveRoute_FallbackChain(t *testing.T) {
	a := newWenxuePreflightTestApp(t)

	t.Run("wenxue 绑定优先", func(t *testing.T) {
		a.cfg.SetFeatureModel("wenxue", "llama", "qwen35")
		a.cfg.SetFeatureModel("gaea", "xai", "grok-4.7")
		a.cfg.SetFeatureModelEnabled("wenxue", true)
		eng, mdl := a.wenxueResolveRoute()
		if eng != "llama" || mdl != "qwen35" {
			t.Fatalf("route = (%q,%q), want (llama,qwen35)", eng, mdl)
		}
	})

	t.Run("wenxue 停用回落 gaea 绑定", func(t *testing.T) {
		a.cfg.SetFeatureModel("wenxue", "llama", "qwen35")
		a.cfg.SetFeatureModel("gaea", "xai", "grok-4.7")
		a.cfg.SetFeatureModelEnabled("wenxue", false)
		eng, mdl := a.wenxueResolveRoute()
		if eng != "xai" || mdl != "grok-4.7" {
			t.Fatalf("route = (%q,%q), want (xai,grok-4.7)", eng, mdl)
		}
	})

	t.Run("双未绑回落全局活跃", func(t *testing.T) {
		a.cfg.SetFeatureModel("wenxue", "", "")
		a.cfg.SetFeatureModel("gaea", "", "")
		a.cfg.SetFeatureModelEnabled("wenxue", false)
		eng, _ := a.wenxueResolveRoute()
		if eng == "" {
			t.Fatal("动态回落应解析出活跃引擎，得空")
		}
	})

	t.Run("绑引擎未绑模型回落引擎默认档", func(t *testing.T) {
		a.cfg.SetFeatureModel("wenxue", "llama", "")
		a.cfg.SetFeatureModelEnabled("wenxue", true)
		eng, mdl := a.wenxueResolveRoute()
		if eng != "llama" || mdl != "qwen35" {
			t.Fatalf("route = (%q,%q), want (llama,qwen35)——与 ai.Client 动态默认档对齐", eng, mdl)
		}
	})
}

func TestWenxueResolveRoute_NormalizesLlamaPathReadOnly(t *testing.T) {
	a := newWenxuePreflightTestApp(t)
	stale := `C:\AI\models\gemma-4-26b-a4b-it-qat\gemma-4-26B-A4B-it-qat-UD-Q4_K_XL.gguf`
	a.cfg.SetFeatureModel("wenxue", "llama", stale)
	a.cfg.SetFeatureModelEnabled("wenxue", true)

	eng, mdl := a.wenxueResolveRoute()
	if eng != "llama" || mdl != "gemma4_26b_a4b" {
		t.Fatalf("route = (%q,%q), want (llama,gemma4_26b_a4b)", eng, mdl)
	}
	// 只读归一：不落档（聊天侧 normalizeChatBoundModel 会持久化，图文不代写）
	if _, got := a.cfg.GetFeatureModel("wenxue"); got != stale {
		t.Fatalf("绑定被改写 = %q, want 原样 %q", got, stale)
	}
}

func TestWenxuePreflight_EnsuresEveryTurnAndTouchesLease(t *testing.T) {
	a := newWenxuePreflightTestApp(t)
	a.cfg.SetFeatureModel("wenxue", "llama", "qwen35")
	a.cfg.SetFeatureModelEnabled("wenxue", true)
	resetWenxueLeases(t)

	original := wenxuePreflightFn
	t.Cleanup(func() { wenxuePreflightFn = original })
	calls := 0
	wenxuePreflightFn = func(_ *App, engineID, model string) (bool, error) {
		calls++
		if engineID != "llama" || model != "qwen35" {
			t.Errorf("ensure call #%d = (%q,%q), want (llama,qwen35)", calls, engineID, model)
		}
		return calls == 1, nil // 首轮冷启动，次轮已在运行
	}

	// 每轮语义：同一 App 连发两轮都要过闸（引擎可能在两轮之间被收割）
	for i := 1; i <= 2; i++ {
		if err := a.wenxuePreflight(); err != nil {
			t.Fatalf("第 %d 轮 preflight: %v", i, err)
		}
	}
	if calls != 2 {
		t.Fatalf("ensure 调用次数 = %d, want 2（每轮一发）", calls)
	}

	now := time.Now()
	if !wenxueLeaseAlive("llama", now) {
		t.Fatal("预检通过应记图文租约")
	}
	if wenxueLeaseAlive("llama", now.Add(wenxueIdleLease+time.Minute)) {
		t.Fatal("租约超时应失效")
	}
}

func TestWenxuePreflight_ErrorSurfacedWithContext(t *testing.T) {
	a := newWenxuePreflightTestApp(t)
	a.cfg.SetFeatureModel("wenxue", "llama", "qwen35")
	a.cfg.SetFeatureModelEnabled("wenxue", true)

	original := wenxuePreflightFn
	t.Cleanup(func() { wenxuePreflightFn = original })
	wenxuePreflightFn = func(_ *App, _, _ string) (bool, error) {
		return true, errors.New("qwen35 300s 未就绪")
	}

	err := a.wenxuePreflight()
	if err == nil {
		t.Fatal("绑定引擎未就绪应报错")
	}
	if !strings.Contains(err.Error(), "图文绑定的模型 llama/qwen35 未能就绪") {
		t.Fatalf("error = %q, want 图文绑定上下文", err)
	}
}

func TestWenxuePreflight_GuardSkipsNonManaged(t *testing.T) {
	a := newWenxuePreflightTestApp(t)

	t.Run("云端引擎直通不拉起", func(t *testing.T) {
		original := wenxuePreflightFn
		t.Cleanup(func() { wenxuePreflightFn = original })
		wenxuePreflightFn = func(_ *App, engineID, _ string) (bool, error) {
			// 生产 wenxueEnsureEngine 对云端直通（默认分支），此处断言绝不
			// 触碰托管本地引擎
			switch engineID {
			case "llama", "strata", "whirl":
				t.Errorf("托管引擎 %s 不应在云端绑定路径被拉起", engineID)
			}
			return false, nil
		}
		a.cfg.SetFeatureModel("wenxue", "xai", "grok-4.7")
		a.cfg.SetFeatureModelEnabled("wenxue", true)
		if err := a.wenxuePreflight(); err != nil {
			t.Fatalf("preflight: %v", err)
		}
	})

	t.Run("离线模式云端绑定守卫拦截", func(t *testing.T) {
		original := wenxuePreflightFn
		t.Cleanup(func() { wenxuePreflightFn = original })
		wenxuePreflightFn = func(_ *App, engineID, _ string) (bool, error) {
			t.Errorf("离线模式云端引擎 %s 不应进预检拉起", engineID)
			return false, nil
		}
		a.cfg.SetOfflineMode(true)
		t.Cleanup(func() { a.cfg.SetOfflineMode(false) })
		if err := a.wenxuePreflight(); err != nil {
			t.Fatalf("preflight: %v", err)
		}
	})
}

func TestWenxueEnsureEngine_DispatchErrors(t *testing.T) {
	a := newWenxuePreflightTestApp(t)

	if _, err := a.wenxueEnsureEngine("llama", "not-a-profile"); err == nil ||
		!strings.Contains(err.Error(), "不在 llama 档位目录") {
		t.Fatalf("err = %v, want 档位目录错误", err)
	}
	// 云端引擎：直通不拉起
	if cold, err := a.wenxueEnsureEngine("xai", "grok-4.7"); err != nil || cold {
		t.Fatalf("云端 = (cold=%v, err=%v), want (false,nil)", cold, err)
	}

	originalProbe := strataPreflightProbe
	t.Cleanup(func() { strataPreflightProbe = originalProbe })
	strataPreflightProbe = func() ([]string, bool) { return nil, false }
	cold, err := a.wenxueEnsureEngine("strata", "qwen3.8-flash-next-uncensored")
	if err == nil || cold || !strings.Contains(err.Error(), "已禁用请求内热启动") {
		t.Fatalf("Strata 未运行 = (cold=%v, err=%v), want immediate explicit-start error", cold, err)
	}
}

func TestWenxueLease_ProtectsEngineFromReaper(t *testing.T) {
	a := newIdleReaperTestApp(t, 5) // 全局阈值 5 分钟（生产默认）
	now := time.Now()
	idle := 5 * time.Minute
	target := func(leaseAt time.Time) []managedIdleTarget {
		return []managedIdleTarget{{
			id: "llama", registered: registeredProc().registered,
			probe: func() bool { return true },
			stop:  func() {},
			lease: func(t time.Time) bool { return wenxueLeaseAlive("llama", t) },
		}}
	}

	t.Run("空闲超阈但租约未满不收", func(t *testing.T) {
		resetWenxueLeases(t)
		a.engineMgr.TouchEngineAt("llama", now.Add(-11*time.Minute)) // 空闲 11min > 5min
		wenxueTouchEngineAt("llama", now.Add(-4*time.Minute))        // 图文租约 4min < 10min
		stopped := a.reapManagedEngineIdleOnce(now, idle, target(now))
		if len(stopped) != 0 {
			t.Errorf("stopped = %v, want 空（租约保护期内）", stopped)
		}
	})

	t.Run("租约也过期即收割", func(t *testing.T) {
		resetWenxueLeases(t)
		a.engineMgr.TouchEngineAt("llama", now.Add(-11*time.Minute))
		wenxueTouchEngineAt("llama", now.Add(-11*time.Minute))
		stopped := a.reapManagedEngineIdleOnce(now, idle, target(now))
		if len(stopped) != 1 || stopped[0] != "llama" {
			t.Errorf("stopped = %v, want [llama]", stopped)
		}
	})

	t.Run("无租约按全局阈值照旧", func(t *testing.T) {
		resetWenxueLeases(t)
		a.engineMgr.TouchEngineAt("llama", now.Add(-11*time.Minute))
		stopped := a.reapManagedEngineIdleOnce(now, idle, target(now))
		if len(stopped) != 1 {
			t.Errorf("stopped = %v, want [llama]", stopped)
		}
	})

	t.Run("生产目标表 llama/strata/whirl 挂租约 comfyui 不挂", func(t *testing.T) {
		targets := a.managedIdleTargets()
		byID := map[string]managedIdleTarget{}
		for _, tg := range targets {
			byID[tg.id] = tg
		}
		for _, id := range []string{"llama", "strata", "whirl"} {
			if byID[id].lease == nil {
				t.Errorf("target %s 缺图文租约谓词", id)
			}
		}
		if byID["comfyui"].lease != nil {
			t.Error("comfyui 不应有图文租约")
		}
	})
}
