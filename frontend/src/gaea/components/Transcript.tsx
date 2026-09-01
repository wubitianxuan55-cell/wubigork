/* eslint-disable react-refresh/only-export-components -- buildSegments 纯函数导出供测试与消息渲染复用 */
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, ArrowDown, Ban, Brain, CheckCircle, ChevronRight, FileText, Loader } from "../icons";
import type { Item } from "../lib/store";
import { useItems, useTurnStartAt } from "../lib/store";
import { useT } from "../lib/i18n";
import { AssistantMessage, UserMessage } from "./Message";
import { SkillCaptureModal } from "./SkillCaptureModal";
import { StreamingIndicator } from "./StreamingIndicator";
import { WorkHeader } from "./WorkHeader";
import { ToolCard } from "./ToolCard";
import { ToolGroup, scanGroups } from "./ToolGroup";
import { ErrorCard } from "./ErrorCard";
import { Welcome } from "./Welcome";
import { useEntranceAnimation } from "../lib/useEntranceAnimation";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { useNow } from "../lib/useNow";
import { displayReasoningText } from "../lib/reasoningDisplay";


// ── 滚动参数 ──────────────────────────────────────────────────────────
const BOTTOM_THRESHOLD_PX = 80;
const NOOP_SCROLL = () => {};

function isNearBottom(el: HTMLElement): boolean {
  return el.scrollHeight - el.scrollTop - el.clientHeight < BOTTOM_THRESHOLD_PX;
}

type ToolItem = Extract<Item, { kind: "tool" }>;

// scrollVersion: 轻量级内容变化信号
function scrollVersion(items: Item[]): string {
  const n = items.length;
  if (n === 0) return "0";
  const last = items[n - 1];
  switch (last.kind) {
    case "assistant":
      return `${n}:${last.id}:${last.text.length}:${last.streaming ? 1 : 0}`;
    case "tool":
      return `${n}:${last.id}:${last.status}`;
    default:
      return `${n}:${last.id}`;
  }
}

// mergeConsecutiveReasoning: 合并连续纯推理消息
function mergeConsecutiveReasoning(items: Item[]): Item[] {
  const out: Item[] = [];
  for (const it of items) {
    let prevIdx = out.length - 1;
    while (prevIdx >= 0) {
      const pi = out[prevIdx];
      if (pi.kind === "phase" || pi.kind === "notice") { prevIdx--; continue; }
      if (pi.kind === "tool" && pi.name === "todo_write") { prevIdx--; continue; }
      break;
    }
    const prev = prevIdx >= 0 ? out[prevIdx] : null;
    if (
      prev && prev.kind === "assistant" && it.kind === "assistant" &&
      !prev.text && !it.text && !prev.streaming && !it.streaming
    ) {
      out[prevIdx] = { ...prev, reasoning: prev.reasoning + "\n\n" + it.reasoning };
    } else {
      out.push(it);
    }
  }
  return out;
}

type AssistantItem = Extract<Item, { kind: "assistant" }>;

// ── 过程卡：把一轮助手回合的“思考 + 工具调用”收进可折叠卡片 ──
// 参考 tianxuan 桌面端 TurnCollapse：运行中自动展开，完成后自动收起，
// 头部显示耗时 / 工具数 / 思考段数，正文按“思考 → 工具批次”分段。
// 2026-08-26 决策（用户）：删除「大过程卡」——已完成轮不再把整轮合并成一张
// 展开的大卡，文本与过程卡分开：正文始终以独立消息显示，过程（思考/工具）
// 单独成小卡、默认折叠。所有轮次统一走 alternatingSegments（与流式一致）。

type Segment = { processItems: Item[]; outsideItems: Item[] };

