/* eslint-disable react-refresh/only-export-components -- 任务模板加载/兜底函数导出供测试与复用 */
import {
  ArrowUpRight, BarChart3, BookOpen, Brain, Clock, FilePpt, FileText, FolderOpen,
  MessageSquare, RefreshCw, ScrollText, Sparkles, Table, Wand2,
} from "../icons";
import { useEffect, useState } from "react";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { app } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { sessionTitle } from "../lib/session";
import type { DictKey } from "../locales/en";
import type { Meta, SessionMeta, TaskTemplate } from "../lib/types";

// 相对时间（zh 原硬编码文案逐字收编）：translator 由渲染层传入，语言切换即时生效。
function formatTimeAgo(ms: number, t: Translator): string {
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60000);
  if (min < 1) return t("welcome.justNow");
  if (min < 60) return t("welcome.minAgo", { n: min });
  const hrs = Math.floor(min / 60);
  if (hrs < 24) return t("welcome.hourAgo", { n: hrs });
  return new Date(ms).toLocaleDateString([], { month: "short", day: "numeric" });
}

// ── 通用办公能力卡片 ──────────────────────────────────────────────
// name/desc 为用户可读 UI 文案 → 走字典；prompt 发给 LLM，保持原文不做 i18n。
interface OfficeCapability {
  icon: React.ReactNode;
  nameKey: DictKey;
  descKey: DictKey;
  prompt: string;
}

const OFFICE_CAPABILITIES: OfficeCapability[] = [
  {
    icon: <FileText size={17} />,
    nameKey: "welcome.capWrite",
    descKey: "welcome.capWriteDesc",
    prompt: "帮我写一份项目总结报告，包含背景、进展、问题和下一步计划，输出为 Markdown。",
  },
  {
    icon: <Table size={17} />,
    nameKey: "welcome.capSheet",
    descKey: "welcome.capSheetDesc",
    prompt: "帮我整理表格数据（xlsx/csv），做分类汇总并说明口径。",
  },
  {
    icon: <RefreshCw size={17} />,
    nameKey: "welcome.capConvert",
    descKey: "welcome.capConvertDesc",
    prompt: "把这份 docx/xlsx/pdf 文档转换成 Markdown，保留标题层级和表格。",
  },
  {
    icon: <BarChart3 size={17} />,
    nameKey: "welcome.capChart",
    descKey: "welcome.capChartDesc",
    prompt: "根据这份数据生成图表（柱状图/折线图），输出 PNG 图片。",
  },
  {
    icon: <ScrollText size={17} />,
    nameKey: "welcome.capReport",
    descKey: "welcome.capReportDesc",
    prompt: "把这几份文档素材拼装成一份完整的报告，包含封面、目录、正文和附录。",
  },
  {
    icon: <FilePpt size={17} />,
    nameKey: "welcome.capDeck",
    descKey: "welcome.capDeckDesc",
    prompt: "根据这份内容生成一份 PPT 演示文稿（.pptx），先列大纲再成稿。",
  },
  {
    icon: <Brain size={17} />,
    nameKey: "welcome.capKnowledge",
    descKey: "welcome.capKnowledgeDesc",
    prompt: "把这段内容加入知识库（分类、标签），方便以后检索复用。",
  },
];

// ── 内置技能 chips ────────────────────────────────────────────────
// label 为技能名（ASCII 字面量），sub 为用户可读说明 → 走字典；prompt 发给 LLM。
interface SkillChip {
  label: string;
  subKey: DictKey;
  prompt: string;
}

const OFFICE_SKILLS: SkillChip[] = [
  { label: "format-convert", subKey: "welcome.skillConvert", prompt: "用 run_skill 调用 format-convert 技能，把文档转换为可编辑 Markdown。" },
  { label: "chart-builder", subKey: "welcome.skillChart", prompt: "用 run_skill 调用 chart-builder 技能，从数据生成统计图表。" },
  { label: "doc-assemble", subKey: "welcome.skillAssemble", prompt: "用 run_skill 调用 doc-assemble 技能，把多份素材拼装成完整报告。" },
  { label: "docx", subKey: "welcome.skillDocx", prompt: "用 run_skill 调用 docx 技能创建或编辑 Word 文档。" },
  { label: "xlsx", subKey: "welcome.skillXlsx", prompt: "用 run_skill 调用 xlsx 技能创建或处理表格文件。" },
  { label: "pdf", subKey: "welcome.skillPdf", prompt: "用 run_skill 调用 pdf 技能读取、合并或创建 PDF 文档。" },
  { label: "pptx", subKey: "welcome.skillPptx", prompt: "用 run_skill 调用 pptx 技能把内容做成 PowerPoint 演示文稿。" },
];

