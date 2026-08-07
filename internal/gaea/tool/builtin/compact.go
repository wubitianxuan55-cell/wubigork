// Package builtin provides gaea's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import "encoding/json"

var compactDesc = map[string]string{
	"read_file":        "读取文件内容(可选行范围/分页)",
	"write_file":       "写入/覆盖文件(自动建父目录)",
	"ls":               "列出目录条目(子目录带/)",
	"bash":             "执行shell命令(5分超时,output_format=json得结构化输出)",
	"bash_output":      "读取后台任务增量输出",
	"kill_shell":       "终止后台任务",
	"wait":             "阻塞等待后台任务结束",
	"web_fetch":        "抓取URL纯文本(去标签,SSRF安全,支持重试)",
	"web_search":       "搜索公开网页，返回结构化JSON(title/url/snippet/source)，支持引用追踪",
	"todo_write":       "更新任务清单(全量替换,最多一个进行中)",
	"complete_step":    "完成计划步骤(须可验证证据,禁止纯manual)",
	"memory_search":    "搜索记忆(关键词,kind过滤,BM25排序)",
	"read_skill":       "读取指定技能(skill)的完整内容",
	"format_convert":   "文档格式转换(docx/xlsx/pdf→Markdown，含OCR扫描件回退)",
	"chart_gen":        "matplotlib图表生成(bar/line/pie/scatter)",
	"knowledge_search": "搜索工程知识库(关键词,分类+标签过滤)",
	"knowledge_add":    "向知识库添加条目(标题+分类+正文)",
}

var compactSchema = map[string]json.RawMessage{
	"read_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"},"line_numbers":{"type":"boolean"}},"required":["path"]}`),
	"write_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
	"ls": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"}}}`),
	"bash": json.RawMessage(
		`{"type":"object","properties":{"command":{"type":"string"},"run_in_background":{"type":"boolean"}},"required":["command"]}`),
	"bash_output": json.RawMessage(
		`{"type":"object","properties":{"job_id":{"type":"string"},"filter":{"type":"string"}},"required":["job_id"]}`),
	"kill_shell": json.RawMessage(
		`{"type":"object","properties":{"job_id":{"type":"string"}},"required":["job_id"]}`),
	"wait": json.RawMessage(
		`{"type":"object","properties":{"job_ids":{"type":"array","items":{"type":"string"}},"timeout_seconds":{"type":"integer"}}}`),
	"web_fetch": json.RawMessage(
		`{"type":"object","properties":{"url":{"type":"string"},"retries":{"type":"integer"}},"required":["url"]}`),
	"web_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"topK":{"type":"integer"}},"required":["query"]}`),
	"todo_write": json.RawMessage(
		`{"type":"object","properties":{"todos":{"type":"array","items":{"type":"object","properties":{"content":{"type":"string"},"status":{"type":"string"},"activeForm":{"type":"string"},"level":{"type":"integer"}},"required":["content","status"]}}},"required":["todos"]}`),
	"complete_step": json.RawMessage(
		`{"type":"object","properties":{"step":{"type":"string"},"step_index":{"type":"integer"},"result":{"type":"string"},"evidence":{"type":"array","items":{"type":"object","properties":{"kind":{"type":"string"},"summary":{"type":"string"},"command":{"type":"string"},"paths":{"type":"array","items":{"type":"string"}}},"required":["kind","summary"]}}},"required":["step","result","evidence"]}`),
	"memory_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"kind":{"type":"string"}},"required":["query"]}`),
	"read_skill": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	"format_convert": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"output":{"type":"string"},"pages":{"type":"string"}},"required":["path"]}`),
	"chart_gen": json.RawMessage(
		`{"type":"object","properties":{"labels":{"type":"array","items":{"type":"string"}},"values":{"type":"array","items":{"type":"number"}},"chart_type":{"type":"string"},"title":{"type":"string"},"output":{"type":"string"}},"required":["labels","values"]}`),
	"knowledge_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"category":{"type":"string"},"tag":{"type":"string"}}}`),
	"knowledge_add": json.RawMessage(
		`{"type":"object","properties":{"title":{"type":"string"},"category":{"type":"string"},"body":{"type":"string"},"tags":{"type":"string"},"phase":{"type":"string"},"discipline":{"type":"string"},"source":{"type":"string"}},"required":["title","category","body"]}`),
}
