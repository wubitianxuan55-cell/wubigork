// 本文件导出面只含组件（ContextView/ContextTrendChart）；六分类键名与语义色
// 常量在 context/cats.ts 单源（CAT_COLORS 收编后不再有非组件导出）。
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type ReactNode, type UIEvent } from "react";
import { app } from "../lib/bridge";
import type { FamilyTotalsView } from "../lib/types/context";
import { useT } from "../lib/i18n";
import type {
  ContextCategory,
  ContextCompactTiers,
  ContextDnaBand,
  ContextDnaSample,
  ContextEvent,
  ContextNodeDetailView,
  ContextRequestRecord,
  ContextSurfaceSnapshot,
  ContextTimeline,
} from "../lib/types";
import type { DictKey } from "../locales/en";
import { fmtTokens } from "../lib/stats";
import { fmtDuration } from "./context/cards";
import { loadContextPrefs, saveContextPref, type CtxTrendGranularity, type CtxTrendMode } from "../lib/contextPrefs";
import { useLiveReload } from "../hooks/useLiveReload";
import { subscribeAgentNetwork, reloadAgentNetwork } from "../lib/agentNetworkStore";
import { CONTEXT_CATS, CONTEXT_CAT_BROWSE, CAT_COLORS } from "./context/cats";

// 六分类语义色 CAT_COLORS 定义已收编至 context/cats.ts 单源（i18n 收口批），
// 本文件经上方导入消费；inspector/cards 也直接从 cats.ts 取。
import { StatsCard, TokenCard, TimingCard, SessionInfoCard, SummaryBar } from "./context/cards";
import { ContextBrowserTree, FileActivityTree } from "./context/inspector";
import type { BrowserCatFocus, FileChipFocus } from "./context/inspector";
import { CommandsCard } from "./context/CommandsCard";
import { AgentRadial } from "./context/AgentRadial";
import "./context/context-view.css";
import { BarChart3, LineChart, List } from "../icons";
import type { AgentNetwork } from "../lib/types";

// 六分类语义色在 context/cats.ts 单源（CAT_COLORS，经文件头导入消费）。

// 六分类单一事实源在 context/cats.ts（FE1-02）：此处派生面板所需的 key+全名。
const CATS: { key: keyof ContextCategory; labelKey: DictKey }[] = CONTEXT_CATS.map(({ key, labelKey }) => ({ key, labelKey }));

// v4.79 delta 徽标的分类短名（与浏览器组行短名同款键；surface 节点 cat 与
// 六分类 key 同字面量）。
const CAT_BROWSE_SHORT: Record<string, DictKey> = CONTEXT_CAT_BROWSE;

function catTotal(c: ContextCategory): number {
  return c.system + c.tools + c.user + c.inject + c.skill + c.assistant + c.tool;
}

interface TrendDeltaPart {
  cat: keyof ContextCategory;
  tokens: number;
}

interface TrendDeltaBar {
  parts: TrendDeltaPart[];
  up: number;
  down: number;
  net: number;
}

function deltaBarOf(current: ContextCategory, previous?: ContextCategory): TrendDeltaBar {
  const parts = CATS.map(({ key }) => ({
    cat: key,
    tokens: current[key] - (previous?.[key] ?? 0),
  }));
  let up = 0;
  let down = 0;
  for (const part of parts) {
    if (part.tokens > 0) up += part.tokens;
    else if (part.tokens < 0) down += Math.abs(part.tokens);
  }
  return { parts, up, down, net: up - down };
}

interface DnaDeltaPart {
  band: ContextDnaBand;
  tokens: number;
}

function dnaTotal(bands: readonly ContextDnaBand[]): number {
  let total = 0;
  for (const band of bands) total += Math.max(0, band.tokens);
  return total;
}

function dnaDeltaOf(current: readonly ContextDnaBand[], previous?: readonly ContextDnaBand[]) {
  if (!previous) return { up: [] as DnaDeltaPart[], down: [] as DnaDeltaPart[], net: 0 };
  const before = new Map(previous.map((band) => [band.key, band] as const));
  const after = new Map(current.map((band) => [band.key, band] as const));
  const up: DnaDeltaPart[] = [];
  const down: DnaDeltaPart[] = [];
  for (const band of current) {
    const delta = band.tokens - (before.get(band.key)?.tokens ?? 0);
    if (delta > 0) up.push({ band, tokens: delta });
    before.delete(band.key);
  }
  for (const band of previous) {
    const delta = (after.get(band.key)?.tokens ?? 0) - band.tokens;
    if (delta < 0) down.push({ band, tokens: -delta });
  }
  const upTotal = up.reduce((sum, part) => sum + part.tokens, 0);
  const downTotal = down.reduce((sum, part) => sum + part.tokens, 0);
  return { up, down, net: upTotal - downTotal };
}

const EMPTY: ContextTimeline = {
  ok: true, window: 0,
  current: { system: 0, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 },
  stats: { turns: 0, steps: 0, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0 },
  requests: [], events: [], nodes: [], archive: [], files: [], commands: [],
};

const contextScrollBySession = new Map<string, number>();

type BrowserSurfaceMode = "live" | "step";
type SurfaceReadState =
  | { status: "live" }
  | { status: "loading"; seq: number }
  | { status: "ready"; seq: number; snapshot: ContextSurfaceSnapshot }
  | { status: "error"; seq: number }
  | { status: "unavailable"; seq: number };

// ─── 当前上下文宽卡（对齐 dsh CurrentComposition：大数字+分段条含空闲+图例） ──
// 压缩水位线（上游 ContextWindowRing 同款语义、gaea 三层水位版）：compact
// 在场且窗口已知时，分段条上叠 soft/ratio/force 三道水位竖标，状态行按
// 「<soft 健康（不渲染）/≥soft 增长中+距压缩/≥ratio 已达压缩/≥force 强制」
// 四档呈现——替换旧静态 ≥70/90 提示（compact 缺席时保留旧提示兼容旧后端）。
function CurrentContextCard({
  used,
  window: win,
  current,
  compact,
}: {
  used: number;
  window: number;
  current: ContextCategory;
  compact?: ContextCompactTiers;
}) {
  const t = useT();
  // v4.69：图例 chip hover 联动（dsh hover-key 同款）——悬停某分类时该段保持
  // 不透明度、其余段淡出（150ms opacity 过渡），视觉焦点即时定位。
  const [activeCat, setActiveCat] = useState<keyof ContextCategory | "idle" | null>(null);
  const segOpacity = (key: keyof ContextCategory | "idle") =>
    activeCat !== null && activeCat !== key ? 0.28 : 1;
  const chipDim = (key: keyof ContextCategory | "idle") =>
    activeCat !== null && activeCat !== key ? "opacity-45" : "opacity-100";
  const leaveChip = (key: keyof ContextCategory | "idle") => setActiveCat((cur) => (cur === key ? null : cur));
  const total = used;
  // 窗口未知（win=0，如原罪估算口径）时百分比诚实显示「—」，不伪造 0%。
  const pctExact = win > 0 ? Math.min(100, (total / win) * 100) : null;
  const pct = pctExact != null ? Math.round(pctExact) : null;
  const idle = Math.max(0, win - total);
  const toolPct = total > 0 ? Math.round((current.tool / total) * 100) : 0;
  // 水位块仅在窗口已知且三值齐备时生效（后端 nil=内核未初始化/窗口未知）。
  const tiers = win > 0 && compact && compact.ratio > 0 && compact.soft > 0 && compact.force > 0 ? compact : null;
  const pctRaw = win > 0 ? (total / win) * 100 : 0;
  const toCompact = tiers ? Math.max(0, Math.round(win * tiers.ratio) - total) : 0;
  const marks = tiers
    ? ([
        { key: "soft", at: tiers.soft * 100, color: "var(--md-sys-color-outline-variant)", label: t("contextview.wmSoft", { pct: Math.round(tiers.soft * 100) }) },
        { key: "ratio", at: tiers.ratio * 100, color: "var(--md-sys-color-warning, #f59e0b)", label: t("contextview.wmRatio", { pct: Math.round(tiers.ratio * 100) }) },
        { key: "force", at: tiers.force * 100, color: "var(--md-sys-color-destructive)", label: t("contextview.wmForce", { pct: Math.round(tiers.force * 100) }) },
      ] as const)
    : [];
  return (
    <div className="ctx-card ctx-hero p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <BarChart3 size={12} />
          </span>
          <span className="text-[12.5px] font-semibold text-fg">{t("contextview.currentTitle")}</span>
        </div>
        {current.tool > 0 && (
          <div className="text-[10.5px] tabular-nums text-fg-faint">
            {t("contextview.toolResultShare", { tokens: fmtTokens(current.tool), pct: toolPct })}
          </div>
        )}
      </div>
      <div className="mt-1.5 flex items-baseline gap-1.5">
        <span className="font-mono text-[26px] font-bold leading-none tracking-tight text-fg">{fmtTokens(total)}</span>
        <span className="text-[12px] text-fg-faint">/ {fmtTokens(win)} tokens</span>
        <span className="ml-auto flex items-baseline gap-1">
          {pct != null ? (
            <>
              <b className="font-mono text-[26px] font-bold leading-none text-fg">{pctExact!.toFixed(1)}%</b>
              <span className="text-[11px] text-fg-faint">{t("contextview.pctUsed")}</span>
            </>
          ) : (
            <b className="font-mono text-[26px] font-bold leading-none text-fg-faint" title={t("contextview.windowUnknown")}>—</b>
          )}
        </span>
      </div>
      <div className="relative mt-2">
        <div className="flex h-4 w-full overflow-hidden rounded-md bg-bg-soft" role="img" aria-label={t("contextview.currentTitle")}>
          {CATS.map((c) => {
            const w = total > 0 ? (current[c.key] / total) * 100 : 0;
            if (w <= 0) return null;
            return (
              <div
                key={c.key}
                data-testid={`comp-seg-${c.key}`}
                data-cat={c.key}
                style={{ width: `${w}%`, background: CAT_COLORS[c.key], opacity: segOpacity(c.key), transition: "opacity 150ms ease" }}
              />
            );
          })}
          {idle > 0 && (pct == null || pct < 100) && (
            <div
              data-testid="comp-seg-idle"
              className="h-full"
              style={{
                width: `${(idle / win) * 100}%`,
                background: "color-mix(in srgb, var(--md-sys-color-text-secondary) 14%, transparent)",
                opacity: segOpacity("idle"),
                transition: "opacity 150ms ease",
              }}
              title={t("contextview.idleWindow")}
            />
          )}
        </div>
        {/* 压缩水位竖标：叠加在分段条上方（宽度=窗口占比，与分段同基准） */}
        {marks.map((m) => (
          <span
            key={m.key}
            data-testid={`comp-wm-${m.key}`}
            className="absolute top-0 h-4 w-px cursor-default"
            style={{ left: `${Math.min(100, m.at)}%`, background: m.color }}
            title={m.label}
          />
        ))}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-0.5">
        {CATS.map((c) => {
          const share = total > 0 ? Math.round((current[c.key] / total) * 100) : 0;
          return (
            <span
              key={c.key}
              data-testid={`comp-chip-${c.key}`}
              data-cat={c.key}
              className={`inline-flex cursor-default items-center gap-1 text-[11.5px] text-fg-dim transition-opacity duration-150 ${chipDim(c.key)} ${
                activeCat === c.key ? "rounded bg-accent/10 px-1" : ""
              }`}
              onMouseEnter={() => setActiveCat(c.key)}
              onMouseLeave={() => leaveChip(c.key)}
            >
              <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[c.key] }} />
              {t(c.labelKey)}
              <span className="tabular-nums font-mono text-fg-faint">
                ≈{fmtTokens(current[c.key])} ({share}%)
              </span>
            </span>
          );
        })}
        <span
          data-testid="comp-chip-idle"
          className={`inline-flex cursor-default items-center gap-1 text-[11.5px] text-fg-dim transition-opacity duration-150 ${chipDim("idle")} ${
            activeCat === "idle" ? "rounded bg-accent/10 px-1" : ""
          }`}
          onMouseEnter={() => setActiveCat("idle")}
          onMouseLeave={() => leaveChip("idle")}
        >
          <span
            className="h-2 w-2 shrink-0 rounded-sm"
            style={{ background: "color-mix(in srgb, var(--md-sys-color-text-secondary) 14%, transparent)" }}
          />
          {t("contextview.idleWindow")}
        </span>
      </div>
      {tiers ? (
        pctRaw >= tiers.force * 100 ? (
          <div className="mt-1.5 text-[9.5px] text-err">{t("contextview.wmStatusForce", { pct: Math.round(tiers.force * 100) })}</div>
        ) : pctRaw >= tiers.ratio * 100 ? (
          <div className="mt-1.5 text-[9.5px] text-warning">{t("contextview.wmStatusRatio", { pct: Math.round(tiers.ratio * 100) })}</div>
        ) : pctRaw >= tiers.soft * 100 ? (
          <div className="mt-1.5 text-[9.5px] text-fg-faint">
            {t("contextview.wmStatusSoft", { tokens: fmtTokens(toCompact) })}
          </div>
        ) : null
      ) : (
        pct != null &&
        pct >= 70 && (
          <div className={`mt-1.5 text-[9.5px] ${pct >= 90 ? "text-err" : "text-warning"}`}>
            {pct >= 90 ? t("contextview.almostFull") : t("contextview.highUsage")}
            <span className="ml-2 opacity-80">{pct >= 90 ? t("contextview.almostFullHint") : t("contextview.highUsageHint")}</span>
          </div>
        )
      )}
    </div>
  );
}

