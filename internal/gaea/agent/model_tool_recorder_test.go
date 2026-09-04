package agent

// v4.62.2：mt_ transcript 正文的信封拆包回归。本地模型工具（vision /
// summarize_file）的 ToolResult.Output 是 JSON 信封串，原样落 transcript 会
// 渲染出「字面 \n」转义墙（实机报告：本地模型工具输出一团乱）。

import "testing"

func TestUnwrapModelToolOutput(t *testing.T) {
	// 标准信封：取 data.result 正文（真实换行保留）
	envelope := "{\"ok\":true,\"success\":true,\"code\":\"ok\",\"data\":{\"result\":\"第一段\\n\\n第二段\",\"tool\":\"summarize_file\"}}"
	if got := unwrapModelToolOutput(envelope); got != "第一段\n\n第二段" {
		t.Fatalf("信封拆包失败：%q", got)
	}

	// 非信封（自由文本）原样返回
	if got := unwrapModelToolOutput("普通正文\n多行"); got != "普通正文\n多行" {
		t.Fatalf("自由文本不应改动：%q", got)
	}

	// JSON 但无 data.result（或为空）原样返回
	if got := unwrapModelToolOutput(`{"ok":false,"error":"boom"}`); got != `{"ok":false,"error":"boom"}` {
		t.Fatalf("无 result 信封应原样返回：%q", got)
	}

	// 破损 JSON 原样返回（不抛错）
	if got := unwrapModelToolOutput("{not-json"); got != "{not-json" {
		t.Fatalf("破损 JSON 应原样返回：%q", got)
	}

	// result 为空白视同缺失，原样返回
	if got := unwrapModelToolOutput(`{"data":{"result":"   "}}`); got != `{"data":{"result":"   "}}` {
		t.Fatalf("空白 result 应原样返回：%q", got)
	}
}
