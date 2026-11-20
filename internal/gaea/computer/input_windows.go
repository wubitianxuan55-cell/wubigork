//go:build windows

package computer

// W5 输入注入（施工单）：SendInput 单通道（鼠标绝对坐标/键盘 Unicode/组合键）
// + SetThreadDpiAwarenessContext 切换与复原（复原失败由调用方置
// coordinatesDisabled——asset-62 §6 失败即禁用）。
// 结构体布局逐字段对齐 Win32（INPUT stride=40；MOUSEINPUT 偏移 dx@8
// dy@12 data@16 flags@20 time@24 extra@32；KEYBDINPUT vk@8 scan@10
// flags@12 time@16 extra@24），尾巴补垫到 40 保证数组步长。
// 注入走 sendMouse/sendKeys seam（测试换桩断言零调用）。

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modUser32In = syscall.NewLazyDLL("user32.dll")

	procSendInput             = modUser32In.NewProc("SendInput")
	procSetThreadDpiCtx       = modUser32In.NewProc("SetThreadDpiAwarenessContext")
	procSwitchToThisWindow    = modUser32In.NewProc("SwitchToThisWindow")
	procSetForegroundWindowIn = modUser32In.NewProc("SetForegroundWindow")
	procGetSystemMetricsIn    = modUser32In.NewProc("GetSystemMetrics")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfMove        = 0x0001
	mouseeventfLeftDown    = 0x0002
	mouseeventfLeftUp      = 0x0004
	mouseeventfRightDown   = 0x0008
	mouseeventfRightUp     = 0x0010
	mouseeventfMiddleDown  = 0x0020
	mouseeventfMiddleUp    = 0x0040
	mouseeventfWheel       = 0x0800
	mouseeventfVirtualDesk = 0x4000
	mouseeventfAbsolute    = 0x8000

	keyeventfKeyup   = 0x0002
	keyeventfUnicode = 0x0004

	smXVirtual  = 76
	smYVirtual  = 77
	smCxVirtual = 78
	smCyVirtual = 79

	// dpiPerMonitorV2 = DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2（(HANDLE)-4；
	// 常量负数不能直接转 uintptr，用位反形态 ^uintptr(3)）。
	dpiPerMonitorV2 = ^uintptr(3)
)

// mouseEvent INPUT(MOUSEINPUT)：stride=40。偏移对齐 Win32：dx@8 dy@12
// data@16 flags@20 time@24 extra@32（Time 后显式中垫，ExtraInfo 8 对齐）。
type mouseEvent struct {
	Type      uint32
	_         uint32
	DX        int32
	DY        int32
	MouseData int32
	DwFlags   uint32
	Time      uint32
	_         uint32
	ExtraInfo uintptr
}

// keybdEvent INPUT(KEYBDINPUT)：vk@8 scan@10 flags@12 time@16 extra@24；
// 联合体按最大成员 MOUSEINPUT(32) 铺，尾垫钉死 40 步长。
type keybdEvent struct {
	Type      uint32
	_         uint32
	WVk       uint16
	WScan     uint16
	DwFlags   uint32
	Time      uint32
	_         [4]byte
	ExtraInfo uintptr
	_         [8]byte
}

var (
	_ = unsafe.Sizeof(mouseEvent{}) // 构建期可见性；尺寸断言在测试（=40）
)

// sendMouse/sendKeys 注入 seam（测试换桩断言零调用）。
var (
	sendMouse = func(events []mouseEvent) error {
		if len(events) == 0 {
			return nil
		}
		n, _, _ := procSendInput.Call(uintptr(len(events)), uintptr(unsafe.Pointer(&events[0])), unsafe.Sizeof(mouseEvent{}))
		if n != uintptr(len(events)) {
			return fmt.Errorf("SendInput(mouse) 注入不完整: %d/%d", n, len(events))
		}
		return nil
	}
	sendKeys = func(events []keybdEvent) error {
		if len(events) == 0 {
			return nil
		}
		n, _, _ := procSendInput.Call(uintptr(len(events)), uintptr(unsafe.Pointer(&events[0])), unsafe.Sizeof(keybdEvent{}))
		if n != uintptr(len(events)) {
			return fmt.Errorf("SendInput(keybd) 注入不完整: %d/%d", n, len(events))
		}
		return nil
	}
)

