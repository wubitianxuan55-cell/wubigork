package rewrite

import (
	"strings"
	"testing"
)

// TestResolveSelection 选段校验与 ±50 模糊重锚全分支（MuMu L4959-L4986 rune 版）。
func TestResolveSelection(t *testing.T) {
	// 67 rune：30 风 + 7 字选段（rune 30..37）+ 30 雨
	content := strings.Repeat("风", 30) + "月落乌啼霜满天" + strings.Repeat("雨", 30)
	sel := "月落乌啼霜满天"
	// 长文验证 ±50 窗口边界：选段在 rune 10..17，其余 130 字填充
	long := strings.Repeat("甲", 10) + "目标段落在这里" + strings.Repeat("乙", 120)
	longSel := "目标段落在这里"
	// 多字节前缀验证重锚「字节偏移 → rune 偏移」换算：αβγδε 共 10 字节 5 rune，
	// 选段在 rune 5..9（字节口径会错锚到 10..14 即 ζηθικ）
	greek := "αβγδε目标段落ζηθικ"
	greekSel := "目标段落"

	cases := []struct {
		name      string
		content   string
		start     int
		end       int
		selected  string
		wantStart int
		wantEnd   int
		wantErr   string
	}{
		{"精确命中", content, 30, 37, sel, 30, 37, ""},
		{"selected 空跳过重锚", content, 0, 7, "", 0, 7, ""},
		{"起始越界", content, 67, 70, sel, 0, 0, "起始位置超出内容范围"},
		{"起始负数", content, -1, 5, sel, 0, 0, "起始位置超出内容范围"},
		{"空内容", "", 0, 1, "", 0, 0, "起始位置超出内容范围"},
		{"结束越界", content, 0, 68, sel, 0, 0, "结束位置超出内容范围"},
		{"起始等于结束", content, 5, 5, sel, 0, 0, "起始位置必须小于结束位置"},
		{"起始大于结束", content, 10, 3, sel, 0, 0, "起始位置必须小于结束位置"},
		{"窗口重锚命中", content, 33, 40, sel, 30, 37, ""},
		{"长文窗口内重锚", long, 20, 27, longSel, 10, 17, ""},
		{"长文窗口外不命中", long, 100, 107, longSel, 0, 0, "选中的文本与章节内容不匹配，请刷新后重试"},
		{"文本不存在", content, 0, 5, "根本不存在的句子", 0, 0, "选中的文本与章节内容不匹配，请刷新后重试"},
		{"多字节重锚 rune 换算", greek, 7, 11, greekSel, 5, 9, ""},
	}
	for _, tc := range cases {
		gotStart, gotEnd, err := ResolveSelection(tc.content, tc.start, tc.end, tc.selected)
		if tc.wantErr != "" {
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("%s: want err %q got %v", tc.name, tc.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: 不应报错: %v", tc.name, err)
		}
		if gotStart != tc.wantStart || gotEnd != tc.wantEnd {
			t.Fatalf("%s: got (%d,%d) want (%d,%d)",
				tc.name, gotStart, gotEnd, tc.wantStart, tc.wantEnd)
		}
	}
}

// TestPartialLengthSpec 四档区间数学与提示文案（MuMu L5054-L5071）+ 未知回 similar。
func TestPartialLengthSpec(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		selRunes int
		target   int
		wantMin  int
		wantMax  int
		wantHint string
	}{
		{"similar", LengthModeSimilar, 100, 0, 80, 120,
			"保持与原文相近的字数（约 100 字，允许 80-120 字浮动）"},
		{"expand", LengthModeExpand, 100, 0, 120, 200, "适当扩展内容（目标 120-200 字）"},
		{"condense", LengthModeCondense, 100, 0, 50, 80, "精简压缩内容（目标 50-80 字）"},
		{"custom", LengthModeCustom, 100, 1000, 800, 1200, "目标字数：约 1000 字（允许±20%浮动）"},
		{"custom 下钳 50", LengthModeCustom, 100, 10, 40, 60, "目标字数：约 50 字（允许±20%浮动）"},
		{"custom 上钳 10000", LengthModeCustom, 100, 99999, 8000, 12000, "目标字数：约 10000 字（允许±20%浮动）"},
		{"未知 mode 回 similar", "bogus", 100, 0, 80, 120,
			"保持与原文相近的字数（约 100 字，允许 80-120 字浮动）"},
		{"空 mode 回 similar", "", 250, 0, 200, 300,
			"保持与原文相近的字数（约 250 字，允许 200-300 字浮动）"},
	}
	for _, tc := range cases {
		got := PartialLengthSpec(tc.mode, tc.selRunes, tc.target)
		if got.MinRunes != tc.wantMin || got.MaxRunes != tc.wantMax || got.Hint != tc.wantHint {
			t.Fatalf("%s: got %+v want min=%d max=%d hint=%q",
				tc.name, got, tc.wantMin, tc.wantMax, tc.wantHint)
		}
	}
}

