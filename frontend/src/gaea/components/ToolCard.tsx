import { memo, useMemo, useRef, useState } from "react";
import {
  Ban,
  Check,
  ChevronRight,
  Loader2,
  Users,
  X,
} from "../icons";


import { ICONS, mcpOr } from "./tool_icons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { boundedOutput, diffStatFor, diffsFor, subjectOf, summarize } from "../lib/tools";
import { getTaskCardActivity, getTaskCardOpenTarget, hasTaskCardActivityProvider, openTaskCardSession, resolveTaskRef, taskResultSummary } from "../lib/taskActivity";
import { formatElapsed } from "../lib/time";
import { useNow } from "../lib/useNow";
import type { Item } from "../lib/store";
import { FileLinkText } from "./FileLinkText";

type ToolItem = Extract<Item, { kind: "tool" }>;

function pretty(json: string): string {
  try {
    return JSON.stringify(JSON.parse(json), null, 2);
  } catch {
    return json;
  }
}

function StatusGlyph({ status, recoverable }: { status: ToolItem["status"]; recoverable?: boolean }) {
  if (status === "running") return <Loader2 className="animate-spin" size={12} />;
  if (status === "error") return <X className={recoverable ? "text-fg-faint/60" : "text-err"} size={12} />;
  if (status === "stopped") return <Ban className="text-fg-faint" size={12} />;
  return <Check className="text-ok" size={12} />;
}

// ── TaskLiveRow：task 工具卡运行中的实时活动行（v4.26 对齐 Codex）──────
// Why：子代理跑起来后主窗此前只有一张静态「运行中」task 卡——子代理的
// lastText/lastTool 只在右栏分工面板可见。本行把 GaeaAgentNetwork/SubagentRuns
// 轮询数据经 setTaskCardActivityProvider 注入的动态渲染成 Codex 式活动预览
// （单行截断），并补已用时 + 「查看分工」入口提示。
// How：ref 解析（args.continue_from / output 引用行，派发初期为 ""）→
// getTaskCardActivity 取动态；未注入 provider（null=按现状渲染契约）→ 整行
// 不渲染；查不到动态 → 只显示已用时与提示，绝不报错。1s tick（useNow）内刷新。
function TaskLiveRow({ item }: { item: ToolItem }) {
  const t = useT();
  const now = useNow();
  // 起跑时刻：Item 契约无时间戳，以挂载时刻近似计时（运行中卡片常驻挂载；
  // 恢复历史会话时 running 一律还原为 stopped，不会进入本行）。
  const startRef = useRef(Date.now());
  const ref = useMemo(() => resolveTaskRef(item.args, item.output), [item.args, item.output]);
  // 1s tick 内取值：App 层轮询更新 provider 数据后最迟 1s 上屏。
  // args 原样透传：派发初期 ref 为空串时，App 侧 provider 可用 args 里的
  // 任务描述文本与并行各 run.task 做唯一命中匹配（taskActivity 契约）。
  const activity = getTaskCardActivity(ref, item.args);
  // v4.63：运行中活动行本身也是「打开会话」入口（与整卡点击同语义）。
  const openRef = ref || getTaskCardOpenTarget("", item.args);
  // 未注入活动数据源（null）：按现状渲染（v4.26 之前没有这一行）
  if (!hasTaskCardActivityProvider()) return null;
  const elapsed = formatElapsed(Math.max(0, now - Math.floor(startRef.current / 1000)));

  return (
    <div
      data-testid="task-live"
      className={`flex items-center gap-1.5 px-2 pb-1 pl-7 text-[11px] text-fg-faint select-none ${openRef ? "cursor-pointer hover:text-fg-dim" : ""}`}
      title={openRef ? t("tool.openSessionHint") : undefined}
      onClick={openRef ? () => { openTaskCardSession(openRef); } : undefined}
    >
      {activity?.lastText && (
        <span className="min-w-0 truncate text-fg-dim/80" title={activity.lastText}>
          {activity.lastText}
        </span>
      )}
      {activity?.lastTool && (
        <span className="min-w-0 shrink-0 max-w-[30%] truncate font-mono" title={activity.lastTool}>
          {activity.lastTool}
        </span>
      )}
      <span className="ml-auto shrink-0 tabular-nums">{elapsed}</span>
      <span className="inline-flex shrink-0 items-center gap-1">
        <Users size={11} className="shrink-0" />
        {t("tool.openSessionHint")}
      </span>
    </div>
  );
}

