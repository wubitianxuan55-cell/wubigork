package dag

// 计划期写路径闸测试（2026-11-08 众神殿裁决刀2）：重叠判定口径（相等/目录
// 前缀/Windows 大小写/边界不对齐不重叠）、分波上下游豁免、缺声明降宽判定。

import (
	"os"
	"testing"
)

func TestWritePathsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"docs/a.xlsx", "docs/a.xlsx", true},
		{"docs", "docs/a.xlsx", true},                                       // 目录含其下全部
		{"docs/a.xlsx", "DOCS/B.XLSX", false},                               // 大小写归一后不重叠
		{"DOCS", "docs/a.xlsx", true},                                       // Windows 大小写不敏感
		{"docs/a.txt", "docs/a.txt.bak", false},                             // 非目录边界不对齐
		{"docs" + string(os.PathSeparator) + "a.xlsx", "docs/a.xlsx", true}, // 反斜杠归一
		{"./docs/a.xlsx", "docs/a.xlsx", true},                              // ./ 前缀归一
		{"docs/a.xlsx", "reports/a.xlsx", false},
	}
	for _, tc := range cases {
		if got := WritePathsOverlap(tc.a, tc.b); got != tc.want {
			t.Errorf("WritePathsOverlap(%q,%q)=%v want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsLiteralWritePath(t *testing.T) {
	for _, p := range []string{"docs/a.xlsx", "out/", "OUT"} {
		if !IsLiteralWritePath(p) {
			t.Errorf("%q 应判字面可判定", p)
		}
	}
	for _, p := range []string{"", "  ", "docs/*.xlsx", "../escape", "a?b"} {
		if IsLiteralWritePath(p) {
			t.Errorf("%q 应判无法判定", p)
		}
	}
}

func TestWaveWriteConflicts(t *testing.T) {
	// 同波重叠 → 冲突点名。
	sameWave := []Node{
		{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"docs/out.docx"}},
		{ID: "b", Title: "t", Prompt: "p", WritePaths: []string{"docs"}},
	}
	if conflicts, err := WaveWriteConflicts(sameWave); err != nil || len(conflicts) != 1 {
		t.Fatalf("同波重叠应报 1 条冲突: %v %v", conflicts, err)
	}
	// 上下游（依赖分波）→ 天然异波豁免。
	chain := []Node{
		{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"docs/mid.xlsx"}},
		{ID: "b", Title: "t", Prompt: "p", DependsOn: []string{"a"}, WritePaths: []string{"docs/mid.xlsx"}},
	}
	if conflicts, err := WaveWriteConflicts(chain); err != nil || len(conflicts) != 0 {
		t.Fatalf("上下游同路径应豁免: %v %v", conflicts, err)
	}
	// 同波不重叠 + glob 不比对。
	disjoint := []Node{
		{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"docs/a.docx"}},
		{ID: "b", Title: "t", Prompt: "p", WritePaths: []string{"docs/b.docx", "charts/*.png"}},
	}
	if conflicts, err := WaveWriteConflicts(disjoint); err != nil || len(conflicts) != 0 {
		t.Fatalf("不重叠+glob 应零冲突: %v %v", conflicts, err)
	}
}

func TestWaveFullyDeclared(t *testing.T) {
	full := []Node{
		{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"a.docx"}},
		{ID: "b", Title: "t", Prompt: "p", WritePaths: []string{"b.docx"}},
	}
	if !WaveFullyDeclared(full) {
		t.Error("全声明波应判真")
	}
	partial := []Node{{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"a.docx"}}, {ID: "b", Title: "t", Prompt: "p"}}
	if WaveFullyDeclared(partial) {
		t.Error("缺声明节点应令波判假（降宽串行）")
	}
	glob := []Node{{ID: "a", Title: "t", Prompt: "p", WritePaths: []string{"*.docx"}}}
	if WaveFullyDeclared(glob) {
		t.Error("glob 属无法判断应判假（降宽串行）")
	}
	if WaveFullyDeclared(nil) {
		t.Error("空波判假")
	}
}