// withThreadDpi 进入 per-monitor-v2 感知上下文执行 fn，结束后复原。
// 复原失败叠加 errDpiRestoreFailed（调用方据此置 coordinatesDisabled）。
func withThreadDpi(fn func() error) error {
	if err := procSetThreadDpiCtx.Find(); err != nil {
		return fn() // 老系统无此 API：按进程级感知继续
	}
	prev, _, _ := procSetThreadDpiCtx.Call(dpiPerMonitorV2)
	err := fn()
	if restored, _, _ := procSetThreadDpiCtx.Call(prev); restored == 0 {
		err = errors.Join(err, errDpiRestoreFailed)
	}
	return err
}

// toAbsolute 屏幕物理坐标 → SendInput 绝对坐标（0..65535 全虚拟屏）。
func toAbsolute(sx, sy int) (uint32, uint32) {
	v, _, _ := procGetSystemMetricsIn.Call(smXVirtual)
	vy, _, _ := procGetSystemMetricsIn.Call(smYVirtual)
	vw, _, _ := procGetSystemMetricsIn.Call(smCxVirtual)
	vh, _, _ := procGetSystemMetricsIn.Call(smCyVirtual)
	vx, vyv, vwv, vhv := int(int32(v)), int(int32(vy)), int(int32(vw)), int(int32(vh))
	if vwv <= 1 || vhv <= 1 {
		return 0, 0
	}
	fx := (sx - vx) * 65535 / (vwv - 1)
	fy := (sy - vyv) * 65535 / (vhv - 1)
	if fx < 0 {
		fx = 0
	}
	if fy < 0 {
		fy = 0
	}
	return uint32(fx), uint32(fy)
}

// mouseButtons 按钮名 → down/up 旗标（double=双击展开两次）。
func mouseButtons(button string) (down, up uint32, count int, err error) {
	switch strings.ToLower(strings.TrimSpace(button)) {
	case "", "left", "click", "single":
		return mouseeventfLeftDown, mouseeventfLeftUp, 1, nil
	case "double":
		return mouseeventfLeftDown, mouseeventfLeftUp, 2, nil
	case "right":
		return mouseeventfRightDown, mouseeventfRightUp, 1, nil
	case "middle":
		return mouseeventfMiddleDown, mouseeventfMiddleUp, 1, nil
	default:
		return 0, 0, 0, fmt.Errorf("未知按钮 %q（left/right/middle/double）", button)
	}
}

// ClickAtScreen 屏幕绝对物理坐标点击（DPI 上下文内换算+注入）。
func ClickAtScreen(button string, count int, sx, sy int) error {
	down, up, n, err := mouseButtons(button)
	if err != nil {
		return err
	}
	if count <= 0 {
		count = n
	}
	return withThreadDpi(func() error {
		ax, ay := toAbsolute(sx, sy)
		events := []mouseEvent{{
			Type: inputMouse, DX: int32(ax), DY: int32(ay),
			DwFlags: mouseeventfMove | mouseeventfAbsolute | mouseeventfVirtualDesk,
		}}
		for i := 0; i < count; i++ {
			events = append(events,
				mouseEvent{Type: inputMouse, DwFlags: down},
				mouseEvent{Type: inputMouse, DwFlags: up})
		}
		return sendMouse(events)
	})
}

// TypeTextRaw Unicode 逐字符注入（换行走 Return 键；\r 忽略）。
func TypeTextRaw(text string) error {
	var events []keybdEvent
	for _, r := range text {
		if r == '\r' {
			continue
		}
		if r == '\n' {
			events = append(events,
				keybdEvent{Type: inputKeyboard, WVk: 0x0D},
				keybdEvent{Type: inputKeyboard, WVk: 0x0D, DwFlags: keyeventfKeyup})
			continue
		}
		scan := uint16(r)
		events = append(events,
			keybdEvent{Type: inputKeyboard, WScan: scan, DwFlags: keyeventfUnicode},
			keybdEvent{Type: inputKeyboard, WScan: scan, DwFlags: keyeventfUnicode | keyeventfKeyup})
	}
	return sendKeys(events)
}

// FocusWindow 把目标窗拉到前台（SwitchToThisWindow 稳定路径 + SetForeground 兜底）。
func FocusWindow(windowID int) {
	procSwitchToThisWindow.Call(uintptr(windowID), 1)
	procSetForegroundWindowIn.Call(uintptr(windowID))
	time.Sleep(60 * time.Millisecond) // 前台切换落定
}

