import { memo, useId, useMemo, useRef, useState } from "react";
import {
  Ban,
  Bot,
  Check,
  ChevronRight,
  Clock,
  LineChart,
  Loader2,
  X,
} from "../icons";


import { mcpOr } from "./tool_icons";
import { lxToolIcon } from "./lingxiIcons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { emitFrontendEvent, FRONTEND_EVENTS } from "../../events";
import {
  boundedOutput,
  commandOf,
  diffStatFor,
  diffsFor,
  READER_JSON_TOOLS,
  subjectOf,
  summarize,
  TOOL_CARD_OUTPUT_PREVIEW_LINES,
  unifiedRows,
} from "../lib/tools";
import type { DiffRow } from "../lib/diff";
import { fireTaskCardAmbiguity, getTaskCardActivity, getTaskCardAmbiguity, getTaskCardOpenTarget, hasTaskCardAmbiguityHandler, isDelegationCall, openTaskCardSession, resolveTaskRef, taskResultSummary } from "../lib/taskActivity";
import { formatElapsed } from "../lib/time";
import { useNow } from "../lib/useNow";
import { MemoryPeekModal } from "./MemoryPeekModal";
import { parseMemoryPeek } from "../lib/memoryPeek";
import type { Item } from "../lib/store";
import { FileLinkText } from "./FileLinkText";
import { ScheduleApplyRollback } from "./ScheduleApplyRollback";
import { ToolSources } from "../../components/evidence/ToolSources";
import { useBoardPorts } from "./boardPorts";

export type ToolItem = Extract<Item, { kind: "tool" }>;
export interface ToolCardProps {
  item: ToolItem;
  subcalls?: ToolItem[];
  isToolVisible?: (name: string) => boolean;
  /** 同一会话内稳定的委派显示名（子代理 A/B/C…），由对话投影统一分配。 */
  delegationName?: string;
}

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

// v4.504 diff 完成刀：行级 ± 视图替代只显 original 的旧渲染（write_file 全增
// diff 的 original 是空串→展开态空盒；edit_file 只显旧文）。着色对齐 diffstat
// 芯片口径：add=text-ok、del=text-err、ctx=text-fg-dim。
function DiffRowsView({ rows, folded }: { rows: DiffRow[]; folded: number }) {
  return (
    <pre className="office-tool-card__diff px-3 py-2 overflow-auto max-h-72 whitespace-pre">
      <code>
        {rows.map((r, i) => (
          <span
            key={i}
            data-diff={r.type}
            className={`block ${r.type === "add" ? "text-ok bg-ok/5" : r.type === "del" ? "text-err bg-err/5" : "text-fg-dim"}`}
          >
            <span className="office-diff__label">{r.type === "add" ? "+ " : r.type === "del" ? "- " : ""}</span>
            {r.text}
          </span>
        ))}
        {folded > 0 && <span className="block text-fg-faint/60">{`… 已折叠 ${folded} 行`}</span>}
      </code>
    </pre>
  );
}

// 任务卡注入点读口：板块壳经 Context 提供时走它（见 boardPorts——文学板块与
// 办公同挂时各读各的槽位），未提供（办公）= 模块单例。
function useTaskSlots() {
  const ports = useBoardPorts();
  return ports ?? {
    activity: getTaskCardActivity,
    fireAmbiguity: fireTaskCardAmbiguity,
    openSession: openTaskCardSession,
    openTarget: getTaskCardOpenTarget,
    ambiguity: getTaskCardAmbiguity,
    hasAmbiguityHandler: hasTaskCardAmbiguityHandler,
  };
}

