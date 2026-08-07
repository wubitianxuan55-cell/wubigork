package skill

// Built-in skills ship with gaea and back general office subagent workflows.
// A user/project file with the same name overrides the built-in (see Store.List / Store.Read).

// negativeClaimRule keeps subagents honest about "found nothing" answers.
const negativeClaimRule = `When you claim something does NOT exist (no caller, no usage, not implemented), say which searches you ran to reach that conclusion — a negative claim is only as trustworthy as the search behind it.`

// tuiFormatting nudges concise, terminal-friendly output.
const tuiFormatting = `Keep the final answer compact and terminal-friendly: short paragraphs or bullets, no walls of text, no restating the question.`

// builtinSkills returns the shipped skills. A fresh slice each call so callers
// can't mutate the shared set.
func builtinSkills() []Skill {
	return []Skill{
		{
			Name:        "format-convert",
			Description: "文档格式转换：docx/xlsx/pdf→Markdown 格式转换，可用于统一不同来源的工程文档为可编辑 Markdown。",
			Body: `你作为格式转换子代理运行。将工程文档转换为 Markdown 格式。

## 可用工具
- format_convert：一键转换 docx/xlsx/pdf 为 Markdown

## 操作方式
1. 确认源文件格式（.docx/.xlsx/.pdf）
2. 使用 format_convert 工具转换
3. 如指定 output 参数，保存为文件；否则返回文本

## 最终输出
- 返回转换后的 Markdown 文本
- 保留原标题层级、表格结构

` + negativeClaimRule + `

` + tuiFormatting + `

父节点的 'task' 是格式转换任务。不要偏离。`,
			Scope:        ScopeBuiltin,
			Path:         "(builtin)",
			RunAs:        RunSubagent,
			AllowedTools: []string{"format_convert", "read_file", "write_file"},
		},
		{
			Name:        "chart-builder",
			Description: "图表生成：从检测数据生成统计图表（柱状图/折线图/饼图/散点图），适用于调查报告数据可视化。",
			Body: `你作为图表生成子代理运行。从工程数据生成可视化图表。

## 可用工具
- chart_gen：生成 matplotlib 图表（bar/line/pie/scatter）

## 操作方式
1. 确认数据类别和数值
2. 选择合适的图表类型
3. 使用 chart_gen 工具生成并保存图片

## 最终输出
- 返回图片文件路径
- 附图表类型和数据摘要

` + negativeClaimRule + `

` + tuiFormatting + `

父节点的 'task' 是图表生成任务。不要偏离。`,
			Scope:        ScopeBuiltin,
			Path:         "(builtin)",
			RunAs:        RunSubagent,
			AllowedTools: []string{"chart_gen", "read_file", "write_file", "xlsx_read", "csv_parse"},
		},
		{
			Name:        "doc-assemble",
			Description: "文档拼装：将多份 Markdown 文档片段合并为完整报告，含封面、目录、正文、附录。",
			Body: `你作为文档拼装子代理运行。将多份文档素材拼装为完整报告。

## 可用工具
- doc_merge：合并多个 docx 文档
- read_file / write_file：读取和写入 Markdown 片段
- docx_write：将最终 Markdown 输出为 docx

## 操作方式
1. 收集所有文档片段（Markdown 或 docx）
2. 按报告结构组织：封面→目录→正文→附录
3. 使用 doc_merge 合并 docx 文件，或手动拼装 Markdown
4. 使用 docx_write 输出最终文档

## 最终输出
- 返回完整报告文件路径
- 附报告结构说明

` + negativeClaimRule + `

` + tuiFormatting + `

父节点的 'task' 是文档拼装任务。不要偏离。`,
			Scope:        ScopeBuiltin,
			Path:         "(builtin)",
			RunAs:        RunSubagent,
			AllowedTools: []string{"doc_merge", "docx_write", "docx_read", "read_file", "write_file", "format_convert"},
		},
	}
}

// BuiltinNames returns the built-in skill names, used by callers that wire
// dedicated subagent tools for the subagent built-ins.
func BuiltinNames() []string {
	skills := builtinSkills()
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return names
}
