package schedule

import "testing"

// opTypesDocumented Op.Type 文档注释枚举的 12 种操作类型（ops.go Op 结构体
// 头注同源）。批 23 IN3-04 表驱动化后，applyOne 的分派正确性完全取决于
// opHandlers 注册表——键拼错/漏注册会静默落入「不支持的操作类型」分支，
// 编译期抓不到，只能靠本守卫把「注册表键集 = 文档枚举」双向锁死。
var opTypesDocumented = []string{
	"upsert_task",
	"patch_task",
	"remove_task",
	"set_links",
	"set_meta",
	"auto_chain",
	"set_baseline",
	"clear_baseline",
	"upsert_resource",
	"patch_resource",
	"remove_resource",
	"set_assignments",
}

// TestOpHandlersComplete 注册表完备性双向锁：文档枚举的每种 op 都有非 nil
// 处理函数（漏注册=新增 op 永远拒）；注册表也不得出现文档外的键（拼错键
// =对应 op 全部拒）。
func TestOpHandlersComplete(t *testing.T) {
	if len(opHandlers) != len(opTypesDocumented) {
		t.Fatalf("注册表键数 %d ≠ 文档枚举 %d（多出的键=%v）",
			len(opHandlers), len(opTypesDocumented), extraKeys())
	}
	for _, typ := range opTypesDocumented {
		h, ok := opHandlers[typ]
		if !ok {
			t.Errorf("文档枚举的 op %q 未注册处理函数", typ)
			continue
		}
		if h == nil {
			t.Errorf("op %q 注册了 nil 处理函数", typ)
		}
	}
}

func extraKeys() []string {
	documented := make(map[string]bool, len(opTypesDocumented))
	for _, k := range opTypesDocumented {
		documented[k] = true
	}
	var out []string
	for k := range opHandlers {
		if !documented[k] {
			out = append(out, k)
		}
	}
	return out
}

// TestApplyOneUnknownTypeFailsClosed 未知类型 fail-closed（拆分前 default
// 分支同文案，ops_golden fixture「未知操作类型」用例同口径的本地图钉）。
func TestApplyOneUnknownTypeFailsClosed(t *testing.T) {
	p := Project{}
	if _, err := applyOne(&p, Op{Type: "nope"}); err == nil || err.Error() != "不支持的操作类型：nope" {
		t.Fatalf("未知类型应拒绝且文案逐字一致，got=%v", err)
	}
}