// ── 委派卡（2026-11-18 简化重设计）：一次委派 = 一行任务简报 ──
// 只保留任务名、状态和一行结果/实时进展；点击整卡进入子代理会话查看详情，
// 不在消息流内展开输出。活动条沿用 v4.26 注入点契约，provider 未注入时
// 显示诚实的等待态。
// ref 解析自 args.continue_from / output 引用行，空 ref 回退 args 文本唯一命中。
function DelegationCard({
  item,
  nested,
  openTarget,
  ambiguous,
  delegationName,
}: {
  item: ToolItem;
  nested: ToolItem[];
  openTarget: string;
  ambiguous: boolean;
  delegationName?: string;
}) {
  const t = useT();
  const compact = useCompact();
  // 起跑时刻：Item 契约无时间戳，以挂载时刻近似计时（运行中卡片常驻挂载；
  // 恢复历史会话时 running 一律还原为 stopped，不会进入计时面）。
  const startRef = useRef(Date.now());
  const ref = useMemo(() => resolveTaskRef(item.args, item.output), [item.args, item.output]);
  // 显示状态以 runs 真态优先（2026-11-18 后台委派实弹：回执即达把卡翻成
  // done，而子代理磁盘态仍在 running——三卡假完成定罪）。provider 未注入/
  // 查不到时回退 item.status（诚实维持现状）。
  const slots = useTaskSlots();
  const activity = slots.activity(ref, item.args);
  const runState = activity?.state;
  const displayStatus = runState === "running" || runState === "queued"
    ? "running"
    : runState === "failed"
      ? "error"
      : runState === "aborted"
        ? "stopped"
        : runState === "completed"
          ? "done"
          : item.status;
  const running = displayStatus === "running";
  const now = useNow(running);
  const subject = useMemo(() => subjectOf(item.name, item.args), [item.name, item.args]);
  // 元信息芯片：模型/引擎名取自 args（五参模型面 engine/model 均可选）。
  const engine = useMemo(() => {
    try {
      const a = JSON.parse(item.args || "{}") as { engine?: string; model?: string };
      return a.engine || a.model || "";
    } catch {
      return "";
    }
  }, [item.args]);
  const background = useMemo(() => {
    try {
      const a = JSON.parse(item.args || "{}") as { run_in_background?: boolean };
      return a.run_in_background === true;
    } catch {
      return false;
    }
  }, [item.args]);
  const result = running ? "" : taskResultSummary(item.output, item.error);
  const clickable = !!openTarget || ambiguous;
  const headClick = clickable
    ? ambiguous
      ? () => { slots.fireAmbiguity(item.args); }
      : () => { slots.openSession(openTarget); }
    : undefined;
  const elapsed = running ? formatElapsed(Math.max(0, now - Math.floor(startRef.current / 1000))) : "";
  const statusWord = running
    ? t("tool.delegateRunning")
    : displayStatus === "error"
      ? t("process.statusError")
      : displayStatus === "stopped"
        ? t("process.statusStopped")
        : t("tool.delegateDone");
  const stepWord = nested.length > 0
    ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length })
    : "";
  const waitingText = t("tool.delegateWaiting");
  const displayName = delegationName || t("tool.delegateKind");
  const stateText = running
    ? activity?.lastText || waitingText
    : result || item.error || statusWord;
  const statusIcon = running
    ? null
    : displayStatus === "error"
      ? <X size={11} />
      : displayStatus === "stopped"
        ? <Ban size={11} />
        : <Check size={11} />;

  return (
    <div
      className="office-tool-card office-delegate-card"
      data-status={displayStatus}
      data-compact={compact ? "true" : "false"}
      data-clickable={clickable ? "true" : "false"}
      data-delegation-name={delegationName}
    >
      <button
        type="button"
        data-testid="delegate-head"
        className={`office-delegate-card__head select-none ${clickable ? "cursor-pointer" : ""}`}
        onClick={headClick}
        title={ambiguous ? t("taskpick.cardAria") : clickable ? t("tool.openSessionHint") : undefined}
        aria-label={ambiguous
          ? t("taskpick.cardAria")
          : clickable
            ? `${displayName} · ${t("tool.delegateOpenAria", { task: subject })}`
            : `${displayName} · ${subject}`}
        disabled={!clickable}
      >
        <span className="office-delegate-card__lamp" data-status={displayStatus} title={statusWord} aria-hidden>
          <Bot size={compact ? 15 : 17} />
        </span>
        <span className="office-delegate-card__copy">
          <span className="office-delegate-card__title-line">
            <span className="office-delegate-card__kind">
              {displayName}
            </span>
            <span className="office-delegate-card__title" title={engine ? `${subject} · ${engine}` : subject}>
              {subject || item.name}
            </span>
            {background && (
              <span className="office-delegate-card__background">
                {t("tool.delegateBackground")}
              </span>
            )}
            {stepWord && <span className="office-delegate-card__step">{stepWord}</span>}
          </span>
          <span
            data-testid={running ? "task-live" : item.status === "done" ? "delegate-result" : undefined}
            className="office-delegate-card__summary"
            data-kind={running ? "running" : item.status}
            role={running ? "status" : undefined}
            aria-live={running ? "polite" : undefined}
          >
            {running && <span className="office-delegate-card__pulse" aria-hidden />}
            <span className="office-delegate-card__summary-text" title={stateText}>{stateText}</span>
            {running && activity?.lastTool && (
              <span className="office-delegate-card__tool" title={activity.lastTool}>{activity.lastTool}</span>
            )}
            {running && elapsed && (
              <span className="office-delegate-card__elapsed">
                <Clock size={10} aria-hidden />
                <span className="tabular-nums">{elapsed}</span>
              </span>
            )}
          </span>
        </span>
        <span className="office-delegate-card__aside">
          <span className="office-delegate-card__status" data-status={item.status}>
            {statusIcon}
            <span>{statusWord}</span>
          </span>
          {clickable && (
            <span className="office-delegate-card__open">
              {t("tool.delegateView")}
              <ChevronRight size={10} aria-hidden />
            </span>
          )}
        </span>
      </button>
    </div>
  );
}