// TestPartialMaxTokens max(500, min(MaxRunes*3, 8000))：下限/中段/上限/零值。
func TestPartialMaxTokens(t *testing.T) {
	cases := []struct {
		name string
		spec PartialSpec
		want int
	}{
		{"下限 500（×3 不足）", PartialSpec{MinRunes: 80, MaxRunes: 120}, 500},
		{"中段 ×3 直取", PartialSpec{MinRunes: 800, MaxRunes: 1200}, 3600},
		{"上限 8000", PartialSpec{MinRunes: 8000, MaxRunes: 12000}, 8000},
		{"零值兜底 500", PartialSpec{}, 500},
	}
	for _, tc := range cases {
		if got := PartialMaxTokens(tc.spec); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

// TestBuildPartialInstruction 四段固定顺序、空上下文占位、指令 trim 与 1000 rune 截断。
func TestBuildPartialInstruction(t *testing.T) {
	spec := PartialSpec{MinRunes: 80, MaxRunes: 120, Hint: "保持与原文相近的字数（约 100 字，允许 80-120 字浮动）"}
	out := BuildPartialInstruction("压缩对白节奏", "选段正文内容", "前文参考片段", "后文参考片段", spec)

	headers := []string{
		"# 局部重写指令",
		"## 上下文（仅参考，不得改动）",
		"## 选中文本（只重写以下【选段】，选段外一字不动）",
		"## 用户指令",
		"## 长度模式提示",
	}
	last := -1
	for _, h := range headers {
		i := strings.Index(out, h)
		if i < 0 {
			t.Fatalf("缺段 %q:\n%s", h, out)
		}
		if i <= last {
			t.Fatalf("四段顺序不对：%q 未出现在上一段之后", h)
		}
		last = i
	}
	for _, want := range []string{"【选段前文】\n前文参考片段", "【选段后文】\n后文参考片段", "选段正文内容", spec.Hint} {
		if !strings.Contains(out, want) {
			t.Fatalf("缺 %q:\n%s", want, out)
		}
	}

	// 空上下文 → 章节开头/结尾占位（含纯空白）
	out2 := BuildPartialInstruction("指令", "选段", "", " \n\t", spec)
	for _, want := range []string{"（这是章节开头）", "（这是章节结尾）"} {
		if !strings.Contains(out2, want) {
			t.Fatalf("空上下文应占位 %q:\n%s", want, out2)
		}
	}

	// 指令 trim 后紧随段头
	out3 := BuildPartialInstruction("  去空格  ", "选段", "前", "后", spec)
	if !strings.Contains(out3, "## 用户指令\n去空格\n") {
		t.Fatalf("指令应 trim 后紧随段头:\n%s", out3)
	}

	// >1000 rune 截断（「霰」不出现在任何固定文案，只可能来自指令段）
	long := strings.Repeat("霰", 1200)
	out4 := BuildPartialInstruction("  "+long+"  ", "选段", "前", "后", spec)
	if n := strings.Count(out4, "霰"); n != 1000 {
		t.Fatalf("指令应截断至 1000 rune，剩 %d", n)
	}
}
