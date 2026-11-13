// 编队页（v4.542 刀5，对标 dsh-context Fleet tab）：Agent 家族视图——摘要条、
// 成员卡树（刀11：每 Agent 卡+费用+任务/消息计数，点击联动事件检查）、
// 协作泳道（派发→回报时间线）、协作任务（状态筛选+搜索）、事件检查（派发/
// 回报合并流）、Agent 网络卡片拓扑、活跃泳道、协作任务与事件检查。
// 数据源：GaeaAgentNetwork（谱系树）+ SubagentRuns（实时运行快照，共享单轮
// 询 store）+ GaeaFamilyEvents（合并流+成员行）。诚实子集：消息级跨
// transcript 合并流、blocked-by 依赖链需新后端面，本版不造（见拍板池）。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useT } from "../../lib/i18n";
import { fmtTokens } from "../../lib/stats";
import { subscribeSubagentRuns, type SubagentRunsMeta } from "../../lib/subagentRunsStore";
import { app } from "../../lib/bridge";
import { subscribeAgentNetwork } from "../../lib/agentNetworkStore";
import type { AgentNetwork } from "../../lib/types";
import type { FamilyAgentKind, FamilyAgentRowView, FamilyEventStreamView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { AgentTopology } from "./AgentTopology";
import { ArrowUpRight, ClipboardList, GitBranch, List, Users } from "../../icons";
import "./fleet-view.css";

type RunStatus = SubagentRunView["status"];

// STATUS_COLOR 与 dsh/子代理既有语义对齐：running 绿、queued 灰、
// completed 深绿、failed 红。
const STATUS_COLOR: Record<RunStatus, string> = {
  running: "var(--md-sys-color-success, #22c55e)",
  queued: "var(--md-sys-color-outline, #9e9e9e)",
  completed: "var(--md-sys-color-primary, var(--color-primary))",
  failed: "var(--md-sys-color-destructive, #ef4444)",
};

function shortTask(task: string, max = 24): string {
  const one = task.split("\n")[0] ?? task;
  return one.length > max ? `${one.slice(0, max)}…` : one;
}

export function Fleet({ sessionPath, onViewContext }: {
  running?: boolean;
  sessionPath?: string;
  onViewContext?: (ref: string) => void;
}) {
  const t = useT();
  const [net, setNet] = useState<AgentNetwork | null>(null);
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [meta, setMeta] = useState<SubagentRunsMeta>({ status: "loading", total: 0, running: 0 });
  // 家族事件流（刀10：main+子代理 transcript 合并流；运行中 4s 轻轮询）。
  const [family, setFamily] = useState<FamilyEventStreamView | null>(null);

  useEffect(
    () => subscribeAgentNetwork((n) => setNet(n), { path: sessionPath }),
    [sessionPath],
  );
  useEffect(
    () => subscribeSubagentRuns(sessionPath ?? "", (r, m) => { setRuns(r); setMeta(m); }),
    [sessionPath],
  );

  const loadFamily = useCallback(() => {
    app.FamilyEvents(sessionPath ?? "")
      .then(setFamily)
      .catch(() => setFamily(null));
  }, [sessionPath]);
  useEffect(() => {
    loadFamily();
  }, [loadFamily]);

  const subagentRuns = useMemo(() => runs.filter((r) => (r.kind ?? "subagent") === "subagent"), [runs]);

  // 刀11：成员卡点击联动事件检查（agent 筛选上提到页级 + 滚动定位）。
  const [eventAgent, setEventAgent] = useState<string>("all");
  const eventsRef = useRef<HTMLDivElement | null>(null);
  const pickMember = useCallback((ref: string) => {
    setEventAgent(ref);
    eventsRef.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, []);
  const familyAgents = family?.agents ?? [];
  const agentCount = Math.max(1 + subagentRuns.length, familyAgents.length);
  const messageCount = family?.events.length ?? 0;

  return (
    <div className="fleet-view" data-testid="ctx-fleet">
      <header className="fleet-hero" data-testid="fleet-hero">
        <div className="fleet-hero-copy">
          <span className="fleet-eyebrow">{t("fleet.heroEyebrow")}</span>
          <div className="fleet-title-row">
            <span className="fleet-beacon" aria-hidden><i /><i /><i /></span>
            <h2>{t("fleet.heroTitle")}</h2>
          </div>
          <p>{t("fleet.heroSubtitle")}</p>
        </div>
        <dl className="fleet-summary" data-testid="fleet-summary">
          <Summary label={t("fleet.kpiAgents")} value={String(agentCount)} />
          <Summary label={t("fleet.kpiRunning")} value={String(meta.running)} tone="running" />
          <Summary label={t("fleet.kpiTasks")} value={String(subagentRuns.length)} />
          <Summary label={t("fleet.kpiMessages")} value={String(messageCount)} tone="messages" />
        </dl>
      </header>

      <div className="fleet-main-grid" data-testid="fleet-main">
        <div className="fleet-network-col">
          {net ? (
            <AgentTopology network={net} runs={subagentRuns} agents={familyAgents} onViewContext={onViewContext} />
          ) : (
            <section className="ctx-card fleet-network-empty">
              <div className="fleet-card-head">
                <span className="fleet-card-title"><Users size={14} aria-hidden />{t("fleet.networkTitle")}</span>
              </div>
              <div className="fleet-empty-state">
                <Users size={20} aria-hidden />
                <span>{t("fleet.empty")}</span>
              </div>
            </section>
          )}
        </div>
        <div className="fleet-side-stack">
          <SwimLanes runs={subagentRuns} family={family} />
          <TaskPanel runs={subagentRuns} />
        </div>
      </div>

      <div ref={eventsRef} className="fleet-events-wrap">
        <EventInspector family={family} runs={subagentRuns} onViewContext={onViewContext} agent={eventAgent} onAgent={setEventAgent} />
      </div>

      {familyAgents.length > 1 && (
        <MemberCards agents={familyAgents} runs={subagentRuns} onPick={pickMember} picked={eventAgent} />
      )}
    </div>
  );
}

function Summary({ label, value, tone }: { label: string; value: string; tone?: "running" | "messages" }) {
  return (
    <div className="fleet-summary-stat">
      <dt>{label}</dt>
      <dd data-tone={tone}>{value}</dd>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="fleet-section-title">{children}</h3>;
}

// ─── 协作泳道 ────────────────────────────────────────────────
// 每个子代理一条道：派发（createdAt）→ 收尾（updatedAt）的 span 画在同一条
// 全局时间轴上；父代理道标出每次派发点。任务编号认同一件事：悬停联动同
// 任务（简化为同 lane 高亮）。
function SwimLanes({ runs, family }: { runs: SubagentRunView[]; family: FamilyEventStreamView | null }) {
  const t = useT();
  const [hovered, setHovered] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [latest, setLatest] = useState(false);
  const needle = query.trim().toLowerCase();
  const visible = runs.filter((run) => needle === "" || `${run.task} ${run.answer ?? ""}`.toLowerCase().includes(needle));
  const ordered = latest ? [...visible].reverse() : visible;
  const times = visible.flatMap((r) => [Date.parse(r.createdAt), Date.parse(r.updatedAt || r.createdAt)]).filter((x) => Number.isFinite(x));
  const min = times.length > 0 ? Math.min(...times) : 0;
  const max = times.length > 0 ? Math.max(...times, min + 1) : 1;
  const pos = (ts: number): number => ((ts - min) / (max - min)) * 100;
  return (
    <section className="ctx-card fleet-compact-card fleet-collab-card" data-testid="fleet-lanes">
      <div className="fleet-card-head">
        <span className="fleet-card-title"><GitBranch size={14} aria-hidden />{t("fleet.lanesTitle")}</span>
        <span className="fleet-card-meta">
          {t("fleet.lanesSummary", { shown: visible.length + 1, total: runs.length + 1, lanes: runs.length + 1 })}
        </span>
      </div>
      <div className="fleet-card-tools">
        <label className="fleet-search">
          <span className="sr-only">{t("fleet.lanesSearch")}</span>
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("fleet.lanesSearch")} />
        </label>
        <div className="fleet-segmented" role="group">
          <button type="button" aria-pressed={!latest} onClick={() => setLatest(false)}>{t("fleet.first")}</button>
          <button type="button" aria-pressed={latest} onClick={() => setLatest(true)}>{t("fleet.latest")}</button>
        </div>
      </div>
      {visible.length > 0 ? (
        <div className="fleet-lane-list">
          <Lane
            name={t("fleet.laneParent")}
            color="var(--fleet-violet)"
            marks={visible.map((r) => ({ at: pos(Date.parse(r.createdAt)), title: `${t("fleet.evDispatch")}: ${shortTask(r.task, 60)}` }))}
            span={null}
            dim={hovered !== null}
          />
          {ordered.map((r) => {
            const start = pos(Date.parse(r.createdAt));
            const end = pos(Date.parse(r.updatedAt || r.createdAt));
            const dim = hovered !== null && hovered !== r.ref;
            return (
              <div key={r.ref} onMouseEnter={() => setHovered(r.ref)} onMouseLeave={() => setHovered(null)}>
                <Lane
                  name={shortTask(r.task)}
                  color={STATUS_COLOR[r.status]}
                  marks={[]}
                  span={{ start, end, status: r.status }}
                  dim={dim}
                  title={`${r.task} · ${t("fleet.laneSpanTip", { from: fmtClock(r.createdAt), to: fmtClock(r.updatedAt || r.createdAt) })}`}
                />
              </div>
            );
          })}
        </div>
      ) : (
        <div className="fleet-empty-state fleet-empty-state-compact">
          <List size={18} aria-hidden />
          <span>{t(runs.length === 0 ? "fleet.noEvents" : "fleet.none")}</span>
        </div>
      )}
      {family?.truncated ? <span className="fleet-card-note">{t("fleet.truncated", { count: family.truncated })}</span> : null}
    </section>
  );
}

function Lane({ name, color, marks, span, dim, title }: {
  name: string;
  color: string;
  marks: { at: number; title: string }[];
  span: { start: number; end: number; status: RunStatus } | null;
  dim?: boolean;
  title?: string;
}) {
  return (
    <div className={`fleet-lane${dim ? " is-dim" : ""}`} title={title}>
      <span className="fleet-lane-name">{name}</span>
      <div className="fleet-lane-track">
        {marks.map((m, i) => (
          <span key={i} title={m.title} className="fleet-lane-mark" style={{ left: `${m.at}%`, background: color }} />
        ))}
        {span && (
          <span
            className="fleet-lane-span"
            style={{
              left: `${span.start}%`,
              width: `${Math.max(span.end - span.start, 1.5)}%`,
              background: STATUS_COLOR[span.status],
            }}
          />
        )}
      </div>
    </div>
  );
}

function fmtClock(iso: string): string {
  const d = new Date(iso);
  return Number.isFinite(d.getTime()) ? d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "—";
}

// ─── 成员卡树（刀11，1:1 上游 Fleet 成员卡）──────────────────
// 每 Agent 一卡：角色徽标（主控/子 Agent）+状态点（join 实时 runs）+
// Token/费用/委派/消息/回报计数（Go FamilyAgentRows 源日志直算口径）。
// 点击卡片=事件检查筛选到该 Agent 并滚动定位（Wave4 chips 联动同款）。
function fmtCost(n: number | undefined): string {
  if (!n || n <= 0) return "0";
  if (n < 0.01) return `<0.01`;
  return n.toFixed(2);
}

function MemberCards({ agents, runs, onPick, picked }: {
  agents: FamilyAgentRowView[];
  runs: SubagentRunView[];
  onPick: (ref: string) => void;
  picked: string;
}) {
  const t = useT();
  const statusByRef = useMemo(() => {
    const m = new Map<string, RunStatus>();
    for (const r of runs) m.set(r.ref, r.status);
    return m;
  }, [runs]);
  return (
    <section className="ctx-card p-3" data-testid="fleet-members">
      <SectionTitle>{t("fleet.membersTitle")}</SectionTitle>
      <div className="grid grid-cols-1 gap-1.5 min-[900px]:grid-cols-2 min-[1400px]:grid-cols-3">
        {agents.map((a) => {
          const isMain = a.ref === "main";
          const status = statusByRef.get(a.ref);
          const dim = picked !== "all" && picked !== a.ref;
          return (
            <button
              key={a.ref}
              type="button"
              data-testid={`fleet-member-${a.ref}`}
              aria-pressed={picked === a.ref}
              onClick={() => onPick(picked === a.ref ? "all" : a.ref)}
              className={`cursor-pointer rounded-md border border-border-soft bg-panel-2 p-2 text-left transition-opacity ${dim ? "opacity-45" : "opacity-100"} hover:border-accent/40`}
            >
              <div className="flex items-center gap-1.5">
                <span
                  className="shrink-0 rounded px-1 py-px text-[9px] font-medium"
                  style={{
                    background: isMain ? "var(--md-sys-color-primary, var(--color-primary))" : "var(--md-sys-color-outline, #9e9e9e)",
                    color: "var(--md-sys-color-surface, #fff)",
                  }}
                >
                  {isMain ? t("fleet.roleLead") : t("fleet.roleSub")}
                </span>
                {status && <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: STATUS_COLOR[status] }} title={status} />}
                <span className="ml-auto truncate text-[10px] text-fg-dim" title={a.ref}>
                  {isMain ? t("fleet.agentMain") : shortTask(a.ref, 18)}
                </span>
              </div>
              <div className="mt-1.5 flex flex-wrap items-center gap-x-2.5 gap-y-0.5 font-mono text-[10px] tabular-nums text-fg-dim">
                <span className="text-fg">{fmtTokens(a.tokens)}</span>
                <span>{t("fleet.cardCost")} {fmtCost(a.cost)}</span>
                <span>{t("fleet.cardDispatches")} {a.dispatches}</span>
                <span>{t("fleet.cardMessages")} {a.messages}</span>
                {!!a.reports && <span>{t("fleet.cardReports")} {a.reports}</span>}
              </div>
            </button>
          );
        })}
      </div>
    </section>
  );
}