// 流式进行中的轮 / 已完成轮：与 tianxuan 一致，正文与过程卡交替出现。
// （用户决策 2026-08-26 起不再区分：删除大过程卡后，全部轮次统一交替。）
function alternatingSegments(turn: Item[]): Segment[] {
  const segments: Segment[] = [];
  let curProcess: Item[] = [];
  let curOutside: Item[] = [];
  const flush = () => {
    if (curProcess.length > 0 || curOutside.length > 0) {
      segments.push({ processItems: curProcess, outsideItems: curOutside });
      curProcess = [];
      curOutside = [];
    }
  };
  for (const it of turn) {
    if (it.kind === "user") {
      flush();
      segments.push({ processItems: [], outsideItems: [it] });
      continue;
    }
    if (it.kind === "assistant") {
      if (it.text) {
        // 与 tianxuan 一致：正文出现时先落盘当前过程，正文留在外面，
        // 后续工具/思考再开新过程段 —— 形成“文本 ↔ 过程卡”交替效果。
        if (curOutside.length > 0) flush();
        if (it.reasoning) curProcess.push({ ...it, text: "" } as Item);
        curOutside.push({ ...it, reasoning: "" } as Item);
      } else {
        if (curOutside.length > 0) flush();
        curProcess.push(it);
      }
      continue;
    }
    if (curOutside.length > 0) flush();
    // v4.26 phase 收编：phase 不再独立成行走消息流（此前堆叠成一行行
    // 「.phase」弱存在感文本）——统一进过程卡：最新 phase 由 WorkHeader
    // 工作态头部展示，历史 phase 折叠在过程卡内。item 形态不变。
    if (it.kind === "tool" || it.kind === "compaction" || it.kind === "notice" || it.kind === "phase") {
      curProcess.push(it);
    } else {
      curOutside.push(it);
    }
  }
  flush();
  return segments;
}

export function buildSegments(items: Item[], _running = false): Segment[] {
  // 先按用户消息切成轮
  const turns: Item[][] = [];
  let curTurn: Item[] = [];
  for (const it of items) {
    if (it.kind === "user") {
      if (curTurn.length > 0) turns.push(curTurn);
      curTurn = [it];
    } else {
      curTurn.push(it);
    }
  }
  if (curTurn.length > 0) turns.push(curTurn);

  // 2026-08-26（用户决策）：删除大过程卡——所有轮次（含已完成）统一交替，
  // 文本独立显示、过程单独成卡，不再把文本和过程包在一张卡片内。
  const segments: Segment[] = [];
  turns.forEach((turn) => {
    segments.push(...alternatingSegments(turn));
  });
  return segments;
}

// segSig 计算一段内容的版本签名（T7-4 轮级缓存用）：流式只改最后一轮时，
// 已完成段的 item 引用与内容都不变 → 签名不变 → Transcript 复用缓存段对象，
// TurnBlock（memo）才能整体跳过重渲染。
function segSig(seg: Segment): string {
  let s = "";
  for (const it of [...seg.processItems, ...seg.outsideItems]) {
    if (it.kind === "assistant") s += `${it.id}:${it.text.length}:${it.reasoning.length}:${it.streaming ? 1 : 0};`;
    else if (it.kind === "tool") s += `${it.id}:${it.status}:${it.output?.length ?? 0};`;
    else s += `${it.id};`;
  }
  return s;
}

// buildSubcalls 收集段内的父子工具调用（parentId → 子调用列表），供本段的
// 过程卡与外部工具卡渲染子调用。
function buildSubcalls(seg: Segment): Map<string, ToolItem[]> {
  const map = new Map<string, ToolItem[]>();
  for (const it of [...seg.processItems, ...seg.outsideItems]) {
    if (it.kind === "tool" && it.parentId) {
      const arr = map.get(it.parentId) ?? [];
      arr.push(it);
      map.set(it.parentId, arr);
    }
  }
  return map;
}

