//go:build windows

package computer

// 真机门控测试（W2 验收 3 的可回归形态）：GAEA_LIVE_COMPUTER_TEST 置值才跑。
// 只读观察（枚举窗口 + UIA 取树 + 全屏截图落盘），不注入任何输入。
//   set GAEA_LIVE_COMPUTER_TEST=1

import (
	"os"
	"strings"
	"testing"
)

func TestListWindowsLive(t *testing.T) {
	if os.Getenv("GAEA_LIVE_COMPUTER_TEST") == "" {
		t.Skip("set GAEA_LIVE_COMPUTER_TEST=1 to run")
	}
	wins, err := ListWindows()
	if err != nil {
		t.Fatalf("ListWindows: %v", err)
	}
	if len(wins) == 0 {
		t.Fatal("可见顶层窗口数为 0（不可能）")
	}
	for _, w := range wins {
		t.Logf("id=%d app=%s title=%.60s rect=(%d,%d %dx%d)", w.ID, w.App, w.Title, w.X, w.Y, w.W, w.H)
	}
}

func TestObserveTreeLive(t *testing.T) {
	if os.Getenv("GAEA_LIVE_COMPUTER_TEST") == "" {
		t.Skip("set GAEA_LIVE_COMPUTER_TEST=1 to run")
	}
	wins, err := ListWindows()
	if err != nil {
		t.Fatalf("ListWindows: %v", err)
	}
	// 选一扇像样的窗：非最小化（-32000 是最小化位）、标题≥5、非输入法悬浮窗。
	var pick *Window
	for i := range wins {
		w := wins[i]
		if w.X <= -30000 || w.Y <= -30000 || len(w.Title) < 5 {
			continue
		}
		if strings.Contains(w.Title, "imestatuspop") {
			continue
		}
		pick = &wins[i]
		break
	}
	if pick == nil {
		t.Fatal("无像样窗口可观察")
	}
	snap, err := Default().Observe(pick.ID, ObserveOpt{IncludeText: true, IncludeScreenshot: true})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if snap.Text == "" {
		t.Fatal("树文本为空")
	}
	if !strings.HasPrefix(snap.Text, "Window: \"") {
		t.Fatalf("树首行文法错: %.60s", snap.Text)
	}
	lines := strings.Count(snap.Text, "\n")
	t.Logf("窗口=%q 树=%d 行 %d 字节 shot=%v", pick.Title, lines, len(snap.Text), snap.Shot != nil)
	t.Logf("文本头 400 字节 %q", snap.Text[:min(400, len(snap.Text))])
	t.Logf("树前 20 行:\n%s", strings.Join(strings.SplitN(snap.Text, "\n", 21)[:min(20, lines+1)], "\n"))
	if lines < 3 {
		t.Fatal("树行数过少（UIA 遍历疑似未展开）")
	}
}
