//go:build windows

// wpsdock —— 本机 WPS 文档窗口「面板化」探针（独立工具，不参与 gaea 构建）
//
// 用途：验证「把 WPS 文档窗口放进 gaea 右侧面板」的两种实现是否可行。
//   list     只读：枚举 WPS 文档窗口（类名 + 标题 + 样式）
//   dock     停靠：把窗口按屏幕矩形摆放，可选去掉标题栏（安全，不改正属性）
//   embed    嵌入：SetParent 到宿主窗口（实验性，风险见 -h）
//   restore  还原：从 %TEMP%\wpsdock-state.json 恢复样式并置回顶层
//
// 用法示例：
//   go run ./tools/wpsdock list
//   go run ./tools/wpsdock dock -hwnd 9179416 -rect 1200,120,700,900 -borderless
//   go run ./tools/wpsdock dock -title 施工成本 -rect 1200,120,700,900 -borderless
//   go run ./tools/wpsdock embed -title 施工成本 -parent 1234567
//   go run ./tools/wpsdock restore -title 施工成本
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	pSetParent                = user32.NewProc("SetParent")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pShowWindow               = user32.NewProc("ShowWindow")
	pIsWindow                 = user32.NewProc("IsWindow")
	pGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	pGetWindowRect            = user32.NewProc("GetWindowRect")

	pOpenProcess               = kernel32.NewProc("OpenProcess")
	pQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	pCloseHandle               = kernel32.NewProc("CloseHandle")
)

const (
	// GWL_STYLE = -16 / GWL_EXSTYLE = -20，用补码常量避免 uintptr 溢出
	gwlStyle   uintptr = ^uintptr(15)
	gwlExStyle uintptr = ^uintptr(19)

	wsChild             = 0x40000000
	wsPopup             = 0x80000000
	wsCaption           = 0x00C00000
	wsThickFrame        = 0x00040000
	wsSysMenu           = 0x00080000
	wsMinimizeBox       = 0x00020000
	wsMaximizeBox       = 0x00010000
	swpNoZOrder         = 0x0004
	swpNoActivate       = 0x0010
	swpFrameChanged     = 0x0020
	swpShowWindow       = 0x0040
	processQueryLimited = 0x1000
)

type winInfo struct {
	HWND     uintptr `json:"hwnd"`
	Proc     string  `json:"proc"`
	Class    string  `json:"class"`
	Title    string  `json:"title"`
	Style    uint32  `json:"style"`
	ExStyle  uint32  `json:"exstyle"`
	Visible  bool    `json:"visible"`
	IsDocWin bool    `json:"is_doc_win"`
}

type saved struct {
	HWND  uintptr `json:"hwnd"`
	Style uint32  `json:"style"`
	Embed bool    `json:"embed"`
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

// 文档窗口类名：WPS 表格沿用 Excel 的 XLMAIN，文字沿用 Word 的 OpusApp；
// 少数版本走 Qt 外壳（Qt5QWindowIcon / KLiteMainWindowShadowBorder）。
func isDocClass(cls string) bool {
	switch cls {
	case "XLMAIN", "OpusApp":
		return true
	}
	return false
}

func isWPSProc(p string) bool {
	p = strings.ToLower(p)
	return p == "wps.exe" || p == "et.exe" || p == "wpp.exe" || p == "wpspdf.exe"
}

func enumWPS() []winInfo {
	var out []winInfo
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
		st, _, _ := pGetWindowLongPtrW.Call(hwnd, gwlStyle)
		ex, _, _ := pGetWindowLongPtrW.Call(hwnd, gwlExStyle)
		vis, _, _ := pIsWindowVisible.Call(hwnd)
		out = append(out, winInfo{
			HWND: hwnd, Proc: pn, Class: cls, Title: title,
			Style: uint32(st), ExStyle: uint32(ex), Visible: vis != 0,
			IsDocWin: isDocClass(cls),
		})
		return 1
	})
	pEnumWindows.Call(cb, 0)
	return out
}

func statePath() string {
	return filepath.Join(os.TempDir(), "wpsdock-state.json")
}

func loadState() []saved {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return nil
	}
	var s []saved
	_ = json.Unmarshal(b, &s)
	return s
}

func saveState(s []saved) {
	b, _ := json.MarshalIndent(s, "", "  ")
	_ = os.WriteFile(statePath(), b, 0o600)
}

func remember(hwnd uintptr, embed bool) {
	st, _, _ := pGetWindowLongPtrW.Call(hwnd, gwlStyle)
	list := loadState()
	for i := range list {
		if list[i].HWND == hwnd {
			list[i].Style = uint32(st)
			list[i].Embed = embed
			saveState(list)
			return
		}
	}
	list = append(list, saved{HWND: hwnd, Style: uint32(st), Embed: embed})
	saveState(list)
}

func pick(hwndArg uintptr, titleArg string) (uintptr, error) {
	if hwndArg != 0 {
		ok, _, _ := pIsWindow.Call(hwndArg)
		if ok == 0 {
			return 0, fmt.Errorf("hwnd %d 不是有效窗口", hwndArg)
		}
		return hwndArg, nil
	}
	for _, w := range enumWPS() {
		if !w.Visible || w.Title == "" {
			continue
		}
		if titleArg != "" && !strings.Contains(w.Title, titleArg) {
			continue
		}
		if w.IsDocWin || titleArg != "" {
			return w.HWND, nil
		}
	}
	return 0, fmt.Errorf("没找到匹配的 WPS 文档窗口，先用 list 看看")
}

