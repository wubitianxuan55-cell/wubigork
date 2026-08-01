import { FolderOpen, MessageSquare, Clock, ScrollText, BarChart3, FileSpreadsheet, FileImage, Puzzle, BookOpen, ClipboardList } from "../icons";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { sessionTitle } from "../lib/session";
import type { Meta, SessionMeta } from "../lib/types";

function formatTimeAgo(ms: number): string {
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min}分钟前`;
  const hrs = Math.floor(min / 60);
  if (hrs < 24) return `${hrs}小时前`;
  return new Date(ms).toLocaleDateString([], { month: "short", day: "numeric" });
}

// ── 工程技能模块 ────────────────────────────────────────────────────
interface SkillCard {
  icon: React.ReactNode;
  name: string;
  desc: string;
  badge: string;   // 类型徽章
  prompt: string;
}

const SKILL_MODULES: SkillCard[] = [
  {
    icon: <ClipboardList size={18} />,
    name: "场地环境调查",
    desc: "初调/详调报告框架，含场地基本信息、历史使用、周边敏感目标",
    badge: "🧬 子代理",
    prompt: "启动场地环境调查初调报告框架，梳理场地基本信息、历史使用情况、周边敏感目标等关键内容。",
  },
  {
    icon: <FileSpreadsheet size={18} />,
    name: "投标方案编制",
    desc: "10章技术标书：工程概况、施工组织、质量控制、安全文明",
    badge: "📄 文档",
    prompt: "启动投标方案编制，生成土壤修复工程技术标投标方案，包括工程概况、施工组织设计、质量保证措施等。",
  },
  {
    icon: <Puzzle size={18} />,
    name: "修复方案设计",
    desc: "工艺参数 + 设备选型 + 施工部署 + 监测验收全流程",
    badge: "🧬 子代理",
    prompt: "撰写污染土壤修复实施方案，包括技术比选、工艺设计、施工部署等内容。",
  },
  {
    icon: <BarChart3 size={18} />,
    name: "数据报告生成",
    desc: "导入检测数据，统计分析、超标识别、空间分布一键生成",
    badge: "📊 图表",
    prompt: "导入检测数据CSV文件，进行统计分析，包括超标识别、空间分布、统计描述等。",
  },
  {
    icon: <ScrollText size={18} />,
    name: "污染风险评估",
    desc: "对标 GB 36600 / GB 15618 进行超标判定与风险计算",
    badge: "🧬 子代理",
    prompt: "根据提供的检测数据，对标 GB 36600 和 GB 15618 进行超标判定，计算污染风险并给出评估结论。",
  },
  {
    icon: <BookOpen size={18} />,
    name: "成本测算",
    desc: "七项汇总成本：钻孔/检测/药剂/土方/设备/人工/评估",
    badge: "📊 图表",
    prompt: "生成土壤修复工程七项汇总成本测算表，涵盖场地调查、风险评估、方案设计、施工实施、监测验收等各阶段费用。",
  },
  {
    icon: <FileImage size={18} />,
    name: "图表生成",
    desc: "柱状/折线/饼图/散点图，自动检测中文字体，PNG/SVG 输出",
    badge: "📊 图表",
    prompt: "根据数据分析结果生成可视化图表，支持柱状图、折线图、饼图、散点图等多种类型。",
  },
  {
    icon: <MessageSquare size={18} />,
    name: "文档汇总",
    desc: "多份 docx 合并、格式转换、PPT 演示稿一键生成",
    badge: "📄 文档",
    prompt: "汇总多份文档资料，合并生成综合报告或演示文稿。",
  },
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
  const recentSessions = sessions?.filter(s => !s.current).slice(0, 3) ?? [];

  return (
    <div className="h-full flex flex-col items-center max-w-2xl mx-auto px-6 overflow-y-auto pt-16">
      {cwdName && (
        <div className={`inline-flex items-center gap-2 px-3 py-1.5 mb-5 rounded-full bg-accent-soft border border-accent/20 text-fg-dim ${compact ? "text-[11px]" : "text-[12px]"}`}>
          <FolderOpen size={compact ? 12 : 13} className="text-accent" />
          <span className="font-medium text-accent">{cwdName}</span>
          {meta?.label && <span className="text-fg-faint">· {meta.label}</span>}
        </div>
      )}
      <div className="welcome-stagger-1">
        <img src={logoSvg} className={`rounded-[10px] mb-5 welcome-logo dark:hidden ${compact ? "w-8 h-8" : "w-10 h-10"}`} alt="gaea" />
        <img src={logoLightSvg} className={`rounded-[10px] mb-5 welcome-logo hidden dark:block ${compact ? "w-8 h-8" : "w-10 h-10"}`} alt="gaea" />
      </div>
      <div className={`welcome-stagger-2 text-fg-dim mb-8 text-center ${compact ? "text-[13px]" : "text-[14px]"}`} style={{fontFamily: "var(--ds-font-display)", fontWeight: 500, letterSpacing: "-0.01em"}}>
        {t("welcome.tagline")}
      </div>

      {/* ── 技能模块网格 ─────────────────────────────────────────────── */}
      <div className="welcome-stagger-3 w-full">
        <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-3 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
          <Puzzle size={12} />
          工程技能模块
        </div>
        <div className="grid grid-cols-2 gap-3">
          {SKILL_MODULES.map((skill) => (
            <button
              key={skill.name}
              onClick={() => onPrompt(skill.prompt)}
              className={`group flex flex-col items-start text-left font-[inherit] bg-bg-elev border border-border-soft rounded-xl p-3.5 cursor-pointer transition-all hover:border-accent/25 hover:bg-bg-elev hover:-translate-y-px hover:shadow-[var(--ds-shadow-card)] ${compact ? "p-3" : "p-3.5"}`}
              title={skill.prompt}
            >
              <div className="flex items-center gap-2 w-full mb-2">
                <span className="text-accent shrink-0">{skill.icon}</span>
                <span className={`font-semibold text-fg truncate ${compact ? "text-[12px]" : "text-[13px]"}`}>{skill.name}</span>
                <span className={`ml-auto text-[10px] text-fg-faint whitespace-nowrap px-1.5 py-0.5 rounded-full bg-bg-soft border border-border-soft ${compact ? "hidden" : ""}`}>
                  {skill.badge}
                </span>
              </div>
              <p className={`text-fg-dim leading-relaxed line-clamp-2 ${compact ? "text-[11px]" : "text-[12px]"}`}>
                {skill.desc}
              </p>
            </button>
          ))}
        </div>
      </div>

      {/* ── 自由提问提示 ─────────────────────────────────────────────── */}
      <div className={`welcome-stagger-3 w-full mt-5 px-3 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-fg-faint text-center ${compact ? "text-[11px]" : "text-[12px]"}`}>
        或直接输入工程问题，开始对话
      </div>

      {/* ── 最近会话 ─────────────────────────────────────────────────── */}
      {recentSessions.length > 0 && onResumeSession && (
        <div className="w-full mt-5 pt-4 border-t border-border-soft mb-8">
          <div className={`font-semibold text-fg-faint uppercase tracking-wider mb-2.5 flex items-center gap-1.5 ${compact ? "text-[10px]" : "text-[11px]"}`}>
            <Clock size={12} />
            最近会话
          </div>
          <div className="flex flex-col gap-1.5">
            {recentSessions.map((s) => (
              <button
                key={s.path}
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-left font-[inherit] text-fg-dim hover:text-fg hover:bg-bg-elev hover:border-fg-faint transition-all ${compact ? "text-[11px]" : "text-[12px]"}`}
                onClick={() => void onResumeSession(s.path)}
              >
                <MessageSquare size={compact ? 12 : 13} className="text-fg-faint shrink-0" />
                <span className="flex-1 truncate font-medium">{sessionTitle(s, "未命名会话")}</span>
                <span className={`text-fg-faint shrink-0 ${compact ? "text-[10px]" : "text-[11px]"}`}>{formatTimeAgo(s.modTime)}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
