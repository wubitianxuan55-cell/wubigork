package skill

import (
	"strings"
	"testing"
)

// T14（codex 蒸馏）验收：确定性 / 平方加成 / 8KiB 查询不报错 / 中文 n-gram 生效。

func mkSkills() []Skill {
	return []Skill{
		{Name: "excel-export", Description: "导出 excel 表格", Body: "把数据导出为 excel 文件，支持多工作表。"},
		{Name: "word-report", Description: "生成 word 报告", Body: "生成 word 文档报告，支持样式。"},
		{Name: "表格导出", Description: "把表格数据导出为 csv", Body: "面向中文用户的表格导出技能。"},
		{Name: "schedule-apply", Description: "应用进度计划", Body: "把进度计划写回文件。"},
	}
}

// ① 同一查询 100 次结果逐元素相同（确定性）。
func TestSelectDeterministic(t *testing.T) {
	skills := mkSkills()
	first := Select(skills, "导出 excel 表格", 3)
	for i := 0; i < 100; i++ {
		again := Select(skills, "导出 excel 表格", 3)
		if len(again.IDs) != len(first.IDs) {
			t.Fatalf("长度漂移 @%d", i)
		}
		for j := range again.IDs {
			if again.IDs[j] != first.IDs[j] {
				t.Fatalf("第 %d 次结果漂移 @%d: %v vs %v", i, j, again.IDs, first.IDs)
			}
		}
	}
	if len(first.IDs) == 0 || first.IDs[0] != 0 {
		t.Fatalf("excel-export 应排第一: %v", first.IDs)
	}
}

// ② 覆盖 3 个不同词的文档排在只覆盖 1 词的前面（平方加成生效）。
func TestSelectMultiTermCoverage(t *testing.T) {
	skills := []Skill{
		{Name: "a-doc", Description: "别的", Body: "无相关内容 placeholder"},
		{Name: "b-doc", Description: "只有 excel 一件事 excel excel excel excel excel"},
		{Name: "c-doc", Description: "导出 excel 表格 全覆盖"},
	}
	got := Select(skills, "导出 excel 表格", 3)
	if len(got.IDs) < 2 || got.IDs[0] != 2 {
		t.Fatalf("覆盖 3 词应排前: %v", got.IDs)
	}
}

// ③ 8KiB 查询不报错且 query_truncated=true。
func TestSelectHugeQueryTruncates(t *testing.T) {
	skills := mkSkills()
	big := strings.Repeat("导出表格 ", 1400) // ~8.4KiB
	got := Select(skills, big, 3)
	if !got.QueryTruncated {
		t.Fatalf("8KiB 查询应上报截断")
	}
	if len(got.IDs) == 0 {
		t.Fatalf("截断后仍应有结果")
	}
}

// ⑤ 中文技能名在中文查询下进 top-3（字符 n-gram 路径生效）。
func TestSelectChineseName(t *testing.T) {
	skills := mkSkills()
	got := Select(skills, "表格导出", 3)
	found := false
	for _, id := range got.IDs {
		if skills[id].Name == "表格导出" {
			found = true
		}
	}
	if !found {
		t.Fatalf("中文技能名应进 top-3: %v", got.IDs)
	}
	// top-1 直取（名与查询完全互为逆序，ngram 共享最高）。
	if got.IDs[0] != 2 {
		t.Fatalf("表格导出应排第一: %v", got.IDs)
	}
}

// Fuse 三级 tie-break：同分取最好名次，同名次取小 id。
func TestFuseTieBreak(t *testing.T) {
	// id1 与 id2 同分（名次和相同），但 id1 最好名次更靠前 → id1 胜。
	got := Fuse([][]int{{1, 2}, {2, 1}}, 0)
	if len(got.IDs) != 2 || got.IDs[0] != 1 {
		t.Fatalf("tie-break 应取最好名次: %v", got.IDs)
	}
	// 同分同最好名次（对称双列表）→ id 升序。
	got = Fuse([][]int{{9, 8}, {8, 9}}, 0)
	if got.IDs[0] != 8 {
		t.Fatalf("同分同名次应取小 id: %v", got.IDs)
	}
	// limit 截断。
	got = Fuse([][]int{{1, 2, 3}}, 2)
	if len(got.IDs) != 2 {
		t.Fatalf("limit 截断失效: %v", got.IDs)
	}
}

// 空查询不猜。
func TestSelectEmptyQuery(t *testing.T) {
	if got := Select(mkSkills(), "  ", 3); len(got.IDs) != 0 {
		t.Fatalf("空查询应返回空: %v", got.IDs)
	}
}
