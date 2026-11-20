package computer

// W5 验收（施工单）：deny 表硬拒（零 syscall）/双闸门（旧索引与坐标拒绝且
// 注入桩零调用）/越界契约文案/坐标闸三态/DPI 禁用/键名解析/INPUT 步长。
import (
	"errors"
	"strings"
	"testing"
	"unsafe"
)

// fakeElement 造一个非nil 元素指针（指向真实分配，COM 侧只作句柄传给桩）。
func fakeElement() unsafe.Pointer { return unsafe.Pointer(new([64]byte)) }

func fakeWindow() Window {
	return Window{ID: 42, App: `process:C:\windows\notepad.exe`, Title: "记事本"}
}

func TestGuardChordForbidden(t *testing.T) {
	for _, chord := range []string{"Win+r", "Meta_L+x", "cmd", "super", "OS+l", "ctrl+win"} {
		if err := GuardChord(chord); !errors.Is(err, ErrForbiddenKey) {
			t.Fatalf("%q 必须硬拒: %v", chord, err)
		}
	}
	if err := GuardChord("Control_L+a"); err != nil {
		t.Fatalf("合法组合不得拒: %v", err)
	}
	if err := GuardChord("Return"); err != nil {
		t.Fatalf("Return 不得拒: %v", err)
	}
}

func TestKeysymChordParse(t *testing.T) {
	mods, key, err := keysymChord("Return")
	if err != nil || key != 0x0D || len(mods) != 0 {
		t.Fatalf("Return: mods=%v key=%#x err=%v", mods, key, err)
	}
	mods, key, err = keysymChord("Control_L+a")
	if err != nil || key != 0x41 || len(mods) != 1 || mods[0] != 0x11 {
		t.Fatalf("Control_L+a: mods=%v key=%#x err=%v", mods, key, err)
	}
	mods, key, err = keysymChord("Ctrl+Alt+Delete")
	if err != nil || key != 0x2E || len(mods) != 2 {
		t.Fatalf("Ctrl+Alt+Delete: mods=%v key=%#x err=%v", mods, key, err)
	}
	if _, _, err := keysymChord("ctrl+whatever"); err == nil {
		t.Fatal("未知键名必须报错")
	}
}

// stubInjection 换注入桩并返回调用计数（零调用断言用）。
func stubInjection(t *testing.T) *int {
	t.Helper()
	calls := 0
	prevM, prevK, prevE := sendMouse, sendKeys, clickElement
	sendMouse = func([]mouseEvent) error { calls++; return nil }
	sendKeys = func([]keybdEvent) error { calls++; return nil }
	clickElement = func(unsafe.Pointer) error { calls++; return nil }
	t.Cleanup(func() { sendMouse, sendKeys, clickElement = prevM, prevK, prevE })
	return &calls
}

// stubObserveSnapshot 让平台观察返回定制快照（含元素缓存）。
func stubObserveSnapshot(t *testing.T, snap *Snapshot) {
	t.Helper()
	stubObserve(t, func(int, ObserveOpt) (*Snapshot, error) {
		clone := *snap
		clone.Revision = 0 // Observe 覆盖
		clone.cached = snap.cached
		return &clone, nil
	})
}

func TestStaleElementGateZeroInjection(t *testing.T) {
	calls := stubInjection(t)
	s, _ := newTestSession(t, 0)
	// 1) 从未观察 → 元素动作必须拒且零注入。
	if err := s.ClickWindowElement(0); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("未观察必须 stale: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("拒绝路径必须零注入: %d", *calls)
	}
	// 2) 纯截图观察（无文本缓存）→ 旧索引同样拒（codex 契约）。
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "", Shot: &Shot{ID: "s0"}})
	if _, err := s.Observe(42, ObserveOpt{IncludeScreenshot: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.ClickWindowElement(0); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("截图型观察后元素动作必须 stale: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("截图型观察后必须零注入: %d", *calls)
	}
	// 3) 带文本观察（含缓存）→ 闸门过、注入桩恰一次（Invoke 失败回落坐标）。
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "Window: \"记事本\", App: notepad.",
		cached: []unsafe.Pointer{fakeElement()}, Elements: 1})
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.ClickWindowElement(0); err != nil {
		t.Fatalf("带文本观察后元素点击应放行: %v", err)
	}
	if *calls != 1 {
		t.Fatalf("放行路径应注入一次: %d", *calls)
	}
}

