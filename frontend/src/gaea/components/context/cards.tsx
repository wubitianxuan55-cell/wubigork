/* eslint-disable react-refresh/only-export-components -- 纯展示卡与格式化助手同文件（对齐 ContextView 收敛模式） */
import { useT } from "../../lib/i18n";
// cards.tsx — dsh-context 头部仪表卡移植（对齐 dsh-context 插件 StatsCard/TokenCard/
// TimingCard/SessionInfoCard/SummaryBar 信息架构，套用 gaea 星枢皮肤）。
//
// i18n 待主代理收口：文案暂硬编码中文，收口时统一替换为 t(DictKey)。
// 本文件保持纯展示（无 hooks、无 bridge、无 i18n provider），便于 jsdom 定向测试。
//
// 语义色豁免说明：六分类语义色 hex 与 ContextView.tsx 的 CAT_COLORS 单源保持一致
// （页面蓝图 context.md 豁免条款）；主代理收口时可将其提升为共享模块再统一 import。
import type { ContextRequestRecord, ContextStats, ContextTiming } from "../../lib/types";
import { fmtTokens } from "../../lib/stats";

// ─── 语义色（与 ContextView CAT_COLORS 同值；hex-exempt 页面级语义色板） ────
const COLORS = {
  donutTrack: "#8080802e", // hex-exempt Donut 轨道色（同 dsh 原版 #8080802e）
  cacheHit: "#22c55e", // hex-exempt 缓存命中绿（= user 绿）
  cacheMiss: "#a855f7", // hex-exempt 未缓存输入紫（= inject 紫）
  output: "#3b82f6", // hex-exempt 输出蓝（= system 蓝）
  modelWait: "#3b82f6", // hex-exempt 模型等待蓝
  modelGen: "#06b6d4", // hex-exempt 模型生成青（= tool 青）
  toolExec: "#f59e0b", // hex-exempt 工具执行橙（= tools 橙）
  other: "#808080", // hex-exempt 其他开销灰
  idle: "#808080", // hex-exempt {t("contextview.idleWindow")}灰
} as const;

// 六分类图例（标签与 ContextView CATS 的 zh 文案一致；i18n 待主代理收口）
const LEGEND_CATS: { key: keyof ContextRequestRecord["category"]; color: string; label: string }[] = [
  { key: "system", color: "#3b82f6", label: "系统提示词" }, // hex-exempt 六分类语义色
  { key: "tools", color: "#f59e0b", label: "工具定义" }, // hex-exempt 六分类语义色
  { key: "user", color: "#22c55e", label: "用户消息" }, // hex-exempt 六分类语义色
  { key: "inject", color: "#a855f7", label: "注入内容" }, // hex-exempt 六分类语义色
  { key: "assistant", color: "#1e40af", label: "助手消息" }, // hex-exempt 六分类语义色
  { key: "tool", color: "#06b6d4", label: "工具结果" }, // hex-exempt 六分类语义色
];

// ─── 纯格式化/聚合助手（导出以便测试） ─────────────────────────

/** ms → 「x 分 x 秒」/「x 秒」（dsh 时长口径；亚秒归入 0 秒，不伪造精度） */
export function fmtDuration(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000));
  if (total < 60) return `${total} 秒`;
  return `${Math.floor(total / 60)} 分 ${total % 60} 秒`;
}

export interface TokenSummary {
  hit: number; // 缓存输入（cacheHitTokens 累计）
  miss: number; // 未缓存输入（cacheMissTokens 累计）
  out: number; // 输出（outputTokens 累计）
}

/** TokenCard 汇总：sum(cacheHitTokens)/sum(cacheMissTokens)/sum(outputTokens) */
export function summarizeTokens(requests: ContextRequestRecord[]): TokenSummary {
  let hit = 0, miss = 0, out = 0;
  for (const r of requests) {
    hit += r.cacheHitTokens ?? 0;
    miss += r.cacheMissTokens ?? 0;
    out += r.outputTokens ?? 0;
  }
  return { hit, miss, out };
}

function fmtPct(part: number, whole: number): string {
  return whole > 0 ? `${Math.round((part / whole) * 100)}%` : "0%";
}