// ScrollAtScreen 屏幕绝对坐标滚轮（dy>0 向下，单位=齿格；横向滚轮 Windows
// 硬件面普遍缺 WHEEL_HDX 支持面，明确拒绝防伪横滚）。
func ScrollAtScreen(sx, sy int, dx, dy int) error {
	if dx != 0 {
		return fmt.Errorf("Windows 滚轮仅支持纵向（dx=%d；横向滚动请用键盘方向键）", dx)
	}
	if dy == 0 {
		return nil
	}
	return withThreadDpi(func() error {
		ax, ay := toAbsolute(sx, sy)
		events := []mouseEvent{{
			Type: inputMouse, DX: int32(ax), DY: int32(ay),
			DwFlags: mouseeventfMove | mouseeventfAbsolute | mouseeventfVirtualDesk,
		}}
		steps := dy
		delta := int32(-120) // dy>0 = 向下（滚轮负向）
		if steps < 0 {
			steps = -steps
			delta = 120
		}
		for i := 0; i < steps; i++ {
			events = append(events, mouseEvent{Type: inputMouse, DwFlags: mouseeventfWheel, MouseData: delta})
		}
		return sendMouse(events)
	})
}

// keysymChord 解析 X11 keysym 风格组合键（"Control_L+a"；别名 Ctrl/Control/
// KP_0/Numpad_0/Numpad_Add 等，asset-62 §7 名字形态全收）；最后一段=终键，
// 其余=修饰键。deny 表（GuardChord）由调用方先过。
func keysymChord(chord string) (mods []uint16, key uint16, err error) {
	var vks []uint16
	for _, p := range strings.Split(chord, "+") {
		name := strings.ToLower(strings.TrimSpace(p))
		if name == "" {
			continue
		}
		vk, ok := keysymVK[name]
		if !ok {
			rs := []rune(name)
			if len(rs) == 1 {
				if vk = charVK(rs[0]); vk != 0 {
					ok = true
				}
			}
		}
		if !ok {
			return nil, 0, fmt.Errorf("未知键名 %q（支持 keysym 名/单字符/别名，标点与中文请用 computer_type）", p)
		}
		vks = append(vks, vk)
	}
	if len(vks) == 0 {
		return nil, 0, fmt.Errorf("组合键为空: %q", chord)
	}
	return vks[:len(vks)-1], vks[len(vks)-1], nil
}

// keysymVK 键名 → VK 表（基础条目；init 程序化补数字/字母/F 键/小键盘）。
var keysymVK = map[string]uint16{
	"return": 0x0D, "enter": 0x0D, "kp_enter": 0x0D,
	"tab": 0x09, "escape": 0x1B, "esc": 0x1B,
	"space": 0x20, "backspace": 0x08, "delete": 0x2E, "del": 0x2E,
	"insert": 0x2D, "home": 0x24, "end": 0x23,
	"pageup": 0x21, "pagedown": 0x22, "prior": 0x21, "next": 0x22,
	"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27,
	"control": 0x11, "control_l": 0x11, "control_r": 0xA3,
	"ctrl": 0x11, "ctrl_l": 0x11, "ctrl_r": 0xA3,
	"shift": 0x10, "shift_l": 0x10, "shift_r": 0xA1,
	"alt": 0x12, "alt_l": 0x12, "alt_r": 0xA5, "menu": 0x12,
	"comma": 0xBC, "period": 0xBE, "slash": 0xBF, "semicolon": 0xBA,
	"minus": 0xBD, "equal": 0xBB, "quote": 0xDE, "backquote": 0xC0,
	"backslash": 0xDC, "bracketleft": 0xDB, "bracketright": 0xDD,
	"greater": 0xBE, "less": 0xBC, "question": 0xBF,
	"numpad_add": 0x6B, "kp_add": 0x6B,
	"numpad_subtract": 0x6D, "kp_subtract": 0x6D,
	"numpad_multiply": 0x6A, "kp_multiply": 0x6A,
	"numpad_divide": 0x6F, "kp_divide": 0x6F,
	"numpad_decimal": 0x6E, "kp_decimal": 0x6E,
}