func parseRect(s string) (x, y, w, h int32, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("rect 需为 x,y,w,h")
	}
	v := make([]int32, 4)
	for i, p := range parts {
		n, e := strconv.Atoi(strings.TrimSpace(p))
		if e != nil {
			return 0, 0, 0, 0, fmt.Errorf("rect 第 %d 段不是整数", i+1)
		}
		v[i] = int32(n)
	}
	return v[0], v[1], v[2], v[3], nil
}

func main() {
	cmd := "list"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	hwnd := fs.Uint64("hwnd", 0, "目标窗口句柄（十进制或 0x 前缀）")
	title := fs.String("title", "", "按标题子串挑选窗口（如 施工成本）")
	rect := fs.String("rect", "", "屏幕矩形 x,y,w,h（dock 用）")
	borderless := fs.Bool("borderless", false, "去掉标题栏/边框，看起来像面板（可 restore 还原）")
	parent := fs.Uint64("parent", 0, "宿主窗口句柄（embed 用，通常是 gaea 主窗口）")
	_ = fs.Parse(args)

	h := uintptr(*hwnd)

	switch cmd {
	case "list":
		list := enumWPS()
		fmt.Printf("%-12s %-8s %-10s %-8s %-9s %s\n", "hwnd", "proc", "class", "visible", "docwin", "title")
		for _, w := range list {
			if !w.Visible && w.Title == "" {
				continue
			}
			fmt.Printf("%-12d %-8s %-10s %-8v %-9v %s\n", w.HWND, w.Proc, w.Class, w.Visible, w.IsDocWin, w.Title)
		}
		for _, w := range list {
			if !w.IsDocWin {
				continue
			}
			var r struct{ L, T, R, B int32 }
			pGetWindowRect.Call(w.HWND, uintptr(unsafe.Pointer(&r)))
			dpi, _, _ := pGetDpiForWindow.Call(w.HWND)
			fmt.Printf("\n[文档窗口] hwnd=%d rect=(%d,%d)-(%d,%d) 尺寸=%dx%d dpi=%d\n",
				w.HWND, r.L, r.T, r.R, r.B, r.R-r.L, r.B-r.T, dpi)
		}
		fmt.Printf("\n合计 %d 个 WPS 相关窗口（只有 class=XLMAIN/OpusApp 是文档窗口）\n", len(list))

	case "dock":
		x, y, w, hh, err := parseRect(*rect)
		if err != nil {
			fmt.Println("✗", err)
			os.Exit(2)
		}
		target, err := pick(h, *title)
		if err != nil {
			fmt.Println("✗", err)
			os.Exit(2)
		}
		remember(target, false)
		if *borderless {
			st, _, _ := pGetWindowLongPtrW.Call(target, gwlStyle)
			style := uint32(st) &^ (wsCaption | wsThickFrame | wsSysMenu | wsMinimizeBox | wsMaximizeBox)
			pSetWindowLongPtrW.Call(target, gwlStyle, uintptr(style))
		}
		pSetWindowPos.Call(target, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(hh),
			swpNoZOrder|swpNoActivate|swpFrameChanged|swpShowWindow)
		fmt.Printf("✓ 已停靠 hwnd=%d → %d,%d %dx%d（borderless=%v）；还原：go run ./tools/wpsdock restore -hwnd %d\n", target, x, y, w, hh, *borderless, target)

	case "embed":
		if *parent == 0 {
			fmt.Println("✗ embed 需要 -parent（宿主窗口句柄）")
			os.Exit(2)
		}
		target, err := pick(h, *title)
		if err != nil {
			fmt.Println("✗", err)
			os.Exit(2)
		}
		remember(target, true)
		st, _, _ := pGetWindowLongPtrW.Call(target, gwlStyle)
		style := (uint32(st) &^ (wsPopup | wsCaption | wsThickFrame)) | wsChild
		pSetWindowLongPtrW.Call(target, gwlStyle, uintptr(style))
		pSetParent.Call(target, uintptr(*parent))
		if *rect != "" {
			if x, y, w, hh, err := parseRect(*rect); err == nil {
				pSetWindowPos.Call(target, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(hh),
					swpNoZOrder|swpNoActivate|swpFrameChanged|swpShowWindow)
			}
		}
		fmt.Printf("✓ 已 SetParent hwnd=%d → parent=%d（实验性）；还原：go run ./tools/wpsdock restore -hwnd %d\n", target, *parent, target)

	case "restore":
		list := loadState()
		if h != 0 {
			target := h
			for _, s := range list {
				if s.HWND == target {
					if s.Embed {
						pSetParent.Call(target, 0)
					}
					pSetWindowLongPtrW.Call(target, gwlStyle, uintptr(s.Style))
					pSetWindowPos.Call(target, 0, 0, 0, 0, 0, 0x0002|0x0001|swpFrameChanged|0x0040) // SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW
					fmt.Printf("✓ 已还原 hwnd=%d\n", target)
					return
				}
			}
			fmt.Println("✗ 状态文件里没有这个 hwnd")
			return
		}
		for _, s := range list {
			if s.Embed {
				pSetParent.Call(s.HWND, 0)
			}
			pSetWindowLongPtrW.Call(s.HWND, gwlStyle, uintptr(s.Style))
		}
		saveState(nil)
		fmt.Printf("✓ 已还原 %d 个窗口\n", len(list))

	default:
		fmt.Println("用法：wpsdock [list|dock|embed|restore] -h 查看参数")
	}
}

func argsToString(a []string) string { return strings.Join(a, " ") }
