package pantheon

import (
	"strings"
	"testing"
)

// 2026-11-03 空结果根修回归（live dump 实证：zcode 大仓流 4,194,323B 正好撞
// 4MB cap，runStream 旧代码只取 head，turn.completed/result/text_delta 全在
// 被丢弃的 tail → 「Agent 返回空结果」）。锁死两条契约：
//   1. cappedBuffer.String() 必须含尾部环留（11-02 设计），runStream 必须用它；
//   2. 截断流（头+标记+尾）喂 parseAgentOutput 仍能出全文。

func TestCappedBufferStringCarriesTail(t *testing.T) {
	buf := &cappedBuffer{limit: 1000}
	head := strings.Repeat("a", 600) + "\n"
	mid := strings.Repeat("b", 600) + "\n"
	tail := `{"type":"turn.completed","payload":{"response":"尾部结论TAILKEEP"}}` + "\n" +
		`{"type":"result","response":"result 尾部结论"}` + "\n"
	buf.Write([]byte(head))
	buf.Write([]byte(mid)) // 触发截断，进入 tail 模式
	buf.Write([]byte(tail))

	got := buf.String()
	if !strings.Contains(got, "进程输出超限") {
		t.Fatalf("truncation marker missing:\n%.200s", got)
	}
	if !strings.Contains(got, "TAILKEEP") {
		t.Fatalf("tail content dropped:\n%.200s", got)
	}
	if !strings.Contains(got, "result 尾部结论") {
		t.Fatalf("tail end dropped — this is the 2026-11-03 empty-result bug:\n%.200s", got)
	}
	if !buf.truncated {
		t.Fatal("truncated flag must be set")
	}
}

func TestParseTruncatedStreamStillYieldsVerdict(t *testing.T) {
	// 模拟修复后 runStream 返回给 parse 的真实截断形态：
	// head（早期事件+被斩断的残行）+ 截断标记 + tail（text_delta 末段+收尾事件）。
	head := `{"type":"turn.started","payload":{}}
{"type":"model.streaming","payload":{"kind":"reasoning_delta","delta":"思考"}}
{"type":"model.streaming","payload":{"kind":"text_delta","delta":"前半段观"`
	marker := "\n【进程输出超限，中段截断，以下保留完整尾部】\n"
	tail := `点已经被尾部承接。
{"type":"model.streaming","payload":{"kind":"turn.completed","response":"完整陈词：截断流仍要出全文KEEPVERDICT。"}}
{"type":"result","response":"完整陈词：截断流仍要出全文KEEPVERDICT。"}
`
	raw := head + marker + tail
	got := parseAgentOutput(raw)
	if !strings.Contains(got, "KEEPVERDICT") {
		t.Fatalf("truncated stream lost the verdict:\n%s", got)
	}
}
