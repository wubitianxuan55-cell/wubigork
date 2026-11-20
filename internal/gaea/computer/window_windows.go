//go:build windows

package computer

// 顶层窗口枚举（施工单 W2）：EnumWindows + DWM 遮罩过滤 + 进程全路径。
// Window.App 形态 process:<exe 全路径>，是授权/审批卡/deny 表的稳定标识
//（对齐 codex list_windows 的 app 字段）。

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modUser32W = syscall.NewLazyDLL("user32.dll")
	modKernelW = syscall.NewLazyDLL("kernel32.dll")
	modDwmapi  = syscall.NewLazyDLL("dwmapi.dll")

	procEnumWindows          = modUser32W.NewProc("EnumWindows")
	procIsWindowW            = modUser32W.NewProc("IsWindow")
	procIsWindowVisibleW     = modUser32W.NewProc("IsWindowVisible")
	procGetWindowTextW       = modUser32W.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = modUser32W.NewProc("GetWindowTextLengthW")
	procGetWindowLongW       = modUser32W.NewProc("GetWindowLongW")
	procGetWindowRectW       = modUser32W.NewProc("GetWindowRect")
	procGetWindowThreadPID   = modUser32W.NewProc("GetWindowThreadProcessId")
	procOpenProcessW         = modKernelW.NewProc("OpenProcess")
	procQueryFullImageNameW  = modKernelW.NewProc("QueryFullProcessImageNameW")
	procCloseHandleW         = modKernelW.NewProc("CloseHandle")
	procDwmGetWindowAttrW    = modDwmapi.NewProc("DwmGetWindowAttribute")
	procCLSIDFromStringW     = syscall.NewLazyDLL("ole32.dll").NewProc("CLSIDFromString")
)

const (
	// gwlExstyle = GWL_EXSTYLE(-20)；常量负数不能直接转 uintptr，用位反形态。
	gwlExstyle      = ^uintptr(19)
	wsExToolWindow  = 0x00000080
	dwmwaCloaked    = 14 // DWM_CLOAKED：UWP 挂起/虚拟桌面隐藏窗口
	pidQueryLimited = 0x1000
	maxPathW        = 32768
)

// ListWindows 枚举当前可见顶层窗口（z 序，最前最先）。过滤：不可见 / DWM
// 遮罩 / 工具窗（WS_EX_TOOLWINDOW）/ 无标题。
func ListWindows() ([]Window, error) {
	var out []Window
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if vis, _, _ := procIsWindowVisibleW.Call(hwnd); vis == 0 {
			return 1
		}
		if isToolWindowW(hwnd) || isCloakedW(hwnd) {
			return 1
		}
		title := windowTextW(hwnd)
		if title == "" {
			return 1
		}
		w := Window{ID: int(int32(hwnd)), Title: title, App: processPathOfW(hwnd)}
		w.X, w.Y, w.W, w.H = windowRectW(hwnd)
		out = append(out, w)
		return 1
	})
	if err := procEnumWindows.Find(); err != nil {
		return nil, fmt.Errorf("computer: %w", err)
	}
	procEnumWindows.Call(cb, 0)
	return out, nil
}

// windowIsAlive 判断 windowID（=HWND）是否仍有效。
func windowIsAlive(windowID int) bool {
	alive, _, _ := procIsWindowW.Call(uintptr(windowID))
	return alive != 0
}

// windowIdentity 取窗口标题与进程标识（观察/枚举共用）。
func windowIdentity(windowID int) (title, app string) {
	return windowTextW(uintptr(windowID)), processPathOfW(uintptr(windowID))
}

// windowRect 窗口 frame（屏幕坐标，物理像素）。
func windowRect(windowID int) (x, y, w, h int) {
	return windowRectW(uintptr(windowID))
}

func isToolWindowW(hwnd uintptr) bool {
	ex, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlExstyle))
	return ex&wsExToolWindow != 0
}

func isCloakedW(hwnd uintptr) bool {
	if err := procDwmGetWindowAttrW.Find(); err != nil {
		return false // 无 dwmapi（老系统）= 不遮罩
	}
	var cloaked uint32
	hr, _, _ := procDwmGetWindowAttrW.Call(hwnd, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), 4)
	return hr == 0 && cloaked != 0
}

func windowTextW(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(n+1))
	return syscall.UTF16ToString(buf)
}

func windowRectW(hwnd uintptr) (x, y, w, h int) {
	var r struct{ Left, Top, Right, Bottom int32 }
	procGetWindowRectW.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top)
}

// processPathOfW 取窗口属主进程的全路径（QueryFullProcessImageNameW），
// 失败返回空串（窗口仍可用，只是授权面拿不到稳定标识）。
func processPathOfW(hwnd uintptr) string {
	var pid uint32
	procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, _, _ := procOpenProcessW.Call(pidQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandleW.Call(h)
	buf := make([]uint16, maxPathW)
	size := uint32(len(buf))
	ret, _, _ := procQueryFullImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 { // BOOL：非零=成功（不是 HRESULT）
		return "process:" + syscall.UTF16ToString(buf[:size])
	}
	return ""
}

// guidFromString 解析 "{xxxxxxxx-…}" 形态 GUID（CLSIDFromString）。
func guidFromString(s string) (g syscall.GUID, err error) {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return g, err
	}
	hr, _, _ := procCLSIDFromStringW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&g)))
	if hr != 0 {
		return g, fmt.Errorf("computer: bad guid %s hr=0x%x", s, hr)
	}
	return g, nil
}