// ─── 协作任务 ────────────────────────────────────────────────
function TaskPanel({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  const [status, setStatus] = useState<RunStatus | "all">("all");
  const [q, setQ] = useState("");
  const filtered = runs.filter((r) => {
    if (status !== "all" && r.status !== status) return false;
    const query = q.trim().toLowerCase();
    if (query && !(`${r.task} ${r.answer ?? ""}`.toLowerCase().includes(query))) return false;
    return true;
  });
  return (
    <section className="ctx-card fleet-compact-card fleet-tasks-card" data-testid="fleet-tasks">
      <div className="fleet-card-head">
        <span className="fleet-card-title"><ClipboardList size={14} aria-hidden />{t("fleet.tasksTitle")}</span>
        <span className="fleet-card-meta">{t("fleet.workSummary", { shown: filtered.length, total: runs.length })}</span>
      </div>
      <div className="fleet-card-tools">
        <span className="fleet-segmented">
          {(["all", "running", "queued", "completed", "failed"] as const).map((s) => (
            <button
              key={s}
              type="button"
              aria-pressed={status === s}
              data-testid={`fleet-task-status-${s}`}
              className="fleet-filter-chip"
              onClick={() => setStatus(s)}
            >
              {s === "all"
                ? t("fleet.filterAll")
                : s === "running"
                  ? t("fleet.stRunning")
                  : s === "queued"
                    ? t("fleet.stQueued")
                    : s === "completed"
                      ? t("fleet.stCompleted")
                      : t("fleet.stFailed")}
            </button>
          ))}
        </span>
        <label className="fleet-search">
          <span className="sr-only">{t("fleet.taskSearch")}</span>
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t("fleet.taskSearch")}
          />
        </label>
      </div>
      <ul className="fleet-task-list">
        {filtered.map((r) => (
          <li key={r.ref} className="fleet-task-row">
            <div className="fleet-task-main">
              <span className="fleet-task-dot" style={{ background: STATUS_COLOR[r.status] }} title={r.status} />
              <span className="fleet-task-title" title={r.task}>{shortTask(r.task, 48)}</span>
              {r.model && <span className="fleet-task-model">{r.model}</span>}
              <span className="fleet-task-calls">×{r.toolCalls}</span>
            </div>
            {r.answer && <div className="fleet-task-answer">{r.answer}</div>}
          </li>
        ))}
        {filtered.length === 0 && (
          <li className="fleet-empty-state fleet-empty-state-compact">
            <ClipboardList size={18} aria-hidden />
            <span>{t("fleet.none")}</span>
          </li>
        )}
      </ul>
    </section>
  );
}

