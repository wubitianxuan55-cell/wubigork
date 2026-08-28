/* eslint-disable react-refresh/only-export-components -- 子组件与容器同文件（Phase A 收敛，避免过早拆文件） */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { app } from "../lib/bridge";
import { AgentNetworkCard } from "./AgentNetworkCard";
import type {
  ContextCategory, ContextEvent, ContextRequestRecord, ContextStats, ContextTimeline,
} from "../lib/types";
import { fmtTokens } from "../lib/stats";

// 六分类语义色（效果图对齐：系统蓝/工具橙/用户绿/注入紫/助手深蓝/工具青）。
export const CAT_COLORS: Record<keyof ContextCategory, string> = {
  system: "#3b82f6",
  tools: "#f59e0b",
  user: "#22c55e",
  inject: "#a855f7",
  assistant: "#1e40af",
  tool: "#06b6d4",
};

const CATS: { key: keyof ContextCategory; label: string }[] = [
  { key: "system", label: "系统提示词" },
  { key: "tools", label: "工具定义" },
  { key: "user", label: "用户消息" },
  { key: "inject", label: "注入内容" },
  { key: "assistant", label: "助手消息" },
  { key: "tool", label: "工具结果" },
];

function catTotal(c: ContextCategory): number {
  return c.system + c.tools + c.user + c.inject + c.assistant + c.tool;
}

const EMPTY: ContextTimeline = {
  ok: true, window: 0,
  current: { system: 0, tools: 0, user: 0, inject: 0, assistant: 0, tool: 0 },
  stats: { turns: 0, steps: 0, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0 },
  requests: [], events: [], nodes: [], archive: [],
};

// ─── 上下文统计卡 ───────────────────────────────────────────
function StatsBoard({ stats }: { stats: ContextStats }) {
  const items: [string, string][] = [
    ["轮次", String(stats.turns)],
    ["步数", String(stats.steps)],
    ["注入", String(stats.injects)],
    ["压缩", String(stats.compacts)],
    ["剪枝", String(stats.prunes)],
    ["工具调用", String(stats.toolCalls)],
    ["图片", String(stats.images)],
    ["缓存命中", stats.cacheHitPercent != null ? `${stats.cacheHitPercent.toFixed(2)}%` : "—"],
    ["预估费用", stats.costEstimate != null ? `¥${stats.costEstimate.toFixed(2)}` : "—"],
  ];
  return (
    <div className="grid grid-cols-3 gap-px overflow-hidden rounded-lg border border-border-soft bg-border-soft/60">
      {items.map(([k, v]) => (
        <div key={k} className="bg-bg px-2.5 py-1.5">
          <div className="text-[9px] text-fg-faint">{k}</div>
          <div className="text-[12px] font-medium text-fg tabular-nums">{v}</div>
        </div>
      ))}
    </div>
  );
}