// 内置任务模板兜底：后端命令库为空或加载失败时（首启/离线），欢迎页仍有常用办公任务可一键发起。
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

// resolveTemplates 空库/失败兜底：后端返回空或 null 时退回内置模板。
export function resolveTemplates(remote: TaskTemplate[] | null | undefined): TaskTemplate[] {
  return remote && remote.length > 0 ? remote : FALLBACK_TEMPLATES;
}

// 模块级缓存：欢迎页每次挂载（新建/切换会话时 items 清空）都会重新拉
// TaskTemplates；与命令面板「任务模板」组共用数据源，缓存避免重复请求。
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

// 测试辅助：清空缓存（vitest 隔离用例间状态）。
export function resetTemplatesCacheForTest(): void {
  templatesCache = null;
}

export function Welcome({
  onPrompt,
  cwd: _cwd,
  cwdName,
  sessions,
  onResumeSession,
  meta,
}: {
  onPrompt: (text: string) => void;
  cwd?: string;
  cwdName?: string;
  sessions?: SessionMeta[];
  onResumeSession?: (path: string) => Promise<void>;
  meta?: Meta;
}) {
  const t = useT();
  const compact = useCompact();
  const [templates, setTemplates] = useState<TaskTemplate[]>([]);
  useEffect(() => {
    let live = true;
    loadTemplates().then((ts) => { if (live) setTemplates(ts); });
    return () => { live = false; };
  }, []);
  const recentSessions = sessions?.filter((s) => !s.current).slice(0, 3) ?? [];

  return (
    <div className="h-full flex flex-col items-center max-w-4xl mx-auto px-6 overflow-y-auto welcome-shell">
      {/* 顶部工作区 pill */}
      <div className="welcome-rise welcome-rise-0 mt-8 mb-6">
        {cwdName && (
          <div className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-accent-soft border border-accent/20 text-fg-dim ${compact ? "text-[11px]" : "text-[12px]"}`}>
            <FolderOpen size={compact ? 12 : 13} className="text-accent" />
            <span className="font-medium text-accent">{cwdName}</span>
            {meta?.label && <span className="text-fg-faint">· {meta.label}</span>}
          </div>
        )}
      </div>

      {/* 主视觉：logo + 标题 */}
      <div className="welcome-rise welcome-rise-1 flex flex-col items-center text-center">
        <div className="relative mb-6">
          <img src={logoSvg} className={`rounded-[12px] welcome-logo dark:hidden ${compact ? "w-12 h-12" : "w-14 h-14"}`} alt="gaea" />
          <img src={logoLightSvg} className={`rounded-[12px] welcome-logo hidden dark:block ${compact ? "w-12 h-12" : "w-14 h-14"}`} alt="gaea" />
          <span className="absolute -right-1 -bottom-1 w-5 h-5 rounded-full bg-accent/15 border border-accent/25 flex items-center justify-center text-accent">
            <Sparkles size={10} />
          </span>
        </div>
        <h1
          className={`text-fg font-semibold leading-tight ${compact ? "text-[24px]" : "text-[30px]"}`}
          style={{ fontFamily: "var(--ds-font-display)", letterSpacing: "-0.02em" }}
        >
          {t("welcome.heading")}
        </h1>
        <p className={`text-fg-dim mt-2.5 ${compact ? "text-[12.5px]" : "text-[13.5px]"}`}>
          {t("welcome.tagline")}
        </p>
      </div>

      {/* 通用办公能力网格 */}
      <div className="welcome-rise welcome-rise-2 w-full mt-9">
        <div className={`flex items-center justify-between mb-3 ${compact ? "text-[10px]" : "text-[11px]"}`}>
          <span className="font-semibold text-fg-faint uppercase tracking-wider flex items-center gap-1.5">
            <BookOpen size={12} />
            {t("welcome.coreCaps")}
          </span>
          <span className="text-fg-faint/60">{t("welcome.clickToStart")}</span>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {OFFICE_CAPABILITIES.map((cap) => (
            <button
              key={cap.nameKey}
              onClick={() => onPrompt(cap.prompt)}
              className="group relative flex flex-col items-start text-left font-[inherit] bg-bg-elev border border-border-soft rounded-xl p-3.5 cursor-pointer transition-all duration-200 hover:border-accent/30 hover:bg-bg-elev hover:-translate-y-0.5 hover:shadow-[var(--ds-shadow-card)] overflow-hidden"
              title={cap.prompt}
            >
              <span className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/25 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
              <div className="flex items-center gap-2.5 w-full mb-2.5">
                <span className="w-8 h-8 rounded-lg bg-accent/10 border border-accent/15 text-accent flex items-center justify-center shrink-0 group-hover:bg-accent/15 transition-colors">
                  {cap.icon}
                </span>
                <span className={`font-semibold text-fg ${compact ? "text-[12.5px]" : "text-[13.5px]"}`}>{t(cap.nameKey)}</span>
                <ArrowUpRight size={12} className="ml-auto rotate-45 text-fg-faint/0 group-hover:text-accent transition-all group-hover:translate-x-0 group-hover:translate-y-0 -translate-x-0.5 translate-y-0.5" />
              </div>
              <p className={`text-fg-dim leading-relaxed line-clamp-2 ${compact ? "text-[11px]" : "text-[12px]"}`}>
                {t(cap.descKey)}
              </p>
            </button>
          ))}
        </div>
      </div>

      {/* 任务模板库：常见办公任务一键发起（与 slash 命令同源） */}
      {templates.length > 0 && (
        <div className="welcome-rise welcome-rise-3 w-full mt-6">
          <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-2.5 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
            <Sparkles size={12} />
            {t("welcome.templates")}
            <span className="text-fg-faint/60 normal-case tracking-normal font-normal">{t("welcome.templatesHint")}</span>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
            {templates.map((tm) => (
              <button
                key={tm.name}
                onClick={() => onPrompt(tm.prompt)}
                className="group flex flex-col items-start text-left font-[inherit] bg-bg-soft border border-border-soft rounded-lg p-2.5 cursor-pointer transition-all duration-200 hover:border-accent/35 hover:bg-bg-elev hover:-translate-y-0.5"
                title={tm.prompt}
              >
                <span className="flex items-center gap-1.5 w-full mb-1">
                  <span className={`font-semibold text-fg ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}>{tm.title}</span>
                  <span className="ml-auto font-mono text-[9px] text-accent/80 bg-accent/10 rounded px-1 py-px">/{tm.name}</span>
                </span>
                <span className="text-fg-faint leading-snug line-clamp-2 text-[10.5px]">{tm.description}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 内置技能 */}
      <div className="welcome-rise welcome-rise-3 w-full mt-6">
        <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-2.5 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
          <Wand2 size={12} />
          {t("welcome.skills")}
        </div>
        <div className="flex flex-wrap gap-2">
          {OFFICE_SKILLS.map((skill) => (
            <button
              key={skill.label}
              onClick={() => onPrompt(skill.prompt)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-border-soft bg-bg-soft text-fg-dim text-[11.5px] font-mono cursor-pointer hover:border-accent/30 hover:bg-accent/5 hover:text-fg transition-all"
              title={skill.prompt}
            >
              <span className="text-accent">{skill.label}</span>
              <span className="text-fg-faint/70 text-[10.5px]">· {t(skill.subKey)}</span>
            </button>
          ))}
        </div>
      </div>

      {/* 自由提问提示 */}
      <div className={`welcome-rise welcome-rise-3 w-full mt-6 px-3 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-fg-faint text-center ${compact ? "text-[11px]" : "text-[12px]"}`}>
        {t("welcome.freeInput")}
      </div>

      {/* 最近会话 */}
      {recentSessions.length > 0 && onResumeSession && (
        <div className="w-full mt-6 pt-5 border-t border-border-soft mb-10">
          <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-3 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
            <Clock size={12} />
            {t("welcome.recent")}
          </div>
          <div className="flex flex-col gap-2">
            {recentSessions.map((s) => (
              <button
                key={s.path}
                className={`group flex items-center gap-3 px-3.5 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-left font-[inherit] text-fg-dim hover:text-fg hover:bg-bg-elev hover:border-fg-faint/40 transition-all ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}
                onClick={() => void onResumeSession(s.path)}
              >
                <span className="w-7 h-7 rounded-md bg-accent/10 border border-accent/15 text-accent flex items-center justify-center shrink-0">
                  <MessageSquare size={compact ? 12 : 13} />
                </span>
                <span className="flex-1 truncate font-medium">{sessionTitle(s, t("welcome.untitledSession"))}</span>
                <span className={`text-fg-faint shrink-0 ${compact ? "text-[10px]" : "text-[11px]"}`}>{formatTimeAgo(s.modTime, t)}</span>
                <ArrowUpRight size={11} className="rotate-45 text-fg-faint/0 group-hover:text-accent transition-colors shrink-0" />
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
