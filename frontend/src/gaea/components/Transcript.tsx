import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ArrowDown, Brain, ChevronRight } from "../icons";
import type { Item } from "../lib/store";
import { useItems, useTurnStartAt } from "../lib/store";
import { AssistantMessage, UserMessage } from "./Message";
import { SkillCaptureModal } from "./SkillCaptureModal";
import { StreamingIndicator } from "./StreamingIndicator";
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

type Segment = { processItems: Item[]; outsideItems: Item[] };

// 流式进行中的轮：与 tianxuan 一致，正文与过程卡交替出现。
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
    if (it.kind === "tool" || it.kind === "compaction" || it.kind === "notice") {
      curProcess.push(it);
    } else {
      curOutside.push(it);
    }
  }
  flush();
  return segments;
}

// 已完成的轮：把思考、工具和中间正文折叠成一张大过程卡，
// 只把最终正文留在外面（“最后输出时前面的折叠成一个大的过程卡”）。
function consolidatedSegments(turn: Item[]): Segment[] {
  const users: Item[] = turn.filter((it) => it.kind === "user");
  const rest = turn.filter((it) => it.kind !== "user");
  const process: Item[] = [];

  // 最后一个带正文的 assistant 视为最终输出；中间文本与思考/工具进过程卡
  let lastTextIdx = -1;
  for (let i = 0; i < rest.length; i++) {
    if (rest[i].kind === "assistant" && (rest[i] as AssistantItem).text) lastTextIdx = i;
  }

  let finalText: Item | null = null;
  rest.forEach((it, i) => {
    if (it.kind === "assistant") {
      if (it.reasoning) process.push({ ...it, text: "" } as Item);
      if (i === lastTextIdx) {
        finalText = { ...it, reasoning: "" } as Item;
      } else if (it.text) {
        process.push({ ...it, reasoning: "" } as Item);
      }
      return;
    }
    if (it.kind === "tool" || it.kind === "compaction" || it.kind === "notice") {
      process.push(it);
    } else {
      users.push(it); // phase 等正文元素与用户消息同段，先于过程卡
    }
  });

  // 渲染顺序必须是 [用户问题] → [过程卡] → [最终输出]：
  // 把用户消息拆成独立段先渲染，过程卡与最终正文放同一段，否则卡片会
  // 跑到用户问题上方（看起来像被顶到窗口最上方）。
  const segments: Segment[] = [];
  if (users.length > 0) segments.push({ processItems: [], outsideItems: users });
  segments.push({ processItems: process, outsideItems: finalText ? [finalText] : [] });
  return segments;
}