// ─── 事件检查 ────────────────────────────────────────────────
// 家族事件流（刀10）：main+子代理 transcript 合并的消息/委派/回报。
// agent chips（refs 单源，子代理 chip 标签优先用其任务摘要）+ kind 筛选 +
// 文本搜索；回报行点击下钻该子代理上下文。
// 刀11：agent 筛选态上提到页级（成员卡点击联动滚动定位）。
function EventInspector({ family, runs, onViewContext, agent, onAgent }: {
  family: FamilyEventStreamView | null;
  runs: SubagentRunView[];
  onViewContext?: (ref: string) => void;
  agent: string;
  onAgent: (a: string) => void;
}) {
  const t = useT();
  const [kind, setKind] = useState<"all" | FamilyAgentKind>("all");
  const [q, setQ] = useState("");
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const taskByRef = useMemo(() => {
    const m = new Map<string, string>();
    for (const r of runs) m.set(r.ref, r.task);
    return m;
  }, [runs]);
  const events = family?.events ?? [];
  const scoped = events.filter((event) => agent === "all" || event.agent === agent);
  const filtered = events.filter((e) => {
    if (kind !== "all" && e.kind !== kind) return false;
    if (agent !== "all" && e.agent !== agent) return false;
    const query = q.trim().toLowerCase();
    if (query && !(e.text ?? "").toLowerCase().includes(query)) return false;
    return true;
  });
  const counts = {
    all: scoped.length,
    message: scoped.filter((event) => event.kind === "message").length,
    dispatch: scoped.filter((event) => event.kind === "dispatch").length,
    report: scoped.filter((event) => event.kind === "report").length,
  };
  const selected = filtered.find((event) => `${event.agent}-${event.kind}-${event.seq}` === selectedKey) ?? null;
  const agentLabel = (a: string): string =>
    a === "main" ? t("fleet.agentMain") : shortTask(taskByRef.get(a) ?? a, 14);
  const kindColor: Record<FamilyAgentKind, string> = {
    dispatch: "var(--md-sys-color-primary, var(--color-primary))",
    report: "var(--md-sys-color-success, #22c55e)",
    message: "var(--md-sys-color-outline, #9e9e9e)",
  };
  return (
    <section className="ctx-card fleet-events-card" data-testid="fleet-events">
      <div className="fleet-card-head">
        <span className="fleet-card-title"><List size={14} aria-hidden />{t("fleet.eventsTitle")}</span>
        {!!family?.truncated && (
          <span className="fleet-card-note">{t("fleet.truncated", { count: family.truncated })}</span>
        )}
        <span className="fleet-card-meta">{t("fleet.eventSummary", { shown: filtered.length, total: events.length })}</span>
      </div>
      <div className="fleet-events-tools">
        <div className="fleet-segmented fleet-event-kinds" role="group">
          {(["all", "message", "dispatch", "report"] as const).map((key) => (
            <button
              key={key}
              type="button"
              aria-pressed={kind === key}
              data-kind={key}
              onClick={() => {
                setKind(key);
                setSelectedKey(null);
              }}
            >
              {key === "all" ? t("fleet.filterAll") : key === "message" ? t("fleet.evMessage") : key === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}
              <b>{counts[key]}</b>
            </button>
          ))}
        </div>
        <label className="fleet-search">
          <span className="sr-only">{t("fleet.eventSearch")}</span>
          <input
            value={q}
            onChange={(event) => {
              setQ(event.target.value);
              setSelectedKey(null);
            }}
            placeholder={t("fleet.eventSearch")}
          />
        </label>
      </div>
      <div className="fleet-agent-filter">
          {(["all", ...(family?.refs ?? ["main"])] as const).map((a) => (
            <button
              key={a}
              type="button"
              aria-pressed={agent === a}
              className="fleet-agent-chip"
              onClick={() => {
                onAgent(a === agent ? "all" : a);
                setSelectedKey(null);
              }}
            >
              {a === "all" ? t("fleet.agentAll") : agentLabel(a)}
            </button>
          ))}
      </div>
      <div className="fleet-events-layout">
        <div className="fleet-event-stream">
          {filtered.length === 0 ? (
            <div className="fleet-empty-state">
              <List size={20} aria-hidden />
              <span>{t(events.length === 0 ? "fleet.noEvents" : "fleet.none")}</span>
            </div>
          ) : (
            <ul className="fleet-event-list">
              {filtered.map((e) => {
                const key = `${e.agent}-${e.kind}-${e.seq}`;
                return (
                  <li key={key}>
                    <button
                      type="button"
                      data-event-key={key}
                      aria-pressed={selectedKey === key}
                      onClick={() => setSelectedKey((current) => current === key ? null : key)}
                      className="fleet-event-row"
                    >
                      <span className="fleet-event-dot" style={{ background: kindColor[e.kind] }} aria-hidden />
                      <span className="fleet-event-agent">{agentLabel(e.agent)}</span>
                      <span className="fleet-event-kind">
                        {e.kind === "message" ? (e.role === "user" ? t("fleet.roleUser") : t("fleet.roleAssistant")) : e.kind === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}
                      </span>
                      <span className="fleet-event-text" title={e.text}>{e.text || "—"}</span>
                      <span className="fleet-event-time">{fmtClock(new Date(e.ts * 1000).toISOString())}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
        <div className="fleet-event-inspector">
          {selected ? (
            <>
              <div className="fleet-event-detail-head">
                <span className="fleet-event-dot" style={{ background: kindColor[selected.kind] }} aria-hidden />
                <span className="fleet-event-agent">{agentLabel(selected.agent)}</span>
                <span className="fleet-event-kind">
                  {selected.kind === "message" ? (selected.role === "user" ? t("fleet.roleUser") : t("fleet.roleAssistant")) : selected.kind === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}
                </span>
                <span className="fleet-event-time">{fmtClock(new Date(selected.ts * 1000).toISOString())}</span>
              </div>
              <p className="fleet-event-detail-text">{selected.text || "—"}</p>
              {selected.kind === "report" && onViewContext && (
                <button type="button" className="fleet-detail-action" onClick={() => onViewContext(selected.ref || selected.agent)}>
                  <ArrowUpRight size={12} aria-hidden />
                  {t("fleet.open")}
                </button>
              )}
            </>
          ) : (
            <div className="fleet-empty-state">
              <List size={20} aria-hidden />
              <span>{t("fleet.selectEvent")}</span>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
