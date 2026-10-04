package strutil

import "testing"

// TestParseTagsJSONIdenticalToOriginal verifies ParseTagsJSON matches the
// local implementations it replaces (cost.parseTagsJSON and
// knowledge.parseTagsJSON — byte-identical bodies).
func TestParseTagsJSONIdenticalToOriginal(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   \t\n ", nil},
		{"[]", nil},
		{`["人工","材料"]`, []string{"人工", "材料"}},
		{`["a","b"] trailing garbage`, nil}, // 整体不合法 → nil（实测 stdlib 行为，语义冻结）
		{`not json`, nil},
		{`["unclosed`, nil},
		{`{"k":1}`, nil}, // 非数组 → Unmarshal 失败 → nil
	}
	for _, c := range cases {
		got := ParseTagsJSON(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("ParseTagsJSON(%q)=%#v, want %#v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("ParseTagsJSON(%q)[%d]=%q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
	if ParseTagsJSON(`["x"]`) == nil {
		t.Fatal("ParseTagsJSON 应返回非 nil 切片（合法输入）")
	}
}
