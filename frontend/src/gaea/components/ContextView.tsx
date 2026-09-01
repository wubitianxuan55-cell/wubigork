/* eslint-disable react-refresh/only-export-components -- 子组件与容器同文件（Phase A 收敛，避免过早拆文件） */
import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2 } from "../icons";
import { app } from "../lib/bridge";
import { AgentNetworkCard } from "./AgentNetworkCard";
import { usePreviewStore } from "../lib/store";
import type {
  ContextCategory, ContextEvent, ContextRequestRecord, ContextStats, ContextSurfaceNode, ContextTimeline, FileActivity,
} from "../lib/types";
import { fmtTokens } from "../lib/stats";
import { useLiveReload } from "../hooks/useLiveReload";

// 六分类语义色（效果图对齐：系统蓝/工具橙/用户绿/注入紫/助手深蓝/工具青）。
export const CAT_COLORS: Record<keyof ContextCategory, string> = {
  system: "#3b82f6", // hex-exempt 上下文六分类语义色（可视化调色板）
  tools: "#f59e0b", // hex-exempt 上下文六分类语义色
  user: "#22c55e", // hex-exempt 上下文六分类语义色
  inject: "#a855f7", // hex-exempt 上下文六分类语义色
  assistant: "#1e40af", // hex-exempt 上下文六分类语义色
  tool: "#06b6d4", // hex-exempt 上下文六分类语义色
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
  requests: [], events: [], nodes: [], archive: [], files: [],
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

// ─── 上下文总览头部：水位（分色）+ 缓存/费用 + 刷新 ─────────────
function ContextOverviewHeader({
  used,
  window,
  stats,
  running,
  loading,
  onRefresh,
}: {
  used: number;
  window: number;
  stats: ContextStats;
  running: boolean;
  loading: boolean;
  onRefresh: () => void;
}) {
  const pct = window > 0 ? Math.min(100, Math.round((used / window) * 100)) : 0;
  const levelCls = pct >= 90 ? "bg-err" : pct >= 70 ? "bg-warning" : "bg-accent";
  const levelText = pct >= 90 ? "text-err" : pct >= 70 ? "text-warning" : "text-fg-dim";
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center gap-2">
        <span className="text-[11px] font-medium text-fg">上下文</span>
        {running && (
          <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-accent animate-pulse" title="运行中实时刷新" aria-hidden />
        )}
        <span className="flex-1" />
        <span className="shrink-0 text-[10px] tabular-nums text-fg-faint">
          缓存 {stats.cacheHitPercent != null ? `${stats.cacheHitPercent.toFixed(2)}%` : "—"}
        </span>
        <span className="shrink-0 text-[10px] tabular-nums text-fg-faint">
          费用 {stats.costEstimate != null ? `¥${stats.costEstimate.toFixed(2)}` : "—"}
        </span>
        <button
          type="button"
          className="flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-md border-0 bg-transparent transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={onRefresh}
          title="刷新上下文"
          aria-label="刷新上下文"
        >
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>
      <div className="mt-2 flex items-center gap-2">
        <div className="h-2 min-w-0 flex-1 overflow-hidden rounded-full bg-bg-soft">
          <div className={`h-full rounded-full transition-[width] duration-500 ${levelCls}`} style={{ width: `${pct}%` }} />
        </div>
        <span className={`shrink-0 font-mono text-[10px] tabular-nums ${levelText}`}>
          {fmtTokens(used)} / {fmtTokens(window)} · {pct}%
        </span>
      </div>
      {pct >= 70 && (
        <div className={`mt-1 text-[9.5px] ${pct >= 90 ? "text-err" : "text-warning"}`}>
          {pct >= 90 ? "上下文接近上限，建议压缩或新建会话" : "上下文占用较高，注意后续注入空间"}
        </div>
      )}
    </div>
  );
}

