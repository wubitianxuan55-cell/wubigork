// 跨会话洞察页（v4.542 刀3，对标 dsh-context Context Insights）：KPI 栏、
// 近 7 天成对柱图、8 周活动热力图、技能/工具/命令三统计、构成环形图、
// 可筛选的会话卡片。数据源 GaeaContextInsights（Go 侧跨会话折叠聚合，
// size+mtime 缓存）。图表全部自绘 SVG（仓内无图表库依赖纪律）。
// 诚实纪律：估算 tokens 分列不混入计费；范围内无活动显示空态；
// truncated 注明未扫描数。
import { useCallback, useEffect, useMemo, useState } from "react";
import { app } from "../../lib/bridge";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import { relativeTime } from "../../lib/time";
import type {
  ContextCategory,
  ContextInsightsView,
  InsightsDay,
  InsightsRangeKey,
  InsightsSession,
  InsightsTally,
} from "../../lib/types/context";
import { loadContextPrefs, saveContextPref } from "../../lib/contextPrefs";
import { CAT_COLORS, CONTEXT_CATS } from "./cats";
import { Donut } from "./cards";

const RANGES: { key: InsightsRangeKey; labelKey: "contextinsights.rangeToday" | "contextinsights.range7d" | "contextinsights.range30d" | "contextinsights.rangeAll" }[] = [
  { key: "today", labelKey: "contextinsights.rangeToday" },
  { key: "7d", labelKey: "contextinsights.range7d" },
  { key: "30d", labelKey: "contextinsights.range30d" },
  { key: "all", labelKey: "contextinsights.rangeAll" },
];

// fmtCost 两位小数成本（$ 前缀与后端 CostRate.currency 口径无关——洞察页
// 只给数量级，币种以会话费率快照为准）。
function fmtCost(n: number | undefined): string {
  if (!n || n <= 0) return "0";
  if (n < 0.01) return `<0.01`;
  return n.toFixed(2);
}

// compactCount 千分位（次数类 KPI）。
function fmtCount(n: number): string {
  return n.toLocaleString("en-US");
}

