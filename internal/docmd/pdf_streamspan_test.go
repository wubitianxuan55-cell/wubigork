package docmd

// pdf_streamspan_test.go — IN3-10：streamScanner（stream..endstream 定位迭代器）
// 表驱动单测。decodeFlateStreams 与 stripNonTextStreams 共用这套 span 定位，
// 四类用例：正常流 / 缺 endstream / 嵌套字面量 / 空输入，另钉住两条消费侧
// 保留的历史差异（\f 空白类、bodyStart 换行跳过、endstream 贴文件尾）。

import (
	"strings"
	"testing"
)

// collectStreamSpans 用指定空白类收集全部 span。
func collectStreamSpans(s string, isWS func(byte) bool) []streamSpan {
	sc := newStreamScanner(s, isWS)
	var out []streamSpan
	for {
		span, ok := sc.next()
		if !ok {
			return out
		}
		out = append(out, span)
	}
}

func TestStreamScannerSpans(t *testing.T) {
	cases := []struct {
		name string
		in   string
		isWS func(byte) bool
		want []streamSpan // 空切片/nil = 不应产出任何 span
	}{
		{
			name: "空输入",
			in:   "",
			isWS: isPDFSpace,
			want: nil,
		},
		{
			name: "空输入_无关键字纯文本",
			in:   "no stream keywords here\n",
			isWS: isPDFSpace,
			want: nil,
		},
		{
			name: "正常流_单块",
			in:   "<< /Length 3 >>\nstream\nabc\nendstream\nendobj\n",
			isWS: isPDFSpace,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex("<< /Length 3 >>", ">"),
					kwStart:   strings.Index("<< /Length 3 >>\nstream", "stream"),
					bodyStart: strings.Index("<< /Length 3 >>\nstream\nabc", "abc"),
					end:       strings.Index("<< /Length 3 >>\nstream\nabc\nendstream", "endstream"),
				},
			},
		},
		{
			name: "正常流_连续两块按序产出",
			in: "<<>>\nstream\nBT (a) Tj ET\nendstream\n" +
				"<<>>\nstream\nimg\nendstream\ntrailer\n",
			isWS: isStreamSepWS,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex("<<>>", ">"),
					kwStart:   strings.Index("<<>>\nstream", "stream"),
					bodyStart: strings.Index("<<>>\nstream\nBT (a) Tj ET", "BT"),
					end:       strings.Index("<<>>\nstream\nBT (a) Tj ET\nendstream", "endstream"),
				},
				{
					dictEnd:   strings.LastIndex("<<>>\nstream\nBT (a) Tj ET\nendstream\n<<>>", ">"),
					kwStart:   strings.LastIndex("<<>>\nstream\nBT (a) Tj ET\nendstream\n<<>>\nstream\n", "\nstream\n") + 1,
					bodyStart: strings.LastIndex("<<>>\nstream\nBT (a) Tj ET\nendstream\n<<>>\nstream\nimg", "img"),
					end:       strings.LastIndex("<<>>\nstream\nBT (a) Tj ET\nendstream\n<<>>\nstream\nimg\nendstream", "endstream"),
				},
			},
		},
		{
			name: "缺endstream_无块",
			in:   "<<>>\nstream\nabc",
			isWS: isPDFSpace,
			want: nil,
		},
		{
			name: "缺endstream_中止后续扫描",
			// 第一个真流往后再无任何 endstream → 整个扫描终止，
			// 后面另一条真流的关键字也不再有产出机会。
			in:   "<<>>stream\nx\njunk stream here\n<<>>stream\nBT Tj ET\n",
			isWS: isStreamSepWS,
			want: nil,
		},
		{
			name: "嵌套字面量_流体内伪endstream被跳过",
			// 流体内的 "endstreamXX" 后跟非空白 → 非独立，真 endstream 才是边界。
			in:   "<<>>\nstream\nAA endstreamXX BB\nendstream\n",
			isWS: isStreamSepWS,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex("<<>>", ">"),
					kwStart:   strings.Index("<<>>\nstream", "stream"),
					bodyStart: strings.Index("<<>>\nstream\nAA", "AA"),
					end:       strings.LastIndex("<<>>\nstream\nAA endstreamXX BB\nendstream", "endstream"),
				},
			},
		},
		{
			name: "嵌套字面量_endstream内的stream子串是伪命中",
			in:   "endstream\nendobj\n",
			isWS: isStreamSepWS,
			want: nil,
		},
		{
			name: "嵌套字面量_二进制里的stream无字典前缀",
			in:   "junk stream junk\n",
			isWS: isStreamSepWS,
			want: nil,
		},
		{
			name: "边界_endstream贴文件尾算独立",
			in:   "<<>>stream\r\n\r\nabendstream",
			isWS: isPDFSpace,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex("<<>>", ">"),
					kwStart:   strings.Index("<<>>stream", "stream"),
					bodyStart: strings.Index("<<>>stream\r\n\r\nab", "ab"),
					end:       strings.LastIndex("<<>>stream\r\n\r\nabendstream", "endstream"),
				},
			},
		},
		{
			name: "空白类差异_\\f前缀仅decode路径认真流",
			// isPDFSpace 含 \f（decode 路径）→ 真流；isStreamSepWS 不含 → 伪命中。
			in:   ">>\fstream\nabc\nendstream\n",
			isWS: isPDFSpace,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex(">>", ">"),
					kwStart:   strings.Index(">>\fstream", "stream"),
					bodyStart: strings.Index(">>\fstream\nabc", "abc"),
					end:       strings.Index(">>\fstream\nabc\nendstream", "endstream"),
				},
			},
		},
		{
			name: "空白类差异_\\f前缀strip路径不认真流",
			in:   ">>\fstream\nabc\nendstream\n",
			isWS: isStreamSepWS,
			want: nil,
		},
		{
			name: "空白类差异_\\f后缀仅decode路径认独立endstream",
			in:   "<<>>\nstream\nab\nendstream\fx",
			isWS: isPDFSpace,
			want: []streamSpan{
				{
					dictEnd:   strings.LastIndex("<<>>", ">"),
					kwStart:   strings.Index("<<>>\nstream", "stream"),
					bodyStart: strings.Index("<<>>\nstream\nab", "ab"),
					end:       strings.Index("<<>>\nstream\nab\nendstream", "endstream"),
				},
			},
		},
		{
			name: "空白类差异_\\f后缀strip路径认不出endstream",
			in:   "<<>>\nstream\nab\nendstream\fx",
			isWS: isStreamSepWS,
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := collectStreamSpans(tc.in, tc.isWS)
			if len(got) != len(tc.want) {
				t.Fatalf("span 数不符: got %d (%+v), want %d", len(got), got, len(tc.want))
			}
			for i, g := range got {
				w := tc.want[i]
				if g != w {
					t.Errorf("span[%d] 边界不符:\ngot  %+v\nwant %+v", i, g, w)
				}
				// 边界自洽：dictEnd 指向 '>'，kwStart/bodyStart/end 落在关键字上。
				if tc.in[g.dictEnd] != '>' {
					t.Errorf("span[%d].dictEnd=%d 不是 '>': %q", i, g.dictEnd, tc.in)
				}
				if !strings.HasPrefix(tc.in[g.kwStart:], "stream") {
					t.Errorf("span[%d].kwStart=%d 不是 stream 关键字: %q", i, g.kwStart, tc.in)
				}
				if !strings.HasPrefix(tc.in[g.end:], "endstream") {
					t.Errorf("span[%d].end=%d 不是 endstream 关键字: %q", i, g.end, tc.in)
				}
				if g.bodyStart < g.kwStart+len("stream") || g.end < g.bodyStart {
					t.Errorf("span[%d] 边界乱序: %+v", i, g)
				}
			}
		})
	}
}
