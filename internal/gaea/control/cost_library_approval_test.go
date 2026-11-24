package control

// cost_library_approval_test.go — 造价库维护工具的确认卡口径（2026-11-25 造价
// 数据库推倒重建）：cost_library 进 hardAsk 名单；演练与只读动作不弹卡；确认
// 摘要给出动作、对象与规模。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/permission"
)

func TestCostLibraryIsHardAsk(t *testing.T) {
	if !hardAskTools["cost_library"] {
		t.Fatal("cost_library 必须在 hardAsk 名单：导入/删除/批量调价逐条经用户确认")
	}
}

func TestCostLibraryArgsNoWrite(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want bool
	}{
		{"健康扫描只读", map[string]any{"action": "health"}, true},
		{"飞轮未点名词干=只出预览", map[string]any{"action": "flywheel"}, true},
		{"飞轮点名了词干=写库", map[string]any{"action": "flywheel", "stems": []string{"WP01"}}, false},
		{"删除演练", map[string]any{"action": "delete_quota", "codes": []string{"A"}, "dry_run": true}, true},
		{"正式删除", map[string]any{"action": "delete_quota", "codes": []string{"A"}}, false},
		{"导入演练", map[string]any{"action": "import_project", "path": "a.xlsx", "dry_run": true}, true},
		{"正式导入", map[string]any{"action": "import_project", "path": "a.xlsx"}, false},
	}
	for _, tc := range cases {
		b, _ := json.Marshal(tc.args)
		if got := costLibraryArgsNoWrite(b); got != tc.want {
			t.Errorf("%s: costLibraryArgsNoWrite = %v, want %v", tc.name, got, tc.want)
		}
	}
	// 参数解析不了：按「会写」处理（fail-closed）。
	if costLibraryArgsNoWrite(json.RawMessage(`{broken`)) {
		t.Fatal("解析不了的参数不应被当成只读")
	}
}

func TestCostLibraryApprovalSubject(t *testing.T) {
	subject := func(args map[string]any) string {
		b, _ := json.Marshal(args)
		return costLibraryApprovalSubject(b)
	}
	if s := subject(map[string]any{"action": "import_project", "path": `D:\新工作\南岸\包3成本测算表.xlsx`}); !strings.Contains(s, "导入五表工作簿") || !strings.Contains(s, "包3成本测算表.xlsx") {
		t.Fatalf("导入摘要应含动作与文件名，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "delete_quota", "codes": []string{"A", "B"}, "force": true}); !strings.Contains(s, "删除定额") || !strings.Contains(s, "2 条") || !strings.Contains(s, "连带解除引用") {
		t.Fatalf("删除摘要应含动作、规模与强制标记，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "set_price", "resource_code": "C20", "price": 470.5}); !strings.Contains(s, "工料机调价") || !strings.Contains(s, "C20") || !strings.Contains(s, "¥470.50") {
		t.Fatalf("调价摘要应含编码与新价，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "save_resource", "resource": map[string]any{"title": "中砂", "current_price": 120}}); !strings.Contains(s, "保存工料机") || !strings.Contains(s, "中砂") {
		t.Fatalf("保存摘要应含名称，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "delete_project", "project_id": 7}); !strings.Contains(s, "删除项目") || !strings.Contains(s, "#7") {
		t.Fatalf("删项目摘要应含项目号，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "dedupe", "keep_id": 3, "merge_ids": []int64{5, 6}}); !strings.Contains(s, "合并 2 条") {
		t.Fatalf("去重摘要应含合并规模，得到 %q", s)
	}
	// 未知动作：委托通用摘要（与 approvalSubjectFor 的默认分支同值）。
	raw := json.RawMessage(`{"action":"something-new","path":"x"}`)
	if got, want := costLibraryApprovalSubject(raw), permission.Subject(raw); got != want {
		t.Fatalf("未知动作应回落通用摘要：got %q, want %q", got, want)
	}
}
