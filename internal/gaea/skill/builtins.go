package skill

import (
	"github.com/gaea/gaea/internal/gaea/genui"
)

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
			Name:        "genui",
			Description: "生成式 UI 词汇手册：回答中用 ```genui 围栏输出卡片/表格/图表/表单/quiz 前必读，含组件词汇、JSON 自检与纪律。",
			Body:        genui.Handbook,
			Scope:       ScopeBuiltin,
			Path:        "(builtin)",
			RunAs:       RunInline,
		},
		{
			Name: "office-edit",
			Description: "办公文件编辑纪律总纲：创建/修改 xlsx、docx、pptx、markdown 思维导图大纲或 .gbase.json 多维表视图前必读——先读后写、写后回读验证、宁拒不误改、逐 run 编辑防丢格式、.gbase.json 列名口径与 JSON 自检、soffice 渲染取证与 Error [FORMAT_*] 错误码路由。",
			Body: `你正在创建或修改用户的办公文件。本技能是办公编辑纪律总纲：xlsx / docx / pptx 二进制文件一律用 bash + python 库处理；markdown 大纲（思维导图）与 .gbase.json（多维表视图）用 read_file / write_file / edit_file 处理。开工前先通读本技能，写完后按验证闭环收尾。

## 铁律（适用于一切写操作）

1. 先读后写：写之前必须读目标文件的现状（read_file，或 bash + python 读关键字段），禁止基于想象修改。
2. 工具成功 ≠ 正确：每次写入后必须回读验证——用 python 读回被改的字段做任务断言；断言不过就继续修，不许只汇报「已完成」。
3. 宁拒不误改：定位不到目标（old_string 不唯一、单元格/段落找不到）就停下向用户说明，绝不猜位置乱写。
4. 保留未触碰部分：能用定点修改（edit_file / openpyxl 单元格赋值）就不要整文件重写；二进制办公格式禁止用文本工具直改。
5. 负面结论给证据：说「没有 / 不存在」时附上查过什么（grep / 遍历结果）。

## 验证闭环（写 → 读回 → 看渲染）

- 结构回读：写后用同一 python 库读回被改范围，逐项对照任务要求。
- 渲染证据（视觉相关改动）：bash 运行 soffice --headless --convert-to pdf 把文件转 PDF，再 pdftoppm -png 出页图自查；截图只补视觉判断，不替代结构回读。
- 汇报口径：只声称实际验证过的结论；未验证的明说未验证。

## xlsx（bash + python openpyxl）

- 定点编辑：load_workbook 后按单元格赋值，禁止重建整个工作簿；保留既有样式、图表、数据验证、合并单元格。
- 公式：写入公式字符串（如 =SUM(B2:B3)），最终数值交给 LibreOffice 重算；不要用手算结果覆盖公式。
- 多维表视图（.gbase.json）：与 xlsx 同目录同名（报告.xlsx 的视图存 报告.gbase.json）。schema（渲染端字段级容错，坏项被丢弃并出横幅）：
    {"version":1,"views":[{"id":"v1","name":"按状态","type":"grid","sheet":"Sheet1","groupBy":"状态","filter":{"op":"and","conditions":[{"column":"金额","op":"gte","value":50}]},"sort":[{"column":"金额","dir":"desc"}],"colorRules":[{"column":"状态","op":"eq","value":"完成","color":"#RRGGBB"}]}]}
  - 视图按列名引用：column 必须与 xlsx 首行表头逐字一致，禁止按列序；groupBy 可省略（平铺）。
  - op 枚举：eq / ne / gt / gte / lt / lte / contains / empty / notEmpty；type 目前仅支持 grid；color 为 #RRGGBB。
  - 写前 JSON 自检四步：① 括号引号配平 ② 列名与表头逐字核对 ③ op 都在枚举内 ④ version=1。
  - 改 xlsx 列名/删列时，同步更新 .gbase.json 里引用该列的视图。

## docx（bash + python-docx）

- 改段落文本优先逐 run 替换（run.text 赋值）；整段 paragraph.text 赋值会摊平 runs 丢格式。
- 样式改动走样式对象，不硬编码字体名；表格定位先读回行列确认再写。

## pptx（bash + python-pptx）

- 同 docx：逐 run 编辑，禁止 paragraph.text 整段赋值。
- 改写文本长度与原文相近（演示文稿文本框防溢出）。
- 每改一页：回读该页全部文本做断言，并 soffice 渲染该页自查三件事——文本越界、容器溢出、文本重叠。

## 思维导图（markdown 大纲 = 权威格式）

- 写法：首个一级标题是根节点；二级/三级标题作次级分支；嵌套列表（- 加两空格缩进逐层）成叶层。
- 纪律：层级 ≤4；每节点 ≤20 字；同层至少 3 项（不足就合并）；相邻层结构多样不重复。
- 改图：read_file 后用 edit_file 定点改行，不要整文件重写（保持 diff 可读）。

## 错误路由（按 code 恢复，不解析散文）

- Error [FORMAT_INVALID_ARGS]：检查参数后重试一次。
- Error [FORMAT_SOURCE_MISSING]：源文件路径错，先 ls 确认再换路径；不要原路径重试。
- Error [FORMAT_UNSUPPORTED]：格式不支持，如实告知用户并给替代方案。
- Error [FORMAT_CONVERT_FAILED] / [FORMAT_OUTPUT_WRITE_FAILED]：读错误详情，换输出路径或报告用户。
- edit_file 报「未找到 / 多处匹配」：先 read_file 缩小上下文再改，不盲试。
- bash / python 报错：读完整 traceback 定位行号，修复后重跑；同类失败两次就换方案并说明原因。

` + negativeClaimRule + "\n\n" + tuiFormatting,
			Scope:       ScopeBuiltin,
			Path:        "(builtin)",
			RunAs:       RunInline,
		},
		{
			Name: "schedule-edit",
			Description: "进度计划编制与调整纪律：用户要编进度计划/排施工计划/改工期/调搭接/分析关键线路时必读——schedule_get 先读后写、schedule_apply 双通道（project 整计划/ops 增量）、CPM 引擎裁决 fail-closed、回执核对「成功≠正确」、工作日口径与 WBS 两级分解、JGJ/T 121-2015 先分解后编网。",
			Body: `你正在通过 schedule_get / schedule_apply / schedule_analyze 三工具编制或调整用户的工程进度计划（「进度计划」板块同款数据，双向实时同步）。开工前先通读本技能。

## 铁律

1. 先读后写：schedule_get 读现状（无文件=空计划），禁止基于想象修改用户已有计划。
2. 工具成功 ≠ 正确：schedule_apply 回执带写入后总工期与关键工作数，必须核对是否与预期一致；不一致就 schedule_analyze 找原因，修到对为止。
3. 引擎裁决：环依赖/悬空引用/非法字段会被整批拒绝并返回错误原文——读错误、修数据、重试；绝不绕过校验或手算工期冒充引擎结果。
4. 汇报口径：改动后向用户报告「改了什么（清单）→ 总工期 X→Y 天 → 关键线路变化 → 风险提示」，未验证的明说未验证。

## 编制流程（整计划生成，走 project 通道）

顺序对齐 JGJ/T 121-2015「先分解、后编网」：
1. 工作分解：WBS 两级——分组行 level=0（分部工程，如 施工准备/基础工程/主体结构/机电与装修/竣工验收），子任务 level=1（分项，命名动词+对象，如 基础钢筋绑扎）。分组行 duration=0、不参与搭接。
2. 逻辑关系：工艺逻辑（工序先后）+组织逻辑（流水段）；搭接 type 缺省 FS lag=0，流水搭接用 SS+时距，收口用里程碑（duration=0，isMilestone=true）。
3. 工期：工作日整数；类似工程经验值估列并说明依据；有合同/定额工期约束时先声明目标再倒排校核（倒排可用手动模式 manual+manualStart 锁定，注意 manual 不参与关键线路）。
4. 日历：缺省周一~五；用户给停工季/节假日就写进 calendar.holidays；工期全部按工作日计。
5. 写入：schedule_apply project 通道一次性写入 → 回执核对总工期 → schedule_analyze 质检（孤立任务/空分组/缺里程碑必须清零）→ 汇报。生成后主动问用户是否要调整（工期目标/资源限制/分段流水）。

## 调整流程（局部修改，走 ops 通道）

- 压缩/延长工期：patch_task{duration}——只改用户点名的任务；改完汇报总工期与关键线路变化（压缩非关键任务不缩总工期，向用户点破）。
- 改逻辑：set_links{toId,links}（整体替换该任务前置）；增删任务 upsert_task/remove_task（remove 级联子孙）。
- 多条调整一次 ops 数组下发（原子应用，单条失败整批拒绝）。
- 进度反馈：patch_task{progress}（0-100），前锋线在板块横道图查看。

## 推荐逻辑关系（编制环节最高痛点）

schedule_analyze 的 checks 会列出「无前置/孤立任务」——这就是推荐输入。推荐步骤：
1. 先 ops auto_chain 补确定性缺省（无前置叶任务按 WBS 顺序 FS 串联；已有逻辑/手动任务不动）。
2. 再按工艺逻辑覆写例外：并行流水改 SS+时距、工艺硬约束（如 钢筋绑扎 必须在 混凝土浇筑 前）逐条 set_links。
3. 每条推荐给理由（工艺先后/验收节点/合同约束）；用户可逐条否决，否决即回滚该条（apply 有快照+证据卡，可回退）。

## 分析报告模板（用 schedule_analyze 输出组织，四段式）

- **进度**：总工期 X 天；关键线路 A→B→C；里程碑 N 个最早第 Y 天达成。
- **风险**：近关键清单（TF≤2，缓冲小易变关键）；无前置/孤立任务（逻辑缺口服质）；日历停工窗口（节假日/停工季）对工期的放大。
- **盲点**：缺收口里程碑、多条并行尾巴、空分组（分解不到位的信号）。
- **建议**：1-3 条可执行项（压缩关键工序/改搭接提前开工/拆并行/补里程碑），说明预期总工期收益；不确定时给「先分析资源再定」的下一步而非硬编。

## 口径

- 工期/时距一律工作日；日期显示由开工日期+工作日历推导，汇报日期时说明是工作日序还是日历日期。
- 关键工作=总时差 0 且非手动；总时差≤2 为近关键，风险提示优先级高。
- 互通：可建议用户在板块里导入/导出 MS Project XML 与 Project/斑马互查。

` + negativeClaimRule,
			Scope:       ScopeBuiltin,
			Path:        "(builtin)",
			RunAs:       RunInline,
		},
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
- read_file / ls / bash：读取数据（xlsx/csv 用 bash + python 的 openpyxl/pandas 提取，再喂给 chart_gen）

