package anchor

import (
	"strings"
	"testing"
)

// A1 锚点语言（语法层 v1）：解析/格式化/校验/往返，纯域函数无 I/O。
// 移植口径=zcode anchor.ts；docx 无持久段 ID 的结论见包注释。
func TestParseRoundTrip(t *testing.T) {
	cases := []struct {
		kind DocKind
		in   string
		want string // 规范形（roundtrip 期望）
	}{
		{KindDocx, "p:14", "p:14"},
		{KindDocx, "p:3-9", "p:3-9"},
		{KindDocx, "t:2", "t:2"},
		{KindDocx, "t:2/r3c1", "t:2/r3c1"},
		{KindPptx, "slide:3", "slide:3"},
		{KindPptx, "slide:2-4", "slide:2-4"},
		{KindPptx, "slide:3/shape:7", "slide:3/shape:7"},
		{KindPdf, "page:5", "page:5"},
		{KindPdf, "page:2-4", "page:2-4"},
		{KindXlsx, "Revenue", "Revenue"},
		{KindXlsx, "Revenue!B2", "Revenue!B2"},
		{KindXlsx, "Revenue!B2:F9", "Revenue!B2:F9"},
		{KindXlsx, "'Q3 data'!A1", "'Q3 data'!A1"},
		{KindXlsx, "Q3 data!A1", "'Q3 data'!A1"}, // 带空格表名规范形加引号
		{KindDocx, " p:12 ", "p:12"},             // 首尾空白宽容
	}
	for _, tc := range cases {
		t.Run(string(tc.kind)+":"+tc.in, func(t *testing.T) {
			loc, err := Parse(tc.kind, tc.in)
			if err != nil {
				t.Fatalf("Parse(%s, %q) 报错: %v", tc.kind, tc.in, err)
			}
			if got := loc.Format(); got != tc.want {
				t.Fatalf("Format=%q, want %q", got, tc.want)
			}
			// 往返恒等：规范形再解析再格式化不变。
			loc2, err := Parse(tc.kind, tc.want)
			if err != nil {
				t.Fatalf("规范形重解析报错: %v", err)
			}
			if loc2.Format() != tc.want {
				t.Fatalf("往返不恒等: %q", loc2.Format())
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	cases := []struct {
		kind DocKind
		in   string
	}{
		{KindDocx, "p:0"},      // 0-based 借用
		{KindDocx, "p:9-3"},    // 区间倒置
		{KindDocx, "x:1"},      // 未知前缀
		{KindDocx, "t:1/r0c2"}, // 行号 0
		{KindPptx, "slide:0"},
		{KindPdf, "page:"},
		{KindXlsx, ""},         // 空表名
		{KindXlsx, "Sheet!"},   // !后无单元格
		{KindXlsx, "Sheet!ZZ"}, // 非单元格
		{KindXlsx, "Sheet!A0"}, // 行 0
	}
	for _, tc := range cases {
		t.Run(string(tc.kind)+":"+tc.in, func(t *testing.T) {
			if _, err := Parse(tc.kind, tc.in); err == nil {
				t.Fatalf("Parse(%s, %q) 应报错", tc.kind, tc.in)
			}
		})
	}
}

func TestColumnName(t *testing.T) {
	cases := []struct {
		col  int
		want string
	}{
		{1, "A"}, {26, "Z"}, {27, "AA"}, {52, "AZ"}, {53, "BA"}, {16384, "XFD"},
	}
	for _, tc := range cases {
		if got := ColumnName(tc.col); got != tc.want {
			t.Errorf("ColumnName(%d)=%q, want %q", tc.col, got, tc.want)
		}
		if c := ParseCell(tc.want + "1"); c == nil || c.Col != tc.col {
			t.Errorf("ParseCell(%q) 列往返失败: %+v", tc.want+"1", c)
		}
	}
}

// Rev1 revision 门原语：同内容同 revision，改一字即变。
func TestRev1(t *testing.T) {
	a := Rev1([]byte("document bytes"))
	if len(a) != 16 || strings.ToLower(a) != a {
		t.Fatalf("revision 形态应为 16 位小写 hex: %q", a)
	}
	if Rev1([]byte("document bytes")) != a {
		t.Fatal("同内容 revision 必须稳定")
	}
	if Rev1([]byte("document bytes!")) == a {
		t.Fatal("内容变 revision 必须变")
	}
}

// Span 各格式区间语义。
func TestSpan(t *testing.T) {
	loc, err := Parse(KindDocx, "p:3-9")
	if err != nil {
		t.Fatal(err)
	}
	if f, to := loc.Span(); f != 3 || to != 9 {
		t.Fatalf("Span=%d,%d", f, to)
	}
}
