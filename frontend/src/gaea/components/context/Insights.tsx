// 跨会话洞察页（v4.542，对标 dsh-context Context Insights 1:1）：标题行
// （自定义日期范围+预设）、KPI 2×3（副行）+近 7 天用量双轴图、Token 统计
// （环形+三列 8 行）、耗时统计（环形+行+输出速度）、活跃度热力图（行标+
// 少→多图例+会话|Token toggle）、技能/工具/命令三统计（列表行+头部计数+
// 搜索）、会话区（搜索+排序+工作区分组+环形构成会话卡）。数据源
// GaeaContextInsights。图表全部自绘 SVG。诚实纪律：估算 tokens 分列不混入
// 计费；truncated 注明未扫描数。
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
import { CAT_COLORS, CONTEXT_CATS } from "./cats";
import { Donut, fmtDuration } from "./cards";

const RANGES: { key: InsightsRangeKey; labelKey: "contextinsights.rangeToday" | "contextinsights.range7d" | "contextinsights.range30d" | "contextinsights.rangeAll" }[] = [
  { key: "today", labelKey: "contextinsights.rangeToday" },
  { key: "7d", labelKey: "contextinsights.range7d" },
  { key: "30d", labelKey: "contextinsights.range30d" },
  { key: "all", labelKey: "contextinsights.rangeAll" },
];

type SessionSort = "recent" | "tokens" | "context";

function fmtCost(n: number | undefined): string {
  if (!n || n <= 0) return "0";
  if (n < 0.01) return `<0.01`;
  return n.toFixed(2);
}

