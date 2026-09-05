package genui

import (
	"strings"
	"testing"
)

// 表驱动：StripUIFences 的行扫描语义（对齐前端 splitGenuiFences）。
func TestStripUIFences(t *testing.T) {
	fence := "```genui\n{\"title\":\"看板\",\"items\":[{\"type\":\"stat\",\"label\":\"营收\",\"value\":\"1\"}]}\n```"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "无围栏逐字节不变",
			in:   "第一段正文\n\n第二段：含 { 花括号 } 与 \"引号\"\n",
			want: "第一段正文\n\n第二段：含 { 花括号 } 与 \"引号\"\n",
		},
		{
			name: "无反引号但有类似内容不误剥",
			in:   "前文\n```\n代码块\n```\n后文",
			want: "前文\n```\n代码块\n```\n后文",
		},
		{
			name: "单围栏折叠为占位，围栏外正文保留",
			in:   "前言\n" + fence + "\n后记",
			want: "前言\n" + UIFencePlaceholder + "\n后记",
		},
		{
			name: "多围栏各占一行占位",
			in:   fence + "\n中间\n" + "```dsh-ui\n{\"items\":[]}\n```",
			want: UIFencePlaceholder + "\n中间\n" + UIFencePlaceholder,
		},
		{
			name: "未闭合围栏折叠到末尾",
			in:   "结论如下\n```genui\n{\"title\":\"未闭合\"",
			want: "结论如下\n" + UIFencePlaceholder,
		},
		{
			name: "dsh-ui 语言同样剥离",
			in:   "```dsh-ui\n{}\n```\n正文",
			want: UIFencePlaceholder + "\n正文",
		},
		{
			name: "围栏标记在行中间不算开栏",
			in:   "看这个：```genui 示例\n正文",
			want: "看这个：```genui 示例\n正文",
		},
		{
			name: "其他语言代码块不误剥",
			in:   "```json\n{\"items\":[{\"type\":\"stat\"}]}\n```\n正文",
			want: "```json\n{\"items\":[{\"type\":\"stat\"}]}\n```\n正文",
		},
		{
			name: "裸 ``` 代码块不误剥",
			in:   "```\n原样保留\n```\n",
			want: "```\n原样保留\n```\n",
		},
		{
			name: "闭栏行允许缩进与尾随空白",
			in:   "```genui\n{\"items\":[]}\n   ```   \n尾行",
			want: UIFencePlaceholder + "\n尾行",
		},
		{
			name: "围栏体内出现 ```lang 不重新开栏",
			in:   "```genui\n{\"a\":1}\n```json\n残留体\n```\n尾行",
			want: UIFencePlaceholder + "\n尾行",
		},
		{
			name: "仅围栏的文本折叠为占位",
			in:   fence,
			want: UIFencePlaceholder,
		},
		{
			name: "空文本",
			in:   "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripUIFences(tc.in); got != tc.want {
				t.Fatalf("StripUIFences(%q) =\n%q\nwant\n%q", tc.in, got, tc.want)
			}
		})
	}
}

// 围栏语言集合与前端 GENUI_FENCE_LANGS 同源锚点。
func TestUIFenceLangsParity(t *testing.T) {
	for _, lang := range []string{"genui", "dsh-ui"} {
		if !isUIFenceLang(lang) {
			t.Fatalf("围栏语言 %s 应被识别", lang)
		}
	}
	for _, lang := range []string{"json", "go", "genui2", "dsh-ui-x", "", "GENUI"} {
		if isUIFenceLang(lang) {
			t.Fatalf("语言 %s 不应被识别为 UI 围栏", lang)
		}
	}
}

// 占位行为：多围栏各产生一个占位行，占位本身不含反引号（不再形成代码块）。
func TestStripUIFencesPlaceholderShape(t *testing.T) {
	got := StripUIFences("```genui\n{}\n```\n---\n```genui\n{}\n```")
	if n := strings.Count(got, UIFencePlaceholder); n != 2 {
		t.Fatalf("占位行数 = %d, want 2 (%q)", n, got)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("剥离结果不应残留围栏标记: %q", got)
	}
}