func init() {
	// 0-9 / a-z / F1-F24 / KP_0-9（Numpad_0..9 别名）程序化入表。
	for i := 0; i < 10; i++ {
		keysymVK[string(rune('0'+i))] = uint16(0x30 + i)
		keysymVK[fmt.Sprintf("kp_%d", i)] = uint16(0x60 + i)
		keysymVK[fmt.Sprintf("numpad_%d", i)] = uint16(0x60 + i)
	}
	for c := byte('a'); c <= 'z'; c++ {
		keysymVK[string(rune(c))] = uint16(0x41 + (c - 'a'))
	}
	for i := 1; i <= 24; i++ {
		keysymVK[fmt.Sprintf("f%d", i)] = uint16(0x70 + i - 1)
	}
}

// charVK 单字符 → VK（大小写字母/数字；其余 0=不支持走 Unicode 通道）。
func charVK(r rune) uint16 {
	switch {
	case r >= 'a' && r <= 'z':
		return uint16(0x41 + (r - 'a'))
	case r >= 'A' && r <= 'Z':
		return uint16(0x41 + (r - 'A'))
	case r >= '0' && r <= '9':
		return uint16(0x30 + (r - '0'))
	}
	return 0
}

// PressKeyChord 组合键注入（修饰键按下→终键→修饰键逆序释放）。
func PressKeyChord(chord string) error {
	mods, key, err := keysymChord(chord)
	if err != nil {
		return err
	}
	var events []keybdEvent
	for _, m := range mods {
		events = append(events, keybdEvent{Type: inputKeyboard, WVk: m})
	}
	events = append(events,
		keybdEvent{Type: inputKeyboard, WVk: key},
		keybdEvent{Type: inputKeyboard, WVk: key, DwFlags: keyeventfKeyup})
	for i := len(mods) - 1; i >= 0; i-- {
		events = append(events, keybdEvent{Type: inputKeyboard, WVk: mods[i], DwFlags: keyeventfKeyup})
	}
	return sendKeys(events)
}

// invokeElement UIA Invoke 通道（按钮/菜单项等首选；失败回落坐标）。
// 缓存元素可能产生于其他 Go 线程——先 uiaSession()（MTA）保证本线程 COM
// 初始化后再调它的 vtable。
func invokeElement(elem unsafe.Pointer) bool {
	u, err := uiaSession()
	if err != nil {
		return false
	}
	defer u.close()
	var pattern unsafe.Pointer
	hr := vcall(elem, vtblElemGetCurrentPattern, uintptr(uiaInvokePatternID), uintptr(unsafe.Pointer(&pattern)))
	if hr != 0 || pattern == nil {
		return false
	}
	defer release(pattern)
	return vcall(pattern, vtblInvokePatternInvoke) == 0
}

// elementCenterScreen 元素包围盒中心（屏幕物理坐标；get_CurrentBoundingRectangle
// 直出 RECT，免 SAFEARRAY 解析）。
func elementCenterScreen(elem unsafe.Pointer) (int, int, bool) {
	u, err := uiaSession()
	if err != nil {
		return 0, 0, false
	}
	defer u.close()
	var r struct{ L, T, R, B int32 }
	hr := vcall(elem, vtblElemBoundingRectangle, uintptr(unsafe.Pointer(&r)))
	if hr != 0 || r.R <= r.L || r.B <= r.T {
		return 0, 0, false
	}
	return int((r.L + r.R) / 2), int((r.T + r.B) / 2), true
}

// clickElement 元素点击 seam（测试换桩：门禁测试的假元素句柄不能走真
// COM vtable）。Windows 默认实现 = clickElementReal。
var clickElement = clickElementReal

// ClickElement 元素点击：Invoke 优先，失败回落包围盒中心坐标。
func clickElementReal(elem unsafe.Pointer) error {
	if invokeElement(elem) {
		return nil
	}
	x, y, ok := elementCenterScreen(elem)
	if !ok {
		return fmt.Errorf("元素既不支持 Invoke 也无有效包围盒（不可点）")
	}
	return ClickAtScreen("left", 1, x, y)
}

// WindowRelativeToScreen 窗口相对坐标 → 屏幕绝对物理坐标（per-monitor 感知
// 上下文内 GetWindowRect 直出物理像素，窗口偏移即物理偏移）。
func WindowRelativeToScreen(win Window, x, y int) (int, int) {
	return win.X + x, win.Y + y
}
