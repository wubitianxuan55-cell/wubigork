package genui

import (
	"strings"
	"testing"
)

// 最小可用示例是单源（examples.go）：手册里的字段表必须逐字收录每一条——改了示例
// 没同步手册，这里拦下；校验器报缺字段时附的也是同一条。
func TestHandbookExamplesInSync(t *testing.T) {
	for typ, ex := range minimalExample {
		if !strings.Contains(Handbook, ex) {
			t.Errorf("手册未逐字收录 %s 的最小示例：%s", typ, ex)
		}
		if !strings.Contains(ex, `"type":"`+typ+`"`) {
			t.Errorf("%s 的示例 type 不符：%s", typ, ex)
		}
		// 示例本身必须是合法规格里的一个节点
		if r := ValidateSpec(`{"items":[` + ex + `]}`); !r.OK {
			t.Errorf("%s 的最小示例未通过校验：%v", typ, r.Errors)
		}
	}
	if MinimalExample("no-such-type") != "" {
		t.Error("未登记的类型应返回空示例")
	}
	// 必填表里出现的每个类型都得有示例（新增组件漏登记在这里暴露）
	for typ := range requiredFields {
		if _, ok := minimalExample[typ]; !ok {
			t.Errorf("必填表里的 %s 没有登记最小示例", typ)
		}
	}
}