// ─── 当前上下文组成（六分类分段条 + 图例） ─────────────────────
function CurrentComposition({ current, window }: { current: ContextCategory; window: number }) {
  const total = catTotal(current);
  const pct = window > 0 ? Math.round((total / window) * 100) : 0;
  const warnCls = pct >= 90 ? "text-err" : pct >= 70 ? "text-warning" : "";
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-baseline justify-between">
        <div className="text-[11px] font-medium text-fg">当前上下文</div>
        <div className={`text-[10px] tabular-nums font-mono ${warnCls || "text-fg-dim"}`}>
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
      {pct >= 70 && (
        <div className={`mt-1.5 text-[9.5px] ${pct >= 90 ? "text-err" : "text-warning"}`}>
          {pct >= 90 ? "已接近上下文上限 —— 压缩/剪枝后此处会回落" : "占用较高 —— 注意剩余注入空间"}
        </div>
      )}
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
  // v4.27 悬停构成详情：hover 柱时在图上展示该步六分类分解（原生 SVG title
  // 之外再给一个随时可见的 HTML 详情条，比浏览器默认 tooltip 更丰富）。
  const [hovered, setHovered] = useState<number | null>(null);

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
              title={m === "delta" ? "每步相对上一步的上下文净变化（绿=净增 · 红=净减）" : undefined}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${mode === m ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setMode(m)}
            >{m === "total" ? "全局" : "增量"}</button>
          ))}
        </div>
      </div>
      <div className="mt-1 text-[9px] text-fg-faint">
        ✂ 表示压缩/剪枝 · 点击柱查看该步构成
        {mode === "delta" && <span className="ml-2"><span className="text-[#22c55e]">■</span> 净增 <span className="ml-1 text-[#ef4444]">■</span> 净减</span>}
      </div>
      {hovered !== null && bars[hovered] && (
        <div
          data-testid="trend-hover-detail"
          className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 rounded-md px-2 py-1 text-[9.5px]"
          style={{ background: "var(--md-sys-color-surface-container-high)" }}
        >
          <span className="font-mono text-fg-dim">第{bars[hovered].turn}轮·第{bars[hovered].step}步</span>
          {CATS.map((c) =>
            bars[hovered].category[c.key] > 0 ? (
              <span key={c.key} className="inline-flex items-center gap-1 text-fg-faint">
                <span className="h-1.5 w-1.5 shrink-0 rounded-sm" style={{ background: CAT_COLORS[c.key] }} />
                {c.label} {fmtTokens(bars[hovered].category[c.key])}
              </span>
            ) : null,
          )}
          <span className="ml-auto font-mono text-fg-dim">合计 {fmtTokens(catTotal(bars[hovered].category))}</span>
        </div>
      )}
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
                <g
                  key={b.seq}
                  onClick={() => pick(i)}
                  onMouseEnter={() => setHovered(i)}
                  onMouseLeave={() => setHovered((cur) => (cur === i ? null : cur))}
                  className="cursor-pointer"
                  opacity={selected === i ? 1 : 0.85}
                >
                  <rect x={x} y={y} width={bw} height={Math.max(h, 1)} rx={1.5} fill={d >= 0 ? "#22c55e" : "#ef4444"}> {/* hex-exempt 增量图语义色（绿=净增/红=净减，可视化调色板） */}
                    <title>{`第${b.turn}轮·第${b.step}步 Δ${d >= 0 ? "+" : ""}${fmtTokens(d)}`}</title>
                  </rect>
                </g>
              );
            }
            let acc = 0;
            return (
              <g
                key={b.seq}
                onClick={() => pick(i)}
                onMouseEnter={() => setHovered(i)}
                onMouseLeave={() => setHovered((cur) => (cur === i ? null : cur))}
                className="cursor-pointer"
                opacity={selected === i ? 1 : 0.9}
              >
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
function StepDetail({ record, window }: { record: ContextRequestRecord | null; window: number }) {
  if (!record) {
    return (
      <div className="rounded-lg border border-border-soft bg-bg p-3 text-[10px] text-fg-faint">
        点击趋势图中的柱查看该请求的输入、回复与上下文构成。
      </div>
    );
  }
  const total = catTotal(record.category);
  const windowPct = window > 0 ? Math.round((total / window) * 100) : null;
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
        {record.estimated
          ? <span className="text-warning/90">估算构成（无用量记录）</span>
          : record.promptTokens ? <span>实际 prompt {fmtTokens(record.promptTokens)}</span> : null}
        {record.outputTokens ? <span>输出 {record.outputTokens}</span> : null}
        {cachePct != null ? <span>缓存 {cachePct.toFixed(2)}%</span> : null}
        <span>合计 ≈{fmtTokens(total)}</span>
        {windowPct != null && <span>占窗口 {windowPct}%</span>}
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

// ─── 文件活动（工具读写工作区文件的时间线） ─────────────────────
const FILE_ACTION_META: Record<FileActivity["action"], { label: string; cls: string }> = {
  read: { label: "读", cls: "bg-cyan-500/15 text-cyan-400" },
  write: { label: "写", cls: "bg-amber-500/15 text-amber-400" },
  move: { label: "移", cls: "bg-purple-500/15 text-purple-400" },
  dir: { label: "目录", cls: "bg-slate-500/15 text-slate-400" },
};

function FileActivityCard({ files }: { files: FileActivity[] }) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  const shown = files.slice(-40).reverse();
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">文件活动</div>
        <div className="text-[9px] text-fg-faint tabular-nums">{files.length} 次文件接触</div>
      </div>
      <div className="mt-1.5 max-h-44 overflow-y-auto">
        {shown.length === 0 && (
          <div className="py-2 text-[10px] text-fg-faint">暂无文件活动 —— 工具读写工作区文件后会出现在这里</div>
        )}
        {shown.map((f) => {
          const meta = FILE_ACTION_META[f.action] ?? { label: f.action, cls: "bg-bg-soft text-fg-dim" };
          const clickable = f.action !== "dir";
          return (
            <button
              key={`${f.tool}-${f.seq}`}
              type="button"
              disabled={!clickable}
              onClick={() => clickable && openFilePreview(f.path)}
              title={clickable ? `在右侧预览 ${f.path}` : undefined}
              className={`flex w-full items-center gap-1.5 border-b border-border-soft/40 py-0.5 text-left text-[10px] last:border-0 ${
                clickable ? "cursor-pointer transition-colors hover:bg-bg-soft/70" : "cursor-default"
              }`}
            >
              <span className={`shrink-0 rounded px-1 text-[9px] ${meta.cls}`}>{meta.label}</span>
              <span className="shrink-0 font-mono text-fg-faint">{f.tool}</span>
              <span className="truncate font-mono text-fg-dim" title={f.path}>{f.path}</span>
              <span className="ml-auto shrink-0 tabular-nums font-mono text-fg-faint">{new Date(f.ts * 1000).toLocaleTimeString()}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

// ─── 上下文浏览器（模型可见 surface 节点 + 归档） ───────────────
const CAT_BROWSE_LABELS: Record<ContextSurfaceNode["cat"], string> = {
  system: "系统",
  tools: "工具",
  user: "用户",
  inject: "注入",
  assistant: "助手",
  tool: "工具结果",
};
const BROWSE_CATS = ["all", "system", "tools", "user", "inject", "assistant", "tool"] as const;

function NodeRow({ node, open, onToggle }: { node: ContextSurfaceNode; open: boolean; onToggle: () => void }) {
  const text = node.text || "(无预览 —— 全文在事件日志 request_header 中)";
  const truncated = text.length > 56;
  const shown = open || !truncated ? text : `${text.slice(0, 56)}…`;
  return (
    <div className="flex items-start gap-1.5 border-b border-border-soft/40 py-1 text-[10px] last:border-0">
      <span className="mt-0.5 h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[node.cat] }} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 text-fg-faint">
          <span className="font-medium" style={{ color: CAT_COLORS[node.cat] }}>{CAT_BROWSE_LABELS[node.cat]}</span>
          <span className="tabular-nums font-mono">≈{fmtTokens(node.tokens)}</span>
          {node.gone != null && <span className="text-warning">已压缩</span>}
          {truncated && (
            <button
              className="ml-auto cursor-pointer border-0 bg-transparent text-accent hover:underline"
              onClick={onToggle}
            >{open ? "收起" : "展开"}</button>
          )}
        </div>
        <div className="mt-0.5 whitespace-pre-wrap break-all font-mono text-fg-dim">{shown}</div>
      </div>
    </div>
  );
}

function ContextBrowserCard({ nodes, archive }: { nodes: ContextSurfaceNode[]; archive: ContextSurfaceNode[] }) {
  const [tab, setTab] = useState<"active" | "archive">("active");
  const [cat, setCat] = useState<"all" | ContextSurfaceNode["cat"]>("all");
  const [openSeq, setOpenSeq] = useState<number | null>(null);
  const list = tab === "active" ? nodes : archive;
  const shown = cat === "all" ? list : list.filter((n) => n.cat === cat);
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">上下文浏览器</div>
        <div className="flex items-center gap-1 text-[10px]">
          {(["active", "archive"] as const).map((t) => (
            <button
              key={t}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${tab === t ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setTab(t)}
            >{t === "active" ? `活跃 ${nodes.length}` : `归档 ${archive.length}`}</button>
          ))}
        </div>
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-1 text-[10px]">
        {BROWSE_CATS.map((c) => (
          <button
            key={c}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${cat === c ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            onClick={() => setCat(c)}
          >{c === "all" ? "全部" : CAT_BROWSE_LABELS[c]}</button>
        ))}
      </div>
      <div className="mt-1.5 max-h-52 overflow-y-auto">
        {shown.length === 0 && <div className="py-2 text-[10px] text-fg-faint">没有该分类的上下文节点</div>}
        {shown.slice(-60).map((n) => (
          <NodeRow key={n.seq} node={n} open={openSeq === n.seq} onToggle={() => setOpenSeq(openSeq === n.seq ? null : n.seq)} />
        ))}
      </div>
      <div className="mt-1 text-[9px] text-fg-faint">
        节点 = 模型可见的上下文元素 · 系统/工具在构成变化时记录 · 归档 = 被压缩移出的内容
      </div>
    </div>
  );
}

// ─── 容器：拉取 + 布局 ──────────────────────────────────────
export function ContextView({ running, sessionPath }: { running: boolean; sessionPath?: string }) {
  const [timeline, setTimeline] = useState<ContextTimeline>(EMPTY);
  const [picked, setPicked] = useState<ContextRequestRecord | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    app.ContextView()
      // 老后端可能把空切片序列化成 null，按数组消费前统一归一化
      .then((tl) => {
        setTimeline({
          ...tl,
          requests: tl.requests ?? [],
          events: tl.events ?? [],
          nodes: tl.nodes ?? [],
          archive: tl.archive ?? [],
          files: tl.files ?? [],
        });
        setError(null);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // 运行中随事件流节流刷新 + 回合结束立即刷新（useLiveReload）。
  useLiveReload(running, load);

  const isEmpty =
    timeline.window <= 0 &&
    catTotal(timeline.current) === 0 &&
    timeline.requests.length === 0 &&
    timeline.events.length === 0 &&
    timeline.nodes.length === 0 &&
    timeline.files.length === 0;

  return (
    <div className="flex h-full flex-col gap-2 overflow-y-auto p-3">
      {error && (
        <div className="rounded-lg border border-err/30 bg-del-bg px-3 py-2 text-[11px] text-err">
          上下文视图加载失败：{error}
        </div>
      )}
      {!error && (
        <ContextOverviewHeader
          used={catTotal(timeline.current)}
          window={timeline.window}
          stats={timeline.stats}
          running={running}
          loading={loading}
          onRefresh={() => void load()}
        />
      )}
      {!error && isEmpty && (
        <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border-soft bg-bg px-6 py-10 text-center">
          <span className="text-[12px] font-medium text-fg">暂无上下文数据</span>
          <span className="max-w-[46ch] text-[10.5px] leading-relaxed text-fg-faint">
            运行任务后，上下文构成、趋势、注入/压缩事件与文件活动会实时显示在这里。
          </span>
        </div>
      )}
      {!error && !isEmpty && (
        <>
      <StatsBoard stats={timeline.stats} />
      <CurrentComposition current={timeline.current} window={timeline.window} />
      <ContextTrendChart requests={timeline.requests} events={timeline.events} onPick={setPicked} />
      <StepDetail record={picked} window={timeline.window} />
      <EventsList events={timeline.events} />
      <FileActivityCard files={timeline.files} />
      <ContextBrowserCard nodes={timeline.nodes} archive={timeline.archive} />
      <AgentNetworkCard running={running} sessionPath={sessionPath} />
        </>
      )}
    </div>
  );
}