export const ToolCard = memo(function ToolCard({ item, subcalls }: { item: ToolItem; subcalls?: ToolItem[] }) {
  const t = useT();
  const compact = useCompact();
  const diffs = useMemo(() => diffsFor(item.name, item.args), [item.name, item.args]);
  const subject = useMemo(() => subjectOf(item.name, item.args), [item.name, item.args]);
  // Codex 式 diffstat 芯片：编辑类工具行内显示 +N−M（不再重复进斜体摘要）
  const stat = useMemo(() => diffStatFor(item.name, item.args), [item.name, item.args]);
  const Icon = ICONS[item.name] ?? mcpOr(item.name);
  const nested = subcalls ?? [];
  const hasNested = nested.length > 0;

  const summary =
    item.status === "running"
      ? ""
      : item.name === "task"
        // v4.26：task 卡完成后显示子代理结果摘要（summarize 对 task 返回空，
        // 完成卡此前除嵌套步数外没有任何结果信息；error 走卡片错误区）。
        ? (taskResultSummary(item.output, item.error) ||
           (hasNested ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length }) : ""))
        : hasNested
          ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length })
          : summarize(item.name, item.args, item.output, item.error);

  const hasArgs = diffs.length > 0 || !!item.args;
  const hasOutput = !!item.output;
  const expandable = hasArgs || hasOutput;

  const [open, setOpen] = useState(false);
  // P2-2 大工具输出有界预览：超长输出折叠为头部 + 展开全部开关
  const bounded = useMemo(() => boundedOutput(item.output), [item.output]);
  const [showFullOutput, setShowFullOutput] = useState(false);

  const bodyRef = useRef<HTMLDivElement>(null);
  useGSAPCollapse(bodyRef, open && expandable);

  const quiet =
    item.readOnly && !hasNested && item.status !== "error" && item.status !== "stopped";

  const outputLines = item.output ? item.output.split("\n").length : 0;

  // v4.63 子代理卡片整卡可点：task / run_skill 卡解析出可跳转的子代理 ref
  // 时，点击头部行直接打开对应会话 tab（与右栏任务树同款跳转），不再只是
  // 折叠展开。空目标 = 不可点（诚实维持现状）。
  const isSubagentCard = item.name === "task" || item.name === "run_skill";
  const openTarget = isSubagentCard
    ? getTaskCardOpenTarget(resolveTaskRef(item.args, item.output), item.args)
    : "";
  const clickable = isSubagentCard && !!openTarget;

  const rowPy = compact ? "py-0" : "py-0.5";
  const rowPx = compact ? "px-1" : "px-2";
  const chevronSize = compact ? 11 : 12;
  const summarySize = compact ? "text-[11px]" : "text-[12px]";
  const innerPx = compact ? "px-1" : "px-2";
  const innerPb = compact ? "pb-1" : "pb-1.5";

  // Codex 式工具行：无边框无背景块，hover 才高亮；只读工具安静降噪。
  return (
    <div
      className={`rounded-md transition-colors duration-150 ${
        expandable ? "hover:bg-(color:--md-sys-color-surface-container-high)" : ""
      } ${item.status === "stopped" ? "opacity-60" : ""}`}
      data-tone={item.status === "error" && !item.recoverable ? "danger" : item.status === "running" ? "info" : item.status === "done" ? "success" : item.status === "stopped" ? "warning" : undefined}
    >
      <div
        className={`flex items-center gap-1.5 ${rowPx} ${rowPy} select-none ${
          expandable ? "cursor-pointer hover:bg-bg-soft" : ""
        } ${quiet ? "text-fg-faint/60" : "text-fg-dim"}`}
        onClick={
          clickable
            ? () => { openTaskCardSession(openTarget); }
            : expandable
              ? () => setOpen((v) => !v)
              : undefined
        }
        title={clickable ? t("tool.openSessionHint") : undefined}
      >
        {expandable ? (
          <ChevronRight
            className={`shrink-0 transition-transform duration-200 ${open ? "rotate-90" : ""}`}
            size={chevronSize}
          />
        ) : (
          <ChevronRight className="shrink-0 invisible" size={chevronSize} />
        )}
        <Icon
          className={`shrink-0 ${item.status === "error" && !item.recoverable ? "text-err" : item.status === "error" && item.recoverable ? "text-fg-faint/60" : item.status === "running" ? "text-accent" : "text-fg-faint"}`}
          size={chevronSize + 2}
        />
        <span className={`font-mono font-medium truncate ${item.status === "error" && !item.recoverable ? "text-err" : item.status === "error" && item.recoverable ? "text-fg-dim/60 line-through" : quiet ? "text-fg-faint/70" : "text-fg"} ${compact ? "text-[11px]" : "text-[12px]"}`}>
          {item.name}
        </span>
        {subject && (
          <span className={`text-fg-faint truncate ${summarySize}`}>{subject}</span>
        )}
        {stat && (stat.add > 0 || stat.del > 0) ? (
          <span
            className="shrink-0 ml-1 inline-flex items-center gap-1 rounded px-1 py-px font-mono text-[10.5px] leading-none bg-bg-soft border border-border-soft text-fg-dim tabular-nums"
            title={t("tool.diffStatTitle")}
          >
            <span className="text-ok">+{stat.add}</span>
            <span className="text-err">−{stat.del}</span>
          </span>
        ) : (
          summary && (
          <span className={`text-fg-faint italic ml-0.5 ${summarySize}`}>{summary}</span>
          )
        )}
        <span className="ml-auto shrink-0">
          <StatusGlyph status={item.status} recoverable={item.recoverable} />
        </span>
      </div>

      {/* v4.26 子代理 task 卡 live 化：运行中在头部行下方渲染实时活动行
          （活动预览 / 已用时 / 查看分工），不进折叠区、常驻可见。 */}
      {item.name === "task" && item.status === "running" && <TaskLiveRow item={item} />}

      <div ref={bodyRef} style={{ overflow: "hidden" }}>
        <div>
          {diffs.map((d, i) => (
            <div className={`${innerPx} ${innerPb}`} key={i}>
              {d.label && <div className="text-[10px] text-fg-faint uppercase tracking-wider mb-0.5">{d.label}</div>}
              <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code>{d.original}</code></pre>
            </div>
          ))}

          {hasNested && (
            <div className="pl-3 border-l border-border-soft ml-3">
              {nested.map((c) => (
                <ToolCard key={c.id} item={c} />
              ))}
            </div>
          )}

          {hasArgs && (
            <div className={`${innerPx} ${innerPb}`}>
              {item.args && <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code>{pretty(item.args)}</code></pre>}
            </div>
          )}
          {hasOutput && (
            <div className={`${innerPx} ${innerPb}`}>
              <div className="text-[9px] text-fg-faint/60 uppercase tracking-wider mb-0.5 select-none">{t("tool.outputHeader", { n: outputLines })}</div>
              <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code><FileLinkText text={showFullOutput ? bounded.full : bounded.preview} compact /></code></pre>
              {bounded.collapsed && (
                <button
                  type="button"
                  className="mt-1 px-2 py-0.5 border border-border-soft rounded bg-bg-soft text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
                  onClick={() => setShowFullOutput((v) => !v)}
                >
                  {showFullOutput ? t("tool.collapseOutput") : t("tool.expandAllLines", { n: bounded.hiddenLines })}
                </button>
              )}
              {item.truncated && (
                <div className="mt-1 px-2 py-0.5 border border-border-soft rounded bg-bg-soft text-fg-dim text-[11px]">
                  {t("tool.truncated")}
                </div>
              )}
            </div>
          )}

          {item.error && !item.recoverable && (
            <div className={`${innerPx} py-1 text-err text-[12px] leading-snug border-t border-err/20`}>
              {item.error}
            </div>
          )}
          {item.error && item.recoverable && (
            <div className={`${innerPx} py-1 text-fg-faint/60 text-[12px] leading-snug border-t border-fg-faint/15`}>
              {item.error}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