function fmtCount(n: number): string {
  return n.toLocaleString("en-US");
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

function catSum(c: ContextCategory): number {
  return c.system + c.tools + c.user + c.skill + c.inject + c.assistant + c.tool;
}

const TIMING_COLORS: Record<string, string> = {
  "contextview.timingWait": "#3b82f6", // hex-exempt 耗时统计语义色（模型等待）
  "contextview.timingGen": "#22c55e", // hex-exempt 耗时统计语义色（模型生成）
  "contextview.timingTools": "#f59e0b", // hex-exempt 耗时统计语义色（工具执行）
  "contextview.timingOther": "#9ca3af", // hex-exempt 耗时统计语义色（其他）
};

const HEAT_LEVELS = [
  "var(--md-sys-color-surface-variant, rgba(128,128,128,0.15))",
  "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 25%, transparent)",
  "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 50%, transparent)",
  "color-mix(in srgb, var(--md-sys-color-primary, var(--color-primary)) 75%, transparent)",
  "var(--md-sys-color-primary, var(--color-primary))",
];

export function Insights({ running, onOpenSession }: {
  running?: boolean;
  onOpenSession?: (path: string) => void;
}) {
  const t = useT();
  const [range, setRange] = useState<InsightsRangeKey>("30d");
  const [customFrom, setCustomFrom] = useState("");
  const [customTo, setCustomTo] = useState("");
  const [includeArchived, setIncludeArchived] = useState(false);
  const [data, setData] = useState<ContextInsightsView | null>(null);
  const [failed, setFailed] = useState(false);
  const [dayFilter, setDayFilter] = useState<string | null>(null);
  const [tallyFilter, setTallyFilter] = useState<{ kind: "skills" | "tools" | "programs"; name: string } | null>(null);
  // 会话区控件（1:1：搜索/排序/工作区分组）
  const [q, setQ] = useState("");
  const [sort, setSort] = useState<SessionSort>("recent");
  const [group, setGroup] = useState<string>("all");
  const [heatMode, setHeatMode] = useState<"sessions" | "tokens">("tokens");

  const effectiveRange = customFrom && customTo ? `custom:${customFrom}:${customTo}` : range;

  const load = useCallback(() => {
    app.ContextInsights(effectiveRange, includeArchived)
      .then((d) => {
        setData(d);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, [effectiveRange, includeArchived]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    setDayFilter(null);
    setTallyFilter(null);
  }, [effectiveRange, includeArchived]);

  useEffect(() => {
    if (!running) return;
    const iv = window.setInterval(load, 4000);
    return () => window.clearInterval(iv);
  }, [running, load]);

  const sessions = useMemo(() => {
    if (!data) return [];
    let list = data.sessions;
    if (dayFilter) list = list.filter((s) => dayKeyOf(s.lastActive) === dayFilter);
    if (tallyFilter) list = list.filter((s) => (s[tallyFilter.kind] ?? []).includes(tallyFilter.name));
    const query = q.trim().toLowerCase();
    if (query) list = list.filter((s) => `${s.title ?? ""} ${s.preview ?? ""} ${s.path}`.toLowerCase().includes(query));
    if (group !== "all") list = list.filter((s) => (s.spaceId || "work") === group);
    if (sort === "recent") list = [...list].sort((a, b) => b.lastActive - a.lastActive);
    else if (sort === "tokens") list = [...list].sort((a, b) => b.tokens - a.tokens);
    else list = [...list].sort((a, b) => catSum(b.composition) - catSum(a.composition));
    return list;
  }, [data, dayFilter, tallyFilter, q, sort, group]);

  const composition = useMemo(() => {
    const acc: ContextCategory = { system: 0, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 };
    for (const s of data?.sessions ?? []) {
      acc.system += s.composition.system;
      acc.tools += s.composition.tools;
      acc.user += s.composition.user;
      acc.skill += s.composition.skill;
      acc.inject += s.composition.inject;
      acc.assistant += s.composition.assistant;
      acc.tool += s.composition.tool;
    }
    return acc;
  }, [data]);

  // Token 统计卡 8 行（1:1 上游顺序；工具调用=结果+调用参数、思考/回复三拆）
  const k = data?.kpi;
  const tokenRows = useMemo(
    () => [
      { key: "system", labelKey: "contextview.catSystem" as const, value: composition.system, color: CAT_COLORS.system },
      { key: "tools", labelKey: "contextview.catTools" as const, value: composition.tools, color: CAT_COLORS.tools },
      { key: "user", labelKey: "contextview.catUser" as const, value: composition.user, color: CAT_COLORS.user },
      { key: "inject", labelKey: "contextview.catInject" as const, value: composition.inject, color: CAT_COLORS.inject },
      { key: "skill", labelKey: "contextview.catSkill" as const, value: composition.skill, color: CAT_COLORS.skill },
      { key: "tool", labelKey: "contextview.statToolCallsRow" as const, value: composition.tool + (k?.assistantCallArgs ?? 0), color: CAT_COLORS.tool },
      { key: "reasoning", labelKey: "contextview.statThinking" as const, value: k?.assistantReasoning ?? 0, color: "#6366f1" }, // hex-exempt Token 统计语义色
      { key: "answers", labelKey: "contextview.statAnswers" as const, value: k?.assistantAnswers ?? 0, color: "#ec4899" }, // hex-exempt Token 统计语义色
    ],
    [composition, k],
  );
  const tokenTotal = tokenRows.reduce((s, r) => s + r.value, 0);

  // 耗时统计行（模型等待/模型生成/工具执行/其他）
  const wall = k?.wallMs ?? 0;
  const ttft = k?.ttftMs ?? 0;
  const gen = k?.genMs ?? 0;
  const toolsMs = k?.toolsMs ?? 0;
  const other = Math.max(0, wall - ttft - gen - toolsMs);
  const timingRows = [
    { key: "contextview.timingWait" as const, ms: ttft, count: k?.calls ?? 0 },
    { key: "contextview.timingGen" as const, ms: gen, count: k?.calls ?? 0 },
    { key: "contextview.timingTools" as const, ms: toolsMs, count: k?.toolCalls ?? 0 },
    { key: "contextview.timingOther" as const, ms: other, count: 0 },
  ];
  const tps = wall > 0 && (k?.outputTokens ?? 0) > 0 ? Math.round(k!.outputTokens! / (wall / 1000)) : 0;

  const groupChips = useMemo(() => {
    const counts = new Map<string, number>();
    for (const s of data?.sessions ?? []) {
      const g = s.spaceId || "work";
      counts.set(g, (counts.get(g) ?? 0) + 1);
    }
    return [...counts.entries()];
  }, [data]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4" data-testid="ctx-insights">
      {/* 标题行：标题 + 自定义日期 + 预设 + 含归档（1:1 上游头行） */}
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="m-0 text-[15px] font-semibold text-fg">{t("contextinsights.title")}</h2>
        <span className="text-[11.5px] text-fg-dim">{t("contextinsights.subtitle")}</span>
        <div className="ml-auto flex flex-wrap items-center gap-1.5">
          <input
            type="date"
            value={customFrom}
            onChange={(e) => setCustomFrom(e.target.value)}
            data-testid="insights-from"
            className="rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10.5px] text-fg"
            aria-label={t("contextinsights.from")}
          />
          <span className="text-fg-faint">→</span>
          <input
            type="date"
            value={customTo}
            onChange={(e) => setCustomTo(e.target.value)}
            data-testid="insights-to"
            className="rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10.5px] text-fg"
            aria-label={t("contextinsights.to")}
          />
          {(customFrom || customTo) && (
            <button type="button" className="cursor-pointer border-0 bg-transparent text-[10.5px] text-fg-dim hover:text-fg" onClick={() => { setCustomFrom(""); setCustomTo(""); }}>
              ✕
            </button>
          )}
          {RANGES.map((r) => (
            <button
              key={r.key}
              type="button"
              onClick={() => { setRange(r.key); saveRange(r.key); }}
              aria-pressed={!customFrom && range === r.key}
              data-testid={`insights-range-${r.key}`}
              className={`rounded border-0 px-1.5 py-0.5 text-[11.5px] cursor-pointer ${!customFrom && range === r.key ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            >
              {t(r.labelKey)}
            </button>
          ))}
          <button
            type="button"
            onClick={() => setIncludeArchived((v) => !v)}
            aria-pressed={includeArchived}
            data-testid="insights-archived"
            className={`rounded border-0 px-1.5 py-0.5 text-[11.5px] cursor-pointer ${includeArchived ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
          >
            {t("contextinsights.includeArchived")}
          </button>
        </div>
      </div>

      {failed && <div className="ctx-card p-3 text-[12px] text-err">{t("contextinsights.loadFailed")}</div>}

      {data && (
        <>
          {/* 行1：KPI 2×3 + 近 7 天用量（1:1 上游布局） */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-[2fr_3fr]">
            <div className="grid grid-cols-2 gap-2" data-testid="insights-kpi">
              <Kpi label={t("contextinsights.kpiSessions")} value={fmtCount(data.kpi.sessions)} sub={t("contextinsights.kpiArchivedSub", { n: data.kpi.archived ?? 0 })} />
              <Kpi label={t("contextinsights.kpiTokens")} value={fmtTokens(data.kpi.tokens)} sub={t("contextinsights.kpiTurnsSub", { n: data.kpi.turns })} />
              <Kpi label={t("contextinsights.kpiCost")} value={fmtCost(data.kpi.cost)} sub={t("contextinsights.kpiBilledSub", { n: data.kpi.billed ?? 0, total: data.kpi.sessions })} />
              <Kpi label={t("contextinsights.kpiCacheHit")} value={data.kpi.cacheHitPercent ? `${data.kpi.cacheHitPercent.toFixed(2)}%` : "—"} sub={t("contextinsights.kpiCacheSub", { n: data.kpi.sessions })} />
              <Kpi label={t("contextinsights.kpiToolCalls")} value={fmtCount(data.kpi.toolCalls)} sub={toolsMs > 0 ? t("contextinsights.kpiToolMsSub", { ms: fmtDuration(toolsMs, t) }) : undefined} />
              <Kpi label={t("contextinsights.kpiWall")} value={wall > 0 ? fmtDuration(wall, t) : "—"} sub={t("contextinsights.kpiCallsSub", { n: data.kpi.calls ?? 0 })} />
            </div>
            <section className="ctx-card p-3" data-testid="insights-daily">
              <SectionTitle>{t("contextinsights.daily7")}</SectionTitle>
              <DailyChart days={last7Days(data.days, data.generatedAt)} />
            </section>
          </div>

          {/* 行2：Token 统计 + 耗时统计（1:1 双卡） */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <section className="ctx-card p-3" data-testid="insights-tokenstats">
              <CardHeadRow title={t("contextinsights.tokenStats")} extra={data.kpi.cacheHitPercent ? `${t("contextinsights.kpiCacheHit")} ${data.kpi.cacheHitPercent.toFixed(2)}%` : undefined} />
              <div className="mt-2 flex items-center gap-4">
                <Donut
                  segments={tokenRows.map((r) => ({ value: r.value, color: r.color }))}
                  center={fmtTokens(tokenTotal)}
                  centerSub={t("contextinsights.totalUsage")}
                />
                <ul className="m-0 grid min-w-0 flex-1 list-none grid-cols-2 gap-x-3 gap-y-1 p-0 text-[11px]">
                  {tokenRows.map((r) => (
                    <TokenRow key={r.key} label={t(r.labelKey)} value={r.value} total={tokenTotal} color={r.color} />
                  ))}
                </ul>
              </div>
            </section>
            <section className="ctx-card p-3" data-testid="insights-timingstats">
              <CardHeadRow title={t("contextinsights.timingStats")} extra={tps > 0 ? t("contextinsights.tps", { n: tps }) : undefined} />
              <div className="mt-2 flex items-center gap-4">
                <Donut
                  segments={timingRows.map((r) => ({ value: r.ms, color: TIMING_COLORS[r.key] }))}
                  center={wall > 0 ? fmtDuration(wall, t) : "—"}
                  centerSub={t("contextinsights.activeTime")}
                />
                <ul className="m-0 grid min-w-0 flex-1 list-none grid-cols-2 gap-x-3 gap-y-1 p-0 text-[11px]">
                  {timingRows.map((r) => (
                    <li key={r.key} className="flex items-center gap-1.5">
                      <Dot color={TIMING_COLORS[r.key]} />
                      <span className="shrink-0 text-fg-dim">{t(r.key)}</span>
                      <span className="ml-auto text-right font-mono tabular-nums text-fg">
                        {r.ms > 0 ? fmtDuration(r.ms, t) : "—"}
                        {r.count > 0 && <span className="ml-1 text-fg-faint">· {fmtCount(r.count)}{t("contextinsights.timesUnit")}</span>}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </section>
          </div>

          {/* 行3：活跃度 + 技能加载 + 工具调用 + 命令执行（1:1 四卡） */}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2 min-[1500px]:grid-cols-4">
            <section className="ctx-card p-3" data-testid="insights-heatmap">
              <div className="mb-1.5 flex items-center gap-1.5">
                <SectionTitle>{t("contextinsights.heatmap")}</SectionTitle>
                <span className="ml-auto flex items-center gap-1 text-[10px]">
                  <button type="button" aria-pressed={heatMode === "sessions"} onClick={() => setHeatMode("sessions")} className={`cursor-pointer rounded border-0 px-1 py-px ${heatMode === "sessions" ? "bg-accent/15 text-accent" : "text-fg-dim"}`}>{t("contextinsights.heatSessions")}</button>
                  <button type="button" aria-pressed={heatMode === "tokens"} onClick={() => setHeatMode("tokens")} className={`cursor-pointer rounded border-0 px-1 py-px ${heatMode === "tokens" ? "bg-accent/15 text-accent" : "text-fg-dim"}`}>{t("contextinsights.heatTokens")}</button>
                </span>
              </div>
              <Heatmap days={data.days} generatedAt={data.generatedAt} onPick={(d) => setDayFilter((cur) => (cur === d ? null : d))} picked={dayFilter} mode={heatMode} />
              <div className="mt-1.5 flex items-center justify-end gap-1 text-[9px] text-fg-faint">
                {t("contextinsights.heatLess")}
                {HEAT_LEVELS.map((c, lv) => <span key={lv} className="inline-block size-[9px] rounded-[2px]" style={{ background: c }} />)}
                {t("contextinsights.heatMore")}
              </div>
            </section>
            <section className="ctx-card p-3">
              <TallyList kind="skills" title={t("contextinsights.tallySkills")} items={data.skills} color="var(--md-sys-color-secondary, var(--color-accent))" onPick={setTallyFilter} picked={tallyFilter} unit={t("contextinsights.unitLoads")} />
            </section>
            <section className="ctx-card p-3">
              <TallyList kind="tools" title={t("contextinsights.tallyTools")} items={data.tools} color="var(--md-sys-color-primary, var(--color-primary))" onPick={setTallyFilter} picked={tallyFilter} unit={t("contextinsights.unitCalls")} />
            </section>
            <section className="ctx-card p-3">
              <TallyList kind="programs" title={t("contextinsights.tallyPrograms")} items={data.programs} color="var(--md-sys-color-tertiary, var(--color-accent))" onPick={setTallyFilter} picked={tallyFilter} unit={t("contextinsights.unitRuns")} />
            </section>
          </div>

          {/* 行4：会话区（搜索+排序+工作区分组 chips + 卡片网格） */}
          <section className="ctx-card p-3">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <SectionTitle>
                {t("contextinsights.sessions")} · {fmtCount(sessions.length)}
              </SectionTitle>
              <input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder={t("contextinsights.sessionSearch")}
                data-testid="insights-session-search"
                className="ml-2 w-44 rounded border border-border-soft bg-bg-soft px-2 py-0.5 text-[10.5px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
              />
              {dayFilter && (
                <button type="button" onClick={() => setDayFilter(null)} data-testid="insights-dayfilter" className="rounded border-0 bg-accent/15 px-1.5 py-0.5 text-[11px] text-accent cursor-pointer">
                  {t("contextinsights.dayFilter", { date: dayFilter })}
                </button>
              )}
              {tallyFilter && (
                <button type="button" onClick={() => setTallyFilter(null)} data-testid="insights-tallyfilter" className="rounded border-0 bg-accent/15 px-1.5 py-0.5 text-[11px] text-accent cursor-pointer">
                  {t("contextinsights.tallyFilter", { name: tallyFilter.name })}
                </button>
              )}
              <div className="ml-auto flex items-center gap-1 text-[10.5px]">
                {(["recent", "tokens", "context"] as const).map((k) => (
                  <button key={k} type="button" aria-pressed={sort === k} onClick={() => setSort(k)} className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${sort === k ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}>
                    {t(k === "recent" ? "contextinsights.sortRecent" : k === "tokens" ? "contextinsights.sortTokens" : "contextinsights.sortContext")}
                  </button>
                ))}
              </div>
            </div>
            <div className="mb-2 flex flex-wrap items-center gap-1">
              <button type="button" aria-pressed={group === "all"} onClick={() => setGroup("all")} data-testid="insights-group-all" className={`cursor-pointer rounded-full border-0 px-2 py-0.5 text-[10.5px] ${group === "all" ? "bg-accent/15 text-accent" : "bg-bg-soft text-fg-dim hover:text-fg"}`}>
                {t("contextinsights.groupAll")} {data.sessions.length}
              </button>
              {groupChips.map(([g, n]) => (
                <button key={g} type="button" aria-pressed={group === g} onClick={() => setGroup(g)} className={`cursor-pointer rounded-full border-0 px-2 py-0.5 text-[10.5px] ${group === g ? "bg-accent/15 text-accent" : "bg-bg-soft text-fg-dim hover:text-fg"}`}>
                  {g} {n}
                </button>
              ))}
              {!!data.truncated && (
                <span className="ml-auto text-[11px] text-fg-dim">{t("contextinsights.truncated", { count: data.truncated })}</span>
              )}
            </div>
            {sessions.length === 0 ? (
              <p className="m-0 py-6 text-center text-[12px] text-fg-dim">{t("contextinsights.sessionsEmpty")}</p>
            ) : (
              <ul className="m-0 grid list-none grid-cols-1 gap-2 p-0 md:grid-cols-2 min-[1500px]:grid-cols-3" data-testid="insights-sessions">
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

function saveRange(r: InsightsRangeKey) {
  void import("../../lib/contextPrefs").then((m) => m.saveContextPref("insightsRange", r));
}

function Kpi({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-md border border-border-soft bg-panel-2 px-2.5 py-2" title={sub}>
      <div className="text-[10.5px] text-fg-dim">{label}</div>
      <div className="font-mono text-[17px] font-semibold leading-tight text-fg" data-testid="insights-kpi-value">{value}</div>
      {sub && <div className="truncate text-[9.5px] text-fg-faint">{sub}</div>}
    </div>
  );
}

function CardHeadRow({ title, extra }: { title: string; extra?: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-[12.5px] font-semibold text-fg">{title}</span>
      {extra && <span className="font-mono text-[10.5px] text-fg-dim">{extra}</span>}
    </div>
  );
}

function Dot({ color }: { color: string }) {
  return <span aria-hidden className="h-2 w-2 shrink-0 rounded-sm" style={{ background: color }} />;
}

function TokenRow({ label, value, total, color }: { label: string; value: number; total: number; color: string }) {
  const pct = total > 0 ? Math.round((value / total) * 1000) / 10 : 0;
  return (
    <li className="flex items-center gap-1.5">
      <Dot color={color} />
      <span className="min-w-0 shrink truncate text-fg-dim" title={label}>{label}</span>
      <span className="ml-auto shrink-0 text-right font-mono tabular-nums text-fg-faint">{pct.toFixed(1)}%</span>
      <span className="w-14 shrink-0 text-right font-mono tabular-nums text-fg">{value > 0 ? `=${fmtTokens(value)}` : "—"}</span>
    </li>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="m-0 text-[12.5px] font-medium text-fg">{children}</h3>;
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
    out.push(byDate.get(localDateKey(d)) ?? { date: localDateKey(d), tokens: 0, requests: 0 });
  }
  return out;
}

// DailyChart 近 7 天成对柱（Token 主色/费用次色）+ 双 Y 轴刻度 + 图例
// （1:1 上游：左轴 tokens、右轴费用、底部图例行）。
function DailyChart({ days }: { days: InsightsDay[] }) {
  const t = useT();
  const maxTokens = Math.max(1, ...days.map((d) => d.tokens));
  const maxCost = Math.max(0.01, ...days.map((d) => d.cost ?? 0));
  const W = 560;
  const H = 150;
  const padL = 44;
  const padR = 44;
  const padT = 10;
  const padB = 30;
  const plotH = H - padT - padB;
  const bw = (W - padL - padR) / days.length;
  const yTicks = [1, 0.5, 0].map((f) => ({ y: padT + plotH - f * plotH, label: fmtTokens(Math.round(maxTokens * f)) }));
  const cTicks = [1, 0.5, 0].map((f) => ({ y: padT + plotH - f * plotH, label: fmtCost(maxCost * f) }));
  return (
    <div>
      <svg width="100%" viewBox={`0 0 ${W} ${H}`} role="img" aria-label="daily usage bars">
        {yTicks.map((tk, i) => (
          <g key={`y${i}`}>
            <line x1={padL} y1={tk.y} x2={W - padR} y2={tk.y} stroke="var(--border-soft)" strokeWidth={0.5} />
            <text x={padL - 4} y={tk.y + 3} fontSize={8.5} fill="var(--fg-faint)" textAnchor="end">{tk.label}</text>
            <text x={W - padR + 4} y={tk.y + 3} fontSize={8.5} fill="var(--fg-faint)">{cTicks[i].label}</text>
          </g>
        ))}
        {days.map((d, i) => {
          const th = Math.round((d.tokens / maxTokens) * plotH);
          const ch = Math.round(((d.cost ?? 0) / maxCost) * plotH);
          const x = padL + i * bw;
          return (
            <g key={d.date}>
              <title>{`${d.date} · ${fmtTokens(d.tokens)} tok · ${fmtCost(d.cost)}`}</title>
              <rect x={x + bw * 0.18} y={padT + plotH - th} width={bw * 0.3} height={Math.max(th, 0)} rx={1.5} fill="var(--md-sys-color-primary, var(--color-primary))" opacity={0.85} />
              <rect x={x + bw * 0.52} y={padT + plotH - ch} width={bw * 0.3} height={Math.max(ch, 0)} rx={1.5} fill="var(--md-sys-color-success, #22c55e)" opacity={0.8} />
              <text x={x + bw / 2} y={H - 16} textAnchor="middle" fontSize={8.5} className="fill-current text-fg-dim">
                {d.date.slice(5)}
              </text>
            </g>
          );
        })}
      </svg>
      <div className="flex items-center justify-center gap-3 text-[9.5px] text-fg-dim">
        <span className="inline-flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm" style={{ background: "var(--md-sys-color-primary, var(--color-primary))" }} />{t("contextinsights.dailyTokens")}</span>
        <span className="inline-flex items-center gap-1"><span className="inline-block h-2 w-2 rounded-sm" style={{ background: "var(--md-sys-color-success, #22c55e)" }} />{t("contextinsights.dailyCost")}</span>
      </div>
    </div>
  );
}

// Heatmap 8 周 × 7 天网格（列=周；强度按 heatMode；今日框标记）。
function Heatmap({ days, generatedAt, onPick, picked, mode }: {
  days: InsightsDay[];
  generatedAt: number;
  onPick: (date: string | null) => void;
  picked: string | null;
  mode: "sessions" | "tokens";
}) {
  const byDate = new Map(days.map((d) => [d.date, d]));
  const max = Math.max(1, ...days.map((d) => (mode === "tokens" ? d.tokens : d.sessions ?? 0)));
  const end = new Date(generatedAt * 1000);
  end.setHours(0, 0, 0, 0);
  end.setDate(end.getDate() + (6 - end.getDay()));
  const cells: { date: string | null; day?: InsightsDay }[] = [];
  for (let i = 55; i >= 0; i--) {
    const d = new Date(end);
    d.setDate(d.getDate() - i);
    const key = localDateKey(d);
    cells.push({ date: key, day: byDate.get(key) });
  }
  const weeks: { date: string | null; day?: InsightsDay }[][] = [];
  for (let w = 0; w < 8; w++) weeks.push(cells.slice(w * 7, w * 7 + 7));
  const level = (v: number): number => {
    if (v <= 0) return 0;
    const f = v / max;
    if (f > 0.66) return 4;
    if (f > 0.33) return 3;
    if (f > 0.08) return 2;
    return 1;
  };
  const todayKey = localDateKey(new Date(generatedAt * 1000));
  return (
    <div className="flex gap-[3px]" data-testid="insights-heatmap-grid">
      {weeks.map((week, wi) => (
        <div key={wi} className="flex flex-col gap-[3px]">
          {week.map((cell, ri) => {
            const v = mode === "tokens" ? (cell.day?.tokens ?? 0) : (cell.day?.sessions ?? 0);
            const lv = cell.day ? level(v) : 0;
            const isPicked = cell.date && cell.date === picked;
            return (
              <button
                key={cell.date ?? ri}
                type="button"
                disabled={!cell.day}
                onClick={() => onPick(cell.date && cell.day ? cell.date : null)}
                title={cell.day ? `${cell.date} · ${fmtTokens(cell.day.tokens)} tok · ${cell.day.requests} req · ${cell.day.sessions ?? 0} 会话` : cell.date ?? ""}
                aria-label={cell.date ?? ""}
                data-picked={isPicked || undefined}
                className={`size-[13px] rounded-[2px] border-0 p-0 ${cell.day ? "cursor-pointer" : "cursor-default"} ${isPicked ? "outline outline-1 outline-offset-0" : ""}`}
                style={{
                  background: HEAT_LEVELS[lv],
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

// TallyList 统计列表（1:1 上游：头部「N 个·M 单位」+搜索+列表行
// 彩色点+名称+×N+相对时间；chip 点击过滤会话保留）。
function TallyList({ kind, title, items, color, onPick, picked, unit }: {
  kind: "skills" | "tools" | "programs";
  title: string;
  items: InsightsTally[];
  color: string;
  onPick?: (f: { kind: "skills" | "tools" | "programs"; name: string } | null) => void;
  picked?: { kind: "skills" | "tools" | "programs"; name: string } | null;
  unit: string;
}) {
  const t = useT();
  const [q, setQ] = useState("");
  const shown = items.filter((x) => !q.trim() || x.name.toLowerCase().includes(q.trim().toLowerCase()));
  const total = items.reduce((s, x) => s + x.count, 0);
  return (
    <div data-testid="insights-tally">
      <div className="mb-1 flex flex-wrap items-center gap-1.5">
        <span className="text-[12.5px] font-semibold text-fg">{title}</span>
        <span className="text-[10px] text-fg-dim">{t("contextinsights.tallyHead", { n: items.length, total, unit })}</span>
      </div>
      {items.length > 3 && (
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t("contextinsights.tallySearch")}
          className="mb-1 h-5 w-full rounded border border-border-soft bg-bg-soft px-1.5 text-[10px] text-fg placeholder:text-fg-faint focus:outline-none"
        />
      )}
      {shown.length === 0 ? (
        <p className="m-0 py-2 text-[11px] text-fg-dim">{t("contextinsights.tallyEmpty")}</p>
      ) : (
        <ul className="m-0 max-h-40 list-none overflow-y-auto p-0 text-[10.5px]">
          {shown.map((x) => {
            const active = picked?.kind === kind && picked.name === x.name;
            return (
              <li key={x.name}>
                <button
                  type="button"
                  data-picked={active || undefined}
                  data-testid={`insights-tally-${kind}-${x.name}`}
                  title={x.last ? `${x.name} ×${x.count} · ${relativeTime(x.last * 1000)}` : x.name}
                  className={`flex w-full cursor-pointer items-center gap-1.5 rounded border-0 bg-transparent px-1 py-1 text-left ${active ? "bg-accent/10" : "hover:bg-bg-soft"}`}
                  onClick={() => onPick?.(active ? null : { kind, name: x.name })}
                >
                  <Dot color={color} />
                  <span className="min-w-0 flex-1 truncate font-mono text-fg">{x.name}</span>
                  <span className="shrink-0 font-mono text-fg-dim">×{x.count}</span>
                  <span className="w-14 shrink-0 text-right text-fg-faint">{x.last ? relativeTime(x.last * 1000) : ""}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

// SessionCard 会话卡（1:1 上游：环形构成图 + 轮次/Token 总用量/总预估费用
// + 最后消息行 + workspace 徽标 + 估算/归档角标）。
function SessionCard({ s, onOpen }: { s: InsightsSession; onOpen?: (path: string) => void }) {
  const t = useT();
  const total = catSum(s.composition);
  const clickable = !!onOpen;
  return (
    <li>
      <button
        type="button"
        onClick={clickable ? () => onOpen?.(s.path) : undefined}
        disabled={!clickable}
        data-testid="insights-session-card"
        className={`w-full rounded-md border border-border-soft bg-panel-2 p-2.5 text-left ${clickable ? "cursor-pointer hover:border-accent/40" : "cursor-default"}`}
        title={s.path}
      >
        <div className="flex items-center gap-1.5">
          <span className="min-w-0 flex-1 truncate text-[12px] font-medium text-fg">
            {s.title || s.preview || s.path.split(/[\\/]/).pop() || s.path}
          </span>
          <span className="shrink-0 text-[9.5px] text-fg-faint" title={new Date(s.lastActive * 1000).toLocaleString()}>
            {relativeTime(s.lastActive * 1000)}
          </span>
        </div>
        <div className="mt-0.5 flex items-center gap-1">
          {s.spaceId && <Badge>{s.spaceId}</Badge>}
          {s.estimated && <Badge>{t("contextinsights.badgeEstimated")}</Badge>}
          {s.archived && <Badge>{t("contextinsights.badgeArchived")}</Badge>}
        </div>
        <div className="mt-1.5 flex items-center gap-3">
          <Donut
            size={64}
            stroke={7}
            segments={CONTEXT_CATS.map((c) => ({ value: s.composition[c.key], color: CAT_COLORS[c.key] }))}
            center={total > 0 ? fmtTokens(total) : "0"}
            centerSub=""
          />
          <ul className="m-0 min-w-0 flex-1 list-none space-y-0.5 p-0 text-[10.5px]">
            <li className="flex gap-2">
              <span className="w-16 shrink-0 text-fg-dim">{t("contextinsights.cardTurns")}</span>
              <span className="font-mono tabular-nums text-fg">{s.turns}{t("contextinsights.turnUnit")} {s.steps ?? s.requests}{t("contextinsights.stepUnit")}</span>
            </li>
            <li className="flex gap-2">
              <span className="w-16 shrink-0 text-fg-dim">{t("contextinsights.cardTokens")}</span>
              <span className="font-mono tabular-nums text-fg">{fmtTokens(s.tokens)}</span>
            </li>
            <li className="flex gap-2">
              <span className="w-16 shrink-0 text-fg-dim">{t("contextinsights.cardCost")}</span>
              <span className="font-mono tabular-nums text-fg">{fmtCost(s.cost)}</span>
            </li>
          </ul>
        </div>
        {s.lastMessage && (
          <div className="mt-1 flex gap-2 border-t border-border-soft/60 pt-1 text-[10px]">
            <span className="shrink-0 text-fg-faint">{t("contextinsights.cardLastMsg")}</span>
            <span className="min-w-0 truncate text-fg-dim" title={s.lastMessage}>{s.lastMessage}</span>
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
