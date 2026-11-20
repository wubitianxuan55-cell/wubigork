package computer

// W1 骨架验收（施工单）：两开关同假拒 / 版次失效闸 / 坐标闸 / TTL 懒回收
//（假时钟注入，确定性；语义对齐 browser/manager_test.go 的 TTL 用例）。

import (
	"errors"
	"testing"
	"time"
)

// stubObserve 临时替换平台观察桩，返回恢复函数。
func stubObserve(t *testing.T, fn func(windowID int, opt ObserveOpt) (*Snapshot, error)) {
	t.Helper()
	prev := platformObserve
	platformObserve = fn
	t.Cleanup(func() { platformObserve = prev })
}

func fakeShot(id string) *Snapshot {
	return &Snapshot{Window: Window{ID: 42, App: "process:C:\\windows\\notepad.exe", Title: "记事本"},
		Text: "Window: \"记事本\", App: notepad.", Shot: &Shot{ID: id, Path: "C:\\shots\\x.jpg", Width: 800, Height: 600, NativeWidth: 800, NativeHeight: 600}}
}

func newTestSession(t *testing.T, ttl time.Duration) (*Session, *time.Time) {
	t.Helper()
	s := NewSession(Options{IdleTTL: ttl})
	clock := time.Unix(1700000000, 0)
	s.SetClock(func() time.Time { return clock })
	return s, &clock
}

func TestObserveRejectsBothFalse(t *testing.T) {
	s, _ := newTestSession(t, 0)
	_, err := s.Observe(42, ObserveOpt{})
	if err == nil || err.Error() != "computer_state must request include_text, include_screenshot, or both" {
		t.Fatalf("两开关同假必须拒（照抄 codex 契约文案）: %v", err)
	}
}

func TestRevisionIncrementsAndStaleGate(t *testing.T) {
	s, _ := newTestSession(t, 0)
	stubObserve(t, func(int, ObserveOpt) (*Snapshot, error) { return fakeShot("s1"), nil })
	r1, err := s.Observe(42, ObserveOpt{IncludeText: true})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	r2, err := s.Observe(42, ObserveOpt{IncludeText: true})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if r1.Revision != 1 || r2.Revision != 2 {
		t.Fatalf("版次必须逐次自增: %d,%d", r1.Revision, r2.Revision)
	}
	if err := s.CheckRevision(2); err != nil {
		t.Fatalf("当前版次应放行: %v", err)
	}
	if err := s.CheckRevision(1); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("旧版次必须拒: %v", err)
	}
	if err := s.CheckRevision(0); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("0 版次必须拒: %v", err)
	}
}

func TestScreenshotIDGate(t *testing.T) {
	s, _ := newTestSession(t, 0)
	if err := s.CheckScreenshotID("x"); !errors.Is(err, ErrScreenshotIDRequired) {
		t.Fatalf("从未截图必须回 required: %v", err)
	}
	stubObserve(t, func(int, ObserveOpt) (*Snapshot, error) { return fakeShot("shot-1"), nil })
	if _, err := s.Observe(42, ObserveOpt{IncludeScreenshot: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := s.CheckScreenshotID("shot-1"); err != nil {
		t.Fatalf("本观察 shotID 应放行: %v", err)
	}
	if err := s.CheckScreenshotID("shot-2"); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("他人 shotID 必须失效: %v", err)
	}
}

func TestIdleTTLReleasesBinding(t *testing.T) {
	s, clock := newTestSession(t, 10*time.Minute)
	stubObserve(t, func(int, ObserveOpt) (*Snapshot, error) { return fakeShot("s1"), nil })
	snap, err := s.Observe(42, ObserveOpt{IncludeText: true})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	// 活动期内：推进但不出 TTL，绑定与版次保持。
	*clock = clock.Add(9 * time.Minute)
	if s.BoundWindow() != 42 {
		t.Fatal("活动期内绑定不得释放")
	}
	// 空闲超时：懒回收把版次归零，旧版次天然失效。
	*clock = clock.Add(11 * time.Minute)
	if err := s.CheckRevision(snap.Revision); !errors.Is(err, ErrStaleObservation) {
		t.Fatalf("TTL 过后旧版次必须失效: %v", err)
	}
	if s.BoundWindow() != 0 || s.LastState() != nil {
		t.Fatal("TTL 过后绑定必须释放")
	}
	// 重新观察：版次从头计数（重新绑定是新会话周期）。
	snap2, err := s.Observe(43, ObserveOpt{IncludeText: true})
	if err != nil {
		t.Fatalf("re-observe: %v", err)
	}
	if snap2.Revision != 1 || s.BoundWindow() != 43 {
		t.Fatalf("回收后重新绑定: rev=%d win=%d", snap2.Revision, s.BoundWindow())
	}
}

func TestIdleTTLZeroDisabled(t *testing.T) {
	s, clock := newTestSession(t, 0)
	stubObserve(t, func(int, ObserveOpt) (*Snapshot, error) { return fakeShot("s1"), nil })
	if _, err := s.Observe(42, ObserveOpt{IncludeText: true}); err != nil {
		t.Fatalf("observe: %v", err)
	}
	*clock = clock.Add(24 * time.Hour)
	if err := s.CheckRevision(1); err != nil {
		t.Fatalf("TTL=0（禁用）不得回收: %v", err)
	}
	if s.BoundWindow() != 42 {
		t.Fatal("TTL=0 绑定必须保持")
	}
}

func TestDefaultSingleton(t *testing.T) {
	prev := Default()
	if prev == nil {
		t.Fatal("Default 必须返回单例")
	}
	if Default() != prev {
		t.Fatal("Default 必须幂等")
	}
	if prev.opts.IdleTTL != defaultIdleTTL {
		t.Fatalf("单例默认 TTL=10m: %v", prev.opts.IdleTTL)
	}
}