// ─── 卡片外壳与公共小件 ─────────────────────────────────────────

function CardHead({ title, sub }: { title: string; sub?: string }) {
  // v4.69（对齐 dsh 仪表头）：标题左置、副注右置同行的单行卡头，数据密度优先。
  return (
    <div className="flex items-baseline justify-between gap-2">
      <div className="min-w-0 truncate text-[11px] font-medium text-fg">{title}</div>
      {sub && (
        <div className="min-w-0 shrink-0 truncate text-right text-[9.5px] leading-none text-fg-faint" title={sub}>
          {sub}
        </div>
      )}
    </div>
  );
}

/** label 上小字 + value 下大字（等宽）的统计格 */
function StatCell({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="truncate text-[9.5px] text-fg-faint">{label}</div>
      <div className={`truncate font-mono tabular-nums text-fg ${strong ? "text-[14px]" : "text-[13px]"}`}>{value}</div>
    </div>
  );
}

function Dot({ color }: { color: string }) {
  return <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: color }} />;
}

// ─── Donut 环形（纯 SVG：circle stroke-dasharray，track 同 dsh #8080802e） ──
function Donut({
  segments,
  center,
  centerSub,
  size = 84,
  stroke = 9,
}: {
  segments: { value: number; color: string }[];
  center: string;
  centerSub: string;
  size?: number;
  stroke?: number;
}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const total = segments.reduce((s, seg) => s + Math.max(0, seg.value), 0);
  let acc = 0;
  const half = size / 2;
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden>
        <circle cx={half} cy={half} r={r} fill="none" strokeWidth={stroke} stroke={COLORS.donutTrack} />
        {total > 0 &&
          segments.map((seg, i) => {
            const len = (Math.max(0, seg.value) / total) * c;
            const dash = `${len} ${c - len}`;
            const offset = -acc;
            acc += len;
            return (
              <circle
                key={i}
                cx={half}
                cy={half}
                r={r}
                fill="none"
                stroke={seg.color}
                strokeWidth={stroke}
                strokeDasharray={dash}
                strokeDashoffset={offset}
                transform={`rotate(-90 ${half} ${half})`}
              />
            );
          })}
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="font-mono text-[14px] tabular-nums leading-none text-fg">{center}</span>
        <span className="mt-0.5 text-[8.5px] leading-none text-fg-faint">{centerSub}</span>
      </div>
    </div>
  );
}

// ─── 1. StatsCard 上下文统计（8 格：4 列 × 2 行） ─────────────────
export function StatsCard({ stats }: { stats: ContextStats }) {
  const t = useT();
  const cells: [string, string][] = [
    [t("contextview.statTurns"), String(stats.turns)],
    [t("contextview.statSteps"), String(stats.steps)],
    [t("contextview.statToolCalls"), String(stats.toolCalls)],
    [t("contextview.statImages"), String(stats.images)],
    [t("contextview.statCost"), stats.costEstimate != null ? `¥${stats.costEstimate.toFixed(2)}` : "—"],
    [t("contextview.statInjects"), String(stats.injects)],
    [t("contextview.statCompacts"), String(stats.compacts)],
    [t("contextview.statPrunes"), String(stats.prunes)],
  ];
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <CardHead title={t("contextview.statsTitle")} sub={t("contextview.statsHint")} />
      <div className="mt-2 grid grid-cols-4 gap-x-3 gap-y-2">
        {cells.map(([label, value]) => (
          <StatCell key={label} label={label} value={value} />
        ))}
      </div>
    </div>
  );
}