// renderOutsideItems 渲染段内正文元素（用户/助手/工具/阶段/通知/压缩卡）。
// 顶层纯函数 + TurnBlock 内 useMemo：段 props 不变时整棵子树跳过重渲染。
function renderOutsideItems(
  outsideItems: Item[],
  ctx: {
    turnNo?: number;
    openTurn: number | null;
    onToggleTurn: (tn: number) => void;
    onRewindTurn: (turn: number, scope: string) => void;
    onCollapse: () => void;
    dismissedErrors: Set<string>;
    onDismissError: (id: string) => void;
    captureForId: (id: string) => ((solution: string) => void) | undefined;
    subcalls: Map<string, ToolItem[]>;
    setTurnEl: (tn: number) => (el: HTMLElement | null) => void;
  },
): React.ReactNode[] {
  const gs = scanGroups(outsideItems);
  return gs.map((g) => {
    if (g.kind === "group") {
      return <ToolGroup key={g.id} tools={g.tools} onCollapse={ctx.onCollapse} />;
    }
    const it = g.item;
    switch (it.kind) {
      case "user": {
        const tn = ctx.turnNo;
        return (
          <div
            key={it.id}
            data-turn={tn != null ? tn : undefined}
            data-entrance={it.id}
            ref={tn != null ? ctx.setTurnEl(tn) : undefined}
          >
            <UserMessage
              text={it.text} turn={tn}
              open={tn != null && ctx.openTurn === tn}
              onToggle={tn != null ? () => ctx.onToggleTurn(tn) : undefined}
              onRewind={ctx.onRewindTurn}
            />
          </div>
        );
      }
      case "assistant":
        return (
          <div key={it.id} data-entrance={it.id}>
            <AssistantMessage
              item={it}
              onCollapse={ctx.onCollapse}
              onCapture={ctx.captureForId(it.id)}
            />
          </div>
        );
      case "tool":
        if (it.parentId) return null;
        if (it.name === "todo_write") return null;
        return (
          <div key={it.id} data-entrance={it.id}>
            <ToolCard item={it} subcalls={ctx.subcalls.get(it.id)} />
          </div>
        );
      case "phase":
        // v4.26 phase 收编：phase 统一进过程卡 + 最新进 WorkHeader 头部，
        // 不再在消息流里渲染独立行（此分支仅防御历史/异常路径）。
        return null;
      case "notice":
        if (it.level === "warn") {
          if (ctx.dismissedErrors.has(it.id)) return null;
          return <ErrorCard key={it.id} item={it as Extract<Item, { kind: "notice" }>} onDismiss={ctx.onDismissError} />;
        }
        if (it.text.startsWith("diagnostics:")) {
          const clean = it.text.includes("— clean");
          return (
            <div key={it.id} className={`flex items-center gap-1.5 px-4 py-1 text-[11px] ${clean ? "text-ok" : "text-warning"}`}>
              <span aria-hidden className="shrink-0">{clean ? <CheckCircle size={12} className="text-ok" /> : <AlertCircle size={12} className="text-warning" />}</span>
              <span>{it.text}</span>
            </div>
          );
        }
        return <div key={it.id} className="notice">{it.text}</div>;
      case "compaction":
        return <CompactionCard key={it.id} item={it} />;
      default:
        return null;
    }
  });
}

// T7-4 轮级 memo：把每一段的渲染拆进独立 memo 组件。流式只追加/修改最后一轮
// 时，其余已完成的段（props 引用不变）整体跳过重渲染——避免每 chunk 全量
// 重建消息树造成的滚动/渲染卡顿。onToggle/onCapture/onRewind 等回调全部
// 由 Transcript 用 useCallback 稳定化后传入，memo 才能生效。
export const TurnBlock = memo(function TurnBlock({
  seg, running, isLast, turnNo, openTurn, onToggleTurn, onRewindTurn, onCollapse,
  dismissedErrors, onDismissError, captureForId, turnElsRef, workHeader,
}: {
  seg: Segment;
  running: boolean;
  isLast: boolean;
  turnNo?: number;
  openTurn: number | null;
  onToggleTurn: (tn: number) => void;
  onRewindTurn: (turn: number, scope: string) => void;
  onCollapse: () => void;
  dismissedErrors: Set<string>;
  onDismissError: (id: string) => void;
  captureForId: (id: string) => ((solution: string) => void) | undefined;
  turnElsRef: React.MutableRefObject<Map<number, HTMLElement>>;
  /** v4.26 工作态头部：锚定在最后一轮的用户消息段（WorkHeader 自订 store 的
   *  running/turnStartAt/items，running→done 转换不依赖本组件重渲染）。 */
  workHeader?: boolean;
}) {
  const toolCount = seg.processItems.filter((it) => it.kind === "tool" && !it.parentId).length;
  const thoughtCount = seg.processItems.filter((it) => it.kind === "assistant" && it.reasoning).length;
  const hasProcess = seg.processItems.length > 0;
  const subcalls = useMemo(() => buildSubcalls(seg), [seg]);
  const setTurnEl = useCallback(
    (tn: number) => (el: HTMLElement | null) => {
      if (el) turnElsRef.current.set(tn, el);
      else turnElsRef.current.delete(tn);
    },
    [turnElsRef],
  );
  const outside = useMemo(
    () =>
      renderOutsideItems(seg.outsideItems, {
        turnNo, openTurn, onToggleTurn, onRewindTurn, onCollapse,
        dismissedErrors, onDismissError, captureForId, subcalls, setTurnEl,
      }),
    [seg.outsideItems, turnNo, openTurn, onToggleTurn, onRewindTurn, onCollapse, dismissedErrors, onDismissError, captureForId, subcalls, setTurnEl],
  );
  return (
    <>
      {hasProcess && (
        <ProcessCard
          items={seg.processItems}
          toolCount={toolCount}
          thoughtCount={thoughtCount}
          running={running && isLast}
          small={running && !isLast}
          subcallsByParent={subcalls}
        />
      )}
      {seg.outsideItems.length > 0 && outside}
      {/* v4.26 工作态头部：紧跟用户消息（items 为空也渲染，消灭 turn_started
          到首条 text/tool 之间的死寂窗口）；轮完成转 Codex 式耗时行。 */}
      {workHeader && <WorkHeader />}
    </>
  );
});

