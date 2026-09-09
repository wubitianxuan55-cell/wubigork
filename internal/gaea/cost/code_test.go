package cost

// 条目匹配键测试（v4.178.0 定额编码贯通）：NormalizeCode 归一化矩阵 +
// Save/Get 往返 + 关键词按编码命中。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestNormalizeCode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"A1-12", "A1-12"},
		{" a1 12 ", "A112"},           // 去空白
		{"ａ１－１２", "A1-12"},            // 全角数字/字母/连字符 → 半角
		{"A－1－1２", "A-1-12"},          // 全角连字符与数字混合
		{"清单 040101001", "清单040101001"}, // 中文保留、空白去除
	}
	for _, c := range cases {
		if got := NormalizeCode(c.in); got != c.want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCodeRoundtripAndSearch(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	if err := s.Save(Entry{Name: "a112", Title: "人工挖沟槽土方", Code: " a1-12 ", Price: 42.5}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("a112")
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "A1-12" {
		t.Fatalf("Get code = %q, want A1-12（保存时归一化）", got.Code)
	}
	// 空编码不误报为命中。
	if err := s.Save(Entry{Name: "nocode", Title: "无编码条目", Price: 1}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get("nocode"); got.Code != "" {
		t.Fatalf("空编码应保持空串, got %q", got.Code)
	}
	// 关键词按编码命中（haystack 含 code）。
	hits := s.Search("a1-12", "", "")
	if len(hits) != 1 || hits[0].Name != "a112" {
		t.Fatalf("Search(编码) 命中 = %v, want [a112]", names(hits))
	}
}

func names(list []Summary) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name
	}
	return out
}