// ─── 2. TokenCard Token 统计（环形缓存命中 + 三分解行） ───────────
export function TokenCard({ requests }: { requests: ContextRequestRecord[] }) {
  const t = useT();
  const { hit, miss, out } = summarizeTokens(requests);
  const graded = hit + miss;
  const total = graded + out;
  const hitPct = graded > 0 ? ((hit / graded) * 100).toFixed(2) + "%" : "—"; // v4.69 对齐 dsh：环心命中率两位小数
  const rows = [
    { label: t("contextview.tokensCached"), value: hit, color: COLORS.cacheHit },
    { label: t("contextview.tokensUncached"), value: miss, color: COLORS.cacheMiss },
    { label: t("contextview.tokensOutput"), value: out, color: COLORS.output },
  ];
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <CardHead title={t("contextview.tokensTitle")} sub={t("contextview.tokensHint")} />
      <div className="mt-2 flex items-center gap-4">
        <Donut
          segments={rows.map((r) => ({ value: r.value, color: r.color }))}
          center={hitPct}
          centerSub={t("contextview.statCacheHit")}
        />
        <div className="min-w-0 flex-1 space-y-1.5">
          {rows.map((r) => (
            <div key={r.label} className="flex items-center gap-1.5 text-[10px]">
              <Dot color={r.color} />
              <span className="shrink-0 text-fg-dim">{r.label}</span>
              <span className="ml-auto min-w-0 truncate text-right font-mono tabular-nums text-fg">
                {total > 0 ? fmtTokens(r.value) : "—"}
              </span>
              <span className="w-8 shrink-0 text-right font-mono tabular-nums text-fg-faint">{fmtPct(r.value, total)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── 3. TimingCard 耗时统计（环形活跃时长 + 四分解行 + 工具 top3） ──
interface TimingRow {
  label: string;
  ms: number | null;
  count: number | null; // 次数；其他开销无对应计数 → null
  color: string;
}

function timingRows(timing: ContextTiming, t: (k: "contextview.timingWait" | "contextview.timingGen" | "contextview.timingTools" | "contextview.timingOther") => string): TimingRow[] {
  const { wallMs: wall, ttftMs: ttft, genMs: gen, toolsMs: tools } = timing;
  // 其他开销 = wall − 等待 − 生成 − 工具；仅在四项齐备时推导，缺项显「—」不伪造
  const other = wall != null && ttft != null && gen != null && tools != null
    ? Math.max(0, wall - ttft - gen - tools)
    : null;
  return [
    { label: t("contextview.timingWait"), ms: ttft ?? null, count: timing.calls ?? null, color: COLORS.modelWait },
    { label: t("contextview.timingGen"), ms: gen ?? null, count: timing.calls ?? null, color: COLORS.modelGen },
    { label: t("contextview.timingTools"), ms: tools ?? null, count: timing.toolCalls ?? null, color: COLORS.toolExec },
    { label: t("contextview.timingOther"), ms: other, count: null, color: COLORS.other },
  ];
}

// timing 未就绪时的空态骨架行（全「—」，不伪造数值）
const TIMING_EMPTY_LABEL_KEYS = [
  "contextview.timingWait",
  "contextview.timingGen",
  "contextview.timingTools",
  "contextview.timingOther",
] as const;
const TIMING_EMPTY_ROWS: TimingRow[] = TIMING_EMPTY_LABEL_KEYS.map((k) => ({
  label: k,
  ms: null,
  count: null,
  color:
    k === "contextview.timingWait" ? COLORS.modelWait
    : k === "contextview.timingGen" ? COLORS.modelGen
    : k === "contextview.timingTools" ? COLORS.toolExec
    : COLORS.other,
}));

export function TimingCard({ timing }: { timing?: ContextTiming }) {
  const t = useT();
  const rows = (timing ? timingRows(timing, t) : TIMING_EMPTY_ROWS).map((r) => ({ ...r, label: t(r.label as Parameters<typeof t>[0]) }));
  const wall = timing?.wallMs ?? null;
  const topTools = (timing?.tools ?? [])
    .map((t) => ({ name: t.name, calls: t.calls, ms: t.ms }))
    .sort((a, b) => b.ms - a.ms || b.calls - a.calls) // 后端已按 ms 降序，此处防御性保序
    .slice(0, 3);
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <CardHead title={t("contextview.timingTitle")} sub={t("contextview.timingHint")} />
      <div className="mt-2 flex items-center gap-4">
        <Donut
          segments={rows.map((r) => ({ value: r.ms ?? 0, color: r.color }))}
          center={wall != null ? fmtDuration(wall) : "—"}
          centerSub="活跃时长"
        />
        <div className="min-w-0 flex-1 space-y-1.5">
          {rows.map((r) => (
            <div key={r.label} className="flex items-center gap-1.5 text-[10px]">
              <Dot color={r.color} />
              <span className="shrink-0 text-fg-dim">{r.label}</span>
              <span className="ml-auto min-w-0 truncate text-right font-mono tabular-nums text-fg">
                {r.ms != null ? fmtDuration(r.ms) : "—"}
              </span>
              <span className="w-10 shrink-0 text-right font-mono tabular-nums text-fg-faint">
                {r.count != null ? `${r.count} 次` : "—"}
              </span>
              <span className="w-8 shrink-0 text-right font-mono tabular-nums text-fg-faint">
                {r.ms != null && wall ? fmtPct(r.ms, wall) : "—"}
              </span>
            </div>
          ))}
        </div>
      </div>
      {topTools.length > 0 && (
        <div className="mt-2 border-t border-border-soft pt-1.5" data-testid="timing-tools">
          {topTools.map((t) => (
            <div key={t.name} className="flex items-center gap-1.5 text-[9.5px]">
              <span className="min-w-0 truncate text-fg-faint">{t.name}</span>
              <span className="ml-auto shrink-0 font-mono tabular-nums text-fg-dim">{t.calls} 次</span>
              <span className="w-14 shrink-0 text-right font-mono tabular-nums text-fg-faint">{fmtDuration(t.ms)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── 4. SessionInfoCard 会话信息（行式键值直显） ──────────────────
export function SessionInfoCard({
  sessionName,
  space,
  model,
  window: win,
  requests,
}: {
  sessionName: string;
  space: string;
  model: string;
  window: number;
  requests: number;
}) {
  const t = useT();
  const rows: [string, string][] = [
    [t("contextview.sessionName"), sessionName],
    [t("contextview.sessionSpace"), space],
    [t("contextview.sessionModel"), model],
    [t("contextview.sessionWindow"), String(win)],
    [t("contextview.sessionRequests"), String(requests)],
  ];
  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <CardHead title="会话信息" />
      <div className="mt-2 space-y-1">
        {rows.map(([k, v]) => (
          <div key={k} className="flex items-baseline justify-between gap-2 text-[10px]">
            <span className="shrink-0 text-fg-faint">{k}</span>
            <span className="min-w-0 truncate text-right font-mono tabular-nums text-fg" title={v}>{v}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── 5. SummaryBar 底部汇总条（单行摘要 + 六分类图例） ────────────
export function SummaryBar({
  sessionName,
  used,
  window: win,
  requests,
  costEstimate,
}: {
  sessionName: string;
  used: number;
  window: number;
  requests: ContextRequestRecord[];
  costEstimate?: number;
}) {
  const t = useT();
  const pct = win > 0 ? Math.min(100, Math.round((used / win) * 100)) : 0;
  return (
    <div className="shrink-0 rounded-lg border border-border-soft bg-bg px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="shrink-0 rounded-full bg-accent/15 px-2 py-px text-[10px] font-medium text-accent">
          {sessionName}
        </span>
        <span className="font-mono text-[10px] tabular-nums text-fg-dim">
          {fmtTokens(used)} / {fmtTokens(win)} · {pct}%
        </span>
        <span className="font-mono text-[10px] tabular-nums text-fg-faint">{t("contextview.summaryRequests", { n: requests.length })}</span>
        <span className="font-mono text-[10px] tabular-nums text-fg-faint">
          {costEstimate != null ? `${t("contextview.summaryCost")} ¥${costEstimate.toFixed(2)}` : `${t("contextview.summaryCost")} —`}
        </span>
        <span className="min-[900px]:flex-1" />
        <span className="flex flex-wrap items-center gap-x-2.5 gap-y-0.5">
          {LEGEND_CATS.map((c) => (
            <span key={c.key} className="inline-flex items-center gap-1 text-[9.5px] text-fg-faint">
              <Dot color={c.color} />
              {c.label}
            </span>
          ))}
          <span className="inline-flex items-center gap-1 text-[9.5px] text-fg-faint">
            <Dot color={COLORS.idle} />
            {t("contextview.idleWindow")}
          </span>
        </span>
      </div>
    </div>
  );
}