// 过程卡内的思考块（复用 .reasoning 样式）
function InlineReasoning({ item }: { item: AssistantItem }) {
  // 思考卡默认折叠：只看到标题，点开才看推理内容。
  const [open, setOpen] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  useGSAPCollapse(bodyRef, open);
  const running = item.streaming && !item.text;
  const reasoning = displayReasoningText(item.reasoning ?? "", {
    streaming: item.streaming ?? false,
    truncateStreaming: true,
  });
  if (!reasoning) return null;
  return (
    <div className="reasoning">
      <button
        type="button"
        className="reasoning__head"
        data-running={running ? "" : undefined}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <Brain size={12} className="reasoning__icon" />
        {running && <span aria-hidden className="w-1 h-1 rounded-full bg-accent animate-pulse shadow-[0_0_6px_var(--accent)] shrink-0" />}
        <span className="reasoning__label">思考</span>
        <ChevronRight size={12} className={`reasoning__chevron${open ? " reasoning__chevron--open" : ""}`} />
      </button>
      <div ref={bodyRef} style={{ overflow: "hidden" }}>
        <div className="reasoning__body">{reasoning}</div>
      </div>
    </div>
  );
}

// T7-4：ProcessCard 也加 memo——流式期间已完成轮的过程卡 props 不变，
// 无需随每 chunk 重渲染内部工具/思考卡。

// ── 过程卡状态四态（推导纯函数在 lib/processStatus.ts：
//    状态不只靠颜色——色 + 图标 + 文字三重传达，12 主题下可区分）──
import { deriveProcessStatus, PROCESS_STATUS_META } from "../lib/processStatus";

const STATUS_ICONS = { alert: AlertCircle, ban: Ban, check: CheckCircle } as const;