export function buildSegments(items: Item[], running = false): Segment[] {
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

  const segments: Segment[] = [];
  turns.forEach((turn, ti) => {
    const isLastTurn = ti === turns.length - 1;
    // 只有“正在流式输出的最后一轮”保持交替；其余已完成轮一律折叠成大过程卡
    if (isLastTurn && running) {
      segments.push(...alternatingSegments(turn));
    } else {
      segments.push(...consolidatedSegments(turn));
    }
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
        return <div key={it.id} className="phase">{it.text}</div>;
      case "notice":
        if (it.level === "warn") {
          if (ctx.dismissedErrors.has(it.id)) return null;
          return <ErrorCard key={it.id} item={it as Extract<Item, { kind: "notice" }>} onDismiss={ctx.onDismissError} />;
        }
        if (it.text.startsWith("diagnostics:")) {
          const clean = it.text.includes("— clean");
          return (
            <div key={it.id} className={`flex items-center gap-1.5 px-4 py-1 text-[11px] ${clean ? "text-ok" : "text-warning"}`}>
              <span className="shrink-0">{clean ? "✔" : "⚠"}</span>
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
  dismissedErrors, onDismissError, captureForId, turnElsRef,
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
    </>
  );
});

// 过程卡内的思考块（复用 .reasoning 样式）
function InlineReasoning({ item }: { item: AssistantItem }) {
  // 思考卡默认折叠：展开大过程卡时只看到标题，点开才看推理内容。
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
  /** 运行中轮次的分段小过程卡（完成后应折叠；整轮合并的大过程卡为 false） */
  small?: boolean;
  subcallsByParent: Map<string, ToolItem[]>;
}) {
  // 大过程卡（整轮合并）默认展开；运行中的分段小过程卡默认折叠。
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
      // 本段刚完成：分段小过程卡折叠收起，整轮合并的大过程卡保持展开；
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

  const body = useMemo(() => {
    const out: React.ReactNode[] = [];
    const grouped = scanGroups(items);
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
  }, [items, subcallsByParent]);

  return (
    <div className={`my-1.5 border rounded-xl overflow-hidden bg-bg-soft/40 transition-colors ${running ? "border-accent/25" : "border-border-soft"}`}>
      <button
        type="button"
        className="flex items-center gap-2 w-full px-3 py-2 text-left cursor-pointer hover:bg-bg-elev/60 transition-colors"
        data-running={running ? "" : undefined}
        onClick={() => { userOverridden.current = true; setOpen((v) => !v); }}
        aria-expanded={open}
      >
        <ChevronRight size={13} className={`shrink-0 text-fg-faint transition-transform duration-200 ${open ? "rotate-90" : ""}`} />
        <Brain size={13} className={`shrink-0 ${running ? "text-accent animate-pulse" : "text-fg-faint"}`} />
        <span className="text-[11px] font-medium text-fg-dim">{label}</span>
        {running && <span className="w-1.5 h-1.5 rounded-full bg-accent animate-pulse" />}
      </button>
      <div ref={bodyRef} style={{ overflow: "hidden" }}>
        <div className="px-2.5 pb-2.5 pt-0.5 space-y-1">{body}</div>
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
  }, [items.length, onNewQuestion]);

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
    <div className="transcript" ref={scrollRef} onScroll={onScroll}>
      <div className="w-full px-8 md:px-12 py-4" ref={entranceRef}>
        {items.length === 0 && (
          <Welcome onPrompt={onPrompt} cwd={cwd} cwdName={cwdName} sessions={sessions} onResumeSession={onResumeSession} meta={meta} />
        )}
        <StreamingIndicator running={running} items={items} />
        {segments.map((seg, segIdx, arr) => {
          const isLast = segIdx === arr.length - 1;
          const segKey = seg.processItems[0]?.id ?? seg.outsideItems[0]?.id ?? `seg${segIdx}`;
          // 大过程卡（整轮结束后的合并卡）用独立 key 全新挂载：
          // 默认展开，且不复用运行中小过程卡的折叠实例（小卡始终折叠）。
          const segKeyFinal = running ? segKey : `done-${segKey}`;
          return (
            <TurnBlock
              key={segKeyFinal}
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
            />
          );
        })}
      </div>
      {/* 回到底部按钮 —— 居中圆形，accent 色调 */}
      {showScrollDown && (
        <button
          className="absolute left-1/2 bottom-8 z-20 flex items-center justify-center w-9 h-9 rounded-full border border-accent/20 bg-bg-elev text-fg-dim cursor-pointer hover:text-accent hover:border-accent/40 hover:bg-bg-elev-2 active:scale-95 transition-all shadow-lg"
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
  const [open, setOpen] = useState(false);
  if (item.pending) {
    return (
      <div className="flex items-center gap-2 my-1 mx-2 px-3 py-2 border border-border-soft rounded-lg bg-bg-soft text-fg-faint text-xs animate-pulse">
        <span className="text-accent font-bold">⋯</span> Compacting conversation…
      </div>
    );
  }
  return (
    <div className="my-1 mx-2 border border-border-soft rounded-lg bg-bg-soft overflow-hidden">
      <button className="flex items-center gap-2 w-full px-3 py-2 bg-transparent border-0 text-fg-dim text-[12.5px] cursor-pointer hover:bg-bg-elev" onClick={() => setOpen((v) => !v)}>
        <span className="text-accent text-xs shrink-0">◆</span>
        <span className="font-medium text-fg">Context compacted</span>
        <span className="text-fg-faint text-[11px] ml-auto">{item.messages} messages · {item.trigger}</span>
        <span className="text-fg-faint text-[10.5px] underline shrink-0">{open ? "hide summary" : "show summary"}</span>
      </button>
      {open && <pre className="m-0 p-3 bg-bg text-fg-dim text-[11.5px] leading-relaxed whitespace-pre-wrap border-t border-border-soft">{item.summary}</pre>}
    </div>
  );
}
