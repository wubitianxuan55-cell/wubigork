package bookimport

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestExtractJSON_FirstCompleteObject 钉住 P0-22 同类缺陷在本包的那份重复实现：
// 回复里出现两个并列对象、正文 + 对象 + 后文再提到花括号、字符串字面量内含花括号时，
// 必须切出「第一个完整对象」，而不是「第一个 { 到最后一个 }」拼出来的非法 JSON。
func TestExtractJSON_FirstCompleteObject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]any
	}{
		{
			name: "两个并列对象取第一个",
			in: `{"title":"甲","n":1}
{"title":"乙","n":2}`,
			want: map[string]any{"title": "甲", "n": float64(1)},
		},
		{
			name: "正文 + 对象 + 后文再提到花括号",
			in: `好的，结果如下：
{"title":"甲","n":1}
补充说明：本章结构见 {上述字段}。`,
			want: map[string]any{"title": "甲", "n": float64(1)},
		},
		{
			name: "字符串字面量内的花括号不计入结构",
			in: `{"note":"引号里的 } 与 { 都不算结构","n":2}
后文再次提到 {占位}。`,
			want: map[string]any{"note": "引号里的 } 与 { 都不算结构", "n": float64(2)},
		},
		{
			name: "第一个对象内嵌子对象、后文再提到花括号",
			in: `分析如下：
{"project":{"title":"甲"},"n":3}
（附注：{可忽略}）`,
			want: map[string]any{"project": map[string]any{"title": "甲"}, "n": float64(3)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractJSON(tc.in)
			if err != nil {
				t.Fatalf("应提取出第一个完整对象，却报错: %v\n输入:\n%s", err, tc.in)
			}
			obj, ok := got.(map[string]any)
			if !ok {
				t.Fatalf("应取到对象，实际 %T: %#v", got, got)
			}
			if !reflect.DeepEqual(obj, tc.want) {
				t.Fatalf("提取结果不符\n got: %#v\nwant: %#v", obj, tc.want)
			}
			// 「完整对象」的硬证据：重新序列化后仍能 json.Unmarshal 回来
			b, err := json.Marshal(obj)
			if err != nil {
				t.Fatalf("提取结果不可序列化: %v", err)
			}
			var back map[string]any
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatalf("提取结果不可 json.Unmarshal: %v (原文 %s)", err, b)
			}
		})
	}
}

// TestExtractJSON_ArrayOfObjectsTakenFirst 守住 ExpectArray 路径（章节大纲批次）：
// 顶层对象数组之后即使再出现裸对象，也必须先取到那个数组。
func TestExtractJSON_ArrayOfObjectsTakenFirst(t *testing.T) {
	in := `[{"chapter_number":1}]
补充：本章结构见 {上述字段}。`
	got, err := ExtractJSON(in)
	if err != nil {
		t.Fatalf("应提取出对象数组: %v", err)
	}
	arr, ok := got.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("应取到 1 元素数组，实际 %T: %#v", got, got)
	}
	if err := CheckExpected(got, ExpectArray); err != nil {
		t.Fatalf("提取结果应满足 ExpectArray: %v", err)
	}
}

// TestExtractJSON_BareFenceStillTolerated 守住原 stripCodeFence 的容错面：
// 去掉本包自带实现后，无语言标记的裸围栏也必须照样能提取。
func TestExtractJSON_BareFenceStillTolerated(t *testing.T) {
	fence := strings.Repeat("`", 3)
	in := fence + "\n" + `{"title":"甲"}` + "\n" + fence
	got, err := ExtractJSON(in)
	if err != nil {
		t.Fatalf("裸围栏应可提取: %v", err)
	}
	obj, ok := got.(map[string]any)
	if !ok || obj["title"] != "甲" {
		t.Fatalf("裸围栏提取结果异常: %#v", got)
	}
}
