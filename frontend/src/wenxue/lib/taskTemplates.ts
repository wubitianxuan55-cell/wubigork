import { app } from "./bridge";
import type { TaskTemplate } from "./types";

// 内置任务模板兜底：后端命令库为空或加载失败时（首启/离线），命令面板仍有常用任务。
export const FALLBACK_TEMPLATES: TaskTemplate[] = [
  {
    name: "weekly-report",
    title: "周报",
    description: "结构化周报：进展 / 数据 / 问题 / 下周计划",
    prompt: "帮我生成一份本周工作周报：按「本周进展 / 关键数据 / 遇到的问题 / 下周计划」四部分撰写，输出 Markdown 并保存到 .gaea/exports/。",
  },
  {
    name: "meeting-minutes",
    title: "会议纪要",
    description: "纪要模板：议题 / 结论 / 行动项",
    prompt: "帮我整理一份会议纪要：按「议题与讨论 / 结论 / 行动项」组织，行动项包含负责人和截止时间。",
  },
  {
    name: "cost-estimate",
    title: "成本测算",
    description: "生成 xlsx 成本测算表：公式 + 图表",
    prompt: "帮我制作一份成本测算表（.xlsx）：先对齐测算范围和科目，测算前用 cost_search 查询成本库历史单价作为依据，用 xlsx 能力创建科目/单位/数量/单价/金额表格（金额用公式），生成费用构成图表，完成后用 cost_save 沉淀本次单价，保存到 .gaea/exports/。",
  },
  {
    name: "proposal-outline",
    title: "方案大纲",
    description: "背景 / 目标 / 方案对比 / 实施 / 预算 / 风险",
    prompt: "帮我撰写一份方案大纲：按「背景与目标 / 现状分析 / 方案设计 / 实施计划 / 预算 / 风险」组织。",
  },
  {
    name: "data-analysis",
    title: "数据分析",
    description: "清洗 → 透视 → 图表 → 结论",
    prompt: "帮我做一份数据分析：清洗数据 → 分类汇总 → 生成图表 → 输出结论。",
  },
  {
    name: "document-convert",
    title: "文档转换",
    description: "docx / xlsx / pdf 与 Markdown 互转",
    prompt: "帮我转换这份文档：用 format_convert 转为 Markdown 并保留标题层级与表格。",
  },
  {
    name: "report-assemble",
    title: "报告拼装",
    description: "多素材合并为完整报告",
    prompt: "帮我拼装一份完整报告：封面 / 目录 / 正文 / 附录，保留来源标注。",
  },
  {
    name: "ppt-deck",
    title: "演示文稿",
    description: "大纲 → PPT 成稿（.pptx）",
    prompt: "帮我生成一份演示文稿（.pptx）：先列 8-12 页大纲再成稿。",
  },
];

export function resolveTemplates(remote: TaskTemplate[] | null | undefined): TaskTemplate[] {
  return remote && remote.length > 0 ? remote : FALLBACK_TEMPLATES;
}

let templatesCache: TaskTemplate[] | null = null;

/** 拉取任务模板（带缓存）：远端成功 → 缓存；失败/空 → 内置模板。 */
export async function loadTemplates(): Promise<TaskTemplate[]> {
  if (templatesCache) return templatesCache;
  try {
    const ts = await app.TaskTemplates();
    templatesCache = resolveTemplates(ts);
  } catch {
    templatesCache = FALLBACK_TEMPLATES;
  }
  return templatesCache;
}

export function resetTemplatesCacheForTest(): void {
  templatesCache = null;
}
