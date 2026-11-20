//go:build windows

package computer

// App 启动（施工单 W3）：ShellExecuteW open 拉起 GUI（无控制台残留），
// 短轮询等窗口出现在枚举里作为启动确认。
import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

// LaunchApp 启动 app（exe 全路径或 process:<exe>）并等待其顶层窗口出现
// （最多 ~4s）。返回窗口快照便于模型立刻 computer_state。
func LaunchApp(app string) (Window, error) {
	path := trimProcessPrefix(app)
	if path == "" {
		return Window{}, fmt.Errorf("computer: app 不能为空")
	}
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return Window{}, err
	}
	f, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Window{}, err
	}
	hr, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(f)), 0, 0, 5 /*SW_SHOW*/)
	if hr <= 32 {
		return Window{}, fmt.Errorf("computer: 启动失败（ShellExecute hr=%d，路径或关联程序不存在？）: %s", hr, path)
	}
	want := "process:" + path
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(120 * time.Millisecond)
		wins, err := ListWindows()
		if err != nil {
			continue
		}
		for _, w := range wins {
			if w.App == want {
				return w, nil
			}
		}
	}
	return Window{}, fmt.Errorf("computer: 已发出启动但 %d 秒内窗口未出现（可能被授权窗/启动器拦住）: %s", 4, path)
}

// trimProcessPrefix 归一 app 标识为裸 exe 路径。
func trimProcessPrefix(app string) string {
	p := app
	if len(p) > 8 && p[:8] == "process:" {
		p = p[8:]
	}
	return p
}