// ─── 当前上下文组成（六分类分段条 + 图例） ─────────────────────
function CurrentComposition({ current, window }: { current: ContextCategory; window: number }) {
  const total = catTotal(current);
  const pct = window > 0 ? Math.round((total / window) * 100) : 0;
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-baseline justify-between">
        <div className="text-[11px] font-medium text-fg">当前上下文</div>
        <div className="text-[10px] text-fg-dim tabular-nums font-mono">
          {fmtTokens(total)} / {fmtTokens(window)} · {pct}%
        </div>
      </div>
      <div className="mt-2 flex h-3 w-full overflow-hidden rounded-full bg-bg-soft">
        {CATS.map((c) => {
          const w = total > 0 ? (current[c.key] / total) * 100 : 0;
          if (w <= 0) return null;
          return <div key={c.key} style={{ width: `${w}%`, background: CAT_COLORS[c.key] }} />;
        })}
      </div>
      <div className="mt-2 grid grid-cols-2 gap-x-3 gap-y-0.5">
        {CATS.map((c) => {
          const share = total > 0 ? Math.round((current[c.key] / total) * 100) : 0;
          return (
            <div key={c.key} className="flex items-center gap-1.5 text-[10px] text-fg-dim">
              <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[c.key] }} />
              <span className="truncate">{c.label}</span>
              <span className="ml-auto tabular-nums font-mono text-fg-faint">
                ≈{fmtTokens(current[c.key])} ({share}%)
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── 上下文趋势（原生 SVG 堆叠柱，全局模式；增量 Phase B） ─────
function ContextTrendChart({ requests, events, onPick }: {
  requests: ContextRequestRecord[];
  events: ContextEvent[];
  onPick: (r: ContextRequestRecord) => void;
}) {
  const [granularity, setGranularity] = useState<"step" | "turn">("step");
  const [mode, setMode] = useState<"total" | "delta">("total");
  const [selected, setSelected] = useState<number | null>(null);

  const bars = useMemo(() => {
    if (granularity === "step") return requests;
    const byTurn = new Map<number, ContextRequestRecord>();
    for (const r of requests) byTurn.set(r.turn, r); // 每轮最后一根代表整轮
    return [...byTurn.values()];
  }, [requests, granularity]);

  const maxY = useMemo(() => {
    let m = 0;
    for (const b of bars) {
      const v = mode === "total" ? catTotal(b.category) : b.promptTokens || catTotal(b.category);
      if (v > m) m = v;
    }
    return m || 1;
  }, [bars, mode]);

  const compactSeqs = useMemo(
    () => new Set(events.filter((e) => e.kind === "compact").map((e) => e.seq)),
    [events],
  );

  const W = Math.max(bars.length * 26, 320);
  const H = 150;
  const padL = 46, padR = 8, padT = 14, padB = 18;
  const plotH = H - padT - padB;
  const bw = 20;

  const yTicks = [0, 0.5, 1].map((f) => ({
    y: padT + plotH - f * plotH,
    label: fmtTokens(Math.round(maxY * f)),
  }));

  const pick = (i: number) => {
    const r = bars[i];
    if (!r) return;
    setSelected(i);
    onPick(r);
  };

  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">上下文趋势</div>
        <div className="flex items-center gap-1 text-[10px]">
          {(["step", "turn"] as const).map((g) => (
            <button
              key={g}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${granularity === g ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setGranularity(g)}
            >{g === "step" ? "步数" : "轮次"}</button>
          ))}
          <span className="mx-0.5 h-3 w-px bg-border-soft" />
          {(["total", "delta"] as const).map((m) => (
            <button
              key={m}
              title={m === "delta" ? "增量模式（Phase B）" : undefined}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${mode === m ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"} ${m === "delta" ? "opacity-40" : ""}`}
              onClick={() => setMode(m)}
            >{m === "total" ? "全局" : "增量"}</button>
          ))}
        </div>
      </div>
      <div className="mt-1 text-[9px] text-fg-faint">✂ 表示压缩/剪枝 · 点击柱查看该步构成</div>
      <div className="overflow-x-auto">
        <svg viewBox={`0 0 ${W} ${H}`} className="mt-1 h-auto min-w-full" style={{ width: W }}>
          {yTicks.map((t, i) => (
            <g key={i}>
              <line x1={padL} y1={t.y} x2={W - padR} y2={t.y} stroke="var(--border-soft)" strokeWidth={0.5} />
              <text x={padL - 4} y={t.y + 3} fontSize={9} fill="var(--fg-faint)" textAnchor="end">{t.label}</text>
            </g>
          ))}
          {bars.map((b, i) => {
            const x = padL + i * (bw + 6);
            const total = catTotal(b.category);
            if (mode === "delta") {
              const prev = i > 0 ? catTotal(bars[i - 1].category) : 0;
              const d = total - prev;
              const h = Math.abs((d / maxY) * plotH);
              const y = d >= 0 ? padT + plotH - h : padT + plotH;
              return (
                <g key={b.seq} onClick={() => pick(i)} className="cursor-pointer" opacity={selected === i ? 1 : 0.85}>
                  <rect x={x} y={y} width={bw} height={Math.max(h, 1)} rx={1.5} fill={d >= 0 ? "#22c55e" : "#ef4444"}>
                    <title>{`第${b.turn}轮·第${b.step}步 Δ${d >= 0 ? "+" : ""}${fmtTokens(d)}`}</title>
                  </rect>
                </g>
              );
            }
            let acc = 0;
            return (
              <g key={b.seq} onClick={() => pick(i)} className="cursor-pointer" opacity={selected === i ? 1 : 0.9}>
                {CATS.map((c) => {
                  const v = b.category[c.key];
                  if (v <= 0) return null;
                  const h = (v / maxY) * plotH;
                  const y = padT + plotH - acc - h;
                  acc += h;
                  return (
                    <rect key={c.key} x={x} y={y} width={bw} height={Math.max(h, 1)} fill={CAT_COLORS[c.key]}>
                      <title>{`第${b.turn}轮·第${b.step}步 ${c.label} ${fmtTokens(v)} · 合计 ${fmtTokens(total)}`}</title>
                    </rect>
                  );
                })}
                {compactSeqs.has(b.seq) && (
                  <text x={x + bw / 2} y={padT - 3} fontSize={10} textAnchor="middle" fill="var(--warning, #f59e0b)">✂</text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </div>
  );
}

// ─── 步骤详情卡 ─────────────────────────────────────────────
function StepDetail({ record }: { record: ContextRequestRecord | null }) {
  if (!record) {
    return (
      <div className="rounded-lg border border-border-soft bg-bg p-3 text-[10px] text-fg-faint">
        点击趋势图中的柱查看该请求的输入、回复与上下文构成。
      </div>
    );
  }
  const total = catTotal(record.category);
  const cacheHit = record.cacheHitTokens ?? 0;
  const cacheMiss = record.cacheMissTokens ?? 0;
  const cachePct = cacheHit + cacheMiss > 0 ? (cacheHit * 100) / (cacheHit + cacheMiss) : null;
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">第{record.turn}轮·第{record.step}步</div>
        <div className="text-[9px] text-fg-faint tabular-nums font-mono">{new Date(record.ts * 1000).toLocaleTimeString()}</div>
      </div>
      <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[10px] text-fg-dim tabular-nums font-mono">
        {record.promptTokens ? <span>实际 prompt {fmtTokens(record.promptTokens)}</span> : null}
        {record.outputTokens ? <span>输出 {record.outputTokens}</span> : null}
        {cachePct != null ? <span>缓存 {cachePct.toFixed(2)}%</span> : null}
        <span>合计 ≈{fmtTokens(total)}</span>
      </div>
      {record.briefUser && (
        <div className="mt-1.5 flex gap-1.5 text-[10px]">
          <span className="shrink-0 text-fg-faint">输入</span>
          <span className="truncate font-mono text-fg-dim">{record.briefUser}</span>
        </div>
      )}
      {record.briefResp && (
        <div className="flex gap-1.5 text-[10px]">
          <span className="shrink-0 text-fg-faint">回复</span>
          <span className="truncate font-mono text-fg-dim">{record.briefResp}</span>
        </div>
      )}
      <div className="mt-2 flex h-2 w-full overflow-hidden rounded-full bg-bg-soft">
        {CATS.map((c) => {
          const w = total > 0 ? (record.category[c.key] / total) * 100 : 0;
          if (w <= 0) return null;
          return <div key={c.key} style={{ width: `${w}%`, background: CAT_COLORS[c.key] }} />;
        })}
      </div>
    </div>
  );
}

// ─── 上下文事件流 ───────────────────────────────────────────
const EVENT_KINDS = ["all", "inject", "compact", "prune", "switch", "mode"] as const;
const EVENT_LABEL: Record<string, string> = { inject: "注入", compact: "压缩", prune: "剪枝", switch: "切换", mode: "模式" };

function EventsList({ events }: { events: ContextEvent[] }) {
  const [filter, setFilter] = useState<(typeof EVENT_KINDS)[number]>("all");
  const shown = filter === "all" ? events : events.filter((e) => e.kind === filter);
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">上下文事件</div>
        <div className="flex items-center gap-1 text-[10px]">
          {EVENT_KINDS.map((k) => (
            <button
              key={k}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${filter === k ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setFilter(k)}
            >{k === "all" ? "全部" : EVENT_LABEL[k]}</button>
          ))}
        </div>
      </div>
      <div className="mt-1.5 max-h-44 overflow-y-auto">
        {shown.length === 0 && <div className="text-[10px] text-fg-faint py-2">暂无上下文事件</div>}
        {shown.slice(-30).reverse().map((e) => (
          <div key={`${e.kind}-${e.seq}`} className="flex items-center gap-1.5 py-0.5 text-[10px] border-b border-border-soft/40 last:border-0">
            <span className={`shrink-0 px-1 rounded text-[9px] ${e.kind === "compact" ? "bg-warning/15 text-warning" : e.kind === "prune" ? "bg-err/15 text-err" : "bg-accent/10 text-accent"}`}>
              {e.delta != null && e.delta < 0 ? "" : "+"}{EVENT_LABEL[e.kind] ?? e.kind}
            </span>
            <span className="truncate text-fg-dim">{e.source || "—"}</span>
            <span className="ml-auto shrink-0 text-fg-faint tabular-nums font-mono">
              第{e.turn}轮·第{e.step}步
              {e.delta != null && e.delta !== 0
                ? ` ${e.delta > 0 ? "+" : "-"}${fmtTokens(Math.abs(e.delta))}`
                : ""}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── 容器：拉取 + 布局 ──────────────────────────────────────
export function ContextView({ running }: { running: boolean }) {
  const [timeline, setTimeline] = useState<ContextTimeline>(EMPTY);
  const [picked, setPicked] = useState<ContextRequestRecord | null>(null);
  const [error, setError] = useState<string | null>(null);
  const runningRef = useRef(running);

  const load = useCallback(() => {
    app.ContextView()
      .then((tl) => { setTimeline(tl); setError(null); })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // 回合结束后刷新（running true→false），运行中由事件订阅兜底（Phase B）。
  useEffect(() => {
    if (runningRef.current && !running) load();
    runningRef.current = running;
  }, [running, load]);

  return (
    <div className="flex h-full flex-col gap-2 overflow-y-auto p-3">
      {error && (
        <div className="rounded-lg border border-err/30 bg-del-bg px-3 py-2 text-[11px] text-err">
          上下文视图加载失败：{error}
        </div>
      )}
      <StatsBoard stats={timeline.stats} />
      <CurrentComposition current={timeline.current} window={timeline.window} />
      <ContextTrendChart requests={timeline.requests} events={timeline.events} onPick={setPicked} />
      <StepDetail record={picked} />
      <EventsList events={timeline.events} />
      <AgentNetworkCard running={running} />
      <div className="text-center text-[9px] text-fg-faint">
        上下文浏览器 / 文件活动将在后续阶段接入
      </div>
    </div>
  );
}
