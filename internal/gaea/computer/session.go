package computer

// 电脑操控会话骨架（施工单 W1，2026-11-19）。
//
// Session 持有「当前绑定窗口 + 观察版次 + 最近截图」三件事，是失效闸门
//（asset-62 §5）的 gaea 侧载体：
//   - revision 每次 Observe 自增；动作前 CheckRevision，不匹配 → ErrStaleObservation；
//   - shotID 只在带截图的观察后非空；坐标动作 CheckScreenshotID；
//   - 空闲 TTL 懒回收（与 browser.Manager 的 watcher 不同：本会话无 OS 资源
//     需要主动回收，纯内存绑定，每次访问时按注入时钟惰性判定即可，确定性
//     且可测）。
//
// 平台分野：观察的真实现按构建标签落在 computer_windows.go / computer_other.go
//（对齐 internal/screen 的平台文件范式）；本文件只做平台无关的版次/绑定/TTL。

import (
	"fmt"
	"sync"
	"time"
)

// defaultIdleTTL 单例默认空闲回收时长（对齐 browser.Manager 10m）。
const defaultIdleTTL = 10 * time.Minute

// Options 会话选项。
type Options struct {
	// IdleTTL 空闲自动释放绑定时长（0=禁用，保持常驻绑定）。
	IdleTTL time.Duration
}

// Window 一个可操控的顶层窗口。App 是 process:<exe 全路径> 形态的唯一
// 稳定标识（授权、审批卡、deny 表都用它）。
type Window struct {
	ID    int    `json:"id"`
	App   string `json:"app"`   // process:<完整 exe 路径>
	Title string `json:"title"` // 窗口标题（DwmGetWindowAttribute，非 GetWindowText）
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
}

// ObserveOpt 观察开关（语义照抄 codex get_window_state：两开关不许同时为假）。
type ObserveOpt struct {
	IncludeText       bool // 渲染无障碍树文本
	IncludeScreenshot bool // 抓窗口截图（W0 多模态通道通了才默认开）
}

// Shot 一次观察附带的截图。Path 永远是落盘路径（给人看/复用）；
// data URL 仅在请求序列化时由 bridge 编码（这里只存路径，省常驻内存）。
type Shot struct {
	ID           string `json:"id"` // <uuid>-<region>，每次观察更换
	Path         string `json:"path"`
	Width        int    `json:"width"`         // 逻辑像素
	Height       int    `json:"height"`        // 逻辑像素
	NativeWidth  int    `json:"native_width"`  // 物理像素
	NativeHeight int    `json:"native_height"` // 物理像素
}

// Snapshot 一次观察的结果。Revision 是本会话内全局递增的版次。
type Snapshot struct {
	Window   Window `json:"window"`
	Revision uint64 `json:"revision"`
	Text     string `json:"text,omitempty"` // 树文本（≤32KB，超限按行截断）
	Shot     *Shot  `json:"shot,omitempty"`
}

// Session 会话状态：绑定窗口 + 版次 + 最近观察 + 空闲回收。
type Session struct {
	opts Options

	mu                  sync.Mutex
	windowID            int              // 当前绑定窗口（0=未绑定）
	revision            uint64           // 每次 Observe 自增；回收归零
	shotID              string           // 最近一次带截图观察的 shot id（""=从未）
	lastState           *Snapshot        // 最近一次观察（动作后回树复用它落库）
	lastUsed            time.Time        // 最近活跃时刻（懒 TTL 判定）
	coordinatesDisabled bool             // DPI 复原失败即整体禁用坐标动作（asset-62 §6）
	now                 func() time.Time // 可注入假时钟（测试）
}

var (
	defaultMu      sync.Mutex
	defaultSession *Session
)

// Default 返回进程级单例（工具层每次 Execute 调用），默认 10m 空闲回收。
func Default() *Session {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultSession == nil {
		defaultSession = NewSession(Options{IdleTTL: defaultIdleTTL})
	}
	return defaultSession
}

// SetForTest 替换单例（返回旧实例），照 browser.Manager 范式。
func SetForTest(s *Session) *Session {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	prev := defaultSession
	defaultSession = s
	return prev
}

// NewSession 构造会话。now 缺省 time.Now。
func NewSession(opts Options) *Session {
	return &Session{opts: opts, now: time.Now}
}

// SetClock 注入假时钟（测试专用）。
func (s *Session) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// platformObserve 平台观察入口：由 computer_windows.go / computer_other.go
// 的 init 按构建标签赋值（包级变量便于测试桩注入）。
var platformObserve func(windowID int, opt ObserveOpt) (*Snapshot, error)

// Observe 观察一个窗口：版次自增、更新绑定、透传平台观察。
// include_text/include_screenshot 同假直接拒（照抄 codex 契约文案）。
func (s *Session) Observe(windowID int, opt ObserveOpt) (*Snapshot, error) {
	if !opt.IncludeText && !opt.IncludeScreenshot {
		return nil, fmt.Errorf("computer_state must request include_text, include_screenshot, or both")
	}
	s.mu.Lock()
	s.reapIfIdleLocked()
	s.windowID = windowID
	s.revision++
	rev := s.revision
	s.lastUsed = s.now()
	s.mu.Unlock()

	snap, err := platformObserve(windowID, opt)
	if err != nil {
		return nil, err
	}
	snap.Revision = rev
	s.mu.Lock()
	s.lastState = snap
	if snap.Shot != nil {
		s.shotID = snap.Shot.ID
	}
	s.mu.Unlock()
	return snap, nil
}

// CheckRevision 失效闸门（元素索引）：rev 与当前版次不符 → ErrStaleObservation。
// 先过懒回收——空闲超时后绑定已释放，旧版次天然失效。
func (s *Session) CheckRevision(rev uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reapIfIdleLocked()
	if rev == 0 || rev != s.revision {
		return ErrStaleObservation
	}
	return nil
}

// CheckScreenshotID 坐标闸门：id 与最近带截图观察不符 → 失效；从未截过 →
// ErrScreenshotIDRequired。
func (s *Session) CheckScreenshotID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reapIfIdleLocked()
	if s.shotID == "" {
		return ErrScreenshotIDRequired
	}
	if id != s.shotID {
		return ErrStaleObservation
	}
	return nil
}

// LastState 返回最近一次观察的快照（动作后回树复用；可能为 nil）。
func (s *Session) LastState() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reapIfIdleLocked()
	return s.lastState
}

// BoundWindow 当前绑定窗口 ID（0=未绑定）。
func (s *Session) BoundWindow() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reapIfIdleLocked()
	return s.windowID
}

// SetCoordinatesDisabled DPI 复原失败后的整体禁用（W5 落实；骨架期先留位）。
func (s *Session) SetCoordinatesDisabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.coordinatesDisabled = v
}

// CoordinatesDisabled 坐标动作是否已被禁用。
func (s *Session) CoordinatesDisabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coordinatesDisabled
}

// Touch 刷新活跃时刻（动作族工具调用前调用）。
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = s.now()
}

// reapIfIdleLocked 懒回收：空闲超过 TTL 释放绑定（调用方持锁）。TTL=0 禁用。
func (s *Session) reapIfIdleLocked() {
	if s.opts.IdleTTL <= 0 || s.lastUsed.IsZero() {
		return
	}
	if s.now().Sub(s.lastUsed) < s.opts.IdleTTL {
		return
	}
	s.windowID = 0
	s.revision = 0
	s.shotID = ""
	s.lastState = nil
}