export const ProcessCard = memo(function ProcessCard({
  items,
  toolCount,
  thoughtCount,
  running = false,
  small = false,
  subcallsByParent,
}: {
  items: Item[];
  toolCount: number;
  thoughtCount: number;
  running?: boolean;
  /** 流式交替段的小过程卡（默认折叠）；false = 展开态（默认展开） */
  small?: boolean;
  subcallsByParent: Map<string, ToolItem[]>;
}) {
  // 展开态段默认展开；流式交替段的小过程卡默认折叠。
  // 用户手动折叠/展开过则不干预。
  const [open, setOpen] = useState(!small);
  const userOverridden = useRef(false);
  const prevRunningRef = useRef(running);
  const bodyRef = useRef<HTMLDivElement>(null);
  const turnStartAt = useTurnStartAt();
  const now = useNow();
  const finalElapsedRef = useRef(0);
  useGSAPCollapse(bodyRef, open);

  useEffect(() => {
    const wasRunning = prevRunningRef.current;
    prevRunningRef.current = running;
    if (running) {
      if (!wasRunning) userOverridden.current = false;
      if (!userOverridden.current) setOpen(true);
    } else if (wasRunning && !userOverridden.current) {
      // 本段刚完成：按 small 收敛（展开态保持展开、小过程卡折叠收起）；
      // 用户手动折叠过则不干预。
      setOpen(!small);
      finalElapsedRef.current = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
    } else if (!userOverridden.current && small) {
      // 运行中已完成的历史分段小过程卡：默认折叠（手动展开过则保留）。
      setOpen(false);
    }
  }, [running, turnStartAt, now, small]);

  const elapsed = running
    ? (turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0)
    : finalElapsedRef.current;
  const elapsedStr = elapsed > 0
    ? (elapsed < 60 ? `${elapsed}s` : `${Math.floor(elapsed / 60)}m${elapsed % 60}s`)
    : "";

  const labelParts: string[] = [];
  if (elapsedStr) labelParts.push(`已工作 ${elapsedStr}`);
  if (toolCount > 0) labelParts.push(`${toolCount} 个工具`);
  if (thoughtCount > 0) labelParts.push(`${thoughtCount} 段思考`);
  const label = labelParts.length > 0
    ? labelParts.join(" · ")
    : (running ? "处理中…" : "过程");

  // 状态四态：色 + 图标 + 文字（头部徽标传达）
  const status = deriveProcessStatus(items, running);
  const statusMeta = status !== "idle" ? PROCESS_STATUS_META[status] : null;
  const StatusIcon = statusMeta?.icon ? STATUS_ICONS[statusMeta.icon] : null;

  // v4.26：运行中最新 phase 已上 WorkHeader 工作态头部，卡内不再重复展示
  // （历史 phase 折叠在卡内）；轮完成后头部转耗时行，全部 phase 回到卡内。
  const lastPhaseId = running
    ? items.reduceRight<string>((acc, it) => acc || (it.kind === "phase" ? it.id : ""), "")
    : "";

  const body = useMemo(() => {
    const out: React.ReactNode[] = [];
    // v4.26 重复工具折叠（Claude Code "Called slack 3 times" 式）：只折叠
    // 全部完成的连续同名调用；running 的保持独立卡不折叠。
    const grouped = scanGroups(items, { skipRunning: true });
    for (const gi of grouped) {
      if (gi.kind === "group") {
        out.push(<ToolGroup key={gi.id} tools={gi.tools} />);
        continue;
      }
      const it = gi.item;
      switch (it.kind) {
        case "assistant":
          if (it.reasoning) out.push(<InlineReasoning key={it.id} item={it as AssistantItem} />);
          if (it.text) {
            out.push(
              <div key={`${it.id}-text`} className="px-2.5 py-1.5 rounded-lg bg-bg/60 border border-border-soft/50 text-fg-dim text-[12.5px] leading-relaxed whitespace-pre-wrap">
                {it.text}
              </div>,
            );
          }
          break;
        case "tool":
          if (it.parentId) break;
          out.push(<ToolCard key={it.id} item={it as ToolItem} subcalls={subcallsByParent.get(it.id)} />);
          break;
        case "phase":
          if (it.id === lastPhaseId) break;
          out.push(
            <div key={it.id} className="phase"><Brain size={12} /><span>{it.text}</span></div>,
          );
          break;
        case "notice":
          out.push(<div key={it.id} className="notice">{it.text}</div>);
          break;
        case "compaction":
          out.push(<CompactionCard key={it.id} item={it as CompactionItem} />);
          break;
      }
    }
    return out;
  }, [items, subcallsByParent, lastPhaseId]);

  // Codex 式过程条：无边框、低噪声；运行中只有左侧细线强调，状态靠徽标传达。
  return (
    <div className={`my-1 rounded-lg overflow-hidden transition-colors duration-[var(--dur-base)] ${
      running ? "bg-accent/[0.03]" : "hover:bg-(color:--md-sys-color-surface-container-high)/40"
    }`}>
      <button
        type="button"
        className="flex items-center gap-2 w-full px-2.5 py-1.5 text-left cursor-pointer rounded-lg transition-colors"
        data-running={running ? "" : undefined}
        onClick={() => { userOverridden.current = true; setOpen((v) => !v); }}
        aria-expanded={open}
      >
        <span className={`shrink-0 w-0.5 self-stretch rounded-full transition-colors ${running ? "bg-accent animate-pulse" : "bg-transparent"}`} />
        <ChevronRight size={12} className={`shrink-0 text-fg-faint transition-transform duration-200 ${open ? "rotate-90" : ""}`} />
        <Brain size={12} className={`shrink-0 ${running ? "text-accent animate-pulse" : "text-fg-faint"}`} />
        <span className="text-[11px] font-medium text-fg-dim">{label}</span>
        {statusMeta && (
          <span className={`inline-flex items-center gap-1 shrink-0 rounded-full border px-1.5 py-px text-[10px] font-medium ${statusMeta.cls}`}>
            {StatusIcon && <StatusIcon size={10} className="shrink-0" />}
            <span>{statusMeta.label}</span>
          </span>
        )}
        <span className="ml-auto text-fg-faint/50 text-[10px] font-mono tabular-nums shrink-0">{elapsedStr}</span>
      </button>
      <div ref={bodyRef} style={{ overflow: "hidden" }}>
        <div className="px-2.5 pb-2 pt-0.5 space-y-0.5">{body}</div>
      </div>
    </div>
  );
});

