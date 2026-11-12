/* eslint-disable react-refresh/only-export-components -- 纯展示卡与格式化助手同文件（对齐 ContextView 收敛模式） */
import { useT, type Translator } from "../../lib/i18n";
import "./context-view.css";
import { Clock, Coins, MessageSquare, type Icon } from "../../icons";
import type { ReactNode } from "react";
// cards.tsx — dsh-context 头部仪表卡移植（对齐 dsh-context 插件 StatsCard/TokenCard/
// TimingCard/SessionInfoCard/SummaryBar 信息架构，套用 gaea 星枢皮肤）。
//
// 文案全部走 t(DictKey)（i18n 收口批，此前硬编码中文）；六分类键名与语义色
// 从 context/cats.ts 单源派生。本文件保持纯展示（无 bridge），便于 jsdom 定向测试。
import type { ContextCategory, ContextCommandRecord, ContextRequestRecord, ContextStats, ContextSurfaceNode, ContextTiming } from "../../lib/types";
import type { DictKey } from "../../locales/en";
import { fmtTokens } from "../../lib/stats";
import { CAT_COLORS, CONTEXT_CATS } from "./cats";

// ─── 语义色（六分类色单源在 cats.ts；下列为卡片专属的非分类语义色） ────
const COLORS = {
  donutTrack: "#8080802e", // hex-exempt Donut 轨道色（同 dsh 原版 #8080802e）
  cacheHit: "#22c55e", // hex-exempt 缓存命中绿（= user 绿）
  cacheMiss: "#a855f7", // hex-exempt 未缓存输入紫（= inject 紫）
  output: "#3b82f6", // hex-exempt 输出蓝（= system 蓝）
  modelWait: "#3b82f6", // hex-exempt 模型等待蓝
  modelGen: "#06b6d4", // hex-exempt 模型生成青（= tool 青）
  toolExec: "#f59e0b", // hex-exempt 工具执行橙（= tools 橙）
  other: "#808080", // hex-exempt 其他开销灰
  idle: "#808080", // hex-exempt 空闲窗口灰
} as const;

// 六分类图例：键名+语义色+文案全部从 cats.ts 单源派生（此前本地另抄一份
// 色值与中文标签，与 CAT_COLORS/CONTEXT_CATS 双源漂移风险）。
const LEGEND_CATS: { key: keyof ContextRequestRecord["category"]; color: string; labelKey: DictKey }[] =
  CONTEXT_CATS.map(({ key, labelKey }) => ({ key, color: CAT_COLORS[key], labelKey }));

// ─── 纯格式化/聚合助手（导出以便测试） ─────────────────────────

/** ms → 「x 分 x 秒」/「x 秒」（dsh 时长口径；亚秒归入 0 秒，不伪造精度）。
 *  单位词由调用方经 t 注入（contextview.durSec/durMinSec），保持纯函数可测。 */
export function fmtDuration(ms: number, t: Translator): string {
  const total = Math.max(0, Math.round(ms / 1000));
  if (total < 60) return t("contextview.durSec", { n: total });
  return t("contextview.durMinSec", { m: Math.floor(total / 60), s: total % 60 });
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

function CardHead({ title, sub, icon, extra }: { title: string; sub?: string; icon?: Icon; extra?: ReactNode }) {
  // v4.71（卡片化）：标题前带 22px 图标章；标题左置（12.5px w600）、
  // 副注右置（10.5px）同行的单行卡头。
  // v4.96：可选 extra 插槽（副注右侧的轻量操作位，如 Token 卡「查看趋势」）；
  // 缺省不渲染，既有外观不变。
  const HeadIcon = icon;
  return (
    <div className="flex items-center justify-between gap-2">
      <div className="flex min-w-0 items-center gap-1.5">
        {HeadIcon && (
          <span className="ctx-head-ic" aria-hidden>
            <HeadIcon size={12} />
          </span>
        )}
        <span className="min-w-0 truncate text-[12.5px] font-semibold text-fg">{title}</span>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {sub && (
          <div className="max-w-[44%] truncate text-right text-[10.5px] leading-none text-fg-faint" title={sub}>
            {sub}
          </div>
        )}
        {extra}
      </div>
    </div>
  );
}

function Dot({ color }: { color: string }) {
  return <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: color }} />;
}

