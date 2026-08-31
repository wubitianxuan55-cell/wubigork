// Package builtin provides gaea's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import "encoding/json"

var compactDesc = map[string]string{
	"read_file":          "读取文件内容(可选行范围/分页)",
	"write_file":         "写入/覆盖文件(自动建父目录)",
	"edit_file":          "精确替换文件内容(old_string→new_string,new_string空串=删除)",
	"multi_edit":         "单文件批量替换(edits数组,内存串行,全部成功才原子写盘)",
	"edit_lines":         "按行号区间替换(1-based含端点,new_content空串=删行)",
	"move_file":          "移动/重命名文件(跨卷回退复制,overwrite覆盖目标)",
	"grep":               "正则搜索文件内容(path:line: content,跳过二进制/噪声目录)",
	"ls":                 "列出目录条目(子目录带/)",
	"bash":               "执行shell命令(5分超时,output_format=json得结构化输出)",
	"bash_output":        "读取后台任务增量输出",
	"kill_shell":         "终止后台任务",
	"wait":               "阻塞等待后台任务结束",
	"web_fetch":          "抓取URL纯文本(去标签,SSRF安全,支持重试)",
	"web_search":         "搜索公开网页，返回结构化JSON(title/url/snippet/source)，支持引用追踪",
	"todo_write":         "更新任务清单(全量替换,最多一个进行中)",
	"complete_step":      "完成计划步骤(须可验证证据,禁止纯manual)",
	"memory_search":      "搜索记忆(关键词,kind过滤,BM25排序)",
	"read_skill":         "读取指定技能(skill)的完整内容",
	"format_convert":     "文档格式转换(docx/xlsx/pdf→Markdown，含OCR扫描件回退)",
	"chart_gen":          "matplotlib图表生成(bar/line/pie/scatter)",
	"diagram_gen":        "框架图/流程图生成(matplotlib,中文清晰,替代文生图画文字)",
	"knowledge_search":   "搜索工程知识库(关键词,分类+标签过滤)",
	"knowledge_add":      "向知识库添加条目(标题+分类+正文)",
	"cost_search":        "搜索成本库(关键词/分类/状态,返回单价表)",
	"cost_save":          "写入/更新成本库条目(同名覆盖,含单价/单位/来源)",
	"screen_capture":     "捕获屏幕截图保存为PNG,返回文件路径(可指定region局部截图)",
	"vision":             "用本地视觉模型识别图片内容(文字/布局/细节),配合screen_capture理解截图",
	"browser_navigate":   "受控Edge打开URL(仅http/https,独立临时profile,不碰用户浏览器)",
	"browser_read":       "读取受控浏览器页面文本(全文或selector局部,可截断,frame可选iframe内)",
	"browser_snapshot":   "列出页面可交互元素([#ref] <tag> 文本),click/type前必须先snapshot拿ref",
	"browser_click":      "点击页面元素(snapshot的ref或CSS selector,跳转后ref失效,frame可选iframe内)",
	"browser_type":       "输入文本到输入框(ref或selector,React兼容,submit提交表单,frame可选iframe内)",
	"browser_press":      "发送键盘级按键(Enter/Tab/Escape/字符等,可带ctrl/alt/shift/meta组合,text真实输入)",
	"browser_scroll":     "滚动页面(up/down,amount像素,可选容器selector)",
	"browser_tabs":       "列出受控浏览器全部标签页([active]标记当前页,含标题/URL)",
	"browser_new_tab":    "新建标签页并切换为当前页(url必填,原页refs失效)",
	"browser_switch_tab": "切换当前标签页(tab_id必填,切换后refs失效需重新snapshot)",
	"browser_close":      "关闭受控浏览器(缺省关页+杀Edge+清临时profile;tab_id只关指定标签)",
}