export function Insights({ running, onOpenSession }: {
  running?: boolean;
  onOpenSession?: (path: string) => void;
}) {
  const t = useT();
  // 默认范围读偏好（刀6；点击范围预设写回——dsh 同款语义）。
  const [range, setRange] = useState<InsightsRangeKey>(() => loadContextPrefs().insightsRange);
  const [includeArchived, setIncludeArchived] = useState(false);
  const [data, setData] = useState<ContextInsightsView | null>(null);
  const [failed, setFailed] = useState(false);
  // dayFilter：热力图点击某日 → 会话列表只看该日活跃（KPI 不动，dsh 同语义
  // 的轻量版；再次点击清除）。
  const [dayFilter, setDayFilter] = useState<string | null>(null);
  // tallyFilter（刀8）：统计行 chip 点击 → 会话列表只看用过该技能/工具/程序
  // 的会话（依据后端 InsightsSession.skills/tools/programs 名称清单）。
  const [tallyFilter, setTallyFilter] = useState<{ kind: "skills" | "tools" | "programs"; name: string } | null>(null);

  const load = useCallback(() => {
    app.ContextInsights(range, includeArchived)
      .then((d) => {
        setData(d);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, [range, includeArchived]);

  useEffect(() => {
    load();
  }, [load]);

  // 范围/归档切换重置两项过滤（过滤语义跟着范围走）。
  useEffect(() => {
    setDayFilter(null);
    setTallyFilter(null);
  }, [range, includeArchived]);

  // 运行中轮询刷新（洞察页是跨会话视图，事件流节流无意义——定时轻轮询，
  // 后端有 mtime 缓存兜底）。
  useEffect(() => {
    if (!running) return;
    const iv = window.setInterval(load, 4000);
    return () => window.clearInterval(iv);
  }, [running, load]);

  const sessions = useMemo(() => {
    if (!data) return [];
    let list = data.sessions;
    if (dayFilter) list = list.filter((s) => dayKeyOf(s.lastActive) === dayFilter);
    if (tallyFilter) {
      list = list.filter((s) => (s[tallyFilter.kind] ?? []).includes(tallyFilter.name));
    }
    return list;
  }, [data, dayFilter, tallyFilter]);

  const composition = useMemo(() => {
    const acc: ContextCategory = { system: 0, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 };
    for (const s of sessions) {
      acc.system += s.composition.system;
      acc.tools += s.composition.tools;
      acc.user += s.composition.user;
      acc.skill += s.composition.skill;
      acc.inject += s.composition.inject;
      acc.assistant += s.composition.assistant;
      acc.tool += s.composition.tool;
    }
    return acc;
  }, [sessions]);

  const donutSegments = useMemo(
    () => CONTEXT_CATS.map((c) => ({ value: composition[c.key], color: CAT_COLORS[c.key], key: c.key })),
    [composition],
  );
  const compTotal = donutSegments.reduce((s, x) => s + x.value, 0);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4" data-testid="ctx-insights">
      {/* 标题 + 范围切换 */}
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="m-0 text-[15px] font-semibold text-fg">{t("contextinsights.title")}</h2>
        <span className="text-[11.5px] text-fg-dim">{t("contextinsights.subtitle")}</span>
        <div className="ml-auto flex items-center gap-1" role="group" aria-label={t("contextinsights.title")}>
          {RANGES.map((r) => (
            <button
              key={r.key}
              type="button"
              onClick={() => {
                setRange(r.key);
                saveContextPref("insightsRange", r.key);
              }}
              aria-pressed={range === r.key}
              data-testid={`insights-range-${r.key}`}
              className={`rounded border-0 px-1.5 py-0.5 text-[11.5px] cursor-pointer ${
                range === r.key ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"
              }`}
            >
              {t(r.labelKey)}
            </button>
          ))}
          <button
            type="button"
            onClick={() => setIncludeArchived((v) => !v)}
            aria-pressed={includeArchived}
            data-testid="insights-archived"
            className={`ml-1 rounded border-0 px-1.5 py-0.5 text-[11.5px] cursor-pointer ${
              includeArchived ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"
            }`}
          >
            {t("contextinsights.includeArchived")}
          </button>
        </div>
      </div>

      {failed && <div className="ctx-card p-3 text-[12px] text-err">{t("contextinsights.loadFailed")}</div>}

      {data && (
        <>
          {/* KPI 栏 */}
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-6" data-testid="insights-kpi">
            <Kpi label={t("contextinsights.kpiSessions")} value={fmtCount(data.kpi.sessions)} />
            <Kpi label={t("contextinsights.kpiTokens")} value={fmtTokens(data.kpi.tokens)} sub={data.kpi.estimatedTokens ? `${fmtTokens(data.kpi.estimatedTokens)} ${t("contextinsights.kpiEstimated")}` : undefined} />
            <Kpi label={t("contextinsights.kpiCost")} value={fmtCost(data.kpi.cost)} />
            <Kpi label={t("contextinsights.kpiCacheHit")} value={data.kpi.cacheHitPercent ? `${data.kpi.cacheHitPercent.toFixed(1)}%` : "—"} />
            <Kpi label={t("contextinsights.kpiTurns")} value={fmtCount(data.kpi.turns)} />
            <Kpi label={t("contextinsights.kpiToolCalls")} value={fmtCount(data.kpi.toolCalls)} sub={data.kpi.skills ? `${fmtCount(data.kpi.skills)} ${t("contextinsights.tallySkills")}` : undefined} />
          </div>

          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {/* 近 7 天成对柱图 */}
            <section className="ctx-card p-3" data-testid="insights-daily">
              <SectionTitle>{t("contextinsights.daily7")}</SectionTitle>
              <DailyBars days={last7Days(data.days, data.generatedAt)} />
            </section>

            {/* 8 周热力图 */}
            <section className="ctx-card p-3" data-testid="insights-heatmap">
              <SectionTitle>{t("contextinsights.heatmap")}</SectionTitle>
              <Heatmap days={data.days} generatedAt={data.generatedAt} onPick={(d) => setDayFilter((cur) => (cur === d ? null : d))} picked={dayFilter} />
            </section>
          </div>

          {/* 三统计行 + 构成环形 */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <section className="ctx-card p-3">
              <TallyRow kind="skills" title={t("contextinsights.tallySkills")} items={data.skills} color="var(--md-sys-color-tertiary, var(--color-accent))" onPick={setTallyFilter} picked={tallyFilter} />
              <TallyRow kind="tools" title={t("contextinsights.tallyTools")} items={data.tools} color="var(--md-sys-color-primary, var(--color-primary))" onPick={setTallyFilter} picked={tallyFilter} />
              <TallyRow kind="programs" title={t("contextinsights.tallyPrograms")} items={data.programs} color="var(--md-sys-color-secondary, var(--color-secondary))" onPick={setTallyFilter} picked={tallyFilter} />
            </section>
            <section className="ctx-card flex items-center gap-4 p-3" data-testid="insights-composition">
              <Donut
                segments={donutSegments}
                center={compTotal > 0 ? fmtTokens(compTotal) : "0"}
                centerSub={t("contextinsights.composition")}
              />
              <ul className="m-0 flex min-w-0 flex-1 list-none flex-col gap-1 p-0 text-[11.5px]">
                {CONTEXT_CATS.map((c) => (
                  <li key={c.key} className="flex items-center gap-1.5">
                    <span aria-hidden className="inline-block size-2 rounded-full" style={{ background: CAT_COLORS[c.key] }} />
                    <span className="text-fg-dim">{t(c.labelKey)}</span>
                    <span className="ml-auto font-mono text-fg">{compTotal > 0 ? `${Math.round((composition[c.key] / compTotal) * 100)}%` : "0%"}</span>
                  </li>
                ))}
              </ul>
            </section>
          </div>

          {/* 会话卡片 */}
          <section className="ctx-card p-3">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <SectionTitle>
                {t("contextinsights.sessions")} · {fmtCount(sessions.length)}
              </SectionTitle>
              {dayFilter && (
                <button
                  type="button"
                  onClick={() => setDayFilter(null)}
                  data-testid="insights-dayfilter"
                  className="rounded border-0 bg-accent/15 px-1.5 py-0.5 text-[11px] text-accent cursor-pointer"
                >
                  {t("contextinsights.dayFilter", { date: dayFilter })}
                </button>
              )}
              {tallyFilter && (
                <button
                  type="button"
                  onClick={() => setTallyFilter(null)}
                  data-testid="insights-tallyfilter"
                  className="rounded border-0 bg-accent/15 px-1.5 py-0.5 text-[11px] text-accent cursor-pointer"
                >
                  {t("contextinsights.tallyFilter", { name: tallyFilter.name })}
                </button>
              )}
              {!!data.truncated && (
                <span className="ml-auto text-[11px] text-fg-dim">{t("contextinsights.truncated", { count: data.truncated })}</span>
              )}
            </div>
            {sessions.length === 0 ? (
              <p className="m-0 py-6 text-center text-[12px] text-fg-dim">{t("contextinsights.sessionsEmpty")}</p>
            ) : (
              <ul className="m-0 grid list-none grid-cols-1 gap-2 p-0 md:grid-cols-2" data-testid="insights-sessions">
                {sessions.map((s) => (
                  <SessionCard key={s.path} s={s} onOpen={onOpenSession} />
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function Kpi({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-md border border-border-soft bg-panel-2 px-2 py-1.5" title={sub}>
      <div className="text-[10.5px] text-fg-dim">{label}</div>
      <div className="font-mono text-[15px] font-semibold text-fg" data-testid="insights-kpi-value">{value}</div>
      {sub && <div className="truncate text-[9.5px] text-fg-dim">{sub}</div>}
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="m-0 mb-1.5 text-[12.5px] font-medium text-fg">{children}</h3>;
}

// last7Days 取含 generatedAt 在内的最近 7 个自然日（不足补零），按日升序。
function last7Days(days: InsightsDay[], now: number): InsightsDay[] {
  const byDate = new Map(days.map((d) => [d.date, d]));
  const out: InsightsDay[] = [];
  const base = new Date(now * 1000);
  base.setHours(0, 0, 0, 0);
  for (let i = 6; i >= 0; i--) {
    const d = new Date(base);
    d.setDate(d.getDate() - i);
    const key = d.toISOString().slice(0, 10) === d.toLocaleDateString("sv") // sv-SE 恒 YYYY-MM-DD
      ? d.toLocaleDateString("sv")
      : localDateKey(d);
    out.push(byDate.get(key) ?? { date: key, tokens: 0, requests: 0 });
  }
  return out;
}

// localDateKey 本地时区 YYYY-MM-DD（与 Go dayKey 同口径）。
function localDateKey(d: Date): string {
  const m = `${d.getMonth() + 1}`.padStart(2, "0");
  const day = `${d.getDate()}`.padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

function dayKeyOf(ts: number): string {
  return localDateKey(new Date(ts * 1000));
}

// DailyBars 近 7 天成对柱（token 主色 / 成本次色，各自独立比例——量纲不同
// 不共轴，dsh 同为成对柱）。悬停 title 给两读数。
function DailyBars({ days }: { days: InsightsDay[] }) {
  const maxTokens = Math.max(1, ...days.map((d) => d.tokens));
  const maxCost = Math.max(0.01, ...days.map((d) => d.cost ?? 0));
  const W = 280;
  const H = 88;
  const bw = W / days.length;
  return (
    <svg width="100%" viewBox={`0 0 ${W} ${H + 14}`} role="img" aria-label="daily usage bars">
      {days.map((d, i) => {
        const th = Math.round((d.tokens / maxTokens) * (H - 8));
        const ch = Math.round(((d.cost ?? 0) / maxCost) * (H - 8));
        const x = i * bw;
        const label = d.date.slice(5); // MM-DD
        return (
          <g key={d.date}>
            <title>{`${d.date} · ${fmtTokens(d.tokens)} tok · ${fmtCost(d.cost)}`}</title>
            <rect x={x + bw * 0.18} y={H - th} width={bw * 0.3} height={th} rx={1.5} fill="var(--md-sys-color-primary, var(--color-primary))" opacity={0.85} />
            <rect x={x + bw * 0.52} y={H - ch} width={bw * 0.3} height={ch} rx={1.5} fill="var(--md-sys-color-tertiary, var(--color-accent))" opacity={0.7} />
            <text x={x + bw / 2} y={H + 11} textAnchor="middle" fontSize={8.5} className="fill-current text-fg-dim">
              {i % 2 === 0 || days.length <= 4 ? label : ""}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

// Heatmap 8 周 × 7 天网格（GitHub 风格，列=周）。颜色四档按当日 tokens
// 相对峰值。点击某日 → 过滤会话列表（onPick）。
function Heatmap({ days, generatedAt, onPick, picked }: {
  days: InsightsDay[];
  generatedAt: number;
  onPick: (date: string | null) => void;
  picked: string | null;
}) {
  const byDate = new Map(days.map((d) => [d.date, d]));
  const max = Math.max(1, ...days.map((d) => d.tokens));
  // 网格：结束于 generatedAt 所在周的周六，向前 8 周（56 格）。
  const end = new Date(generatedAt * 1000);
  end.setHours(0, 0, 0, 0);
  end.setDate(end.getDate() + (6 - end.getDay())); // 本周六
  const cells: { date: string | null; day?: InsightsDay }[] = [];
  for (let i = 55; i >= 0; i--) {
    const d = new Date(end);
    d.setDate(d.getDate() - i);
    const key = localDateKey(d);
    cells.push({ date: key, day: byDate.get(key) });
  }
  const weeks: { date: string | null; day?: InsightsDay }[][] = [];
  for (let w = 0; w < 8; w++) weeks.push(cells.slice(w * 7, w * 7 + 7));
  const level = (tokens: number): number => {
    if (tokens <= 0) return 0;
    const f = tokens / max;
    if (f > 0.66) return 4;
    if (f > 0.33) return 3;
    if (f > 0.08) return 2;
    return 1;
  };
  const cellColor = (lv: number): string =>
    [
      "var(--md-sys-color-surface-variant, rgba(128,128,128,0.15))",
      "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 25%, transparent)",
      "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 50%, transparent)",
      "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 75%, transparent)",
      "var(--md-sys-color-primary, var(--color-primary))",
    ][lv];
  const todayKey = localDateKey(new Date(generatedAt * 1000));
  return (
    <div className="flex gap-[3px]" data-testid="insights-heatmap-grid">
      {weeks.map((week, wi) => (
        <div key={wi} className="flex flex-col gap-[3px]">
          {week.map((cell) => {
            const lv = cell.day ? level(cell.day.tokens) : 0;
            const isPicked = cell.date && cell.date === picked;
            return (
              <button
                key={cell.date ?? wi}
                type="button"
                disabled={!cell.day}
                onClick={() => onPick(cell.date && cell.day ? cell.date : null)}
                title={cell.day ? `${cell.date} · ${fmtTokens(cell.day.tokens)} tok · ${cell.day.requests} req` : cell.date ?? ""}
                aria-label={cell.date ?? ""}
                data-picked={isPicked || undefined}
                className={`size-[11px] rounded-[2px] border-0 p-0 ${cell.day ? "cursor-pointer" : "cursor-default"} ${isPicked ? "outline outline-1 outline-offset-0" : ""}`}
                style={{
                  background: cellColor(lv),
                  outlineColor: "var(--md-sys-color-error, var(--color-destructive))",
                  ...(cell.date === todayKey ? { boxShadow: "inset 0 0 0 1px rgba(128,128,128,0.6)" } : {}),
                }}
              />
            );
          })}
        </div>
      ))}
    </div>
  );
}

// TallyRow 一条统计行（标题 + chip 流：名称 ×次数，悬停给最近时间）。
// 刀8：chip 可点击 → 过滤会话卡（依据 InsightsSession 名称清单；再点清除，
// 活动项高亮）。
function TallyRow({ kind, title, items, color, onPick, picked }: {
  kind: "skills" | "tools" | "programs";
  title: string;
  items: InsightsTally[];
  color: string;
  onPick?: (f: { kind: "skills" | "tools" | "programs"; name: string } | null) => void;
  picked?: { kind: "skills" | "tools" | "programs"; name: string } | null;
}) {
  const t = useT();
  return (
    <div className="mb-2 last:mb-0" data-testid="insights-tally">
      <div className="mb-1 flex items-baseline gap-2">
        <span className="text-[12px] font-medium text-fg">{title}</span>
        <span className="text-[10.5px] text-fg-dim">{items.length ? fmtCount(items.reduce((s, x) => s + x.count, 0)) : ""}</span>
      </div>
      {items.length === 0 ? (
        <p className="m-0 text-[11px] text-fg-dim">{t("contextinsights.tallyEmpty")}</p>
      ) : (
        <div className="flex flex-wrap gap-1">
          {items.map((x) => {
            const active = picked?.kind === kind && picked.name === x.name;
            const tip = x.last
              ? t("contextinsights.heatmapTip", { date: x.name, tokens: fmtCount(x.count), requests: relativeTime(x.last * 1000) })
              : x.name;
            if (!onPick) {
              return (
                <span
                  key={x.name}
                  title={tip}
                  className="inline-flex max-w-full items-center gap-1 truncate rounded-full px-1.5 py-0.5 text-[10.5px]"
                  style={{ background: `color-mix(in srgb, ${color} 12%, transparent)`, color: "var(--md-sys-color-on-surface, var(--color-fg))" }}
                >
                  <span className="max-w-[160px] truncate font-mono">{x.name}</span>
                  <span className="font-mono text-fg-dim">×{x.count}</span>
                </span>
              );
            }
            return (
              <button
                key={x.name}
                type="button"
                data-picked={active || undefined}
                title={tip}
                data-testid={`insights-tally-${kind}-${x.name}`}
                className={`inline-flex max-w-full cursor-pointer items-center gap-1 truncate rounded-full border-0 px-1.5 py-0.5 text-[10.5px] ${active ? "outline outline-1" : ""}`}
                style={{
                  background: `color-mix(in srgb, ${color} ${active ? 26 : 12}%, transparent)`,
                  color: "var(--md-sys-color-on-surface, var(--color-fg))",
                  outlineColor: color,
                }}
                onClick={() => onPick(active ? null : { kind, name: x.name })}
              >
                <span className="max-w-[160px] truncate font-mono">{x.name}</span>
                <span className="font-mono text-fg-dim">×{x.count}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// SessionCard 会话卡片：标题/预览 + 计费 token + 成本 + 轮次请求 + 构成
// 迷你堆叠条 + 徽标（估算/归档/空间）。点击回调打开会话（无回调则纯展示）。
function SessionCard({ s, onOpen }: { s: InsightsSession; onOpen?: (path: string) => void }) {
  const t = useT();
  const total = CONTEXT_CATS.reduce((acc, c) => acc + s.composition[c.key], 0);
  const clickable = !!onOpen;
  return (
    <li>
      <button
        type="button"
        onClick={clickable ? () => onOpen?.(s.path) : undefined}
        disabled={!clickable}
        data-testid="insights-session-card"
        className={`w-full rounded-md border border-border-soft bg-panel-2 p-2 text-left ${clickable ? "cursor-pointer hover:border-accent/40" : "cursor-default"}`}
        title={s.path}
      >
        <div className="flex items-center gap-1.5">
          <span className="min-w-0 flex-1 truncate text-[12px] font-medium text-fg">
            {s.title || s.preview || s.path.split(/[\\/]/).pop() || s.path}
          </span>
          {s.estimated && <Badge>{t("contextinsights.badgeEstimated")}</Badge>}
          {s.archived && <Badge>{t("contextinsights.badgeArchived")}</Badge>}
          {s.spaceId && <Badge>{s.spaceId}</Badge>}
        </div>
        {!!s.preview && !s.title && (
          <div className="mt-0.5 truncate text-[10.5px] text-fg-dim">{s.preview}</div>
        )}
        <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[10.5px] text-fg-dim">
          <span className="text-fg">{fmtTokens(s.tokens)}</span>
          {(s.cost ?? 0) > 0 && <span>{t("contextinsights.sessionCost", { cost: fmtCost(s.cost) })}</span>}
          <span>{t("contextinsights.sessionMeta", { turns: s.turns, requests: s.requests, ago: relativeTime(s.lastActive * 1000) })}</span>
          {s.cacheHitPercent ? <span>{t("contextinsights.kpiCacheHit")} {s.cacheHitPercent.toFixed(1)}%</span> : null}
        </div>
        {/* 构成迷你堆叠条（同 CAT_COLORS 单源色板） */}
        {total > 0 && (
          <div className="mt-1.5 flex h-1.5 w-full overflow-hidden rounded-full" aria-hidden>
            {CONTEXT_CATS.map((c) => {
              const v = s.composition[c.key];
              if (v <= 0) return null;
              return <span key={c.key} style={{ width: `${(v / total) * 100}%`, background: CAT_COLORS[c.key] }} />;
            })}
          </div>
        )}
      </button>
    </li>
  );
}

function Badge({ children }: { children: React.ReactNode }) {
  return (
    <span className="shrink-0 rounded-full bg-panel px-1.5 py-px text-[9.5px] text-fg-dim">{children}</span>
  );
}
