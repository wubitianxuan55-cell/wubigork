import {
  ArrowUpRight, BarChart3, BookOpen, Brain, Clock, FilePpt, FileText, FolderOpen,
  MessageSquare, RefreshCw, ScrollText, Sparkles, Table, Wand2,
} from "../icons";
import { useEffect, useState } from "react";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { sessionTitle } from "../lib/session";
import type { Meta, SessionMeta, TaskTemplate } from "../lib/types";

function formatTimeAgo(ms: number): string {
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min}分钟前`;
  const hrs = Math.floor(min / 60);
  if (hrs < 24) return `${hrs}小时前`;
  return new Date(ms).toLocaleDateString([], { month: "short", day: "numeric" });
}

// ── 通用办公能力卡片 ──────────────────────────────────────────────
interface OfficeCapability {
  icon: React.ReactNode;
  name: string;
  desc: string;
  prompt: string;
}

const OFFICE_CAPABILITIES: OfficeCapability[] = [
  {
    icon: <FileText size={17} />,
    name: "文档撰写",
    desc: "报告、方案、公文，Word 与 Markdown 一键成稿",
    prompt: "帮我写一份项目总结报告，包含背景、进展、问题和下一步计划，输出为 Markdown。",
  },
  {
    icon: <Table size={17} />,
    name: "表格处理",
    desc: "xlsx / csv 数据整理、公式计算与分类汇总",
    prompt: "帮我整理表格数据（xlsx/csv），做分类汇总并说明口径。",
  },
  {
    icon: <RefreshCw size={17} />,
    name: "格式转换",
    desc: "docx / xlsx / pdf 与 Markdown 互转，保留结构",
    prompt: "把这份 docx/xlsx/pdf 文档转换成 Markdown，保留标题层级和表格。",
  },
  {
    icon: <BarChart3 size={17} />,
    name: "图表生成",
    desc: "柱状、折线、饼图、散点图，数据可视化",
    prompt: "根据这份数据生成图表（柱状图/折线图），输出 PNG 图片。",
  },
  {
    icon: <ScrollText size={17} />,
    name: "报告拼装",
    desc: "多份素材合并为完整报告，含封面、目录、附录",
    prompt: "把这几份文档素材拼装成一份完整的报告，包含封面、目录、正文和附录。",
  },
  {
    icon: <FilePpt size={17} />,
    name: "演示文稿",
    desc: "PPT 大纲与成稿，汇报材料一键生成",
    prompt: "根据这份内容生成一份 PPT 演示文稿（.pptx），先列大纲再成稿。",
  },
  {
    icon: <Brain size={17} />,
    name: "知识沉淀",
    desc: "规范、结论存入知识库，跨会话复用",
    prompt: "把这段内容加入知识库（分类、标签），方便以后检索复用。",
  },
];

// ── 内置技能 chips ────────────────────────────────────────────────
interface SkillChip {
  label: string;
  sub: string;
  prompt: string;
}

const OFFICE_SKILLS: SkillChip[] = [
  { label: "format-convert", sub: "格式转换", prompt: "用 format-convert 把文档转换为可编辑 Markdown。" },
  { label: "chart-builder", sub: "图表生成", prompt: "用 chart-builder 从数据生成统计图表。" },
  { label: "doc-assemble", sub: "文档拼装", prompt: "用 doc-assemble 把多份素材拼装成完整报告。" },
  { label: "docx", sub: "Word 文档", prompt: "用 docx 技能创建或编辑 Word 文档。" },
  { label: "xlsx", sub: "表格", prompt: "用 xlsx 技能创建或处理表格文件。" },
  { label: "pdf", sub: "PDF 文档", prompt: "用 pdf 技能读取、合并或创建 PDF 文档。" },
  { label: "pptx", sub: "演示文稿", prompt: "用 pptx 技能把内容做成 PowerPoint 演示文稿。" },
];

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
    app.TaskTemplates()
      .then((ts) => { if (live) setTemplates(ts ?? []); })
      .catch(() => {});
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
        <div
          className="text-fg-faint uppercase tracking-[0.18em] text-[10.5px] mb-2 flex items-center gap-1.5"
        >
          <Wand2 size={11} />
          GAEA OFFICE
        </div>
        <h1
          className={`text-fg font-semibold leading-tight ${compact ? "text-[24px]" : "text-[30px]"}`}
          style={{ fontFamily: "var(--ds-font-display)", letterSpacing: "-0.02em" }}
        >
          今天想做什么？
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
            核心能力
          </span>
          <span className="text-fg-faint/60">点击卡片快速发起</span>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {OFFICE_CAPABILITIES.map((cap) => (
            <button
              key={cap.name}
              onClick={() => onPrompt(cap.prompt)}
              className="group relative flex flex-col items-start text-left font-[inherit] bg-bg-elev border border-border-soft rounded-xl p-3.5 cursor-pointer transition-all duration-200 hover:border-accent/30 hover:bg-bg-elev hover:-translate-y-0.5 hover:shadow-[var(--ds-shadow-card)] overflow-hidden"
              title={cap.prompt}
            >
              <span className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-accent/25 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
              <div className="flex items-center gap-2.5 w-full mb-2.5">
                <span className="w-8 h-8 rounded-lg bg-accent/10 border border-accent/15 text-accent flex items-center justify-center shrink-0 group-hover:bg-accent/15 transition-colors">
                  {cap.icon}
                </span>
                <span className={`font-semibold text-fg ${compact ? "text-[12.5px]" : "text-[13.5px]"}`}>{cap.name}</span>
                <ArrowUpRight size={12} className="ml-auto rotate-45 text-fg-faint/0 group-hover:text-accent transition-all group-hover:translate-x-0 group-hover:translate-y-0 -translate-x-0.5 translate-y-0.5" />
              </div>
              <p className={`text-fg-dim leading-relaxed line-clamp-2 ${compact ? "text-[11px]" : "text-[12px]"}`}>
                {cap.desc}
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
            任务模板
            <span className="text-fg-faint/60 normal-case tracking-normal font-normal">/weekly-report、/cost-estimate 等斜杠命令同样可用</span>
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
          内置技能
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
              <span className="text-fg-faint/70 text-[10.5px]">· {skill.sub}</span>
            </button>
          ))}
        </div>
      </div>

      {/* 自由提问提示 */}
      <div className={`welcome-rise welcome-rise-3 w-full mt-6 px-3 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-fg-faint text-center ${compact ? "text-[11px]" : "text-[12px]"}`}>
        或直接输入你的办公需求，开始对话
      </div>

      {/* 最近会话 */}
      {recentSessions.length > 0 && onResumeSession && (
        <div className="w-full mt-6 pt-5 border-t border-border-soft mb-10">
          <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-3 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
            <Clock size={12} />
            最近会话
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
                <span className="flex-1 truncate font-medium">{sessionTitle(s, "未命名会话")}</span>
                <span className={`text-fg-faint shrink-0 ${compact ? "text-[10px]" : "text-[11px]"}`}>{formatTimeAgo(s.modTime)}</span>
                <ArrowUpRight size={11} className="rotate-45 text-fg-faint/0 group-hover:text-accent transition-colors shrink-0" />
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