func TestElementIndexOutOfBoundsContract(t *testing.T) {
	stubInjection(t)
	s, _ := newTestSession(t, 0)
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "t",
		cached: []unsafe.Pointer{fakeElement(), fakeElement()}, Elements: 2})
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	err := s.ClickWindowElement(5)
	if err == nil || !strings.Contains(err.Error(), "UI Automation element index is out of bounds: 5") {
		t.Fatalf("越界契约文案必须原样透出: %v", err)
	}
}

func TestCoordinateGateTriStates(t *testing.T) {
	stubInjection(t)
	s, _ := newTestSession(t, 0)
	// 从未截图 → required。
	if err := s.ClickWindow(10, 10, "left", 1); !errors.Is(err, ErrScreenshotIDRequired) {
		t.Fatalf("无截图必须 required: %v", err)
	}
	// 带截图观察 → 放行（注入桩一次）。
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "t", Shot: &Shot{ID: "s1"}})
	if _, err := s.Observe(42, ObserveOpt{IncludeScreenshot: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.ClickWindow(10, 10, "left", 1); err != nil {
		t.Fatalf("截图后坐标点击应放行: %v", err)
	}
	// 再观察一次（纯文本）→ 截图版次落后 → stale。
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "t"})
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.ClickWindow(10, 10, "left", 1); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("旧截图坐标必须 stale: %v", err)
	}
}

func TestCoordinatesDisabledAndGuardWindow(t *testing.T) {
	stubInjection(t)
	s, _ := newTestSession(t, 0)
	s.SetCoordinatesDisabled(true)
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "t", Shot: &Shot{ID: "s1"}})
	if _, err := s.Observe(42, ObserveOpt{IncludeScreenshot: true, IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.ClickWindow(1, 1, "left", 1); !errors.Is(err, ErrCoordinatesDisabled) {
		t.Fatalf("DPI 禁用后坐标动作必须拒: %v", err)
	}
	// deny 表：终端窗直接拒且零注入。
	s2, _ := newTestSession(t, 0)
	calls := stubInjection(t)
	stubObserveSnapshot(t, &Snapshot{Window: Window{ID: 7, App: `process:C:\Windows\system32\cmd.exe`, Title: "管理员: 命令提示符"}, Text: "t", Shot: &Shot{ID: "s1"}})
	if _, err := s2.Observe(7, ObserveOpt{IncludeScreenshot: true, IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s2.ClickWindow(1, 1, "left", 1); err == nil || !strings.Contains(err.Error(), "终端") {
		t.Fatalf("终端窗必须被 deny 表拒: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("deny 表拒绝路径必须零注入: %d", *calls)
	}
}

func TestCheckBoundWindowStale(t *testing.T) {
	stubInjection(t)
	s, _ := newTestSession(t, 0)
	stubObserveSnapshot(t, &Snapshot{Window: fakeWindow(), Text: "t",
		cached: []unsafe.Pointer{fakeElement()}, Elements: 1})
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.CheckBoundWindow(42); err != nil {
		t.Fatalf("同窗应放行: %v", err)
	}
	if err := s.CheckBoundWindow(43); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("换窗必须 stale: %v", err)
	}
}

func TestMouseKeybEventSizes(t *testing.T) {
	// INPUT stride 必须 40（SendInput 数组步长契约）。
	if got := unsafe.Sizeof(mouseEvent{}); got != 40 {
		t.Fatalf("mouseEvent=%d want 40", got)
	}
	if got := unsafe.Sizeof(keybdEvent{}); got != 40 {
		t.Fatalf("keybdEvent=%d want 40", got)
	}
}
