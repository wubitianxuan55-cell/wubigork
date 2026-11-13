//go:build windows

// Package wpshost — 本机 WPS 文档窗口「面板化」宿主（灵犀 optimize plan P1-4
// 第一步 dock；探针 tools/wpsdock 的产品化收编，win32 syscall 移植）。
//
// 定位：把 WPS 文档窗口**摆放**到 gaea 右侧面板矩形上（不改父子关系，
// 不 SetParent——embed 是第二步，默认关闭，须用户拍板另开）。
// 还原：停靠前记住 GWL_STYLE，restore 恢复样式并置回顶层。
package wpshost

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pEnumWindows              = user32.NewProc("EnumWindows")
	pGetClassNameW            = user32.NewProc("GetClassNameW")
	pGetWindowTextW           = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	pGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	pIsWindowVisible          = user32.NewProc("IsWindowVisible")
	pGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	pSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pShowWindow               = user32.NewProc("ShowWindow")
	pIsWindow                 = user32.NewProc("IsWindow")

	pOpenProcess               = kernel32.NewProc("OpenProcess")
	pQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	pCloseHandle               = kernel32.NewProc("CloseHandle")
)

const (
	// GWL_STYLE = -16，用补码常量避免 uintptr 溢出（wpsdock 同款）。
	gwlStyle uintptr = ^uintptr(15)

	wsCaption     = 0x00C00000
	wsThickFrame  = 0x00040000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsMaximizeBox = 0x00010000

	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
	swpShowWindow   = 0x0040
	swpNoMove       = 0x0002
	swpNoSize       = 0x0001

	processQueryLimited = 0x1000
)

// Window 是一个 WPS 相关顶层窗口的描述。
type Window struct {
	HWND     uint64 `json:"hwnd"`
	Proc     string `json:"proc"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Visible  bool   `json:"visible"`
	IsDocWin bool   `json:"is_doc_win"` // 类名=XLMAIN(表格)/OpusApp(文字) 的文档窗口
}

// dockState 是一个窗口停靠前的样式快照（还原依据）。
type dockState struct {
	HWND  uint64 `json:"hwnd"`
	Style uint32 `json:"style"`
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	var b []uint16
	for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; ptr = unsafe.Add(ptr, 2) {
		b = append(b, *(*uint16)(ptr))
	}
	return syscall.UTF16ToString(b)
}

func procName(pid uint32) string {
	h, _, _ := pOpenProcess.Call(processQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return "?"
	}
	defer pCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	r, _, _ := pQueryFullProcessImageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return "?"
	}
	full := syscall.UTF16ToString(buf[:n])
	if i := strings.LastIndexByte(full, '\\'); i >= 0 {
		return full[i+1:]
	}
	return full
}

// isDocClass：WPS 表格沿用 Excel 的 XLMAIN，文字沿用 Word 的 OpusApp
//（wpsdock 探针实测；PPT 类名版本差异大，v1 不收）。
func isDocClass(cls string) bool {
	return cls == "XLMAIN" || cls == "OpusApp"
}

func isWPSProc(p string) bool {
	p = strings.ToLower(p)
	return p == "wps.exe" || p == "et.exe" || p == "wpp.exe" || p == "wpspdf.exe"
}

// List 枚举本机全部 WPS 相关顶层窗口。
func List() []Window {
	var out []Window
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var pid uint32
		pGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		pn := procName(pid)
		if !isWPSProc(pn) {
			return 1
		}
		clsBuf := make([]uint16, 256)
		pGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&clsBuf[0])), 256)
		cls := syscall.UTF16ToString(clsBuf)
		ln, _, _ := pGetWindowTextLengthW.Call(hwnd)
		var title string
		if ln > 0 {
			tb := make([]uint16, ln+1)
			pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&tb[0])), ln+1)
			title = syscall.UTF16ToString(tb)
		}
		vis, _, _ := pIsWindowVisible.Call(hwnd)
		out = append(out, Window{
			HWND: uint64(hwnd), Proc: pn, Class: cls, Title: title,
			Visible: vis != 0, IsDocWin: isDocClass(cls),
		})
		return 1
	})
	pEnumWindows.Call(cb, 0)
	return out
}

func statePath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "gaea", "wps-dock-state.json")
}

func loadStates() []dockState {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return nil
	}
	var s []dockState
	_ = json.Unmarshal(b, &s)
	return s
}

func saveStates(s []dockState) {
	_ = os.MkdirAll(filepath.Dir(statePath()), 0o700)
	b, _ := json.MarshalIndent(s, "", "  ")
	_ = os.WriteFile(statePath(), b, 0o600)
}

func rememberStyle(hwnd uintptr) {
	st, _, _ := pGetWindowLongPtrW.Call(hwnd, gwlStyle)
	list := loadStates()
	for i := range list {
		if list[i].HWND == uint64(hwnd) {
			list[i].Style = uint32(st)
			saveStates(list)
			return
		}
	}
	saveStates(append(list, dockState{HWND: uint64(hwnd), Style: uint32(st)}))
}

// Dock 把窗口摆放到屏幕矩形 (x,y,w,h)，borderless=true 时去掉标题栏/边框
//（样式已先记档，restore 可还原）。
func Dock(hwnd uint64, x, y, w, h int, borderless bool) error {
	target := uintptr(hwnd)
	if ok, _, _ := pIsWindow.Call(target); ok == 0 {
		return fmt.Errorf("hwnd %d 不是有效窗口（可能已被关闭）", hwnd)
	}
	rememberStyle(target)
	if borderless {
		st, _, _ := pGetWindowLongPtrW.Call(target, gwlStyle)
		style := uint32(st) &^ (wsCaption | wsThickFrame | wsSysMenu | wsMinimizeBox | wsMaximizeBox)
		pSetWindowLongPtrW.Call(target, gwlStyle, uintptr(style))
	}
	pSetWindowPos.Call(target, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		swpNoZOrder|swpNoActivate|swpFrameChanged|swpShowWindow)
	return nil
}

// Restore 还原窗口样式并置回顶层（ NOMOVE|NOSIZE 只改样式与可见性）。
func Restore(hwnd uint64) error {
	target := uintptr(hwnd)
	for _, s := range loadStates() {
		if s.HWND == hwnd {
			pSetWindowLongPtrW.Call(target, gwlStyle, uintptr(s.Style))
			pSetWindowPos.Call(target, 0, 0, 0, 0, 0,
				swpNoMove|swpNoSize|swpFrameChanged|swpShowWindow)
			return nil
		}
	}
	return fmt.Errorf("状态档里没有 hwnd %d 的停靠记录（可能未曾停靠或已被还原）", hwnd)
}

// RestoreAll 还原全部已停靠窗口（gaea 退出/面板关闭兜底）。
func RestoreAll() int {
	list := loadStates()
	for _, s := range list {
		if ok, _, _ := pIsWindow.Call(uintptr(s.HWND)); ok != 0 {
			pSetWindowLongPtrW.Call(uintptr(s.HWND), gwlStyle, uintptr(s.Style))
			pSetWindowPos.Call(uintptr(s.HWND), 0, 0, 0, 0, 0,
				swpNoMove|swpNoSize|swpFrameChanged|swpShowWindow)
		}
	}
	saveStates(nil)
	return len(list)
}
