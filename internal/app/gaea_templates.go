package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// TaskTemplate 是预置办公任务模板（P1-③ 任务模板库）。
// 欢迎页「任务模板」区直接填入 Prompt；同名 slash 命令由 ensureTaskTemplateCommands
// 落盘为 .gaea/commands/<name>.md，复用既有自定义命令管线（/ 菜单 + Submit 解析）。
type TaskTemplate struct {
	Name        string `json:"name"`        // slash 命令名（如 weekly-report）
	Title       string `json:"title"`       // 展示名（如 周报）
	Description string `json:"description"` // 一句话说明
	Prompt      string `json:"prompt"`      // 结构化任务指令
}

// taskTemplates 是内置模板的唯一数据源（欢迎页 + slash 命令共用）。
var taskTemplates = []TaskTemplate{
	{
		Name: "weekly-report", Title: "周报", Description: "结构化周报：进展 / 数据 / 问题 / 下周计划",
		Prompt: "帮我生成一份本周工作周报：\n1. 先读取工作区的本周相关资料（如有），没有就基于以下要点组织；\n2. 按「本周进展 / 关键数据 / 遇到的问题 / 下周计划」四部分撰写；\n3. 输出为 Markdown 并保存到工作区 .gaea/exports/ 下（文件名含日期），正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "meeting-minutes", Title: "会议纪要", Description: "纪要模板：议题 / 结论 / 行动项（负责人 + 截止）",
		Prompt: "帮我整理一份会议纪要：\n1. 先说明会议主题、时间、参会人（缺失内容用「待补充」占位）；\n2. 按「议题与讨论 / 结论 / 行动项」组织，行动项必须包含负责人和截止时间；\n3. 输出为 Markdown 并保存到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "cost-estimate", Title: "成本测算", Description: "生成 xlsx 成本测算表：公式计算 + 原生图表",
		Prompt: "帮我制作一份成本测算表（.xlsx）：\n1. 先与我对齐测算范围和科目（人工/材料/机械/管理费/税费等）；\n2. 测算前先用 cost_search 查询成本库中的历史单价作为定价依据：命中的科目直接引用并在正文注明依据的条目名称，缺失科目与用户确认或给出合理估价并说明假设；\n3. 用 xlsx 能力创建表格：科目、单位、数量、单价、金额，金额用公式计算（数量×单价），并提供汇总行；\n4. 为费用构成生成原生图表（柱状/饼图）；\n5. 测算完成后用 cost_save 把本次采用的单价沉淀为成本条目（来源标注本次项目/文件，同名覆盖），并在正文汇报新增/更新条数；\n6. 保存到 .gaea/exports/ 并在正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "proposal-outline", Title: "方案大纲", Description: "方案结构：背景 / 目标 / 方案对比 / 实施 / 预算 / 风险",
		Prompt: "帮我撰写一份方案大纲：\n1. 按「背景与目标 / 现状分析 / 方案设计（含对比）/ 实施计划 / 预算与资源 / 风险与应对 / 附录」组织；\n2. 每部分先列要点，再按需展开成文；\n3. 输出为 Markdown（或按我要求导出 docx）并保存到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "data-analysis", Title: "数据分析", Description: "表格清洗 → 透视汇总 → 图表 → 结论",
		Prompt: "帮我做一份数据分析：\n1. 读取指定的表格数据（xlsx/csv），先说明数据口径并做清洗（缺失值/重复/格式）；\n2. 做分类汇总（按关键维度分组统计）；\n3. 生成合适的图表（柱状/折线/饼图）；\n4. 输出「数据概况 / 关键发现 / 建议」三段结论，并把结果保存到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "document-convert", Title: "文档转换", Description: "docx / xlsx / pdf 与 Markdown 互转，保留结构",
		Prompt: "帮我转换这份文档：\n1. 用 format_convert 把 docx/xlsx/pdf 转为 Markdown；\n2. 检查转换结果：标题层级、表格、公式是否保留，缺失内容手动补齐；\n3. 保存转换结果到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "report-assemble", Title: "报告拼装", Description: "多份素材合并为完整报告：封面 / 目录 / 正文 / 附录",
		Prompt: "帮我拼装一份完整报告：\n1. 先盘点提供的素材（@ 引用的文件），梳理可用内容与缺口；\n2. 按「封面 / 目录 / 正文 / 附录」结构组装，正文分章并保留来源标注；\n3. 输出为 Markdown 或 docx 并保存到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
	{
		Name: "ppt-deck", Title: "演示文稿", Description: "内容大纲 → PPT 成稿（.pptx）",
		Prompt: "帮我生成一份演示文稿（.pptx）：\n1. 先把内容整理为 8-12 页大纲（封面 / 目录 / 正文页 / 结束页）；\n2. 每页要点化表达，配一页图表（如有数据）；\n3. 用 pptx 技能生成 .pptx 保存到 .gaea/exports/，正文给出可点击的 [文件名](路径)。",
	},
}

// templateExportsRoot 返回模板 prompt 文本中的产物目录根段（S4 参数化）：
// work（缺省/空/非法值）= ".gaea/exports/"（现状逐字），play = ".gaea/play/exports/"。
const (
	templateExportsRootWork = ".gaea/exports/"
	templateExportsRootPlay = ".gaea/play/exports/"
)

func templateExportsRoot(space string) string {
	if spaces.Normalize(space) == spaces.SpacePlay {
		return templateExportsRootPlay
	}
	return templateExportsRootWork
}

// renderTaskTemplates 按会话空间渲染内置模板库（设计 §5 参数化）：play 空间把
// prompt 里的 .gaea/exports/ 根段替换为 .gaea/play/exports/；work 缺省输出与
// taskTemplates 原文逐字一致（欢迎页/命令面板的既有文本与测试锚定不动）。
func renderTaskTemplates(space string) []TaskTemplate {
	root := templateExportsRoot(space)
	out := make([]TaskTemplate, len(taskTemplates))
	for i, t := range taskTemplates {
		if root != templateExportsRootWork {
			t.Prompt = strings.ReplaceAll(t.Prompt, templateExportsRootWork, root)
		}
		out[i] = t
	}
	return out
}

// GaeaTaskTemplates 返回内置办公任务模板库（按当前生效空间渲染产物路径）。
func (a *App) GaeaTaskTemplates() []TaskTemplate {
	return renderTaskTemplates(gaeaEffectiveSpace())
}

// ensureTaskTemplateCommands 把模板落盘为 .gaea/commands/<name>.md（幂等）：
// 已存在的文件不覆盖（保留用户自己的修改/同名命令），让 / 菜单与 Submit
// 能通过既有自定义命令管线解析模板。命令文件是工作区级共享资产（跨空间
// 复用、只播种一次），恒用 work 缺省文本，不随会话空间渲染（S4）。
func ensureTaskTemplateCommands(cwd string) error {
	dir := filepath.Join(cwd, ".gaea", "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, t := range taskTemplates {
		path := filepath.Join(dir, t.Name+".md")
		if _, err := os.Stat(path); err == nil {
			continue // 用户已有同名文件：保留
		}
		content := fmt.Sprintf("---\ndescription: %s\n---\n\n%s", t.Description, t.Prompt)
		if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}