export const ToolCard = memo(function ToolCard({ item, subcalls, isToolVisible, delegationName }: ToolCardProps) {
  const t = useT();
  const compact = useCompact();
  const slots = useTaskSlots();
  const diffs = useMemo(() => diffsFor(item.name, item.args), [item.name, item.args]);
  // 波3.5 实弹反馈：提炼记忆（remember/fact_add）行点击直接弹窗查看完整内容，
  // 行内主体显示记忆正文首行（不再要求展开翻 JSON args）。
  const memoryPeek = useMemo(
    () => (item.name === "remember" || item.name === "fact_add" ? parseMemoryPeek(item.name, item.args) : null),
    [item.name, item.args],
  );
  const [peekOpen, setPeekOpen] = useState(false);
  const rawSubject = useMemo(() => subjectOf(item.name, item.args), [item.name, item.args]);
  const subject = useMemo(() => {
    if (memoryPeek) {
      const first = memoryPeek.body.split("\n").find((l: string) => l.trim()) ?? "";
      return first.length > 60 ? first.slice(0, 60) + "…" : first || memoryPeek.title;
    }
    return rawSubject;
  }, [memoryPeek, rawSubject]);
  // Codex 式 diffstat 芯片：编辑类工具行内显示 +N−M（不再重复进斜体摘要）
  const stat = useMemo(() => diffStatFor(item.name, item.args), [item.name, item.args]);
  const Icon = lxToolIcon(item.name) ?? mcpOr(item.name);
  const nested = subcalls ?? [];
  const hasNested = nested.length > 0;

  // v4.350：summary/pretty/outputLines memo 化——write_file 等工具 args 内联
  // 整文件内容（数百 KB 常见），此前折叠态每次自身更新仍全付 JSON round-trip
  // 与全文 split（diffsFor/boundedOutput 已 memo，此处补齐漏网三处）。
  const summary = useMemo(() => {
    if (item.status === "running") return "";
    if (item.name === "task") {
      // v4.26：task 卡完成后显示子代理结果摘要（summarize 对 task 返回空，
      // 完成卡此前除嵌套步数外没有任何结果信息；error 走卡片错误区）。
      return taskResultSummary(item.output, item.error) ||
        (nested.length > 0 ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length }) : "");
    }
    return nested.length > 0
      ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length })
      : summarize(item.name, item.args, item.output, item.error);
  }, [item.status, item.name, item.args, item.output, item.error, nested.length, t]);

  // Codex 式对齐（2026-10）：展开态不再整包摊 JSON args——
  // ① bash 类把 args.command 直接渲染成终端命令块（多行命令在 JSON 里
  //    是 \n 转义长串，是「太长」的主源）；
  // ② 写文件类 diffs 已是 args 的呈现，args JSON 不再重复渲染（此前
  //    write_file 整文件内容 diff 一遍、JSON 再摊一遍）。
  const command = useMemo(() => commandOf(item.name, item.args), [item.name, item.args]);
  const showArgsJson = !command && diffs.length === 0 && !READER_JSON_TOOLS.has(item.name) && !!item.args;
  const hasOutput = !!item.output;
  const [open, setOpen] = useState(false);
  // 三层预算派生：披露按真实隐藏量，而不是“有 output 就一定有内容可展开”。
  const bounded = useMemo(() => boundedOutput(item.output, TOOL_CARD_OUTPUT_PREVIEW_LINES), [item.output]);
  const expandable = !!command || diffs.length > 0 || showArgsJson || bounded.collapsed || !!item.error;
  const [showFullOutput, setShowFullOutput] = useState(false);

  const bodyRef = useRef<HTMLDivElement>(null);
  useGSAPCollapse(bodyRef, open && expandable);

  const quiet =
    item.readOnly && !hasNested && item.status !== "error" && item.status !== "stopped";

  const prettyArgs = useMemo(() => (showArgsJson && item.args ? pretty(item.args) : ""), [showArgsJson, item.args]);

  // v4.63 子代理卡片整卡可点：task / run_skill 卡解析出可跳转的子代理 ref
  // 时，点击头部行直接打开对应会话 tab（与右栏任务树同款跳转），不再只是
  // 折叠展开。空目标 = 不可点（诚实维持现状）。
  // v4.68：空 ref 多候选歧义（≥2 文本匹配 running）时不再死路——整卡可点，
  // 点击派发歧义处理器由 App 弹轻量选择器人工挑一个（宁缺勿错：绝不自动
  // 跳转；0/1 候选行为与 v4.63 完全一致）。可达性：cursor-pointer +
  // aria-label 说明「多个运行中任务，点击选择」。
  // 委派卡分派走调用级判定（2026-11-20 家族收窄）：task 恒委派；run_skill
  // 仅当回执携带 "Subagent reference: sa_…"（真起了隔离子会话）。inline/
  // pipeline 形态按普通工具行渲染；运行中（无回执）同为普通行，回执落地后
  // 段签名变更自然升级——宁迟半拍勿错标。
  const isSubagentCard = isDelegationCall(item.name, item.output);
  const resolvedRef = isSubagentCard ? resolveTaskRef(item.args, item.output) : "";
  const openTarget = isSubagentCard
    ? slots.openTarget(resolvedRef, item.args)
    : "";
  const ambiguous = isSubagentCard && !openTarget
    && slots.ambiguity(resolvedRef, item.args)
    && slots.hasAmbiguityHandler();
  const clickable = isSubagentCard && (!!openTarget || ambiguous);
  const bodyId = useId();
  const statusLabel = item.status === "running"
    ? t("process.statusRunning")
    : item.status === "error"
      ? t("process.statusError")
      : item.status === "stopped"
        ? t("process.statusStopped")
        : t("process.statusDone");

  const chevronSize = compact ? 11 : 12;
  const innerPx = compact ? "px-1" : "px-2";
  const innerPb = compact ? "pb-1" : "pb-1.5";
  if (isToolVisible && !isToolVisible(item.name)) return null;

  // 2026-11-18 重设计：委派家族走专属一等卡形态（交互语义不变，见
  // DelegationCard 头注释）；其余工具维持通用工具行。
  if (isSubagentCard) {
    return (
      <DelegationCard
        item={item}
        nested={nested}
        openTarget={openTarget}
        ambiguous={ambiguous}
        delegationName={delegationName}
      />
    );
  }

  // 工具行先回答“运行了什么、结果是什么”，再允许下钻输出。整行使用真实
  // button 语义，键盘与屏幕阅读器都能展开。
  return (
    <div
      className={`office-tool-card ${item.status === "stopped" ? "opacity-70" : ""}`}
      data-tone={item.status === "error" && !item.recoverable ? "danger" : item.status === "running" ? "info" : item.status === "done" ? "success" : item.status === "stopped" ? "warning" : undefined}
      data-open={open && expandable ? "true" : "false"}
      data-quiet={quiet ? "true" : "false"}
      data-compact={compact ? "true" : "false"}
    >
      <button
        type="button"
        className={`office-tool-card__head select-none ${clickable ? "cursor-pointer" : ""}`}
        onClick={
          memoryPeek
            ? () => setPeekOpen(true)
            : clickable
              ? ambiguous
                ? () => { slots.fireAmbiguity(item.args); }
                : () => { slots.openSession(openTarget); }
              : expandable
                ? () => setOpen((v) => !v)
                : undefined
        }
        title={ambiguous ? t("taskpick.cardAria") : clickable ? t("tool.openSessionHint") : undefined}
        aria-label={ambiguous ? t("taskpick.cardAria") : undefined}
        aria-expanded={!memoryPeek && !clickable && expandable ? open : undefined}
        aria-controls={!clickable && expandable ? bodyId : undefined}
        disabled={!clickable && !expandable}
      >
        <span className="office-tool-card__status" data-status={item.status} title={statusLabel}>
          <StatusGlyph status={item.status} recoverable={item.recoverable} />
          <span className="sr-only">{statusLabel}</span>
        </span>
        <Icon
          className="office-tool-card__icon shrink-0"
          width={chevronSize + 2}
          height={chevronSize + 2}
          aria-hidden
        />
        <span className="office-tool-card__identity">
          <span className={`office-tool-card__name ${item.status === "error" && item.recoverable ? "line-through opacity-70" : ""}`}>
            {item.name}
          </span>
          {/* bash 的行内主体=命令本身；展开后由下方命令块独占。 */}
          {subject && !(open && command) && (
            <span className="office-tool-card__subject" data-command={command ? "true" : "false"}>
              {command ? `$ ${subject}` : subject}
            </span>
          )}
        </span>
        <span className="office-tool-card__trail">
          {stat && (stat.add > 0 || stat.del > 0) ? (
            <span
              className="office-tool-card__diffstat inline-flex items-center gap-1 border border-border-soft bg-bg-soft px-1 py-px font-mono text-fg-dim tabular-nums"
              title={t("tool.diffStatTitle")}
            >
              <span className="text-ok">+{stat.add}</span>
              <span className="text-err">−{stat.del}</span>
            </span>
          ) : (
            summary && <span className="office-tool-card__summary">{summary}</span>
          )}
        </span>
        <ChevronRight
          size={chevronSize}
          className={`office-tool-card__chevron shrink-0 ${!clickable && !expandable ? "invisible" : ""}`}
          data-open={open ? "true" : "false"}
          aria-hidden
        />
      </button>

      <ToolSources sources={item.sources} />

      {/* schedule_apply 回执卡常驻动作行（v4.146 刀A + 工具化 2026-10）：
          「回滚本次」（Journal 按 target 匹配最新基线快照，无记录该段不渲染）
          +「展开查看」开右栏「进度计划」Tab（2026-11-14 起右栏一级 Tab，原 ToolSheet 退役；
          docs/archive/gaea-ai-workbench-ui-2026-10.md §4 工具卡=工具入口之一）。 */}
      {item.name === "schedule_apply" && item.status === "done" && (
        <div className="px-3 py-1 flex items-center gap-2 border-b border-border-soft" data-testid="sched-apply-actions">
          <ScheduleApplyRollback args={item.args} />
          <span className="flex-1" />
          <button
            className="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-accent/40 bg-transparent text-accent text-[10px] cursor-pointer hover:bg-accent/10 transition-colors"
            onClick={() => emitFrontendEvent(FRONTEND_EVENTS.OPEN_TOOL, { tool: "schedule" })}
            title="在对话上方打开进度计划工具面板（临时面板，关闭回到对话）"
            data-testid="sched-apply-open-tool"
          >
            <LineChart size={10} aria-hidden />
            展开查看
          </button>
        </div>
      )}

      <div id={bodyId} ref={bodyRef} style={{ overflow: "hidden" }}>
        <div className="office-tool-card__body">
          {diffs.map((d, i) => {
            const { rows, folded } = unifiedRows(d.original, d.modified);
            if (rows.length === 0 && folded === 0) return null;
            return (
              <div className={`${innerPx} ${innerPb}`} key={i}>
                {d.label && <div className="office-tool-card__section-label">{d.label}</div>}
                <DiffRowsView rows={rows} folded={folded} />
              </div>
            );
          })}

          {hasNested && (
            <div className="pl-3 border-l border-border-soft ml-3">
              {nested.map((c) => (
                <ToolCard key={c.id} item={c} isToolVisible={isToolVisible} />
              ))}
            </div>
          )}

          {/* Codex 式 bash 命令块：$ 前缀终端风格，多行命令原样换行、超长行折行 */}
          {command && (
            <div className={`${innerPx} ${innerPb}`}>
              <pre className="office-tool-card__command px-3 py-2 whitespace-pre-wrap break-words text-fg-dim"><code>{`$ ${command}`}</code></pre>
            </div>
          )}

          {showArgsJson && (
            <div className={`${innerPx} ${innerPb}`}>
              {item.args && <pre className="office-tool-card__args px-3 py-2 overflow-auto whitespace-pre text-fg-dim"><code>{prettyArgs}</code></pre>}
            </div>
          )}
          {hasOutput && (
            <div className={`${innerPx} ${innerPb}`}>
              <div className="office-tool-card__section-label select-none">{t("tool.outputHeader", { n: bounded.totalLines })}</div>
              <pre className="office-tool-card__output px-3 py-2 overflow-auto max-h-72 whitespace-pre-wrap break-words text-fg-dim"><code><FileLinkText text={showFullOutput ? bounded.full : bounded.preview} compact /></code></pre>
              {bounded.preview !== bounded.full && (
                <button
                  type="button"
                  className="mt-1 px-2 py-0.5 border border-border-soft rounded bg-bg-soft text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
                  onClick={() => setShowFullOutput((v) => !v)}
                >
                  {showFullOutput
                    ? t("tool.collapseOutput")
                    : bounded.collapsed
                      ? t("tool.expandAllLines", { n: bounded.hiddenLines })
                      : t("tool.expandFullOutput")}
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
            <div className="office-tool-card__error" data-recoverable="false" role="alert">
              {item.error}
            </div>
          )}
          {item.error && item.recoverable && (
            <div className="office-tool-card__error" data-recoverable="true">
              {item.error}
            </div>
          )}
        </div>
      </div>
      {peekOpen && memoryPeek && (
        <MemoryPeekModal payload={{ tool: item.name, args: item.args }} onClose={() => setPeekOpen(false)} />
      )}
    </div>
  );
});