// ─── 上下文趋势（原生 SVG 堆叠柱，全局模式；增量 Phase B） ─────
// v4.69：detail 插槽把「选中步骤详情」收进趋势卡内部（对齐 dsh：趋势卡即
// master-detail 单元，详情不再单独占一行跨屏滚动）。
// v4.96：导出供定向测试（对齐本文件 react-refresh 豁免的收敛模式）；
// focus 锚点（Token 卡「查看趋势」下钻最后一跳）——tick 去重，滚动定位到卡
// （scrollIntoView block:nearest，与 v4.82 brief→浏览器锚点同款机制）+ 3s
// accent 外圈强调；只定位不改选中态（默认最新请求语义保持）。
export function ContextTrendChart({
  requests,
  events,
  onPick,
  detail,
  focus,
  onHover,
  onPickCategory,
  granularity: granularityProp,
  onGranularity,
  selectedSeq,
  sessionKey,
  fetchDna,
}: {
  requests: ContextRequestRecord[];
  events: ContextEvent[];
  onPick: (r: ContextRequestRecord) => void;
  detail?: ReactNode;
  focus?: { tick: number } | null;
  onHover?: (record: ContextRequestRecord | null) => void;
  onPickCategory?: (cat: keyof ContextCategory, record: ContextRequestRecord) => void;
  // 刀12：粒度态上提（父层持有以驱动 StepDetail 的 turn 模式标题——上游
  // detail.turn「第 {t} 轮 · 共 {n} 步」同款）；直连渲染（测试）时可省略，
  // 回落卡内自持态。
  granularity?: CtxTrendGranularity;
  onGranularity?: (g: CtxTrendGranularity) => void;
  /** 父层钉住的请求；用于同步柱高亮、横向定位与历史事件下钻。 */
  selectedSeq?: number | null;
  /** 会话切换时重置自动总览的手动锁与视图局部状态。 */
  sessionKey?: string;
  /** 批量读取横向可见请求的逐节点读取序；缺省时 DNA 诚实不可用。 */
  fetchDna?: (seqs: number[]) => Promise<ContextDnaSample[]>;
}) {
  const t = useT();
  // 2.5d：粒度/模式初值读设置中心偏好（卡内 toggle 交互不变，变更写回）。
  const [granState, setGranState] = useState<CtxTrendGranularity>(() => loadContextPrefs().trendGranularity);
  const granularity = granularityProp ?? granState;
  const [mode, setMode] = useState<CtxTrendMode>(() => loadContextPrefs().trendMode);
  const [adaptive, setAdaptive] = useState(false);
  const [overview, setOverview] = useState(false);
  const [overviewAuto] = useState(() => loadContextPrefs().trendOverviewAuto);
  const [dnaOn, setDnaOn] = useState(false);
  const [dnaBandsBySeq, setDnaBandsBySeq] = useState<Map<number, ContextDnaBand[]>>(() => new Map());
  const [dnaLoading, setDnaLoading] = useState(false);
  const [dnaFailed, setDnaFailed] = useState(false);
  const [dnaHoverCat, setDnaHoverCat] = useState<keyof ContextCategory | null>(null);
  const dnaCache = useRef<Map<number, ContextDnaBand[]>>(new Map());
  const dnaMisses = useRef<Set<number>>(new Set());
  const [selected, setSelected] = useState<number | null>(null);
  // v4.27 悬停构成详情：hover 柱时在图上展示该步六分类分解（原生 SVG title
  // 之外再给一个随时可见的 HTML 详情条，比浏览器默认 tooltip 更丰富）。
  const [hovered, setHovered] = useState<number | null>(null);
  // v4.96 Token 卡→趋势锚点：handledFocus 去重（同 ContextBrowserTree focus
  // 机制）；flash 强调 3s 后自动解除。
  const cardRef = useRef<HTMLDivElement | null>(null);
  const chartScrollRef = useRef<HTMLDivElement | null>(null);
  const [flash, setFlash] = useState(false);
  const handledFocus = useRef(0);
  const autoOverviewSession = useRef<string | undefined>(undefined);
  const autoOverviewFired = useRef(false);
  const manualOverview = useRef(false);
  const lastBarsSignature = useRef("");

  useEffect(() => {
    if (!focus || focus.tick === handledFocus.current) return;
    handledFocus.current = focus.tick;
    setFlash(true);
    requestAnimationFrame(() => {
      cardRef.current?.scrollIntoView({ block: "nearest" });
    });
    const timer = window.setTimeout(() => setFlash(false), 3000);
    return () => window.clearTimeout(timer);
  }, [focus]);

  const pickGranularity = (g: CtxTrendGranularity) => {
    setGranState(g);
    onGranularity?.(g);
    saveContextPref("trendGranularity", g);
  };
  const pickMode = (m: CtxTrendMode) => {
    setMode(m);
    if (m === "delta" && !overview) setAdaptive(true);
    saveContextPref("trendMode", m);
  };

  const bars = useMemo(() => {
    if (granularity === "step") return requests;
    const byTurn = new Map<number, ContextRequestRecord>();
    for (const r of requests) byTurn.set(r.turn, r); // 每轮最后一根代表整轮
    return [...byTurn.values()];
  }, [requests, granularity]);

  const W = Math.max(bars.length * 26, 320);
  const H = 150;
  const padL = 46, padR = 40, padT = 14, padB = 18;
  const plotH = H - padT - padB;
  const bw = 20;
  const pitch = bw + 6;
  const [visibleRange, setVisibleRange] = useState(() => ({ from: 0, to: Math.max(0, bars.length - 1) }));

  const measureVisibleRange = useCallback((el: HTMLDivElement) => {
    const count = bars.length;
    if (count === 0 || el.clientWidth <= 0 || el.scrollWidth <= 0) {
      setVisibleRange((current) => (
        current.from === 0 && current.to === Math.max(0, count - 1)
          ? current
          : { from: 0, to: Math.max(0, count - 1) }
      ));
      return;
    }
    const from = Math.max(0, Math.min(count - 1, Math.floor((el.scrollLeft - padL) / pitch)));
    const to = Math.max(from, Math.min(count - 1, Math.ceil((el.scrollLeft + el.clientWidth - padL) / pitch)));
    setVisibleRange((current) => (current.from === from && current.to === to ? current : { from, to }));
  }, [bars.length, padL, pitch]);

  const onChartScroll = (event: UIEvent<HTMLDivElement>) => {
    measureVisibleRange(event.currentTarget);
  };

  useLayoutEffect(() => {
    const el = chartScrollRef.current;
    if (!el) return;
    const newest = bars.at(-1)?.seq ?? 0;
    const signature = `${granularity}:${bars.length}:${newest}`;
    const previous = lastBarsSignature.current;
    const atEnd = el.scrollWidth - el.scrollLeft - el.clientWidth < 28;
    if (previous === "" || (signature !== previous && atEnd)) {
      el.scrollLeft = Math.max(0, el.scrollWidth);
    }
    lastBarsSignature.current = signature;
    measureVisibleRange(el);
  }, [bars, granularity, measureVisibleRange]);

  useLayoutEffect(() => {
    const el = chartScrollRef.current;
    if (!el) return;
    measureVisibleRange(el);
    if (typeof ResizeObserver !== "function") return;
    const observer = new ResizeObserver(() => measureVisibleRange(el));
    observer.observe(el);
    return () => observer.disconnect();
  }, [measureVisibleRange]);

  useEffect(() => {
    if (selectedSeq == null) return;
    const index = bars.findIndex((bar) => bar.seq === selectedSeq);
    if (index >= 0) setSelected(index);
  }, [bars, selectedSeq]);

  useLayoutEffect(() => {
    if (selectedSeq == null) return;
    const el = chartScrollRef.current;
    const index = bars.findIndex((bar) => bar.seq === selectedSeq);
    if (!el || index < 0 || el.clientWidth <= 0) return;
    el.scrollLeft = Math.max(0, Math.min(
      el.scrollWidth - el.clientWidth,
      padL + index * pitch + bw / 2 - el.clientWidth / 2,
    ));
    measureVisibleRange(el);
  }, [bars, bw, measureVisibleRange, padL, pitch, selectedSeq]);

  useEffect(() => {
    if (autoOverviewSession.current === sessionKey) return;
    autoOverviewSession.current = sessionKey;
    autoOverviewFired.current = false;
    manualOverview.current = false;
    setOverview(false);
    setAdaptive(false);
    setSelected(null);
    setDnaOn(false);
    setDnaLoading(false);
    setDnaFailed(false);
    setDnaHoverCat(null);
    dnaCache.current.clear();
    dnaMisses.current.clear();
    setDnaBandsBySeq(new Map());
  }, [sessionKey]);

  useEffect(() => {
    if (overviewAuto === "off" || autoOverviewFired.current || manualOverview.current) return;
    const threshold = Number(overviewAuto);
    if (bars.length < threshold || threshold <= 0) return;
    autoOverviewFired.current = true;
    setOverview(true);
    setAdaptive(false);
    setDnaOn(false);
  }, [bars.length, overviewAuto]);

  useEffect(() => {
    if (!dnaOn || overview || bars.length === 0) {
      setDnaLoading(false);
      return;
    }
    if (!fetchDna) {
      setDnaFailed(true);
      return;
    }
    const from = Math.max(0, visibleRange.from - 1);
    const to = Math.min(bars.length - 1, visibleRange.to);
    const needed: number[] = [];
    for (let index = from; index <= to; index += 1) {
      const seq = bars[index]?.seq;
      if (seq != null && !dnaCache.current.has(seq) && !dnaMisses.current.has(seq)) needed.push(seq);
    }
    if (needed.length === 0) {
      setDnaLoading(false);
      return;
    }
    let alive = true;
    setDnaLoading(true);
    setDnaFailed(false);
    const timer = window.setTimeout(() => {
      fetchDna(needed)
        .then((samples) => {
          if (!alive) return;
          for (const sample of samples) {
            if (sample.ok) dnaCache.current.set(sample.seq, sample.bands ?? []);
            else dnaMisses.current.add(sample.seq);
          }
          setDnaBandsBySeq(new Map(dnaCache.current));
          setDnaFailed(samples.length > 0 && samples.every((sample) => !sample.ok));
        })
        .catch(() => {
          if (alive) setDnaFailed(true);
        })
        .finally(() => {
          if (alive) setDnaLoading(false);
        });
    }, 90);
    return () => {
      alive = false;
      window.clearTimeout(timer);
    };
  }, [bars, dnaOn, fetchDna, overview, visibleRange.from, visibleRange.to]);

  const dnaData = useMemo(
    () => bars.map((bar, index) => {
      const bands = dnaBandsBySeq.get(bar.seq);
      const previous = index > 0 ? dnaBandsBySeq.get(bars[index - 1].seq) : undefined;
      return {
        bands,
        delta: bands ? dnaDeltaOf(bands, previous) : null,
      };
    }),
    [bars, dnaBandsBySeq],
  );

  const setHover = (index: number | null) => {
    setHovered(index);
    onHover?.(index == null ? null : bars[index] ?? null);
  };

  const deltaBars = useMemo(
    () => bars.map((bar, index) => deltaBarOf(bar.category, index > 0 ? bars[index - 1].category : undefined)),
    [bars],
  );
  const visibleBars = useMemo(
    () => (adaptive ? bars.slice(visibleRange.from, visibleRange.to + 1) : bars),
    [adaptive, bars, visibleRange.from, visibleRange.to],
  );
  const visibleDeltaBars = useMemo(
    () => (adaptive ? deltaBars.slice(visibleRange.from, visibleRange.to + 1) : deltaBars),
    [adaptive, deltaBars, visibleRange.from, visibleRange.to],
  );
  const visibleDnaData = useMemo(
    () => (adaptive ? dnaData.slice(visibleRange.from, visibleRange.to + 1) : dnaData),
    [adaptive, dnaData, visibleRange.from, visibleRange.to],
  );
  const maxTotal = useMemo(
    () => Math.max(1, ...visibleBars.map((bar, index) => {
      const bands = visibleDnaData[index]?.bands;
      return dnaOn && bands ? dnaTotal(bands) : catTotal(bar.category);
    })),
    [dnaOn, visibleBars, visibleDnaData],
  );
  const maxUp = useMemo(
    () => Math.max(0, ...visibleDnaData.map((item, index) => (
      dnaOn
        ? item.delta?.up.reduce((sum, part) => sum + part.tokens, 0) ?? 0
        : visibleDeltaBars[index]?.up ?? 0
    ))),
    [dnaOn, visibleDeltaBars, visibleDnaData],
  );
  const maxDown = useMemo(
    () => Math.max(0, ...visibleDnaData.map((item, index) => (
      dnaOn
        ? item.delta?.down.reduce((sum, part) => sum + part.tokens, 0) ?? 0
        : visibleDeltaBars[index]?.down ?? 0
    ))),
    [dnaOn, visibleDeltaBars, visibleDnaData],
  );
  const deltaSpan = Math.max(1, maxUp + maxDown);
  const deltaBaseline = maxUp + maxDown > 0
    ? padT + (maxUp / deltaSpan) * plotH
    : padT + plotH;
  const maxY = mode === "total" ? maxTotal : deltaSpan;

  const compactSeqs = useMemo(
    () => new Set(events.filter((e) => e.kind === "compact").map((e) => e.seq)),
    [events],
  );

  const yTicks = mode === "total"
    ? [0, 0.25, 0.5, 0.75, 1].map((f) => ({
        y: padT + plotH - f * plotH,
        label: fmtTokens(Math.round(maxTotal * f)),
      }))
    : [
        ...(maxUp > 0
          ? [
              { y: padT, label: `+${fmtTokens(maxUp)}` },
              { y: padT + (deltaBaseline - padT) / 2, label: `+${fmtTokens(Math.round(maxUp / 2))}` },
            ]
          : []),
        { y: deltaBaseline, label: "0" },
        ...(maxDown > 0
          ? [
              { y: deltaBaseline + (padT + plotH - deltaBaseline) / 2, label: `-${fmtTokens(Math.round(maxDown / 2))}` },
              { y: padT + plotH, label: `-${fmtTokens(maxDown)}` },
            ]
          : []),
      ];

  // 1:1 上游 Duration 耗时曲线（右上切换）：每请求 activeMs 折线+右轴时间
  // 刻度。旧日志/纯工具步无 activeMs 的点断开（诚实缺段）。
  // 刀15：开关持久化（上游 settings.durationCurve 同款）——初值读偏好，
  // 切换写回，重开页面保持上次选择。
  const [durationOn, setDurationOn] = useState<boolean>(() => loadContextPrefs().trendDuration);
  const toggleDuration = () =>
    setDurationOn((v) => {
      const nv = !v;
      saveContextPref("trendDuration", nv);
      return nv;
    });
  const durPoints = useMemo(() => {
    if (!durationOn) return [];
    const pts: { x: number; y: number; ms: number }[] = [];
    let maxMs = 0;
    bars.forEach((b, i) => {
      if (adaptive && (i < visibleRange.from || i > visibleRange.to)) return;
      if ((b.activeMs ?? 0) > 0) {
        maxMs = Math.max(maxMs, b.activeMs!);
        pts.push({ x: padL + i * pitch + bw / 2, y: 0, ms: b.activeMs! });
      }
    });
    const yOf = (ms: number) => padT + plotH - (ms / Math.max(maxMs, 1)) * plotH;
    return pts.map((p) => ({ ...p, y: yOf(p.ms), maxMs }));
  }, [adaptive, durationOn, bars, bw, padL, padT, pitch, plotH, visibleRange.from, visibleRange.to]);
  const durMax = durPoints[0]?.maxMs ?? 0;
  const durTicks = [0, 0.5, 1].map((f) => ({
    y: padT + plotH - f * plotH,
    label: fmtDuration(Math.round(durMax * f), t),
  }));

  const pick = (i: number) => {
    const r = bars[i];
    if (!r) return;
    setSelected(i);
    onPick(r);
  };

  // v4.542 刀4 总览模式（dsh overview 同款）：整会话压到面板宽成连续堆叠带，
  // 压缩事件标竖线；悬停读数、点击定位主图该步。视图局部态（不写偏好——
  // dsh 同口径：切换项仅对当前视图生效）。
  const [overviewCat, setOverviewCat] = useState<keyof ContextCategory | null>(null);
  const compactMarks = useMemo(() => {
    // 每个压缩事件 → 其后第一根柱的下标（竖线画在该柱左缘）。
    const marks: number[] = [];
    const sorted = [...compactSeqs].sort((a, b) => a - b);
    for (const eseq of sorted) {
      const idx = bars.findIndex((b) => b.seq > eseq);
      if (idx > 0) marks.push(idx);
    }
    return marks;
  }, [compactSeqs, bars]);
  const overviewMax = useMemo(
    () => Math.max(1, ...bars.map((bar, index) => (
      mode === "total" ? catTotal(bar.category) : deltaBars[index]?.up ?? 0
    ))),
    [bars, deltaBars, mode],
  );
  const overviewPick = (clientX: number, el: SVGSVGElement) => {
    const rect = el.getBoundingClientRect();
    const f = Math.min(0.9999, Math.max(0, (clientX - rect.left) / rect.width));
    pick(Math.floor(f * bars.length));
  };
  const toggleOverview = () => {
    manualOverview.current = true;
    const next = !overview;
    setOverview(next);
    if (next) {
      setAdaptive(false);
      setDnaOn(false);
      setDnaHoverCat(null);
    }
  };
  const selectedIndex = selected ?? (
    selectedSeq != null ? bars.findIndex((bar) => bar.seq === selectedSeq) : -1
  );
  const overviewCursor = hovered ?? (selectedIndex >= 0 ? selectedIndex : Math.max(0, bars.length - 1));
  const onOverviewKeyDown = (event: ReactKeyboardEvent<SVGSVGElement>) => {
    if (bars.length === 0) return;
    if (event.key === "ArrowLeft" || event.key === "ArrowRight" || event.key === "Home" || event.key === "End") {
      event.preventDefault();
      const next = event.key === "Home"
        ? 0
        : event.key === "End"
          ? bars.length - 1
          : Math.max(0, Math.min(bars.length - 1, overviewCursor + (event.key === "ArrowRight" ? 1 : -1)));
      setHover(next);
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      const index = Math.max(0, Math.min(bars.length - 1, overviewCursor));
      pick(index);
    }
  };
  const signedTokens = (value: number) => `${value >= 0 ? "+" : "-"}${fmtTokens(Math.abs(value))}`;
  const hoveredDeltaBar = hovered != null ? deltaBars[hovered] : null;
  const hoveredDnaBands = hovered != null && dnaOn ? dnaData[hovered]?.bands : undefined;
  const dnaLabelOf = (band: ContextDnaBand) => {
    if (band.label) return band.label;
    return t(CATS.find((cat) => cat.key === band.cat)?.labelKey ?? "contextview.catTool");
  };

  return (
    <div
      ref={cardRef}
      data-testid="trend-card"
      data-flash={flash ? "true" : "false"}
      className={`ctx-card p-3 ${flash ? "outline outline-1 -outline-offset-1 outline-accent/50" : ""}`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <LineChart size={12} />
          </span>
          <span className="text-[12.5px] font-semibold text-fg">{t("contextview.trendTitle")}</span>
        </div>
        <div className="flex items-center gap-1 text-[11px]">
          {(["step", "turn"] as const).map((g) => (
            <button
              key={g}
              data-testid={`trend-granularity-${g}`}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${granularity === g ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => pickGranularity(g)}
            >{g === "step" ? t("contextview.granStep") : t("contextview.granTurn")}</button>
          ))}
          <span className="mx-0.5 h-3 w-px bg-border-soft" />
          {(["total", "delta"] as const).map((m) => (
            <button
              key={m}
              title={m === "delta" ? t("contextview.deltaTip") : undefined}
              data-testid={`trend-mode-${m}`}
              className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${mode === m ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => pickMode(m)}
            >{m === "total" ? t("contextview.modeTotal") : t("contextview.modeDelta")}</button>
          ))}
          <button
            title={t("contextview.dnaTrendTip")}
            aria-pressed={dnaOn}
            data-testid="trend-dna"
            className={`rounded border-0 px-1.5 py-0.5 ${dnaOn ? "bg-accent/15 text-accent" : "text-fg-dim"} cursor-pointer hover:text-fg`}
            onClick={() => {
              const next = !dnaOn;
              setDnaOn(next);
              setDnaHoverCat(null);
              if (next) setOverview(false);
            }}
          >DNA</button>
          <button
            title={t("contextview.adaptiveHint")}
            aria-pressed={adaptive}
            disabled={overview}
            data-testid="trend-adaptive"
            className={`rounded border-0 px-1.5 py-0.5 ${adaptive && !overview ? "bg-accent/15 text-accent" : "text-fg-dim"} ${overview ? "cursor-not-allowed opacity-45" : "cursor-pointer hover:text-fg"}`}
            onClick={() => setAdaptive((value) => !value)}
          >{t("contextview.adaptive")}</button>
          <button
            title={t("contextview.durationTip")}
            aria-pressed={durationOn}
            data-testid="trend-duration"
            className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${durationOn ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            onClick={toggleDuration}
          >{t("contextview.duration")}</button>
          <span className="mx-0.5 h-3 w-px bg-border-soft" />
          <button
            title={t("contextview.overviewTip")}
            aria-pressed={overview}
            data-testid="trend-overview"
            className={`px-1.5 py-0.5 rounded border-0 cursor-pointer ${overview ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            onClick={toggleOverview}
          >{t("contextview.overview")}</button>
        </div>
      </div>
      <div className="mt-1 text-[10px] text-fg-faint">
        {t("contextview.trendLegend")}
        {mode === "delta" && <span className="ml-2"><span className="text-(color:--color-success)">▲</span> {t("contextview.netIncrease")} <span className="ml-1 text-(color:--color-destructive)">▼</span> {t("contextview.netDecrease")}</span>}
        {adaptive && !overview && <span className="ml-2">{t("contextview.adaptiveOn")}</span>}
        {dnaOn && <span className="ml-2">{dnaLoading ? t("contextview.dnaLoading") : dnaFailed ? t("contextview.dnaUnavailable") : t("contextview.dnaOn")}</span>}
      </div>
      {dnaOn && mode === "delta" && bars.length <= 1 && (
        <div className="mt-1 rounded-md bg-bg-soft px-2 py-1 text-[9.5px] text-fg-faint" data-testid="ctx-dna-first-note">
          {t("contextview.dnaDeltaFirst")}
        </div>
      )}
      {overview && (
        <svg
          data-testid="trend-overview-band"
          viewBox="0 0 100 34"
          preserveAspectRatio="none"
          role="group"
          tabIndex={0}
          aria-label={t("contextview.overviewTip")}
          className="mt-1.5 h-9 w-full cursor-crosshair rounded-md bg-bg-soft"
          onClick={(e) => overviewPick(e.clientX, e.currentTarget)}
          onKeyDown={onOverviewKeyDown}
          onFocus={() => setHover(overviewCursor)}
          onBlur={() => setHover(null)}
          onMouseLeave={() => {
            setHover(null);
            setOverviewCat(null);
          }}
        >
          {/* 每请求等宽分段 × 七分类堆叠（连续带，无需横向滚动） */}
          {bars.map((b, i) => {
            const x = (i / bars.length) * 100;
            const w = 100 / bars.length;
            const delta = deltaBars[i];
            const total = mode === "delta" ? delta.up : catTotal(b.category);
            if (total <= 0) return null;
            let acc = 0;
            return (
              <g key={b.seq} data-overview-step={b.seq}>
                {CATS.map((c) => {
                  const v = mode === "delta"
                    ? Math.max(0, delta.parts.find((part) => part.cat === c.key)?.tokens ?? 0)
                    : b.category[c.key];
                  if (v <= 0) return null;
                  const h = (v / overviewMax) * 30;
                  const y = 32 - acc - h;
                  acc += h;
                  return (
                    <rect
                      key={c.key}
                      x={x}
                      y={y}
                      width={Math.max(w, 0.12)}
                      height={Math.max(h, 0.12)}
                      fill={CAT_COLORS[c.key]}
                      opacity={overviewCat != null && overviewCat !== c.key ? 0.2 : 0.85}
                      data-overview-cat={c.key}
                      onMouseEnter={() => {
                        setHover(i);
                        setOverviewCat(c.key);
                      }}
                      onClick={(event) => {
                        event.stopPropagation();
                        pick(i);
                        onPickCategory?.(c.key, b);
                      }}
                    >
                      <title>{`${t("contextview.turnStep", { turn: b.turn, step: b.step })} · ${mode === "delta" ? signedTokens(total) : fmtTokens(total)}`}</title>
                    </rect>
                  );
                })}
              </g>
            );
          })}
          {selectedIndex >= 0 && bars.length > 0 && (
            <line
              data-testid="trend-overview-selected"
              x1={((selectedIndex + 0.5) / bars.length) * 100}
              y1={0}
              x2={((selectedIndex + 0.5) / bars.length) * 100}
              y2={34}
              stroke="var(--accent)"
              strokeWidth={0.45}
              pointerEvents="none"
            />
          )}
          {/* 压缩事件竖线（位置=其后第一根柱左缘） */}
          {compactMarks.map((mi) => (
            <line
              key={`cm-${mi}`}
              x1={(mi / bars.length) * 100}
              y1={0}
              x2={(mi / bars.length) * 100}
              y2={34}
              stroke="var(--md-sys-color-warning, #f59e0b)"
              strokeWidth={0.4}
            />
          ))}
          {[0.25, 0.5, 0.75].map((at, index) => (
            <g key={`quartile-${at}`} data-testid="trend-overview-quartile">
              <line x1={at * 100} y1={0} x2={at * 100} y2={34} stroke="var(--fg-faint)" strokeWidth={0.25} strokeDasharray="1 1" opacity={0.65} />
              <text x={at * 100 + 0.8} y={4.5} fontSize={2.8} fill="var(--fg-faint)">Q{index + 1}</text>
            </g>
          ))}
        </svg>
      )}
      {hovered !== null && bars[hovered] && (
        <div
          data-testid="trend-hover-detail"
          className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 rounded-md px-2 py-1 text-[10.5px]"
          style={{ background: "var(--md-sys-color-surface-container-high)" }}
        >
          <span className="font-mono text-fg-dim">{t("contextview.turnStep", { turn: bars[hovered].turn, step: bars[hovered].step })}</span>
          {hoveredDnaBands
            ? hoveredDnaBands.filter((band) => band.tokens > 0).slice(0, 8).map((band) => (
                <span key={band.key} className="inline-flex min-w-0 items-center gap-1 text-fg-faint">
                  <span className="h-1.5 w-1.5 shrink-0 rounded-sm" style={{ background: CAT_COLORS[band.cat] }} />
                  <span className="max-w-28 truncate">{dnaLabelOf(band)}</span> {fmtTokens(band.tokens)}
                </span>
              ))
            : CATS.map((c) => {
                const value = mode === "delta"
                  ? hoveredDeltaBar?.parts.find((part) => part.cat === c.key)?.tokens ?? 0
                  : bars[hovered].category[c.key];
                if (value === 0 || (mode === "total" && value < 0)) return null;
                return (
                  <span key={c.key} className="inline-flex items-center gap-1 text-fg-faint">
                    <span className="h-1.5 w-1.5 shrink-0 rounded-sm" style={{ background: CAT_COLORS[c.key] }} />
                    {t(c.labelKey)} {mode === "delta" ? signedTokens(value) : fmtTokens(value)}
                  </span>
                );
              })}
          <span className="ml-auto font-mono text-fg-dim">
            {t("contextview.total", {
              tokens: hoveredDnaBands
                ? mode === "delta" ? signedTokens(dnaData[hovered]?.delta?.net ?? 0) : fmtTokens(dnaTotal(hoveredDnaBands))
                : mode === "delta" ? signedTokens(hoveredDeltaBar?.net ?? 0) : fmtTokens(catTotal(bars[hovered].category)),
            })}
          </span>
          {/* 刀9：有真实 usage 时并列服务商报告的 Actual（估算 vs 实际诚实并列，
              dsh 悬停同款）；Estimated 关闭的请求无 usage，不造数。 */}
          {(bars[hovered].promptTokens ?? 0) > 0 && (
            <span className="ml-auto font-mono text-fg-dim" data-testid="trend-hover-actual">
              {t("contextview.trendActual", {
                prompt: fmtTokens(bars[hovered].promptTokens ?? 0),
                out: fmtTokens(bars[hovered].outputTokens ?? 0),
                cache: fmtTokens(bars[hovered].cacheHitTokens ?? 0),
              })}
            </span>
          )}
        </div>
      )}
      <div
        ref={chartScrollRef}
        className="overflow-x-auto"
        data-testid="trend-scroll"
        onScroll={onChartScroll}
      >
        <svg
          viewBox={`0 0 ${W} ${H}`}
          className="mt-1 h-auto min-w-full"
          style={{ width: W }}
          role="group"
          aria-label={t("contextview.trendTitle")}
        >
          {yTicks.map((t, i) => (
            <g key={i}>
              <line x1={padL} y1={t.y} x2={W - padR} y2={t.y} stroke="var(--border-soft)" strokeWidth={0.5} />
              <text x={padL - 4} y={t.y + 3} fontSize={9} fill="var(--fg-faint)" textAnchor="end">{t.label}</text>
            </g>
          ))}
          {mode === "delta" && (
            <line
              data-testid="trend-delta-baseline"
              x1={padL}
              y1={deltaBaseline}
              x2={W - padR}
              y2={deltaBaseline}
              stroke="var(--fg-faint)"
              strokeWidth={0.8}
            />
          )}
          {durationOn && durMax > 0 && durTicks.map((tk, i) => (
            <text key={`dt${i}`} x={W - padR + 4} y={tk.y + 3} fontSize={8.5} fill="var(--fg-faint)">{tk.label}</text>
          ))}
          {durationOn && durPoints.length > 1 && (
            <polyline
              points={durPoints.map((p) => `${p.x},${p.y}`).join(" ")}
              fill="none"
              stroke="var(--fg-dim, var(--md-sys-color-outline, #9e9e9e))"
              strokeWidth={1.2}
              data-testid="trend-duration-curve"
            />
          )}
          {bars.map((b, i) => {
            const x = padL + i * pitch;
            const total = catTotal(b.category);
            const selectedNow = selectedIndex === i;
            const dna = dnaOn ? dnaData[i] : null;
            const onBarKeyDown = (event: ReactKeyboardEvent<SVGGElement>) => {
              if (event.key !== "Enter" && event.key !== " ") return;
              event.preventDefault();
              pick(i);
            };
            if (dna?.bands) {
              const dnaBands = dna.bands;
              if (mode === "delta") {
                const delta = dna.delta;
                let upAcc = 0;
                let downAcc = 0;
                const renderPart = (part: DnaDeltaPart, positive: boolean) => {
                  const h = (part.tokens / maxY) * plotH;
                  const y = positive
                    ? deltaBaseline - upAcc - h
                    : deltaBaseline + downAcc;
                  if (positive) upAcc += h;
                  else downAcc += h;
                  const label = dnaLabelOf(part.band);
                  return (
                    <rect
                      key={`${part.band.key}:${positive ? "up" : "down"}`}
                      data-dna-key={part.band.key}
                      x={x}
                      y={y}
                      width={bw}
                      height={Math.max(h, 1)}
                      fill={CAT_COLORS[part.band.cat]}
                      opacity={dnaHoverCat != null && dnaHoverCat !== part.band.cat ? 0.2 : 1}
                      onMouseEnter={() => {
                        setHover(i);
                        setDnaHoverCat(part.band.cat);
                      }}
                      onClick={(event) => {
                        event.stopPropagation();
                        pick(i);
                        onPickCategory?.(part.band.cat, b);
                      }}
                    >
                      <title>{`${label} ${positive ? "+" : "-"}${fmtTokens(part.tokens)} · ${t("contextview.total", { tokens: signedTokens(delta?.net ?? 0) })}`}</title>
                    </rect>
                  );
                };
                return (
                  <g
                    key={b.seq}
                    data-testid={`trend-bar-${b.seq}`}
                    data-dna="true"
                    onClick={() => pick(i)}
                    onKeyDown={onBarKeyDown}
                    onMouseEnter={() => setHover(i)}
                    onMouseLeave={() => {
                      setHover(null);
                      setDnaHoverCat(null);
                    }}
                    onFocus={() => setHover(i)}
                    onBlur={() => {
                      setHover(null);
                      setDnaHoverCat(null);
                    }}
                    className="cursor-pointer"
                    role="button"
                    tabIndex={0}
                    aria-label={`${t("contextview.turnStep", { turn: b.turn, step: b.step })} · ${t("contextview.total", { tokens: signedTokens(delta?.net ?? 0) })}`}
                    aria-pressed={selectedNow}
                    opacity={selectedNow ? 1 : 0.9}
                  >
                    {delta?.up.map((part) => renderPart(part, true))}
                    {delta?.down.map((part) => renderPart(part, false))}
                  </g>
                );
              }
              let acc = 0;
              return (
                <g
                  key={b.seq}
                  data-testid={`trend-bar-${b.seq}`}
                  data-dna="true"
                  onClick={() => pick(i)}
                  onKeyDown={onBarKeyDown}
                  onMouseEnter={() => setHover(i)}
                  onMouseLeave={() => {
                    setHover(null);
                    setDnaHoverCat(null);
                  }}
                  onFocus={() => setHover(i)}
                  onBlur={() => {
                    setHover(null);
                    setDnaHoverCat(null);
                  }}
                  className="cursor-pointer"
                  role="button"
                  tabIndex={0}
                  aria-label={`${t("contextview.turnStep", { turn: b.turn, step: b.step })} · ${t("contextview.total", { tokens: fmtTokens(dnaTotal(dnaBands)) })}`}
                  aria-pressed={selectedNow}
                  opacity={selectedNow ? 1 : 0.9}
                >
                  {dnaBands.map((band) => {
                    if (band.tokens <= 0) return null;
                    const h = (band.tokens / maxY) * plotH;
                    const y = padT + plotH - acc - h;
                    acc += h;
                    const label = dnaLabelOf(band);
                    return (
                      <rect
                        key={band.key}
                        data-dna-key={band.key}
                        x={x}
                        y={y}
                        width={bw}
                        height={Math.max(h, 1)}
                        fill={CAT_COLORS[band.cat]}
                        opacity={dnaHoverCat != null && dnaHoverCat !== band.cat ? 0.2 : 1}
                        onMouseEnter={() => {
                          setHover(i);
                          setDnaHoverCat(band.cat);
                        }}
                        onClick={(event) => {
                          event.stopPropagation();
                          pick(i);
                          onPickCategory?.(band.cat, b);
                        }}
                      >
                        <title>{`${label} ${fmtTokens(band.tokens)} · ${t("contextview.total", { tokens: fmtTokens(dnaTotal(dnaBands)) })}`}</title>
                      </rect>
                    );
                  })}
                </g>
              );
            }
            if (mode === "delta") {
              const delta = deltaBars[i];
              let upAcc = 0;
              let downAcc = 0;
              return (
                <g
                  key={b.seq}
                  data-testid={`trend-bar-${b.seq}`}
                  onClick={() => pick(i)}
                  onKeyDown={onBarKeyDown}
                  onMouseEnter={() => setHover(i)}
                  onFocus={() => setHover(i)}
                  onMouseLeave={() => setHover(null)}
                  onBlur={() => setHover(null)}
                  className="cursor-pointer"
                  role="button"
                  tabIndex={0}
                  aria-label={`${t("contextview.turnStep", { turn: b.turn, step: b.step })} · ${t("contextview.total", { tokens: signedTokens(delta.net) })}`}
                  aria-pressed={selectedNow}
                  opacity={selectedNow ? 1 : 0.85}
                >
                  {CATS.map((c) => {
                    const value = delta.parts.find((part) => part.cat === c.key)?.tokens ?? 0;
                    if (value === 0) return null;
                    const h = (Math.abs(value) / maxY) * plotH;
                    const positive = value > 0;
                    const y = positive
                      ? deltaBaseline - upAcc - h
                      : deltaBaseline + downAcc;
                    if (positive) upAcc += h;
                    else downAcc += h;
                    return (
                      <rect
                        key={c.key}
                        x={x}
                        y={y}
                        width={bw}
                        height={Math.max(h, 1)}
                        rx={1.5}
                        fill={CAT_COLORS[c.key]}
                        stroke={hovered === i ? "var(--color-primary, var(--md-sys-color-primary))" : undefined}
                        strokeWidth={hovered === i ? 1.2 : undefined}
                      >
                        <title>{t("contextview.deltaTitle", {
                          turn: b.turn,
                          step: b.step,
                          delta: `${t(c.labelKey)} ${signedTokens(value)} · ${t("contextview.total", { tokens: signedTokens(delta.net) })}`,
                        })}</title>
                      </rect>
                    );
                  })}
                </g>
              );
            }
            let acc = 0;
            return (
              <g
                key={b.seq}
                data-testid={`trend-bar-${b.seq}`}
                onClick={() => pick(i)}
                onKeyDown={onBarKeyDown}
                onMouseEnter={() => setHover(i)}
                onFocus={() => setHover(i)}
                onMouseLeave={() => setHover(null)}
                onBlur={() => setHover(null)}
                className="cursor-pointer"
                role="button"
                tabIndex={0}
                aria-label={`${t("contextview.turnStep", { turn: b.turn, step: b.step })} · ${t("contextview.total", { tokens: fmtTokens(total) })}`}
                aria-pressed={selectedNow}
                opacity={selectedNow ? 1 : 0.9}
                style={hovered === i ? { filter: "brightness(1.2)" } : undefined} /* hover 提亮=悬停可感知（v4.412，对齐 delta 柱描边语义） */
              >
                {CATS.map((c) => {
                  const v = b.category[c.key];
                  if (v <= 0) return null;
                  const h = (v / maxY) * plotH;
                  const y = padT + plotH - acc - h;
                  acc += h;
                  return (
                    <rect key={c.key} x={x} y={y} width={bw} height={Math.max(h, 1)} fill={CAT_COLORS[c.key]}>
                      <title>{t("contextview.catBarTitle", { turn: b.turn, step: b.step, cat: t(c.labelKey), tokens: fmtTokens(v), total: fmtTokens(total) })}</title>
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
      {detail}
    </div>
  );
}

// ─── 选中步骤详情（v4.69 起内联进趋势卡：无独立外框，顶部分隔线区隔） ──
// 2.5d：brief 行带 Go 折叠给出的跳转锚点（briefUserSeq/briefRespSeq）时整行
// 可点 → 跳到上下文浏览器对应节点（组自动展开+滚动+高亮）；无锚点保持纯文本。
function StepDetail({
  record,
  window,
  onJump,
  stepsInTurn,
  turnMode,
}: {
  record: ContextRequestRecord | null;
  window: number;
  onJump?: (seq: number) => void;
  // 刀12（1:1 上游 detail.step/turn）：本轮步数合计 + 轮粒度标题形态——
  // 步模式「第{turn}轮·第{step}步 (共{total}步)」，轮模式「第{turn}轮 · 共{total}步」。
  stepsInTurn?: number;
  turnMode?: boolean;
}) {
  const t = useT();
  // 容器已默认选中最新请求，null 只存在于快照过渡瞬间 → 不渲染，避免空态闪烁。
  if (!record) return null;
  const total = catTotal(record.category);
  const windowPct = window > 0 ? Math.round((total / window) * 100) : null;
  const cacheHit = record.cacheHitTokens ?? 0;
  const cacheMiss = record.cacheMissTokens ?? 0;
  const cachePct = cacheHit + cacheMiss > 0 ? (cacheHit * 100) / (cacheHit + cacheMiss) : null;
  const title = turnMode
    ? t("contextview.turnTitle", { turn: record.turn, total: stepsInTurn ?? 0 })
    : t("contextview.turnStepTotal", { turn: record.turn, step: record.step, total: stepsInTurn ?? 0 });
  return (
    <div className="mt-2 border-t border-border-soft/70 pt-2">
      <div className="flex items-center justify-between">
        <div className="text-[12.5px] font-semibold text-fg">{title}</div>
        <div className="text-[10px] text-fg-faint tabular-nums font-mono">{new Date(record.ts * 1000).toLocaleTimeString()}</div>
      </div>
      <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11.5px] text-fg-dim tabular-nums font-mono">
        {record.estimated
          ? <span className="text-warning/90">{t("contextview.estimated")}</span>
          : record.promptTokens ? <span>{t("contextview.actualPrompt", { tokens: fmtTokens(record.promptTokens) })}</span> : null}
        {record.outputTokens ? <span>{t("contextview.outputTokens", { n: record.outputTokens })}</span> : null}
        {cachePct != null ? <span>{t("contextview.cachePct", { pct: cachePct.toFixed(2) })}</span> : null}
        <span>{t("contextview.totalApprox", { tokens: fmtTokens(total) })}</span>
        {windowPct != null && <span>{t("contextview.windowPct", { pct: windowPct })}</span>}
      </div>
      {/* v4.79 对比上一步（dsh 同名能力）：逐类 signed delta 徽标；跨压缩标
          「近似」（差分基线被结构性改写）；首个请求基线=空。语义色与趋势
          delta 模式同款（#22c55e 净增 / #ef4444 净减，hex-exempt 可视化调色板）。 */}
      {record.delta && (
        <div data-testid="ctx-delta-strip" className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[10.5px]">
          <span className="shrink-0 text-fg-faint">{t("contextview.vsPrev")}</span>
          {record.delta.first ? (
            <span className="text-fg-dim">{t("contextview.deltaFirst")}</span>
          ) : record.delta.items === 0 && record.delta.tokens === 0 ? (
            <span className="text-fg-dim">{t("contextview.deltaNoChange")}</span>
          ) : (
            <>
              <span className="font-mono font-medium tabular-nums" style={{ color: record.delta.tokens >= 0 ? "#22c55e" : "#ef4444" }}> {/* hex-exempt 增量语义色（与趋势 delta 模式同款：绿=净增/红=净减） */}
                {record.delta.tokens >= 0 ? "+" : "-"}{fmtTokens(Math.abs(record.delta.tokens))}
              </span>
              {record.delta.approx && (
                <span className="text-warning" title={t("contextview.deltaApproxTip")}>≈ {t("contextview.deltaApprox")}</span>
              )}
              {(record.delta.byCat ?? []).map((c) => (
                <span key={c.cat} className="inline-flex items-center gap-1 text-fg-dim">
                  <span className="h-1.5 w-1.5 shrink-0 rounded-sm" style={{ background: CAT_COLORS[c.cat] }} />
                  {t(CAT_BROWSE_SHORT[c.cat] ?? (c.cat as DictKey))}
                  <span className="font-mono tabular-nums">
                    {c.items ? `${c.items > 0 ? "+" : "-"}${Math.abs(c.items)}` : ""}
                    {c.items && c.tokens ? "·" : ""}
                    {c.tokens ? `${c.tokens > 0 ? "+" : "-"}${fmtTokens(Math.abs(c.tokens))}` : ""}
                  </span>
                </span>
              ))}
            </>
          )}
        </div>
      )}
      {record.briefUser && (
        <div className="mt-1.5 flex gap-1.5 text-[11.5px]">
          <span className="shrink-0 text-fg-faint">{t("contextview.inputLabel")}</span>
          {record.briefUserSeq ? (
            <button
              data-testid="ctx-brief-user"
              title={t("contextview.briefJumpTip")}
              className="min-w-0 cursor-pointer truncate border-0 bg-transparent p-0 text-left font-mono text-fg-dim hover:text-accent hover:underline"
              onClick={() => onJump?.(record.briefUserSeq!)}
            >{record.briefUser}</button>
          ) : (
            <span className="truncate font-mono text-fg-dim">{record.briefUser}</span>
          )}
        </div>
      )}
      {(record.briefIn?.length ?? 0) > 0 && (
        <div className="flex gap-1.5 text-[11.5px]" data-testid="ctx-step-in">
          <span className="shrink-0 text-fg-faint">{t("contextview.inLabel")}</span>
          <span className="flex min-w-0 flex-wrap gap-1">
            {record.briefIn!.map((item, index) => (
              <span key={`${item}-${index}`} className="max-w-full truncate rounded bg-bg-soft px-1.5 py-px font-mono text-[10px] text-fg-dim" title={item}>
                {item}
              </span>
            ))}
          </span>
        </div>
      )}
      {record.briefResp && (
        <div className="flex gap-1.5 text-[11.5px]">
          <span className="shrink-0 text-fg-faint">{t("contextview.respLabel")}</span>
          {record.briefRespSeq ? (
            <button
              data-testid="ctx-brief-resp"
              title={t("contextview.briefJumpTip")}
              className="min-w-0 cursor-pointer truncate border-0 bg-transparent p-0 text-left font-mono text-fg-dim hover:text-accent hover:underline"
              onClick={() => onJump?.(record.briefRespSeq!)}
            >{record.briefResp}</button>
          ) : (
            <span className="truncate font-mono text-fg-dim">{record.briefResp}</span>
          )}
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
// v4.69：事件筛选对齐 dsh 多选 chips（去「全部」单选项）——默认五类全亮；
// 全亮时点某类=仅看该类；多选态点某类=排除它；单选态再点=恢复全选。
const EVENT_KINDS = ["inject", "compact", "prune", "switch", "mode"] as const;
type EventKind = (typeof EVENT_KINDS)[number];
// 事件流默认窗口（新事件在上；超出给「显示全部」——浏览器树分页同款纪律）。
const EVENTS_PAGE = 30;
const EVENT_LABEL: Record<EventKind, DictKey> = { inject: "contextview.evInject", compact: "contextview.evCompact", prune: "contextview.evPrune", switch: "contextview.evSwitch", mode: "contextview.evMode" };

function EventsList({ events, focusKind, onFocusEvent }: {
  events: ContextEvent[];
  focusKind?: { kind: EventKind; tick: number } | null;
  onFocusEvent?: (event: ContextEvent) => void;
}) {
  const t = useT();
  const [kinds, setKinds] = useState<readonly EventKind[]>(EVENT_KINDS);
  const handledFocusKind = useRef(0);
  // v4.524 截断诚实化：默认只列最近 EVENTS_PAGE 条（新事件在上），超出时
  // 如实提示总数并给「显示全部」展开（对齐浏览器树分页与记忆中枢侧栏同款
  // 纪律——此前静默砍掉更早的事件）。
  const [showAll, setShowAll] = useState(false);
  useEffect(() => {
    if (!focusKind || focusKind.tick === handledFocusKind.current) return;
    handledFocusKind.current = focusKind.tick;
    setKinds([focusKind.kind]);
    requestAnimationFrame(() => {
      document.querySelector('[data-testid="ctx-events"]')?.scrollIntoView({ block: "center" });
    });
  }, [focusKind]);
  const allOn = kinds.length === EVENT_KINDS.length;
  const shown = allOn ? events : events.filter((e) => kinds.includes(e.kind));
  // 各类计数按全量 events 口径（与文件活动 chips 同款——计数不随选中集合抖动）。
  const countOf = (k: EventKind) => events.reduce((s, e) => (e.kind === k ? s + 1 : s), 0);
  const hidden = showAll ? 0 : Math.max(0, shown.length - EVENTS_PAGE);
  const listed = hidden === 0 ? [...shown].reverse() : shown.slice(-EVENTS_PAGE).reverse();
  const toggle = (k: EventKind) => {
    setKinds((cur) => {
      if (allOn) return [k]; // 全选态点一 → 只留该分类（dsh 排他单选）
      if (!cur.includes(k)) return [...cur, k]; // 点加
      return cur.length === 1 ? [...EVENT_KINDS] : cur.filter((x) => x !== k); // 点掉；最后一项点掉 = 恢复全选
    });
  };
  return (
    <div className="ctx-card p-3 ctx-events-card" data-testid="ctx-events">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <List size={12} />
          </span>
          <span className="text-[12.5px] font-semibold text-fg">{t("contextview.eventsTitle")}</span>
        </div>
        <div className="ctx-kind-filters flex items-center gap-1 text-[11px]">
          {EVENT_KINDS.map((k) => {
            const on = kinds.includes(k);
            return (
              <button
                key={k}
                aria-pressed={on}
                data-kind={k}
                data-testid={`ctx-kind-filter-${k}`}
                className="ctx-kind-filter"
                onClick={() => toggle(k)}
              >{t(EVENT_LABEL[k])}<span className="ml-1 tabular-nums">{countOf(k)}</span></button>
            );
          })}
        </div>
      </div>
      <div className="ctx-events-list mt-1.5 flex max-h-44 flex-col gap-1 overflow-y-auto pr-0.5">
        {shown.length === 0 && <div className="text-[11px] text-fg-faint py-2">{t("contextview.noEvents")}</div>}
        {listed.map((e) => (
          <button
            key={`${e.kind}-${e.seq}`}
            type="button"
            className="ctx-event-row w-full cursor-pointer border-0 bg-transparent text-left"
            data-kind={e.kind}
            data-testid={`ctx-event-row-${e.kind}-${e.seq}`}
            title={t("contextview.eventJumpTip")}
            onClick={() => onFocusEvent?.(e)}
          >
            <span className="ctx-event-sign" aria-hidden>{e.delta != null && e.delta < 0 ? "−" : "+"}</span>
            <span className="ctx-event-kind">
              {t(EVENT_LABEL[e.kind])}
            </span>
            {/* 刀13：注入源形态分类 chip（1:1 上游 form 细分；无标签旧格式不渲染） */}
            {e.kind === "inject" && e.form && (
              <span className="shrink-0 rounded bg-bg-soft px-1 text-[10px] text-fg-dim" title={t("contextview.injectSrcTip")}>
                {t(`contextview.injectForm.${e.form}` as DictKey)}
              </span>
            )}
            <span className="ctx-event-label">
              {e.kind === "switch" && (e.from || e.to)
                ? `${e.from || "—"} → ${e.to || "—"}`
                : e.source || "—"}
            </span>
            <span className="ctx-event-meta">T{e.turn} S{e.step}</span>
            <span className="ctx-event-delta" data-sign={(e.delta ?? 0) >= 0 ? "up" : "down"}>
              {e.delta != null && e.delta !== 0
                ? `${e.delta > 0 ? "+" : "-"}${fmtTokens(Math.abs(e.delta))}`
                : ""}
            </span>
            <span className="ctx-event-time">{new Date(e.ts * 1000).toLocaleTimeString()}</span>
          </button>
        ))}
        {hidden > 0 && (
          <div className="flex items-center gap-1.5 border-t border-border-soft/50 pt-1 text-[10px] text-fg-faint" data-testid="events-truncated">
            <span>{t("contextview.eventsRecent", { n: EVENTS_PAGE, total: shown.length })}</span>
            <button
              type="button"
              className="cursor-pointer border-0 bg-transparent p-0 text-accent hover:underline"
              onClick={() => setShowAll(true)}
            >{t("contextview.eventsShowAll", { n: shown.length })}</button>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── 容器：拉取 + 布局 ──────────────────────────────────────
export function ContextView({
  running,
  sessionPath,
  sessionName: sessionNameProp,
  model,
  fetchTimeline,
  fetchNodeDetail,
  fetchSurface,
  fetchDna,
  spaceOverride,
  showAgentNetwork = false,
  onViewSubagentContext,
}: {
  running: boolean;
  sessionPath?: string;
  /** 当前会话标题（App 由 sidebarSessions+sessionTitle 解析）；缺省回落文件名 */
  sessionName?: string;
  /** 顶栏当前模型 label（state.meta.label） */
  model?: string;
  /** 2.5e 后半：自定义数据源（如子代理会话的 GaeaSubagentContextView）；缺省当前会话。 */
  fetchTimeline?: () => Promise<ContextTimeline>;
  /** 节点详情自定义源（v4.412 原罪复用：SinContextNodeDetail 按故事 id 回读）；缺省会话日志。 */
  fetchNodeDetail?: (seq: number) => Promise<ContextNodeDetailView>;
  /** 历史步骤 surface 自定义源；配自定义 timeline 时必须提供，否则诚实禁用。 */
  fetchSurface?: (seq: number) => Promise<ContextSurfaceSnapshot>;
  /** 趋势 DNA 自定义批量源；配自定义 timeline 时缺省则诚实不可用。 */
  fetchDna?: (seqs: number[]) => Promise<ContextDnaSample[]>;
  /** 会话归属空间覆盖（v4.412 原罪复用：无 sessionPath 可判，显式传「乐园」）；缺省按路径推断。 */
  spaceOverride?: string;
  /** Agent 网络径向图开关（v4.412 原罪复用关闭：网络属办公内核会话，不串台）。 */
  showAgentNetwork?: boolean;
  /** Agent 网络子代理节点「查看上下文」回调（sa_ 节点；提供才渲染入口）。 */
  onViewSubagentContext?: (ref: string) => void;
}) {
  const t = useT();
  const [timeline, setTimeline] = useState<ContextTimeline>(EMPTY);
  const [loadedSessionPath, setLoadedSessionPath] = useState<string | undefined>(undefined);
  const currentSessionPath = useRef(sessionPath);
  currentSessionPath.current = sessionPath;
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const restoredViewport = useRef<string | undefined>(undefined);
  const [picked, setPicked] = useState<ContextRequestRecord | null>(null);
  const [hoveredRequest, setHoveredRequest] = useState<ContextRequestRecord | null>(null);
  const [browserMode, setBrowserMode] = useState<BrowserSurfaceMode>("live");
  const [surfaceRead, setSurfaceRead] = useState<SurfaceReadState>({ status: "live" });
  const surfaceCache = useRef<Map<number, ContextSurfaceSnapshot>>(new Map());
  // 刀12：趋势粒度态上提（驱动 StepDetail turn 模式标题；toggle 交互仍在卡内，
  // 偏好写回仍在卡内 pickGranularity 单点）。
  const [granularity, setGranularity] = useState<CtxTrendGranularity>(() => loadContextPrefs().trendGranularity);
  const [error, setError] = useState<string | null>(null);
  // 2.5d 趋势→浏览器联动：StepDetail brief 行点击的目标节点。tick 让同一
  // 节点重复点击也能重新触发滚动/高亮。
  const [focusNode, setFocusNode] = useState<{ seq: number; tick: number } | null>(null);
  const focusTick = useRef(0);
  const jumpToNode = useCallback((seq: number) => {
    focusTick.current += 1;
    setFocusNode({ seq, tick: focusTick.current });
  }, []);
  const [browserCatFocus, setBrowserCatFocus] = useState<BrowserCatFocus | null>(null);
  const browserCatTick = useRef(0);
  const focusBrowserCat = useCallback((cat: BrowserCatFocus["cat"], query?: string) => {
    browserCatTick.current += 1;
    setBrowserCatFocus({ cat, tick: browserCatTick.current, query });
    requestAnimationFrame(() => {
      document.querySelector('[data-testid="ctx-browser"]')?.scrollIntoView({ block: "center" });
    });
  }, []);
  const [fileChipFocus, setFileChipFocus] = useState<FileChipFocus | null>(null);
  const fileChipTick = useRef(0);
  const focusFileChip = useCallback((chip: FileChipFocus["chip"]) => {
    fileChipTick.current += 1;
    setFileChipFocus({ chip, tick: fileChipTick.current });
    requestAnimationFrame(() => {
      document.querySelector('[data-testid="ctx-files"]')?.scrollIntoView({ block: "center" });
    });
  }, []);
  const [commandFocus, setCommandFocus] = useState<{ program: string; tick: number } | null>(null);
  const commandFocusTick = useRef(0);
  const focusCommandProgram = useCallback((program: string) => {
    commandFocusTick.current += 1;
    setCommandFocus({ program, tick: commandFocusTick.current });
    requestAnimationFrame(() => {
      document.querySelector('[data-testid="ctx-commands"]')?.scrollIntoView({ block: "center" });
    });
  }, []);
  const [eventKindFocus, setEventKindFocus] = useState<{ kind: EventKind; tick: number } | null>(null);
  const eventKindTick = useRef(0);
  const focusEventKind = useCallback((kind: EventKind) => {
    eventKindTick.current += 1;
    setEventKindFocus({ kind, tick: eventKindTick.current });
  }, []);

  // v4.96 Token 卡→趋势联动（三级下钻收尾）：与 focusNode 同款 tick 锚点，
  // 重复点击重新触发定位/强调；不触碰趋势卡选中态。
  const [trendFocus, setTrendFocus] = useState<{ tick: number } | null>(null);
  const trendFocusTick = useRef(0);
  const jumpToTrend = useCallback(() => {
    trendFocusTick.current += 1;
    setTrendFocus({ tick: trendFocusTick.current });
  }, []);
  const focusContextEvent = useCallback((event: ContextEvent) => {
    if (event.kind !== "compact") {
      const hasNode = timeline.nodes.some((node) => node.seq === event.seq)
        || timeline.archive.some((node) => node.seq === event.seq);
      if (hasNode) {
        jumpToNode(event.seq);
        return;
      }
    }
    const target = timeline.requests.find((request) => request.seq >= event.seq);
    if (!target) return;
    setPicked(target);
    setBrowserMode("step");
    jumpToTrend();
  }, [jumpToNode, jumpToTrend, timeline.archive, timeline.nodes, timeline.requests]);

  const [net, setNet] = useState<AgentNetwork | null>(null);
  useLayoutEffect(() => {
    if (loadedSessionPath !== sessionPath) return;
    if (restoredViewport.current === sessionPath) return;
    if (timeline.requests.length === 0 && timeline.events.length === 0 && timeline.nodes.length === 0) return;
    const frame = requestAnimationFrame(() => {
      const el = viewportRef.current;
      if (!el) return;
      el.scrollTop = contextScrollBySession.get(sessionPath ?? "") ?? 0;
      restoredViewport.current = sessionPath;
    });
    return () => cancelAnimationFrame(frame);
  }, [loadedSessionPath, sessionPath, timeline.events.length, timeline.nodes.length, timeline.requests.length]);
  useEffect(() => {
    restoredViewport.current = undefined;
  }, [sessionPath]);
  // 家族用量聚合（1:1 上游 Context Stats 编队费用行；会话切换加载一次）
  const [familyTotals, setFamilyTotals] = useState<FamilyTotalsView | null>(null);
  // dsh-context 的 main row 用左侧「当前上下文 + 趋势」高度钳制右侧浏览器：
  // 两列等高，浏览器滚动区不把左侧卡片拉长。ResizeObserver 只量左列卡片。
  const mainLeftRef = useRef<HTMLDivElement | null>(null);
  const [browserCap, setBrowserCap] = useState<number | null>(null);
  // 1:1 上游 chips 联动：工具 chip 点击 → 浏览器按该工具过滤（再点同 chip 清除）
  const [browserToolFilter, setBrowserToolFilter] = useState<string | null>(null);
  // Agent 网络：订阅共享 store（按 UI 会话读取，v4.181）；running 时
  // useLiveReload 驱动的 load() 顺带 reload。会话切换由依赖重建触发 declarePath。
  // showAgentNetwork=false（原罪复用）不订阅不拉取——网络属办公内核会话，避免串台。
  useEffect(
    () => (showAgentNetwork ? subscribeAgentNetwork((n) => setNet(n), { path: sessionPath }) : undefined),
    [sessionPath, showAgentNetwork],
  );
  useEffect(() => {
    if (!showAgentNetwork) return;
    let alive = true;
    app.FamilyEvents(sessionPath ?? "")
      .then((f) => { if (alive && f.totals) setFamilyTotals(f.totals); })
      .catch(() => {});
    return () => { alive = false; };
  }, [sessionPath, showAgentNetwork]);
  useEffect(() => {
    const column = mainLeftRef.current;
    if (!column) return;
    const measure = () => {
      const cards = Array.from(column.children)
        .map((el) => el.getBoundingClientRect())
        .filter((rect) => rect.height > 0);
      const first = cards[0];
      const last = cards[cards.length - 1];
      const span = first && last ? last.bottom - first.top : 0;
      setBrowserCap(span > 0 ? Math.ceil(span) : null);
    };
    measure();
    if (typeof ResizeObserver !== "function") return;
    const observer = new ResizeObserver(measure);
    observer.observe(column);
    for (const child of Array.from(column.children)) observer.observe(child);
    return () => observer.disconnect();
  }, [timeline]);
  const sessionName = useMemo(() => {
    if (sessionNameProp) return sessionNameProp;
    if (!sessionPath) return "—";
    const base = sessionPath.split(/[\\/]/).pop() ?? sessionPath;
    return base.replace(/\.jsonl$/, "");
  }, [sessionNameProp, sessionPath]);
  const trendDnaFetcher = useMemo(() => {
    if (fetchDna) return fetchDna;
    if (fetchTimeline) return undefined;
    return (seqs: number[]) => app.ContextDna(seqs, sessionPath ?? "");
  }, [fetchDna, fetchTimeline, sessionPath]);
  // 空间归属：显式覆盖（原罪复用）优先；缺省按会话路径推断（/play/ = 乐园）。
  const space = spaceOverride ?? ((sessionPath ?? "").includes("/play/") ? t("contextview.spacePlay") : t("contextview.spaceWork"));

  const load = useCallback(() => {
    if (showAgentNetwork) void reloadAgentNetwork(sessionPath);
    const p = fetchTimeline
      ? fetchTimeline()
      : app.ContextView(sessionPath ?? "");
    p
      // 老后端可能把空切片序列化成 null，按数组消费前统一归一化
      .then((tl) => {
        if (currentSessionPath.current !== sessionPath) return;
        setTimeline({
          ...tl,
          requests: tl.requests ?? [],
          events: tl.events ?? [],
          nodes: tl.nodes ?? [],
          archive: tl.archive ?? [],
          files: tl.files ?? [],
          commands: tl.commands ?? [],
        });
        setLoadedSessionPath(sessionPath);
        setError(null);
      })
      .catch((err: unknown) => {
        if (currentSessionPath.current === sessionPath) {
          setError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {});
  }, [fetchTimeline, sessionPath, showAgentNetwork]);

  useEffect(() => {
    load();
  }, [load]);

  // 运行中随事件流节流刷新 + 回合结束立即刷新（useLiveReload）。
  useLiveReload(running, load);

  useEffect(() => {
    surfaceCache.current.clear();
    setBrowserMode("live");
    setSurfaceRead({ status: "live" });
  }, [sessionPath, fetchTimeline]);

  const effectiveBrowserRequest = hoveredRequest ?? (browserMode === "step" ? picked : null);
  const effectiveBrowserSeq = effectiveBrowserRequest?.seq ?? null;

  useEffect(() => {
    if (effectiveBrowserSeq == null) {
      setSurfaceRead({ status: "live" });
      return;
    }
    const cached = surfaceCache.current.get(effectiveBrowserSeq);
    if (cached) {
      setSurfaceRead({ status: "ready", seq: effectiveBrowserSeq, snapshot: cached });
      return;
    }
    if (fetchTimeline && !fetchSurface) {
      setSurfaceRead({ status: "unavailable", seq: effectiveBrowserSeq });
      return;
    }
    let alive = true;
    setSurfaceRead({ status: "loading", seq: effectiveBrowserSeq });
    const delay = hoveredRequest ? 140 : 0;
    const timer = window.setTimeout(() => {
      const read = fetchSurface
        ? fetchSurface(effectiveBrowserSeq)
        : app.ContextSurface(effectiveBrowserSeq, sessionPath ?? "");
      read
        .then((snapshot) => {
          if (!alive) return;
          surfaceCache.current.set(effectiveBrowserSeq, snapshot);
          setSurfaceRead({ status: "ready", seq: effectiveBrowserSeq, snapshot });
        })
        .catch(() => {
          if (alive) setSurfaceRead({ status: "error", seq: effectiveBrowserSeq });
        });
    }, delay);
    return () => {
      alive = false;
      window.clearTimeout(timer);
    };
  }, [effectiveBrowserSeq, fetchSurface, fetchTimeline, hoveredRequest, sessionPath]);

  // v4.69：默认选中最新请求（对齐 dsh——趋势卡详情随快照实时跟随最新一步，
  // 打开页面即有内容可读）；用户点柱钉住的选择在请求仍存在时保留不动，请求
  // 消失（如压缩重折叠移除）时回退最新。
  useEffect(() => {
    const reqs = timeline.requests;
    setPicked((cur) => {
      if (cur && reqs.some((r) => r.seq === cur.seq)) return cur;
      return reqs.length > 0 ? reqs[reqs.length - 1] : null;
    });
  }, [timeline.requests]);

  const pickBrowserRequest = useCallback((seq: number | null) => {
    if (seq == null) {
      setBrowserMode("live");
      return;
    }
    const request = timeline.requests.find((item) => item.seq === seq);
    if (!request) return;
    setPicked(request);
    setBrowserMode("step");
  }, [timeline.requests]);

  const isEmpty =
    timeline.window <= 0 &&
    catTotal(timeline.current) === 0 &&
    timeline.requests.length === 0 &&
    timeline.events.length === 0 &&
    timeline.nodes.length === 0 &&
    timeline.files.length === 0;
  const browserReady =
    surfaceRead.status === "ready" &&
    surfaceRead.seq === effectiveBrowserSeq;
  const browserNodes = effectiveBrowserSeq == null
    ? timeline.nodes
    : browserReady
      ? surfaceRead.snapshot.nodes
      : [];
  const browserArchive = effectiveBrowserSeq == null
    ? timeline.archive
    : browserReady
      ? surfaceRead.snapshot.archive
      : [];
  const browserSurfaceState = effectiveBrowserSeq == null
    ? "live"
    : surfaceRead.status === "ready" && browserReady
      ? "ready"
      : surfaceRead.status === "loading"
        ? "loading"
        : surfaceRead.status === "unavailable"
          ? "unavailable"
          : "error";
  return (
    <div
      ref={viewportRef}
      data-testid="context-cockpit"
      className="ctx-cockpit flex h-full min-h-0 flex-col gap-2 overflow-y-auto px-2.5 py-2"
      onScroll={(event) => {
        const key = sessionPath ?? "";
        contextScrollBySession.set(key, event.currentTarget.scrollTop);
        if (contextScrollBySession.size > 24) {
          const oldest = contextScrollBySession.keys().next().value;
          if (oldest !== undefined) contextScrollBySession.delete(oldest);
        }
      }}
    >
      {error && (
        <div className="shrink-0 rounded-lg border border-err/30 bg-del-bg px-3 py-2 text-[11px] text-err">
          {t("contextview.loadFail", { msg: error })}
        </div>
      )}
      {!error && isEmpty && (
        <div className="grid grid-cols-1 gap-2 min-[900px]:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
          <CurrentContextCard used={catTotal(timeline.current)} window={timeline.window} current={timeline.current} compact={timeline.compact} />
          <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border-soft bg-bg px-6 py-10 text-center">
            <span className="text-[12px] font-medium text-fg">{t("contextview.empty")}</span>
            <span className="max-w-[46ch] text-[10.5px] leading-relaxed text-fg-faint">
              {t("contextview.emptyHint")}
            </span>
          </div>
        </div>
      )}
      {!error && !isEmpty && (
        <>
          {/* dsh-context contextView：lc-cols.lc-head = StatsContext + 右侧信息列。 */}
          <div className="ctx-cols ctx-head-row" data-context-row="summary">
            <div className="ctx-col ctx-col-stats">
              <StatsCard
                stats={timeline.stats}
                nodes={timeline.nodes}
                files={timeline.files}
                commands={timeline.commands}
                family={familyTotals}
                timings={timeline.timing}
                onToolPick={(tool) => {
                  setBrowserToolFilter((cur) => (cur === tool ? null : tool));
                  focusBrowserCat("tool");
                }}
                onToolMore={() => {
                  setBrowserToolFilter(null);
                  focusBrowserCat("tool");
                }}
                onIOClick={(action) => {
                  if (action === "user") {
                    focusBrowserCat("user");
                    return;
                  }
                  focusFileChip(action);
                }}
                onCommandPick={focusCommandProgram}
                onEventsClick={focusEventKind}
              />
            </div>
            <div className="ctx-col ctx-col-session">
              <SessionInfoCard
                sessionName={sessionName}
                space={space}
                model={model || t("contextview.estimateModel")}
                window={timeline.window}
                requests={timeline.requests.length}
              />
              <SummaryBar
                sessionName={sessionName}
                used={catTotal(timeline.current)}
                window={timeline.window}
                requests={timeline.requests}
                costEstimate={timeline.stats.costEstimate}
                rate={timeline.rate}
              />
            </div>
          </div>

          {/* dsh-context contextView：Token / 耗时各占 lc-head 的一半。 */}
          <div className="ctx-cols ctx-instruments-row" data-context-row="instruments">
            <TokenCard requests={timeline.requests} current={timeline.current} stats={timeline.stats} onViewTrend={jumpToTrend} />
            <TimingCard timing={timeline.timing} />
          </div>

          {/* dsh-context contextView：左列 CurrentComposition + Trend，
              右列 ContextBrowser；浏览器最大高度跟随左列卡片总高度。 */}
          <div className="ctx-cols ctx-main-row" data-context-row="main">
            <div ref={mainLeftRef} className="ctx-col">
              <CurrentContextCard
                used={catTotal(timeline.current)}
                window={timeline.window}
                current={timeline.current}
                compact={timeline.compact}
              />
              <ContextTrendChart
                requests={timeline.requests}
                events={timeline.events}
                onPick={(request) => {
                  setPicked(request);
                  setBrowserMode("step");
                }}
                onHover={setHoveredRequest}
                onPickCategory={(cat, request) => {
                  setPicked(request);
                  setBrowserMode("step");
                  focusBrowserCat(cat);
                }}
                granularity={granularity}
                onGranularity={setGranularity}
                selectedSeq={picked?.seq ?? null}
                sessionKey={sessionPath}
                fetchDna={trendDnaFetcher}
                detail={picked ? (
                  <StepDetail
                    record={picked}
                    window={timeline.window}
                    onJump={jumpToNode}
                    stepsInTurn={timeline.requests.filter((r) => r.turn === picked.turn).length}
                    turnMode={granularity === "turn"}
                  />
                ) : null}
                focus={trendFocus}
              />
            </div>
            <div
              className="ctx-col ctx-col-browser"
              style={browserCap !== null ? { maxHeight: browserCap } : undefined}
            >
              <ContextBrowserTree
                nodes={browserNodes}
                archive={browserArchive}
                focus={focusNode}
                catFocus={browserCatFocus}
                sessionPath={sessionPath}
                fetchNodeDetail={fetchNodeDetail}
                toolFilter={browserToolFilter}
                requests={timeline.requests}
                previewRequest={effectiveBrowserRequest}
                previewing={hoveredRequest != null}
                surfaceMode={browserMode}
                onSurfaceModeChange={setBrowserMode}
                pinnedSeq={picked?.seq ?? null}
                onPickRequest={pickBrowserRequest}
                surfaceState={browserSurfaceState}
                surfaceDropped={browserReady ? surfaceRead.snapshot.nodesDropped : undefined}
                delta={effectiveBrowserRequest?.delta}
              />
            </div>
          </div>

          {/* dsh-context contextView：事件 / 文件活动并排。 */}
          <div className="ctx-cols ctx-audit-row" data-context-row="audit">
            <EventsList events={timeline.events} focusKind={eventKindFocus} onFocusEvent={focusContextEvent} />
            <FileActivityTree
              files={timeline.files}
              sessionPath={sessionPath}
              fetchNodeDetail={fetchNodeDetail}
              chipFocus={fileChipFocus}
              onLocateOp={jumpToNode}
              step={browserMode === "step" ? picked : null}
            />
          </div>

          {/* 命令面板独立整行；Agent 图保留 gaea 能力，置于主栅格之后。 */}
          {(timeline.commands.length > 0 || (timeline.commandStats?.runs ?? 0) > 0) && (
            <div className="ctx-cols" data-context-row="commands">
              <CommandsCard
                commands={timeline.commands}
                stats={timeline.commandStats}
                sessionPath={sessionPath}
                fetchNodeDetail={fetchNodeDetail}
                focusProgram={commandFocus}
              />
            </div>
          )}
          {net && (
            <div className="ctx-cols" data-context-row="agent-network">
              <AgentRadial
                network={net}
                running={running}
                sessionPath={sessionPath}
                onViewContext={onViewSubagentContext}
              />
            </div>
          )}
          {/* 底部只看估算口径；会话汇总并入顶部右列，对应源码 PluginInfo/RecomCard 两卡列。 */}
          <div className="text-[9.5px] leading-relaxed text-fg-faint">{t("contextview.estimateNote")}</div>
        </>
      )}
    </div>
  );
}
