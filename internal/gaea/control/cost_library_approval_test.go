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
		// 库定规矩新增动作（2026-11-25）：盘点与查看字典只读；整理归类/借鉴生成/字典增改演练不弹卡。
		{"维度盘点只读", map[string]any{"action": "dims"}, true},
		{"查看字典只读", map[string]any{"action": "dict"}, true},
		{"查看字典只读（显式 list）", map[string]any{"action": "dict", "dict": map[string]any{"op": "list", "kind": "region"}}, true},
		{"新增字典项=写库", map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "region", "name": "太原市"}}, false},
		{"字典演练", map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "region", "name": "太原市"}, "dry_run": true}, true},
		{"整理归类演练", map[string]any{"action": "classify", "classify": map[string]any{"bills": []any{map[string]any{"id": 1, "specialty": "市政"}}}, "dry_run": true}, true},
		{"整理归类正式", map[string]any{"action": "classify", "classify": map[string]any{"bills": []any{map[string]any{"id": 1, "specialty": "市政"}}}}, false},
		{"借鉴生成演练", map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政", "dry_run": true}, true},
		{"借鉴生成正式", map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政"}, false},
		// 模型读表导入：暂存只在内存里，开导入 / 入暂存 / 勾稽 / 丢弃都不写库。
		{"开导入不写库", map[string]any{"action": "import_open", "source": "a.xlsx"}, true},
		{"入暂存不写库", map[string]any{"action": "import_add", "import_id": "imp-1", "resources": []any{}}, true},
		{"导入勾稽只读", map[string]any{"action": "import_check", "import_id": "imp-1"}, true},
		{"丢弃导入不写库", map[string]any{"action": "import_discard", "import_id": "imp-1"}, true},
		{"提交导入演练", map[string]any{"action": "import_commit", "import_id": "imp-1", "dry_run": true}, true},
		{"提交导入正式=写库", map[string]any{"action": "import_commit", "import_id": "imp-1"}, false},
		// 生成测算模版（六表，2026-11-25）：只读库、只在工作区新增一个文件，不写库。
		{"生成测算模版不写库", map[string]any{"action": "template", "template": map[string]any{"specialty": "市政", "region": "重庆中心城区"}}, true},
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
	// 入库红线（2026-11-27）：导入测算工作簿的卡上要提醒用户「确认已检查过这份表」。
	if s := subject(map[string]any{"action": "import_project", "path": `D:\新工作\南岸\包3成本测算表.xlsx`}); !strings.Contains(s, "导入测算工作簿（请确认你已检查过这份测算表）") || !strings.Contains(s, "包3成本测算表.xlsx") {
		t.Fatalf("导入摘要应含动作、检查提醒与文件名，得到 %q", s)
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
	// 库定规矩新增动作：摘要要让人一眼看到归类规模、借鉴的是哪条到哪个专业、动的是哪条字典项。
	if s := subject(map[string]any{"action": "classify", "classify": map[string]any{
		"projects": []any{map[string]any{"id": 1}}, "bills": []any{map[string]any{"id": 2}, map[string]any{"id": 3}},
		"quotas": []any{map[string]any{"code": "A"}}, "resources": []any{map[string]any{"code": "L01"}},
	}}); !strings.Contains(s, "整理归类") || !strings.Contains(s, "项目 1 · 清单 2 · 定额 1 · 工料机 1") {
		t.Fatalf("整理归类摘要应含四组规模，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政", "item_ids": []int64{5, 6}}); !strings.Contains(s, "借鉴生成定额") || !strings.Contains(s, "· A ·") || !strings.Contains(s, "→ 市政专业") || !strings.Contains(s, "改挂 2 行清单") {
		t.Fatalf("借鉴生成摘要应含源定额、目标专业与改挂规模，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "specialty", "name": "市政"}}); !strings.Contains(s, "新增字典项") || !strings.Contains(s, "专业「市政」") {
		t.Fatalf("新增字典摘要应含种类与名称，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "dict", "dict": map[string]any{"op": "update", "kind": "region", "name": "四川", "rename": "四川省"}}); !strings.Contains(s, "修改字典项") || !strings.Contains(s, "地区「四川」") || !strings.Contains(s, "改名为「四川省」") {
		t.Fatalf("改名字典摘要应含原名与新名，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "dict", "dict": map[string]any{"op": "delete", "kind": "category", "name": "措施费"}}); !strings.Contains(s, "删除字典项") || !strings.Contains(s, "工料机分类「措施费」") {
		t.Fatalf("删除字典摘要应含种类与名称，得到 %q", s)
	}
	// 提交导入：暂存内容在 app 层的会话里，经登记的取数口写进卡片；取不到时只显会话号。
	commit := map[string]any{"action": "import_commit", "import_id": "imp-7"}
	if s := subject(commit); !strings.Contains(s, "提交导入") || !strings.Contains(s, "会话 imp-7") {
		t.Fatalf("取不到会话时提交导入摘要应显会话号，得到 %q", s)
	}
	SetCostImportSummaryHook(func(id string) (string, int, int, int, bool) {
		if id != "imp-7" {
			return "", 0, 0, 0, false
		}
		return "某某地块修复", 3, 1, 120, true
	})
	s := subject(commit)
	SetCostImportSummaryHook(nil)
	if !strings.Contains(s, "项目「某某地块修复」") || !strings.Contains(s, "工料机 3 · 定额 1 · 清单 120") {
		t.Fatalf("提交导入摘要应写出项目与三类行数，得到 %q", s)
	}
	if s := subject(map[string]any{"action": "import_commit", "import_id": "imp-8"}); !strings.Contains(s, "会话 imp-8") {
		t.Fatalf("登记的取数口取不到的会话仍应显会话号，得到 %q", s)
	}
	// 未知动作：委托通用摘要（与 approvalSubjectFor 的默认分支同值）。
	raw := json.RawMessage(`{"action":"something-new","path":"x"}`)
	if got, want := costLibraryApprovalSubject(raw), permission.Subject(raw); got != want {
		t.Fatalf("未知动作应回落通用摘要：got %q, want %q", got, want)
	}
}