var compactSchema = map[string]json.RawMessage{
	"read_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"offset":{"type":"integer"},"limit":{"type":"integer"},"line_numbers":{"type":"boolean"}},"required":["path"]}`),
	"write_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
	"edit_file": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"old_string":{"type":"string"},"new_string":{"type":"string"},"replace_all":{"type":"boolean"}},"required":["path","old_string","new_string"]}`),
	"multi_edit": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"edits":{"type":"array","items":{"type":"object","properties":{"old_string":{"type":"string"},"new_string":{"type":"string"},"replace_all":{"type":"boolean"}},"required":["old_string","new_string"]}}},"required":["path","edits"]}`),
	"edit_lines": json.RawMessage(
		`{"type":"object","properties":{"path":{"type":"string"},"start_line":{"type":"integer"},"end_line":{"type":"integer"},"new_content":{"type":"string"}},"required":["path","start_line","end_line","new_content"]}`),
	"move_file": json.RawMessage(
		`{"type":"object","properties":{"source":{"type":"string"},"destination":{"type":"string"},"overwrite":{"type":"boolean"}},"required":["source","destination"]}`),
	"grep": json.RawMessage(
		`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"},"include":{"type":"string"},"max_results":{"type":"integer"}},"required":["pattern"]}`),
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
	"diagram_gen": json.RawMessage(
		`{"type":"object","properties":{"title":{"type":"string"},"kind":{"type":"string","enum":["framework","flow"]},"nodes":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"label":{"type":"string"},"level":{"type":"integer"},"group":{"type":"string"}},"required":["id","label"]}},"edges":{"type":"array","items":{"type":"object","properties":{"from":{"type":"string"},"to":{"type":"string"},"label":{"type":"string"}},"required":["from","to"]}},"output":{"type":"string"}},"required":["nodes"]}`),
	"knowledge_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"category":{"type":"string"},"tag":{"type":"string"}}}`),
	"knowledge_add": json.RawMessage(
		`{"type":"object","properties":{"title":{"type":"string"},"category":{"type":"string"},"body":{"type":"string"},"tags":{"type":"string"},"phase":{"type":"string"},"discipline":{"type":"string"},"source":{"type":"string"}},"required":["title","category","body"]}`),
	"cost_search": json.RawMessage(
		`{"type":"object","properties":{"query":{"type":"string"},"category":{"type":"string"},"status":{"type":"string"},"limit":{"type":"integer"}}}`),
	"cost_save": json.RawMessage(
		`{"type":"object","properties":{"name":{"type":"string"},"title":{"type":"string"},"category":{"type":"string"},"unit":{"type":"string"},"price":{"type":"number"},"spec":{"type":"string"},"source":{"type":"string"},"tags":{"type":"string"},"status":{"type":"string"},"body":{"type":"string"}},"required":["title","price"]}`),
	"screen_capture": json.RawMessage(
		`{"type":"object","properties":{"region":{"type":"object","properties":{"x":{"type":"integer"},"y":{"type":"integer"},"width":{"type":"integer"},"height":{"type":"integer"}}}}}`),
	"vision": json.RawMessage(
		`{"type":"object","properties":{"image_path":{"type":"string"},"prompt":{"type":"string"}},"required":["image_path"]}`),
	"browser_navigate": json.RawMessage(
		`{"type":"object","properties":{"url":{"type":"string"},"timeout_secs":{"type":"integer"}},"required":["url"]}`),
	"browser_read": json.RawMessage(
		`{"type":"object","properties":{"selector":{"type":"string"},"max_chars":{"type":"integer"},"frame":{"type":"string"}}}`),
	"browser_snapshot": json.RawMessage(
		`{"type":"object"}`),
	"browser_click": json.RawMessage(
		`{"type":"object","properties":{"ref":{"type":"integer"},"selector":{"type":"string"},"frame":{"type":"string"}}}`),
	"browser_type": json.RawMessage(
		`{"type":"object","properties":{"ref":{"type":"integer"},"selector":{"type":"string"},"text":{"type":"string"},"submit":{"type":"boolean"},"frame":{"type":"string"}},"required":["text"]}`),
	"browser_press": json.RawMessage(
		`{"type":"object","properties":{"key":{"type":"string"},"modifiers":{"type":"array","items":{"type":"string"}},"text":{"type":"string"}},"required":["key"]}`),
	"browser_scroll": json.RawMessage(
		`{"type":"object","properties":{"direction":{"type":"string"},"amount":{"type":"integer"},"selector":{"type":"string"}}}`),
	"browser_tabs": json.RawMessage(
		`{"type":"object"}`),
	"browser_new_tab": json.RawMessage(
		`{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`),
	"browser_switch_tab": json.RawMessage(
		`{"type":"object","properties":{"tab_id":{"type":"string"}},"required":["tab_id"]}`),
	"browser_close": json.RawMessage(
		`{"type":"object","properties":{"tab_id":{"type":"string"}}}`),
}
