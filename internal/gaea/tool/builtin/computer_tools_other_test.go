//go:build !windows

package builtin

// W3 验收 3（施工单）：非 Windows 平台三件注册后 Execute 回 ErrUnsupported
// 的信封错误（不 panic）。
import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestComputerToolsUnsupportedOnNonWindows(t *testing.T) {
	RegisterComputerTools(nil, 0)
	for _, name := range []string{"computer_windows", "computer_state", "computer_launch"} {
		tl := computerToolByName(name)
		if tl == nil {
			t.Fatalf("缺 %s", name)
		}
		out, _ := tl.Execute(context.Background(), json.RawMessage(`{}`))
		if !strings.Contains(out, "unsupported") {
			t.Fatalf("%s 应回 unsupported 信封: %.160s", name, out)
		}
	}
}