export function Transcript({
  onPrompt, onRewind, running, onThreadEl, onScrollToTurnReady,
  cwd, cwdName, sessions, onResumeSession, meta,
}: {
  onPrompt: (text: string) => void;
  onRewind?: (turn: number, scope: string) => void;
  running: boolean;
  onThreadEl?: (el: HTMLElement | null) => void;
  onScrollToTurnReady?: (fn: (turn: number) => void) => void;
  cwd?: string;
  cwdName?: string;
  sessions?: import("../lib/types").SessionMeta[];
  onResumeSession?: (path: string) => Promise<void>;
  meta?: import("../lib/types").Meta;
}) {
  const items = useItems();
  const scrollRef = useRef<HTMLDivElement>(null);
  const stick = useRef(true);
  const rAF = useRef<number | null>(null);

  useEffect(() => {
    onThreadEl?.(scrollRef.current);
    return () => onThreadEl?.(null);
  }, [onThreadEl]);

  useEffect(() => {
    return () => { if (rAF.current !== null) cancelAnimationFrame(rAF.current); };
  }, []);

  const [showScrollDown, setShowScrollDown] = useState(false);

  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const atBottom = isNearBottom(el);
    stick.current = atBottom;
    setShowScrollDown(!atBottom && el.scrollHeight > el.clientHeight);
  }, []);

  // ── 智能滚动 ──────────────────────────────────────────────────────
  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    // 运行中强制跟随底部（过程卡/工具卡持续产出时，视图必须实时跟进，
    // 否则一旦 stick 被布局变化置 false，界面会“冻住”像卡死）；
    // 运行结束后恢复智能滚动：用户上翻浏览时不再强拉。
    if (!stick.current && !running) return;

    if (rAF.current !== null) cancelAnimationFrame(rAF.current);
    rAF.current = requestAnimationFrame(() => {
      rAF.current = null;
      if (!stick.current && !running) return;
      el.scrollTop = el.scrollHeight;
    });
  }, [running]);

  const onNewQuestion = useCallback(() => {
    stick.current = true;
    setShowScrollDown(false);
    scrollToBottom();
  }, [scrollToBottom]);

  // ── 内容变化时自动跟随 ──────────────────────────────────────────
  const contentVersion = useMemo(() => scrollVersion(items), [items]);
  const prevItemsLen = useRef(items.length);
  useEffect(() => {
    if (items.length > prevItemsLen.current) {
      const last = items[items.length - 1];
      if (last && last.kind === "user") onNewQuestion();
    }
    prevItemsLen.current = items.length;
  }, [onNewQuestion, items]);

  useEffect(() => {
    scrollToBottom();
  }, [contentVersion, scrollToBottom]);

  // ── ResizeObserver ─────────────────────────────────────────────────
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => {
      if (!stick.current) return;
      scrollToBottom();
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, [scrollToBottom]);

  // ── 预处理 ──────────────────────────────────────────────────────
  // items 重置（新会话/切换会话）时清空 turnEls，防止残留旧 DOM 引用。
  useEffect(() => {
    if (items.length === 0) turnEls.current.clear();
  }, [items.length]);
  const merged = useMemo(() => mergeConsecutiveReasoning(items), [items]);

  // T7-4 轮级缓存：按「段内容签名」复用已构建的 Segment 对象。流式只改
  // 最后一轮时，已完成段的签名不变 → 引用不变 → TurnBlock memo 生效。
  const segCache = useRef(new Map<number, { sig: string; seg: Segment }>());
  const segments = useMemo(() => {
    const built = buildSegments(merged, running);
    const cache = segCache.current;
    const out: Segment[] = [];
    built.forEach((seg, i) => {
      const sig = segSig(seg);
      const prev = cache.get(i);
      if (prev && prev.sig === sig) {
        out.push(prev.seg);
      } else {
        cache.set(i, { sig, seg });
        out.push(seg);
      }
    });
    // 剪枝：段数收缩（新会话）时丢弃多余缓存
    for (const k of cache.keys()) if (k >= built.length) cache.delete(k);
    return out;
  }, [merged, running]);

  // turn→DOM 元素映射（用于跳转）
  const turnEls = useRef(new Map<number, HTMLElement>());
  const scrollToTurnRef = useRef((turn: number) => {
    const el = turnEls.current.get(turn);
    if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
  });
  // V10.17.1: Transcript 卸载时清除 App 中的 scrollToTurn，避免
  // 重新挂载后 MessageNavigator/JumpBar 仍持有旧实例的 turnEls 引用导致跳转失效。
  useEffect(() => {
    onScrollToTurnReady?.(scrollToTurnRef.current);
    return () => onScrollToTurnReady?.(NOOP_SCROLL);
  }, [onScrollToTurnReady]);
  // ── 折叠/展开保持滚动 ──────────────────────────────────────────
  // 250ms 与 GSAP collapse 动画时长耦合（useGSAPCollapse 默认 duration）。
  // 若动画时长变更，此处需同步调整。
  const scheduleMeasure = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const savedTop = el.scrollTop;
    setTimeout(() => {
      if (scrollRef.current) scrollRef.current.scrollTop = savedTop;
    }, 250);
  }, []);

  // ── 入场动画 ──────────────────────────────────────────────────────
  const entranceRef = useEntranceAnimation<HTMLDivElement>(
    items.length > 0 ? `${items[0].id}|${items[items.length - 1].id}` : undefined,
    items.length,
  );

  const [dismissedErrors, setDismissedErrors] = useState(new Set<string>());
  const [openTurn, setOpenTurn] = useState<number | null>(null);
  const [capture, setCapture] = useState<{ task: string; solution: string } | null>(null);
  useEffect(() => {
    if (openTurn === null) return;
    const onDown = (e: MouseEvent) => {
      const el = e.target as Element | null;
      if (!el || !el.closest(".rewind")) setOpenTurn(null);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [openTurn]);

  // 每段对应的用户轮次号（data-turn / 回退弹窗 open 判定用）
  const turnNos = useMemo(() => {
    const map = new Map<number, number>();
    let n = 0;
    segments.forEach((seg, i) => {
      const users = seg.outsideItems.filter((it) => it.kind === "user");
      if (users.length > 0) map.set(i, n);
      n += users.length;
    });
    return map;
  }, [segments]);

  // v4.26 工作态头部锚点：最后一轮的用户消息段。WorkHeader 只挂这一处——
  // 运行中常驻（spinner + 阶段 + 用时 + 步数），轮完成转「已完成 · 用时 · 步数」；
  // 新一轮发出后锚点随最后一条 user 消息移动，上一轮头部自然卸载。
  const lastUserSegIdx = useMemo(() => {
    let last = -1;
    segments.forEach((seg, i) => {
      if (seg.outsideItems.some((it) => it.kind === "user")) last = i;
    });
    return last;
  }, [segments]);

  // T7-4：onToggle/onRewind/onDismiss 全部 useCallback 稳定化，UserMessage/
  // ErrorCard 的 memo 才不会被每次渲染的新函数击穿。
  const toggleTurn = useCallback((tn: number) => {
    setOpenTurn((cur) => (cur === tn ? null : tn));
  }, []);

  const handleRewindTurn = useCallback((turn: number, scope: string) => {
    onRewind?.(turn, scope);
    setOpenTurn(null);
  }, [onRewind]);

  const dismissError = useCallback((id: string) => {
    setDismissedErrors((p) => new Set(p).add(id));
  }, []);

  // onCapture 稳定化：同一 assistant id 复用同一闭包（task 文本来自按 items
  // 重建的映射），AssistantMessage 的 memo 才有效。
  const captureTaskMapRef = useRef(new Map<string, string>());
  const captureFnsRef = useRef(new Map<string, (solution: string) => void>());
  const captureTaskMap = useMemo(() => {
    const map = new Map<string, string>();
    let last = "";
    for (const it of items) {
      if (it.kind === "user") last = it.text;
      else if (it.kind === "assistant") map.set(it.id, last);
    }
    return map;
  }, [items]);
  useEffect(() => {
    captureTaskMapRef.current = captureTaskMap;
    if (items.length === 0) captureFnsRef.current.clear();
  }, [captureTaskMap, items.length]);
  const captureForId = useCallback((id: string): ((solution: string) => void) | undefined => {
    const task = captureTaskMapRef.current.get(id);
    if (!task) return undefined;
    let fn = captureFnsRef.current.get(id);
    if (!fn) {
      fn = (solution: string) => setCapture({ task, solution });
      captureFnsRef.current.set(id, fn);
    }
    return fn;
  }, []);

  const scrollDown = useCallback(() => {
    stick.current = true;
    setShowScrollDown(false);
    scrollToBottom();
  }, [scrollToBottom]);

  return (
    <div className="transcript v3-zone" ref={scrollRef} onScroll={onScroll}>
      <div className="w-full px-4 sm:px-6 md:px-8 py-4" ref={entranceRef}>
        {items.length === 0 && (
          <Welcome onPrompt={onPrompt} cwd={cwd} cwdName={cwdName} sessions={sessions} onResumeSession={onResumeSession} meta={meta} />
        )}
        {/* 正文 74ch 阅读宽度（.v3-reading 居中）；欢迎页作为启动器保持铺满 */}
        <div className="v3-reading">
          {/* v4.26：StreamingIndicator 降级为最底兜底（工作态头部已接管主反馈），
              文案收敛为「连接中…/仍在等待事件…」。items 为空且 running 时
              WorkHeader 在此独立兜底挂载（无 segments 可锚定的死寂窗口）。 */}
          <StreamingIndicator running={running} />
          {items.length === 0 && running && <WorkHeader />}
          {segments.map((seg, segIdx, arr) => {
            const isLast = segIdx === arr.length - 1;
            // 2026-08-26：删除大过程卡后全部走交替段，key 直接用段内首条 id
            // （不再有 done- 重挂载——交替段跨 running 复用实例，用户手动
            // 折叠/展开的过程卡状态不丢）。
            const segKey = seg.processItems[0]?.id ?? seg.outsideItems[0]?.id ?? `seg${segIdx}`;
            return (
              <TurnBlock
                key={segKey}
                seg={seg}
                running={running}
                isLast={isLast}
                turnNo={turnNos.get(segIdx)}
                openTurn={openTurn}
                onToggleTurn={toggleTurn}
                onRewindTurn={handleRewindTurn}
                onCollapse={scheduleMeasure}
                dismissedErrors={dismissedErrors}
                onDismissError={dismissError}
                captureForId={captureForId}
                turnElsRef={turnEls}
                workHeader={segIdx === lastUserSegIdx}
              />
            );
          })}
        </div>
      </div>
      {/* 回到底部按钮 —— 居中圆形，v3 柔光 */}
      {showScrollDown && (
        <button
          className="absolute left-1/2 bottom-8 z-20 flex items-center justify-center w-9 h-9 rounded-full border border-accent/25 bg-bg-elev/85 backdrop-blur-md text-fg-dim cursor-pointer hover:text-accent hover:border-accent/40 hover:bg-bg-elev-2 active:scale-95 transition-all shadow-[var(--v3-glow-faint)]"
          style={{ transform: "translateX(-50%)" }}
          onClick={scrollDown}
          aria-label="回到底部"
        >
          <ArrowDown size={15} />
        </button>
      )}
      <SkillCaptureModal
        open={capture !== null}
        task={capture?.task ?? ""}
        solution={capture?.solution ?? ""}
        onClose={() => setCapture(null)}
      />
    </div>
  );
}




// ── CompactionCard ──────────────────────────────────────────────────
type CompactionItem = Extract<Item, { kind: "compaction" }>;
function CompactionCard({ item }: { item: CompactionItem }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  if (item.pending) {
    return (
      <div className="flex items-center gap-2 my-1 px-2.5 py-1.5 rounded-lg text-fg-faint text-xs animate-pulse">
        <Loader size={12} className="animate-spin text-accent" />
        <span>{t("msg.compacting")}</span>
      </div>
    );
  }
  return (
    <div className="my-1 rounded-lg overflow-hidden">
      <button
        className="flex items-center gap-2 w-full px-2.5 py-1.5 rounded-lg bg-transparent border-0 text-fg-dim text-[12px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high) transition-colors"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span className={`shrink-0 w-0.5 self-stretch rounded-full ${open ? "bg-accent/60" : "bg-transparent"}`} />
        <FileText size={12} className="text-accent shrink-0" />
        <span className="font-medium text-fg">{t("msg.compacted")}</span>
        <span className="text-fg-faint text-[11px] ml-auto tabular-nums">{t("msg.compactedMeta", { n: item.messages, trigger: item.trigger })}</span>
        <span className="text-fg-faint text-[10.5px] shrink-0">{open ? t("msg.hideSummary") : t("msg.showSummary")}</span>
      </button>
      {open && <pre className="m-0 px-3 pb-2 pt-1 text-fg-dim text-[11.5px] leading-relaxed whitespace-pre-wrap">{item.summary}</pre>}
    </div>
  );
}
