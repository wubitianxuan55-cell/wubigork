// Package builtin provides Tianxuan's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import "encoding/json"

var compactDesc = map[string]string{
	"read_file":        "读取文件(可选行范围/分页)",
	"write_file":       "写入/覆盖文件(自动建父目录)",
	"glob":             "通配符匹配文件名(支持**递归)",
	"grep":             "正则搜索文件内容(返回path:行:文本,支持sort_by=relevance)",
	"ls":               "列目录条目(子目录带/)",
	"bash":             "执行shell命令(5分超时,output_format=json得结构化输出)",
	"bash_output":      "读取后台任务增量输出",
	"kill_shell":       "终止后台任务",
	"wait":             "阻塞等待后台任务结束",
	"web_fetch":        "抓取URL纯文本(去标签,SSRF安全,支持重试)",
	"web_search":       "搜索公开网页，返回结构化JSON(title/url/snippet/source)，支持引用追踪",
	"todo_write":       "更新任务清单(全量替换,最多一个进行中)",
	"complete_step":    "完成计划步骤(须可验证证据,禁止纯manual)",
	"git_status":       "显示工作区状态(分支/暂存/未暂存/未跟踪/冲突)",
	"git_diff":         "显示行级别变更(--staged可选,path可限文件)",
	"git_log":          "显示提交历史(支持count/path/author过滤)",
	"memory_search":    "搜索记忆(关键词+kind过滤,BM25排序)",
	"read_skill":       "读取指定技能(skill)的完整内容",
	"csv_parse":        "CSV解析(自动编码/分隔符检测+列统计)",
	"calc_math":        "数学表达式计算(AST解析+math函数)",
	"calc_stats":       "基础统计分析(均值/中位数/标准差)",
	"calc_unit":        "单位转换(长度/重量/温度/压力)",
	"gantt_gen":        "生成Mermaid甘特图(Markdown代码块)",
	"project_init":     "初始化项目目录结构(工程标准模板)",
	"xlsx_read":        "读取Excel文件(解析xlsx表格数据)",
	"xlsx_write":       "创建Excel文件(表头+数据→xlsx,支持多工作表和内置图表)",
	"docx_read":        "读取Word文件(提取docx段落文本)",
	"docx_write":       "创建Word文件(标题+多段正文→docx)",
	"pdf_extract":      "PDF文本提取(支持页码范围)",
	"pdf_create":       "创建PDF文件(标题+段落→pdf)",
	"pptx_create":      "创建PPT文件(标题+要点+图表→pptx)",
	"format_convert":   "文档格式转换(docx/xlsx/pdf→Markdown，含OCR扫描件回退)",
	"chart_gen":        "matplotlib图表生成(bar/line/pie/scatter)",
	"run_template":     "加载运行工具链模板(参数替换)",
	"knowledge_search": "搜索工程知识库(关键词+分类+标签过滤)",
	"knowledge_add":    "向知识库添加条目(标题+分类+正文)",
}
var compactSchema = map[string]json.RawMessage{
	"read_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"},"line_numbers":{"type":"boolean"}},"required":["path"]}`),
	"write_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
	"glob": json.RawMessage(
		`{"type":"object","properties":{"pattern":{"type":"string"}},"required":["pattern"]}`),
	"grep": json.RawMessage(
		`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`),
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
	"git_status": json.RawMessage(
		`{"type":"object","properties":{},"required":[]}`),
	"git_diff": json.RawMessage(
		`{"type":"object","properties":{"staged":{"type":"boolean"},"path":{"type":"string"}}}`),
	"git_log": json.RawMessage(
		`{"type":"object","properties":{"count":{"type":"integer"},"path":{"type":"string"},"author":{"type":"string"}}}`),
	"memory_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"kind":{"type":"string"}},"required":["query"]}`),
	"read_skill": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	"csv_parse": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"delimiter":{"type":"string"},"has_header":{"type":"boolean"},"encoding":{"type":"string"},"limit":{"type":"integer"}},"required":["path"]}`),
	"calc_math": json.RawMessage(
		`{"type":"object","properties":{"expression":{"type":"string"}},"required":["expression"]}`),
	"calc_stats": json.RawMessage(
		`{"type":"object","properties":{"values":{"type":"array","items":{"type":"number"}}},"required":["values"]}`),
	"calc_unit": json.RawMessage(
		`{"type":"object","properties":{"value":{"type":"number"},"from_unit":{"type":"string"},"to_unit":{"type":"string"}},"required":["value","from_unit","to_unit"]}`),
	"gantt_gen": json.RawMessage(
		`{"type":"object","properties":{"title":{"type":"string"},"tasks":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"start":{"type":"string"},"end":{"type":"string"},"duration":{"type":"string"},"depends":{"type":"string"},"section":{"type":"string"}},"required":["name"]}}},"required":["tasks"]}`),
	"project_init": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"},"type":{"type":"string"}},"required":["name","type"]}`),
	"xlsx_read": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"all_sheets":{"type":"boolean"},"sheet_index":{"type":"integer"}},"required":["path"]}`),
	"xlsx_write": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"sheet_name":{"type":"string"},"headers":{"type":"array","items":{"type":"string"}},"rows":{"type":"array","items":{"type":"array","items":{"type":"string"}}},"sheets":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"headers":{"type":"array","items":{"type":"string"}},"rows":{"type":"array","items":{"type":"array","items":{"type":"string"}}},"chart":{"type":"object"}}}}}},"anyOf":[{"required":["path","rows"]},{"required":["path","sheets"]}]}`),
	"docx_read": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
	"docx_write": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"title":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
	"pdf_extract": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"pages":{"type":"string"}},"required":["path"]}`),
	"pdf_create": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"title":{"type":"string"},"content":{"type":"string"},"footer":{"type":"string"}},"required":["path","content"]}`),
	"pptx_create": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"slides":{"type":"array","items":{"type":"object","properties":{"title":{"type":"string"},"content":{"type":"array","items":{"type":"string"}},"layout":{"type":"string"},"chart":{"type":"string"}}}},"required":["path","slides"]}`),
	"format_convert": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"output":{"type":"string"},"pages":{"type":"string"}},"required":["path"]}`),
	"chart_gen": json.RawMessage(
		`{"type":"object","properties":{"labels":{"type":"array","items":{"type":"string"}},"values":{"type":"array","items":{"type":"number"}},"chart_type":{"type":"string"},"title":{"type":"string"},"output":{"type":"string"}},"required":["labels","values"]}`),
	"doc_merge": json.RawMessage(
		`{"type":"object","properties":{"files":{"type":"array","items":{"type":"string"}},"output":{"type":"string"},"add_page_breaks":{"type":"boolean"}},"required":["files","output"]}`),
	"archive": json.RawMessage(
		`{"type":"object","properties":{"action":{"type":"string"},"source":{"type":"string"},"target":{"type":"string"},"files":{"type":"array","items":{"type":"string"}}},"required":["action","source"]}`),
	"save_template": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"},"description":{"type":"string"},"steps":{"type":"array","items":{"type":"object","properties":{"tool":{"type":"string"},"args":{"type":"object"}},"required":["tool"]}}},"required":["name","steps"]}}`),
	"run_template": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"},"params":{"type":"object"}},"required":["name"]}`),
	"knowledge_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"category":{"type":"string"},"tag":{"type":"string"}}}}`),
	"knowledge_add": json.RawMessage(
		`{"type":"object","properties":{"title":{"type":"string"},"category":{"type":"string"},"body":{"type":"string"},"tags":{"type":"string"},"phase":{"type":"string"},"discipline":{"type":"string"},"source":{"type":"string"}},"required":["title","category","body"]}`),
}
