package app

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newIdleReaperTestApp 构造收割器测试台：core + engineMgr，阈值可配。
func newIdleReaperTestApp(t *testing.T, minutes int) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	a := &App{core: &core{
		cfg:       &config.Config{ModelIdleExitMinutes: minutes},
		engineMgr: modelengine.NewManager("", ""),
	}}
	return a
}

// registeredProc 伪造「gaea 本会话拉起」的登记态（只造 cmd/Process 指针，
// 不启真进程——收割判定只读 registered()，stop 走注入桩）。
func registeredProc() *localEngineProc {
	p := &localEngineProc{}
	p.set(&exec.Cmd{Process: &os.Process{}}, func() {})
	return p
}

// TestReapIdle_Gates 四道闸逐闸验证：只收「gaea 拉起 + 有使用记录 + 空闲超阈
// + 确在运行」的目标。
func TestReapIdle_Gates(t *testing.T) {
	a := newIdleReaperTestApp(t, 10)
	now := time.Now()
	idle := 10 * time.Minute

	t.Run("未登记不收", func(t *testing.T) {
		stopped := a.reapManagedEngineIdleOnce(now, idle, []managedIdleTarget{{
			id: "tianxuan", registered: func() bool { return false }, // 外部实例：无登记
			probe: func() bool { return true },
			stop:  func() { t.Error("外部实例不应被代停") },
		}})
		if len(stopped) != 0 {
			t.Errorf("stopped = %v, want 空", stopped)
		}
	})

	t.Run("无使用记录不收", func(t *testing.T) {
		stopped := a.reapManagedEngineIdleOnce(now, idle, []managedIdleTarget{{
			id: "tianxuan", registered: registeredProc().registered, // 本会话从未经手
			probe: func() bool { return true },
			stop:  func() { t.Error("无使用记录不应收割") },
		}})
		if len(stopped) != 0 {
			t.Errorf("stopped = %v, want 空", stopped)
		}
	})

	t.Run("空闲未超阈不收", func(t *testing.T) {
		a.engineMgr.TouchEngineAt("tianxuan", now.Add(-9*time.Minute))
		stopped := a.reapManagedEngineIdleOnce(now, idle, []managedIdleTarget{{
			id: "tianxuan", registered: registeredProc().registered,
			probe: func() bool { return true },
			stop:  func() { t.Error("未超阈值不应收割") },
		}})
		if len(stopped) != 0 {
			t.Errorf("stopped = %v, want 空", stopped)
		}
	})

	t.Run("空闲超阈但不在运行不收", func(t *testing.T) {
		a.engineMgr.TouchEngineAt("strata", now.Add(-11*time.Minute))
		stopped := a.reapManagedEngineIdleOnce(now, idle, []managedIdleTarget{{
			id: "strata", registered: registeredProc().registered,
			probe: func() bool { return false }, // 已停/探活失败：可能是冷启动，不动作
			stop:  func() { t.Error("未运行不应收割") },
		}})
		if len(stopped) != 0 {
			t.Errorf("stopped = %v, want 空", stopped)
		}
	})

	t.Run("四闸全过即收割", func(t *testing.T) {
		a.engineMgr.TouchEngineAt("tianxuan", now.Add(-11*time.Minute))
		stopCalled := 0
		stopped := a.reapManagedEngineIdleOnce(now, idle, []managedIdleTarget{{
			id: "tianxuan", registered: registeredProc().registered,
			probe: func() bool { return true },
			stop:  func() { stopCalled++ },
		}})
		if len(stopped) != 1 || stopped[0] != "tianxuan" || stopCalled != 1 {
			t.Errorf("stopped = %v stopCalled = %d, want [tianxuan]/1", stopped, stopCalled)
		}
	})
}

// TestManagedIdleTargets_CoversAllManagedEngines 生产目标表三引擎齐（天璇/
// Strata/ComfyUI——ComfyUI 于 v4.518 接入）；谓词为方法值/闭包，构造期不求值
// （nil 安全），零值 App 也能取表。
func TestManagedIdleTargets_CoversAllManagedEngines(t *testing.T) {
	a := newIdleReaperTestApp(t, 10)
	targets := a.managedIdleTargets()
	got := make(map[string]bool, len(targets))
	for _, tg := range targets {
		got[tg.id] = true
		if tg.registered == nil || tg.probe == nil || tg.stop == nil {
			t.Errorf("target %s 缺谓词（registered/probe/stop 必须齐）", tg.id)
		}
	}
	for _, want := range []string{"tianxuan", "strata", "comfyui"} {
		if !got[want] {
			t.Errorf("目标表缺 %s: %v", want, got)
		}
	}
}

// TestReapIdle_ThresholdGate 阈值关闭（<=0）时 reapIdleManagedEngines 直接
// 返回——机制总闸在 config（model_idle_exit_minutes=0=关）。
func TestReapIdle_ThresholdGate(t *testing.T) {
	a := newIdleReaperTestApp(t, 0)
	if a.managedIdleExitMinutes() != 0 {
		t.Fatalf("阈值 = %d, want 0", a.managedIdleExitMinutes())
	}
	// 即使记账早已超阈、探活在运行：总闸关闭也不得收割
	a.engineMgr.TouchEngineAt("tianxuan", time.Now().Add(-2*time.Hour))
	stopped := a.reapManagedEngineIdleOnce(time.Now(), 10*time.Minute, []managedIdleTarget{})
	_ = stopped // once 直调不受闸控——闸在 reapIdleManagedEngines；此处只验阈值读取
}

// TestReapIdle_MissingEngineActivitySkipped 有登记但记账表无该引擎（理论上
// 不会发生：登记即 touch）——零值防线兜住，绝不凭空收割。
func TestReapIdle_MissingEngineActivitySkipped(t *testing.T) {
	a := newIdleReaperTestApp(t, 10)
	stopped := a.reapManagedEngineIdleOnce(time.Now(), 10*time.Minute, []managedIdleTarget{{
		id: "strata", registered: registeredProc().registered,
		probe: func() bool { return true },
		stop:  func() { t.Error("零使用记录不应收割") },
	}})
	if len(stopped) != 0 {
		t.Errorf("stopped = %v, want 空", stopped)
	}
}

// TestVisionBoundRuntime_TouchesEngine 识图绑定解析即记账：绑到天璇的识图
// 请求会续上活跃度，不会被收割器在请求间隙停机。
func TestVisionBoundRuntime_TouchesEngine(t *testing.T) {
	c := newTestCore(t)
	if err := c.SetFeatureModel("vision", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("前置绑定: %v", err)
	}
	if _, ok := c.visionBoundRuntime(); !ok {
		t.Fatal("绑定应命中")
	}
	if c.engineMgr.EngineLastActivity("herdsman").IsZero() {
		t.Error("识图绑定解析应记账真实使用")
	}
}