// ─── Donut 环形（纯 SVG：circle stroke-dasharray，track 同 dsh #8080802e） ──
// v4.542 刀3 导出：洞察页构成环形复用（同源色板与轨道样式）。
export function Donut({
  segments,
  center,
  centerSub,
  size = 92,
  stroke = 10,
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
        <span className="font-mono text-[16px] font-semibold tabular-nums leading-none text-fg">{center}</span>
        <span className="mt-0.5 text-[9.5px] leading-none text-fg-faint">{centerSub}</span>
      </div>
    </div>
  );
}

// ─── 1. StatsCard 上下文统计（v4.71：8 个独立小卡，不再合成 1 张大卡） ──
export function StatsCard({ stats, nodes, files, commands, family, timings, onToolPick, onIOClick }: {
  stats: ContextStats;
  // 1:1 上游 Context Stats 复合卡数据面：nodes（工具 chips/user 数聚合）、
  // files（Input/Output 聚合）、commands（程序 chips）、family（编队费用行
  // =编队合计/当前 Agent/子 Agent ×N，1:1 上游三行）、timings（模型调用次数）。
  nodes: ContextSurfaceNode[];
  files: FileActivityLite[];
  commands: ContextCommandRecord[];
  family?: { rootTokens: number; rootCost?: number; subTokens: number; subCost?: number; subCount: number } | null;
  timings?: ContextTiming;
  // 1:1 上游 chips 点击联动：工具 chip→浏览器该工具过滤；IO chip→文件活动
  // 对应动作过滤（上游同款交互：点 chip 即过滤到该面）。
  onToolPick?: (tool: string) => void;
  onIOClick?: (action: "read" | "write" | "search") => void;
}) {
  const t = useT();
  // 4 tiles（1:1：Turns/Steps/Skill Loads/模型回复次数）
  const tiles = [
    { key: "turns", label: t("contextview.statTurns"), value: String(stats.turns) },
    { key: "steps", label: t("contextview.statSteps"), value: String(stats.steps) },
    { key: "skills", label: t("contextview.statSkills"), value: String(stats.skills ?? 0) },
    { key: "calls", label: t("contextview.statModelCalls"), value: String(timings?.calls ?? 0) },
  ];
  // Context Events 计数（stats 全量口径）
  const evLine = [
    { label: t("contextview.evInject"), n: stats.injects, cls: "text-(color:--color-success)" },
    { label: t("contextview.evCompact"), n: stats.compacts, cls: "text-err" },
    { label: t("contextview.evPrune"), n: stats.prunes, cls: "text-warning" },
  ];
  // Input / Output chips（用户消息/读/写/图——文件活动聚合+图片计数）
  let reads = 0, writes = 0;
  for (const f of files) {
    if (f.action === "read") reads++;
    else if (f.action === "write") writes++;
  }
  let userCount = 0;
  for (const n of nodes) if (n.cat === "user") userCount++;
  const ioChips = [
    { label: "User", n: userCount, onClick: undefined },
    { label: "Read", n: reads, onClick: onIOClick ? () => onIOClick("read") : undefined },
    { label: "Write", n: writes, onClick: onIOClick ? () => onIOClick("write") : undefined },
    { label: "Image", n: stats.images, onClick: undefined },
  ];
  // 工具调用 chips（结果节点聚合 top5）
  const toolTally = new Map<string, number>();
  for (const n of nodes) {
    if (n.cat === "tool" && n.tool) toolTally.set(n.tool, (toolTally.get(n.tool) ?? 0) + 1);
  }
  const toolChips = [...toolTally.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  const toolMore = Math.max(0, toolChips.length - 5);
  // 命令程序 chips
  const progTally = new Map<string, number>();
  for (const c of commands) {
    const pName = commandProgramOf(c.cmd ?? "");
    progTally.set(pName, (progTally.get(pName) ?? 0) + 1);
  }
  const progChips = [...progTally.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  const progMore = Math.max(0, progChips.length - 5);
  const chip = (label: string, n: number, onClick?: () => void) =>
    onClick ? (
      <button key={label} type="button" onClick={onClick} title={label} className="inline-flex cursor-pointer items-center gap-1 rounded border-0 bg-bg-soft px-1.5 py-px text-[10px] text-fg-dim hover:text-fg">
        <span aria-hidden className="h-1.5 w-1.5 rounded-sm" style={{ background: "var(--md-sys-color-outline, #9e9e9e)" }} />
        {label} <b className="font-mono text-fg">{n}</b>
      </button>
    ) : (
      <span key={label} className="inline-flex items-center gap-1 rounded bg-bg-soft px-1.5 py-px text-[10px] text-fg-dim">
        <span aria-hidden className="h-1.5 w-1.5 rounded-sm" style={{ background: "var(--md-sys-color-outline, #9e9e9e)" }} />
        {label} <b className="font-mono text-fg">{n}</b>
      </span>
    );
  return (
    <div data-testid="stats-kpi-grid" className="ctx-card p-3">
      <div className="grid grid-cols-2 gap-2 min-[1100px]:grid-cols-4">
        {tiles.map((c) => (
          <div key={c.key} className="ctx-tile" data-testid={`stat-tile-${c.key}`}>
            <span className="truncate text-[10px] font-medium leading-none text-fg-faint">{c.label}</span>
            <span className="mt-0.5 truncate font-mono text-[15px] font-semibold leading-tight tabular-nums text-fg">
              {c.value}
            </span>
          </div>
        ))}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2 border-t border-border-soft/60 pt-2 text-[10.5px]">
        <span className="shrink-0 text-fg-dim">{t("contextview.eventsTitle")}</span>
        {evLine.map((e) => (
          <span key={e.label} className="font-mono tabular-nums">
            <span className={e.n > 0 ? e.cls : "text-fg-faint"}>{e.label} {e.n}</span>
          </span>
        ))}
      </div>
      <StatChipRow label={t("contextview.ioRow")} total={userCount + reads + writes}>
        {ioChips.map((c) => chip(c.label, c.n, c.onClick))}
      </StatChipRow>
      <StatChipRow label={t("contextview.statToolCallsRow")} total={stats.toolCalls}>
        {toolChips.slice(0, 5).map(([name, n]) => chip(name, n, onToolPick ? () => onToolPick(name) : undefined))}
        {toolMore > 0 && <span className="text-[10px] text-fg-faint">+{toolMore} {t("contextview.moreChips")}</span>}
      </StatChipRow>
      <StatChipRow label={t("contextview.cmdRow")} total={commands.length}>
        {progChips.slice(0, 5).map(([name, n]) => chip(name, n))}
        {progMore > 0 && <span className="text-[10px] text-fg-faint">+{progMore} {t("contextview.moreChips")}</span>}
      </StatChipRow>
      {family && (family.rootTokens > 0 || family.subTokens > 0) && (
        <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border-soft/60 pt-2 text-[10.5px]">
          <span className="shrink-0 text-fg-dim">{t("contextview.fleetRow")}</span>
          <span className="font-mono tabular-nums text-fg">
            {t("contextview.fleetTotal")} {fmtTokens(family.rootTokens + family.subTokens)}
            {(family.rootCost != null && family.subCost != null) && ` / ¥${(family.rootCost + family.subCost).toFixed(2)}`}
          </span>
          <span className="font-mono tabular-nums text-fg-dim">
            {t("contextview.fleetCurrent")} {fmtTokens(family.rootTokens)}
            {family.rootCost != null && ` / ¥${family.rootCost.toFixed(2)}`}
          </span>
          {family.subCount > 0 && (
            <span className="font-mono tabular-nums text-fg-dim">
              {t("contextview.fleetSubsN", { n: family.subCount })} {fmtTokens(family.subTokens)}
              {family.subCost != null && ` / ¥${family.subCost.toFixed(2)}`}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function StatChipRow({ label, total, children }: { label: string; total: number; children: React.ReactNode }) {
  return (
    <div className="mt-1.5 flex flex-wrap items-center gap-2 text-[10.5px]">
      <span className="w-28 shrink-0 text-fg-dim">{label}</span>
      <b className="shrink-0 font-mono tabular-nums text-fg">{total}</b>
      <span className="flex min-w-0 flex-wrap items-center gap-1">{children}</span>
    </div>
  );
}

// commandProgramOf 与 CommandsCard 的程序提取同口径（前端侧）。
function commandProgramOf(cmd: string): string {
  for (const tok of cmd.split(/\s+/).filter(Boolean)) {
    if (tok.includes("=") && !tok.startsWith("/") && !tok.startsWith("-")) continue;
    if (tok === "sudo" || tok === "nice" || tok === "nohup" || tok === "command") continue;
    const base = tok.replace(/^["']|["']$/g, "").split(/[/]/).pop();
    return base || tok;
  }
  return "other";
}

// FileActivityLite 是 StatsCard 复合卡的文件聚合输入（避免整卡依赖 FileActivity 类型）。
export interface FileActivityLite {
  action: string;
}

// ─── 2. TokenCard Token 统计（环形缓存命中 + 三分解行） ───────────
// v4.96 三级下钻收尾（聚合 Token 卡 → 趋势）：可选 onViewTrend 回调（缺省不
// 渲染入口，既有消费方/测试零破坏）；点击只触发外层定位+强调锚点，不改趋势
// 卡选中态（保持默认最新请求语义）。纯展示不持 ref——滚动由 ContextView 完成。
export function TokenCard({
  requests,
  current,
  stats,
  onViewTrend,
}: {
  requests: ContextRequestRecord[];
  // current 是当前七分类构成（1:1 上游 Token Stats：构成口径为主卡）。
  current?: ContextCategory;
  // stats 可选：缓存命中右上+思考/回复/调用参数三拆行（刀1 字段消费）。
  stats?: ContextStats;
  onViewTrend?: () => void;
}) {
  const t = useT();
  const { hit, miss, out } = summarizeTokens(requests);
  const graded = hit + miss;
  const total = graded + out;
  const hitPct = graded > 0 ? ((hit / graded) * 100).toFixed(2) + "%" : "—"; // v4.69 对齐 dsh：环心命中率两位小数
  // 1:1 上游 8 行（工具调用=结果+调用参数；思考/回复三拆；助手其余=正文）
  const call = stats?.assistantCallTokens ?? 0;
  const reasoning = stats?.assistantReasoning ?? 0;
  const answers = stats?.assistantAnswerTokens ?? 0;
  const cat = current;
  const compRows = cat
    ? [
        { key: "system", label: t("contextview.catSystem"), value: cat.system, color: CAT_COLORS.system },
        { key: "tools", label: t("contextview.catTools"), value: cat.tools, color: CAT_COLORS.tools },
        { key: "user", label: t("contextview.catUser"), value: cat.user, color: CAT_COLORS.user },
        { key: "inject", label: t("contextview.catInject"), value: cat.inject, color: CAT_COLORS.inject },
        { key: "skill", label: t("contextview.catSkill"), value: cat.skill, color: CAT_COLORS.skill },
        { key: "tool", label: t("contextview.statToolCallsRow"), value: cat.tool + call, color: CAT_COLORS.tool },
        { key: "reasoning", label: t("contextview.statThinking"), value: reasoning, color: "#6366f1" }, // hex-exempt Token 统计语义色
        { key: "answers", label: t("contextview.statAnswers"), value: answers, color: "#ec4899" }, // hex-exempt Token 统计语义色
      ]
    : [];
  const compTotal = compRows.reduce((s, r) => s + r.value, 0);
  return (
    <div className="ctx-card p-3">
      <CardHead
        title={t("contextview.tokensTitle")}
        sub={t("contextview.tokensHint")}
        icon={Coins}
        extra={
          <span className="flex items-center gap-2">
            {stats?.cacheHitPercent != null && (
              <span className="font-mono text-[10.5px] text-fg-dim">
                {t("contextview.statCacheHit")} {stats.cacheHitPercent.toFixed(2)}%
              </span>
            )}
            {onViewTrend && (
              <button
                type="button"
                data-testid="token-view-trend"
                title={t("contextview.trendJumpTip")}
                className="shrink-0 cursor-pointer border-0 bg-transparent p-0 text-[10.5px] leading-none text-accent hover:underline"
                onClick={onViewTrend}
              >
                {t("contextview.viewTrend")}
              </button>
            )}
          </span>
        }
      />
      {cat ? (
        <>
          <div className="mt-2 flex items-center gap-4">
            <Donut
              segments={compRows.map((r) => ({ value: r.value, color: r.color }))}
              center={fmtTokens(compTotal)}
              centerSub={t("contextinsights.totalUsage")}
            />
            <ul className="m-0 grid min-w-0 flex-1 list-none grid-cols-3 gap-x-3 gap-y-1 p-0 text-[10.5px]">
              {compRows.map((r) => {
                const pct = compTotal > 0 ? Math.round((r.value / compTotal) * 1000) / 10 : 0;
                return (
                  <li key={r.key} className="flex items-center gap-1.5">
                    <Dot color={r.color} />
                    <span className="min-w-0 shrink truncate text-fg-dim" title={r.label}>{r.label}</span>
                    <span className="ml-auto shrink-0 text-right font-mono tabular-nums text-fg-faint">{pct.toFixed(1)}%</span>
                    <span className="w-14 shrink-0 text-right font-mono tabular-nums text-fg">{r.value > 0 ? `=${fmtTokens(r.value)}` : "—"}</span>
                  </li>
                );
              })}
            </ul>
          </div>
          {/* 缓存三分解行（gaea 增强维度保留：cached/uncached/output） */}
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-0.5 border-t border-border-soft/60 pt-1.5 text-[10px] text-fg-faint">
            <span>{t("contextview.tokensCached")} {fmtTokens(hit)}</span>
            <span>{t("contextview.tokensUncached")} {fmtTokens(miss)}</span>
            <span>{t("contextview.tokensOutput")} {fmtTokens(out)}</span>
          </div>
        </>
      ) : (
        <div className="mt-2 flex items-center gap-4">
          <Donut
            segments={[{ value: hit, color: COLORS.cacheHit }, { value: miss, color: COLORS.cacheMiss }, { value: out, color: COLORS.output }]}
            center={hitPct}
            centerSub={t("contextview.statCacheHit")}
          />
          <ul className="m-0 min-w-0 flex-1 list-none space-y-1.5 p-0 text-[11.5px]">
            {[{ label: t("contextview.tokensCached"), value: hit, color: COLORS.cacheHit }, { label: t("contextview.tokensUncached"), value: miss, color: COLORS.cacheMiss }, { label: t("contextview.tokensOutput"), value: out, color: COLORS.output }].map((r) => (
              <li key={r.label} className="flex items-center gap-1.5">
                <Dot color={r.color} />
                <span className="shrink-0 text-fg-dim">{r.label}</span>
                <span className="ml-auto min-w-0 truncate text-right font-mono tabular-nums text-fg">
                  {total > 0 ? fmtTokens(r.value) : "—"}
                </span>
                <span className="w-8 shrink-0 text-right font-mono tabular-nums text-fg-faint">{fmtPct(r.value, total)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
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

// timing 未就绪时的空态骨架行（全「—」，不伪造数值）
const TIMING_EMPTY_LABEL_KEYS = [
  "contextview.timingWait",
  "contextview.timingGen",
  "contextview.timingTools",
  "contextview.timingOther",
] as const;
const TIMING_COLORS = {
  wait: "#3b82f6", // hex-exempt 耗时统计语义色（模型等待）
  gen: "#22c55e", // hex-exempt 耗时统计语义色（模型生成）
  reasoning: "#8b5cf6", // hex-exempt 耗时统计语义色（模型思考）
  text: "#06b6d4", // hex-exempt 耗时统计语义色（模型回复）
  tools: "#f59e0b", // hex-exempt 耗时统计语义色（工具执行）
  other: "#9ca3af", // hex-exempt 耗时统计语义色（其他）
};

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
  // 1:1 上游六行：模型等待/模型思考/模型回复/工具参数（gaea 日志无流式参数
  // 增量，诚实缺席）/工具执行/其他开销；右上输出速度（TPS）。
  const wall = timing?.wallMs ?? 0;
  const ttft = timing?.ttftMs ?? 0;
  const gen = timing?.genMs ?? 0;
  const tools = timing?.toolsMs ?? 0;
  const reasoning = timing?.reasoningMs ?? 0;
  const text = timing?.textMs ?? 0;
  // 其他 = wall − 等待 − 生成 − 工具（生成已含思考/回复段，不重复扣）
  const other = wall > 0 ? Math.max(0, wall - ttft - gen - tools) : 0;
  const rows: { labelKey: "contextview.timingWait" | "contextview.timingReasoning" | "contextview.timingText" | "contextview.timingTools" | "contextview.timingOther"; ms: number; count: number | null; blocks?: number; color: string }[] = [
    { labelKey: "contextview.timingWait", ms: ttft, count: timing?.calls ?? null, color: TIMING_COLORS.wait },
    { labelKey: "contextview.timingReasoning", ms: reasoning, count: timing?.calls ?? null, blocks: timing?.reasoningBlocks, color: TIMING_COLORS.reasoning },
    { labelKey: "contextview.timingText", ms: text, count: timing?.calls ?? null, blocks: timing?.textBlocks, color: TIMING_COLORS.text },
    { labelKey: "contextview.timingTools", ms: tools, count: timing?.toolCalls ?? null, color: TIMING_COLORS.tools },
    { labelKey: "contextview.timingOther", ms: other, count: null, color: TIMING_COLORS.other },
  ];
  const denom = wall > 0 ? wall : rows.reduce((s2, r) => s2 + r.ms, 0);
  const tps = timing?.outputTokens != null && gen > 0 ? Math.round(timing.outputTokens / (gen / 1000)) : 0;
  // 时间轴条带（1:1 上游「耗时时间轴」：各段按占比横排，刻度 0/四分位/全程）
  const segments = rows.filter((r) => r.ms > 0 && denom > 0);
  return (
    <div className="ctx-card p-3">
      <CardHead
        title={t("contextview.timingTitle")}
        sub={t("contextview.timingHint")}
        icon={Clock}
        extra={tps > 0 ? <span className="font-mono text-[10.5px] text-fg-dim">{t("contextview.timingTps", { n: tps })}</span> : undefined}
      />
      {timing ? (
        <>
          <div className="mt-2 flex items-center gap-4">
            <Donut
              segments={rows.map((r) => ({ value: r.ms, color: r.color }))}
              center={denom > 0 ? fmtDuration(denom, t) : "—"}
              centerSub={t("contextview.timingActive")}
            />
            <ul className="m-0 min-w-0 flex-1 list-none space-y-1 p-0 text-[10.5px]">
              {rows.map((r) => (
                <li key={r.labelKey} className="flex items-center gap-1.5">
                  <Dot color={r.color} />
                  <span className="shrink-0 text-fg-dim">{t(r.labelKey)}</span>
                  <span className="ml-auto shrink-0 font-mono tabular-nums text-fg-faint">
                    {denom > 0 ? `${((r.ms / denom) * 100).toFixed(1)}%` : "—"}
                  </span>
                  <span className="w-28 shrink-0 text-right font-mono tabular-nums text-fg">
                    {r.ms > 0 ? fmtDuration(r.ms, t) : "—"}
                    {r.count != null && r.count > 0 && <span className="ml-1 text-fg-faint">{`· ${r.count}${t("contextview.timesUnit")}`}</span>}
                    {r.blocks != null && r.blocks > 0 && <span className="ml-1 text-fg-faint">{`· ${r.blocks}${t("contextview.blocksUnit")}`}</span>}
                  </span>
                </li>
              ))}
            </ul>
          </div>
          {segments.length > 0 && (
            <div className="mt-2" data-testid="timing-strip">
              <div className="flex h-2 w-full overflow-hidden rounded-full">
                {segments.map((r) => (
                  <span key={r.labelKey} title={`${t(r.labelKey)} ${fmtDuration(r.ms, t)}`} style={{ width: `${(r.ms / denom) * 100}%`, background: r.color }} />
                ))}
              </div>
              <div className="mt-0.5 flex justify-between font-mono text-[8.5px] text-fg-faint">
                <span>0</span>
                <span>{fmtDuration(Math.round(denom / 4), t)}</span>
                <span>{fmtDuration(Math.round(denom / 2), t)}</span>
                <span>{fmtDuration(Math.round((denom * 3) / 4), t)}</span>
                <span>{fmtDuration(denom, t)}</span>
              </div>
            </div>
          )}
          {(timing?.tools?.length ?? 0) > 0 && (
            <div className="mt-2 flex flex-wrap gap-x-3 gap-y-0.5 border-t border-border-soft/60 pt-1.5 text-[10px] text-fg-faint" data-testid="timing-tools">
              {(timing?.tools ?? []).slice(0, 3).map((tool) => (
                <span key={tool.name} className="inline-flex items-center gap-1">
                  <span className="font-mono text-fg-dim">{tool.name}</span>
                  <span>{`${fmtDuration(tool.ms, t)} · ${tool.calls}${t("contextview.timesUnit")}`}</span>
                </span>
              ))}
            </div>
          )}
        </>
      ) : (
        <div className="mt-2 flex items-center gap-4">
          <Donut segments={TIMING_EMPTY_ROWS.map((r) => ({ value: 0, color: r.color }))} center="—" centerSub={t("contextview.timingActive")} />
          <ul className="m-0 min-w-0 flex-1 list-none space-y-1 p-0 text-[10.5px]">
            {TIMING_EMPTY_ROWS.map((r) => (
              <li key={r.label} className="flex items-center gap-1.5">
                <Dot color={r.color} />
                <span className="shrink-0 text-fg-dim">{r.label}</span>
                <span className="ml-auto font-mono tabular-nums text-fg-faint">—</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

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
    <div className="ctx-card p-3">
      <CardHead title={t("contextview.sessionTitle")} icon={MessageSquare} />
      <div className="mt-2 space-y-1">
        {rows.map(([k, v]) => (
          <div key={k} className="flex items-baseline justify-between gap-2 text-[11.5px]">
            <span className="shrink-0 text-fg-faint">{k}</span>
            <span className="min-w-0 truncate text-right font-mono tabular-nums text-fg" title={v}>{v}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// 2.5e 成本 hover：费率明细（每 1M tokens，供应商口径）；无费率诚实说明。
export function costHoverTitle(
  rate: { inputPer1M?: number; outputPer1M?: number; cacheHitPer1M?: number; currency?: string } | undefined,
  costEstimate: number | undefined,
  t: (k: DictKey, vars?: Record<string, string | number>) => string,
): string {
  const cur = rate?.currency === "USD" ? "$" : "¥";
  if (!rate || (!rate.inputPer1M && !rate.outputPer1M && !rate.cacheHitPer1M)) {
    return t("contextview.costNoRate");
  }
  const line = (label: string, v?: number) => (v ? `${label} ${cur}${v} / 1M tok` : "");
  const parts = [
    line(t("contextview.costRateIn"), rate.inputPer1M),
    line(t("contextview.costRateOut"), rate.outputPer1M),
    line(t("contextview.costRateCache"), rate.cacheHitPer1M),
  ].filter(Boolean);
  const total = costEstimate != null ? `${t("contextview.summaryCost")} ${cur}${costEstimate.toFixed(2)}` : "";
  return [t("contextview.costRateSource"), ...parts, total].filter(Boolean).join("\n");
}

// ─── 5. SummaryBar 底部汇总条（单行摘要 + 六分类图例） ────────────
export function SummaryBar({
  sessionName,
  used,
  window: win,
  requests,
  costEstimate,
  rate,
}: {
  sessionName: string;
  used: number;
  window: number;
  requests: ContextRequestRecord[];
  costEstimate?: number;
  /** 2.5e 成本 hover：最近一次用量上报的单价快照（每 1M tokens）。 */
  rate?: { inputPer1M?: number; outputPer1M?: number; cacheHitPer1M?: number; currency?: string };
}) {
  const t = useT();
  // 窗口未知（win=0）时百分比诚实「—」，与 hero 卡同口径。
  const pct = win > 0 ? Math.min(100, Math.round((used / win) * 100)) : null;
  return (
    <div className="ctx-card ctx-summary shrink-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="shrink-0 rounded-full bg-accent/15 px-2 py-px text-[10px] font-medium text-accent">
          {sessionName}
        </span>
        <span className="font-mono text-[10px] tabular-nums text-fg-dim">
          {fmtTokens(used)} / {fmtTokens(win)} · {pct != null ? `${pct}%` : "—"}
        </span>
        <span className="font-mono text-[10px] tabular-nums text-fg-faint">{t("contextview.summaryRequests", { n: requests.length })}</span>
        <span
          className="font-mono text-[10px] tabular-nums text-fg-faint"
          data-testid="ctx-cost-cell"
          title={costHoverTitle(rate, costEstimate, t)}
        >
          {costEstimate != null
            ? `${t("contextview.summaryCost")} ${rate?.currency === "USD" ? "$" : "¥"}${costEstimate.toFixed(2)}`
            : `${t("contextview.summaryCost")} —`}
        </span>
        <span className="min-[900px]:flex-1" />
        <span className="flex flex-wrap items-center gap-x-2.5 gap-y-0.5">
          {LEGEND_CATS.map((c) => (
            <span key={c.key} className="inline-flex items-center gap-1 text-[9.5px] text-fg-faint">
              <Dot color={c.color} />
              {t(c.labelKey)}
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