## 操作方式
1. 确认数据类别和数值
2. 数据在表格文件（xlsx/csv）中时，先用 bash + python 提取为 labels/values
3. 选择合适的图表类型（bar/line/pie/scatter）
4. 使用 chart_gen 工具生成并保存图片

## 最终输出
- 返回图片文件路径
- 附图表类型和数据摘要

` + negativeClaimRule + `

` + tuiFormatting + `

父节点的 'task' 是图表生成任务。不要偏离。`,
			Scope:        ScopeBuiltin,
			Path:         "(builtin)",
			RunAs:        RunSubagent,
			AllowedTools: []string{"chart_gen", "read_file", "write_file", "bash", "ls"},
		},
		{
			Name:        "doc-assemble",
			Description: "文档拼装：将多份 Markdown 文档片段合并为完整报告，含封面、目录、正文、附录。",
			Body: `你作为文档拼装子代理运行。将多份文档素材拼装为完整报告。

## 可用工具
- read_file / ls：读取素材片段
- format_convert：将 docx/xlsx/pdf 源文件转成 Markdown 片段
- write_file：写出拼装好的 Markdown 报告
- bash：按需把最终 Markdown 转成 docx（全局 node 'docx' 库或 LibreOffice soffice）

## 操作方式
1. 收集所有文档片段（Markdown 或 docx）
2. 按报告结构组织：封面→目录→正文→附录
3. docx 源先用 format_convert 转为 Markdown
4. 手动拼装 Markdown，用 write_file 输出报告（默认 .md）
5. 用户明确要 .docx 时，再用 bash 调用 node 'docx' 或 soffice 转换

## 最终输出
- 返回完整报告文件路径
- 附报告结构说明

` + negativeClaimRule + `

` + tuiFormatting + `

父节点的 'task' 是文档拼装任务。不要偏离。`,
			Scope:        ScopeBuiltin,
			Path:         "(builtin)",
			RunAs:        RunSubagent,
			AllowedTools: []string{"read_file", "write_file", "format_convert", "bash", "ls"},
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
