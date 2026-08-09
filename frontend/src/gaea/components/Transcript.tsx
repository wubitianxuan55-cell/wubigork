import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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

// 过程卡内的思考块（复用 .reasoning 样式）
function InlineReasoning({ item }: { item: AssistantItem }) {
  const [open, setOpen] = useState(true);
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

function ProcessCard({
  items,
  toolCount,
  thoughtCount,
  running = false,
  subcallsByParent,
}: {
  items: Item[];
  toolCount: number;
  thoughtCount: number;
  running?: boolean;
  subcallsByParent: Map<string, ToolItem[]>;
}) {
  const [open, setOpen] = useState(running);
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
      setOpen(false);
      finalElapsedRef.current = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
    }
  }, [running, turnStartAt, now]);

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
}

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
    if (!stick.current) return;

    if (rAF.current !== null) cancelAnimationFrame(rAF.current);
    rAF.current = requestAnimationFrame(() => {
      rAF.current = null;
      if (!stick.current) return;
      el.scrollTop = el.scrollHeight;
    });
  }, []);

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
  const segments = useMemo(() => buildSegments(merged, running), [merged, running]);

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

  // ── 子调用收集 ──────────────────────────────────────────────────
  const subcallsByParent = useMemo(() => {
    const map = new Map<string, ToolItem[]>();
    for (const it of items) {
      if (it.kind === "tool" && it.parentId) {
        const arr = map.get(it.parentId) ?? [];
        arr.push(it);
        map.set(it.parentId, arr);
      }
    }
    return map;
  }, [items]);

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

  const userTurn = useMemo(() => {
    const map = new Map<string, number>();
    let nt = 0;
    for (const it of items) {
      if (it.kind === "user") map.set(it.id, nt++);
    }
    return map;
  }, [items]);

  // ── 分段渲染：过程（思考/工具）进过程卡，正文留在外面 ──
  const renderSegment = useCallback((outsideItems: Item[]) => {
    const gs = scanGroups(outsideItems);
    let lastUserText = "";
    return gs.map((g) => {
      if (g.kind === "group") {
        return <ToolGroup key={g.id} tools={g.tools} onCollapse={scheduleMeasure} />;
      }
      const it = g.item;
      if (it.kind === "user") lastUserText = it.text;
      switch (it.kind) {
        case "user": {
          const tn = userTurn.get(it.id);
          return (
            <div
              key={it.id}
              data-turn={tn != null ? tn : undefined}
              data-entrance={it.id}
              ref={(el) => {
                if (el && tn != null) {
                  turnEls.current.set(tn, el);
                } else if (tn != null) {
                  turnEls.current.delete(tn);
                }
              }}
            >
              <UserMessage
                text={it.text} turn={tn}
                open={tn != null && openTurn === tn}
                onToggle={() => setOpenTurn((cur) => (cur === tn ? null : (tn ?? null)))}
                onRewind={(turn, scope) => { onRewind?.(turn, scope); setOpenTurn(null); }}
              />
            </div>
          );
        }
        case "assistant":
          return (
            <div key={it.id} data-entrance={it.id}>
              <AssistantMessage
                item={it}
                onCollapse={scheduleMeasure}
                onCapture={
                  lastUserText
                    ? (solution) => setCapture({ task: lastUserText, solution })
                    : undefined
                }
              />
            </div>
          );
        case "tool":
          if (it.parentId) return null;
          if (it.name === "todo_write") return null;
          return (
            <div key={it.id} data-entrance={it.id}>
              <ToolCard item={it} subcalls={subcallsByParent.get(it.id)} />
            </div>
          );
        case "phase":
          return <div key={it.id} className="phase">{it.text}</div>;
        case "notice":
          if (it.level === "warn") {
            if (dismissedErrors.has(it.id)) return null;
            return <ErrorCard key={it.id} item={it as Extract<Item, { kind: "notice" }>} onDismiss={(id) => setDismissedErrors((p) => new Set(p).add(id))} />;
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
  }, [userTurn, openTurn, onRewind, scheduleMeasure, subcallsByParent, dismissedErrors]);

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
          const toolCount = seg.processItems.filter((it) => it.kind === "tool" && !it.parentId).length;
          const thoughtCount = seg.processItems.filter((it) => it.kind === "assistant" && it.reasoning).length;
          const hasProcess = seg.processItems.length > 0;
          const isLast = segIdx === arr.length - 1;
          const segKey = seg.processItems[0]?.id ?? seg.outsideItems[0]?.id ?? `seg${segIdx}`;
          return (
            <div key={segKey}>
              {hasProcess && (
                <ProcessCard
                  items={seg.processItems}
                  toolCount={toolCount}
                  thoughtCount={thoughtCount}
                  running={running && isLast}
                  subcallsByParent={subcallsByParent}
                />
              )}
              {seg.outsideItems.length > 0 && renderSegment(seg.outsideItems)}
            </div>
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
