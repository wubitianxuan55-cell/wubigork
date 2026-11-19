package comhelper

// format_edit_boundary_test.go — R2 分档诚实恢复契约（pan_1795058214142 裁决
// action-1）静态契约测试：FormatEdit 成功/失败 × word/excel 四象限都必须带
// 分档撤销边界；任何回执不得暗示 gaea 侧「可回滚」（COM 档无基线快照）。

import (
	"strings"
	"testing"
)

func TestFormatEditCarriesTieredUndoBoundary(t *testing.T) {
	cases := []struct {
		name string
		kind AppKind
		word bool
	}{
		{"成功 word", KindWord, true},
		{"成功 excel", KindExcel, false},
		{"成功 ppt", KindPPT, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatEdit(EditResult{Total: 2, Done: []string{"set_text 3", "set_text 4"}}, tc.kind)
			if !strings.Contains(out, "All 2 operation(s) applied (not saved yet):") {
				t.Fatalf("成功回执缺主体段:\n%s", out)
			}
			assertBoundary(t, out, tc.word)
		})
	}
}

func TestFormatEditFailureCarriesTieredUndoBoundary(t *testing.T) {
	cases := []struct {
		name string
		kind AppKind
		code string
		word bool
	}{
		// 非 prechecked 码：失败警告句 + 边界都在。
		{"失败 word 非 prechecked", KindWord, "COM_FAIL", true},
		{"失败 excel 非 prechecked", KindExcel, "COM_FAIL", false},
		// prechecked 码（动文档前就失败）：无警告句，但边界仍必须在。
		{"失败 word prechecked", KindWord, "BAD_ARGS", true},
		{"失败 ppt prechecked", KindPPT, "NOT_FOUND", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatEdit(EditResult{
				Total:  3,
				Done:   []string{"set_text 3"},
				Failed: &EditFailed{Index: 1, Op: "set_text", Error: "boom", Code: tc.code},
			}, tc.kind)
			if !strings.Contains(out, "Stopped at operation 2 (set_text): boom") {
				t.Fatalf("失败回执缺中止点段:\n%s", out)
			}
			if !precheckedCodes[tc.code] && !strings.Contains(out, "may have changed part of the document") {
				t.Fatalf("非 prechecked 失败缺部分修改警告:\n%s", out)
			}
			if precheckedCodes[tc.code] && strings.Contains(out, "may have changed part of the document") {
				t.Fatalf("prechecked 失败不应带部分修改警告:\n%s", out)
			}
			assertBoundary(t, out, tc.word)
		})
	}
}

func TestUndoBoundaryWordVersusSheets(t *testing.T) {
	if got := UndoBoundary(KindWord); !strings.Contains(got, "一次 Ctrl+Z 整批撤销") || !strings.Contains(got, "仅限当前文档与当前会话") {
		t.Fatalf("Word 边界缺会话域限定: %s", got)
	}
	for _, kind := range []AppKind{KindExcel, KindPPT} {
		got := UndoBoundary(kind)
		if strings.Contains(got, "Ctrl+Z") {
			t.Fatalf("%s 边界不得承诺整批 Ctrl+Z 撤销: %s", kind, got)
		}
		if !strings.Contains(got, "不提供整批撤销") || !strings.Contains(got, "未核验") {
			t.Fatalf("%s 边界缺不承诺句: %s", kind, got)
		}
	}
}

// assertBoundary 分档断言：Word=整批撤销+会话域；Excel/PPT=不承诺整批/不承诺
// 逐操作；四象限都不得出现「可回滚」（gaea 证据链恢复是虚假承诺）。
func assertBoundary(t *testing.T, out string, word bool) {
	t.Helper()
	if strings.Contains(out, "可回滚") || strings.Contains(out, "可恢复") || strings.Contains(out, "基线快照可") {
		t.Errorf("回执不得暗示 gaea 侧恢复能力:\n%s", out)
	}
	if !strings.Contains(out, "撤销边界：") {
		t.Errorf("回执缺撤销边界行:\n%s", out)
		return
	}
	if word {
		if !strings.Contains(out, "一次 Ctrl+Z 整批撤销") || !strings.Contains(out, "仅限当前文档与当前会话") {
			t.Errorf("Word 回执缺整批撤销+会话域边界:\n%s", out)
		}
	} else {
		if !strings.Contains(out, "不提供整批撤销") || !strings.Contains(out, "未核验") {
			t.Errorf("Excel/PPT 回执缺不承诺句:\n%s", out)
		}
		if strings.Contains(out, "逐操作撤销") {
			t.Errorf("Excel/PPT 撤销粒度未核验，不得写成逐操作可撤:\n%s", out)
		}
	}
}
