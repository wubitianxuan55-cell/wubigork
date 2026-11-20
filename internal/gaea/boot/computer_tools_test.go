package boot

// W3 默认关验收（施工单）：默认装配（Enabled=false）下 Registry.Schemas()
// 无 computer_*；本测试二进制内无任何 RegisterComputerTools 调用点，
// 断言稳定（注册侧的正例在 tool/builtin 包测试）。

import (
	"io"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/netclient"
)

func TestDefaultOffNoComputerTools(t *testing.T) {
	reg := tool.NewRegistry()
	// enabled 空=全量装配（现状语义）；只要三件未走 RegisterComputerTools，
	// Schemas() 就不该出现它们。
	addBuiltins(reg, t.TempDir(), nil, nil, sandbox.Spec{}, netclient.ProxySpec{}, io.Discard, "", nil)
	for _, s := range reg.Schemas() {
		if strings.HasPrefix(s.Name, "computer_") {
			t.Fatalf("默认关不得注册 computer_*: %s", s.Name)
		}
	}
}
