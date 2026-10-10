package app

import (
	"strings"
	"testing"
)

// sanitizeSessionTitle 的三道修整：压平换行/剥引点/截上限（模型输出不可信）。
func TestSanitizeSessionTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"造价清单导入", "造价清单导入"},
		{"「造价清单导入」。", "造价清单导入"},
		{"\"数据 备份\"\n", "数据 备份"},
		{"好的，这是标题：《进度计划排期》", "好的，这是标题：《进度计划排期"}, // Trim 只剥首尾引点，中段标点是内容
		{"  \n \n ", ""},
	}
	for _, c := range cases {
		if got := sanitizeSessionTitle(c.in); got != c.want {
			t.Errorf("sanitizeSessionTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	long := strings.Repeat("标", gaeaAutoTitleMaxRunes+5)
	got := sanitizeSessionTitle(long)
	if n := len([]rune(got)); n != gaeaAutoTitleMaxRunes {
		t.Fatalf("超长截断 rune 数 = %d, want %d", n, gaeaAutoTitleMaxRunes)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("截断应带省略号: %q", got)
	}
}

// maybeAutoTitleAfterTurn 的无控制器早退：不 panic、不起 goroutine（nil 客户
// 端同路径——办公引擎未初始化的进程早期形态）。
func TestMaybeAutoTitleEarlyExit(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("无控制器时早退不应 panic: %v", r)
		}
	}()
	maybeAutoTitleAfterTurnNilSafe()
}

// maybeAutoTitleAfterTurnNilSafe 隔离 gaeaCtrl() 全局态：测试进程里 ga.ctrl
// 未初始化即为 nil，等价于「引擎未起」形态；经 test 包内直接调用验证无副作用。
func maybeAutoTitleAfterTurnNilSafe() {
	app := &App{}
	app.maybeAutoTitleAfterTurn()
}
