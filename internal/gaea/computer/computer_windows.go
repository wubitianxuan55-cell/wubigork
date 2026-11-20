//go:build windows

package computer

// Windows 平台观察入口（施工单 W2）：窗口有效性 → UIA 树文本 →
// 可选全屏截图。截图失败不拖垮观察（文本仍可用；shotID 缺席时坐标
// 动作被 CheckScreenshotID 天然拦下，语义自洽）。
func init() { platformObserve = observeWindows }

func observeWindows(windowID int, opt ObserveOpt) (*Snapshot, error) {
	if !windowIsAlive(windowID) {
		return nil, ErrWindowGone
	}
	title, app := windowIdentity(windowID)
	snap := &Snapshot{Window: Window{ID: windowID, Title: title, App: app}}
	if opt.IncludeText {
		text, err := uiaTreeText(uintptr(windowID), title, app)
		if err != nil {
			return nil, err
		}
		snap.Text = text
	}
	if opt.IncludeScreenshot {
		if shot, err := captureFullScreen(); err == nil {
			snap.Shot = shot
		}
	}
	// 填充窗口几何（观察结果自带 frame，模型做坐标换算用）。
	snap.Window.X, snap.Window.Y, snap.Window.W, snap.Window.H = windowRect(windowID)
	return snap, nil
}
