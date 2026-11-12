// 编队页（v4.542 刀5，对标 dsh-context Fleet tab）：Agent 家族视图——摘要条、
// 成员卡树（刀11：每 Agent 卡+费用+任务/消息计数，点击联动事件检查）、
// 协作泳道（派发→回报时间线）、协作任务（状态筛选+搜索）、事件检查（派发/
// 回报合并流）、Agent 网络径向图（复用 AgentRadial）。
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
import { AgentRadial } from "./AgentRadial";

type RunStatus = SubagentRunView["status"];

// STATUS_COLOR 与 dsh/子代理既有语义对齐：running 绿（脉冲在 AgentRadial）、
// queued 灰、completed 深绿、failed 红。
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

export function Fleet({ running, sessionPath, onViewContext }: {
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

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4" data-testid="ctx-fleet">
      {/* 摘要条 */}
      <div className="grid grid-cols-4 gap-2" data-testid="fleet-summary">
        <Summary label={t("fleet.kpiAgents")} value={String(1 + subagentRuns.length)} />
        <Summary label={t("fleet.kpiRunning")} value={String(meta.running)} accent={meta.running > 0} />
        <Summary label={t("fleet.kpiTasks")} value={String(subagentRuns.length)} />
        <Summary label={t("fleet.kpiMessages")} value={String((family?.events ?? []).length)} />
      </div>

      {/* 成员卡树（刀11，1:1 上游 Fleet 成员卡）：每 Agent 费用+任务/消息计数 */}
      {(family?.agents?.length ?? 0) > 0 && (
        <MemberCards agents={family!.agents!} runs={subagentRuns} onPick={pickMember} picked={eventAgent} />
      )}

      {/* Agent 网络（谱系径向图，复用实时拓扑组件） */}
      {net && (
        <section className="ctx-card p-3">
          <SectionTitle>{t("fleet.networkTitle")}</SectionTitle>
          <AgentRadial network={net} running={!!running} sessionPath={sessionPath} onViewContext={onViewContext} />
        </section>
      )}

      {/* 协作泳道 */}
      {subagentRuns.length > 0 && <SwimLanes runs={subagentRuns} />}

      {/* 协作任务 + 事件检查 */}
      {subagentRuns.length > 0 && (
        <div className="grid grid-cols-1 gap-3 min-[1100px]:grid-cols-2">
          <TaskPanel runs={subagentRuns} />
          <div ref={eventsRef}>
            <EventInspector family={family} runs={subagentRuns} onViewContext={onViewContext} agent={eventAgent} onAgent={setEventAgent} />
          </div>
        </div>
      )}

      {subagentRuns.length === 0 && meta.status !== "loading" && (
        <p className="py-6 text-center text-[12px] text-fg-dim" data-testid="fleet-empty">{t("fleet.empty")}</p>
      )}
    </div>
  );
}

function Summary({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="rounded-md border border-border-soft bg-panel-2 px-2 py-1.5">
      <div className="text-[10.5px] text-fg-dim">{label}</div>
      <div className={`font-mono text-[15px] font-semibold ${accent ? "text-(color:--color-success)" : "text-fg"}`}>{value}</div>
    </div>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="m-0 mb-1.5 text-[12.5px] font-medium text-fg">{children}</h3>;
}

// ─── 协作泳道 ────────────────────────────────────────────────
// 每个子代理一条道：派发（createdAt）→ 收尾（updatedAt）的 span 画在同一条
// 全局时间轴上；父代理道标出每次派发点。任务编号认同一件事：悬停联动同
// 任务（简化为同 lane 高亮）。
function SwimLanes({ runs }: { runs: SubagentRunView[] }) {
  const t = useT();
  const [hovered, setHovered] = useState<string | null>(null);
  const times = runs.flatMap((r) => [Date.parse(r.createdAt), Date.parse(r.updatedAt || r.createdAt)]).filter((x) => Number.isFinite(x));
  if (times.length === 0) return null;
  const min = Math.min(...times);
  const max = Math.max(...times, min + 1);
  const pos = (ts: number): number => ((ts - min) / (max - min)) * 100;
  // 父代理道：派发点 = 各 run.createdAt（0 偏移开窗给 hover 空间）。
  return (
    <section className="ctx-card p-3" data-testid="fleet-lanes">
      <SectionTitle>{t("fleet.lanesTitle")}</SectionTitle>
      <div className="flex flex-col gap-1">
        <Lane
          name={t("fleet.laneParent")}
          color="var(--md-sys-color-primary, var(--color-primary))"
          marks={runs.map((r) => ({ at: pos(Date.parse(r.createdAt)), title: `${t("fleet.evDispatch")}: ${shortTask(r.task, 60)}` }))}
          span={null}
          dim={hovered !== null}
        />
        {runs.map((r) => {
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
    <div className={`flex items-center gap-2 ${dim ? "opacity-40" : "opacity-100"} transition-opacity`} title={title}>
      <span className="w-40 shrink-0 truncate text-[10.5px] text-fg-dim">{name}</span>
      <div className="relative h-3.5 flex-1 rounded bg-bg-soft">
        {marks.map((m, i) => (
          <span
            key={i}
            title={m.title}
            className="absolute top-0 h-full w-0.5 -translate-x-1/2"
            style={{ left: `${m.at}%`, background: color }}
          />
        ))}
        {span && (
          <span
            className="absolute top-0.5 h-2.5 rounded-full"
            style={{
              left: `${span.start}%`,
              width: `${Math.max(span.end - span.start, 1.5)}%`,
              background: STATUS_COLOR[span.status],
              opacity: 0.8,
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
    <section className="ctx-card p-3" data-testid="fleet-tasks">
      <div className="mb-1.5 flex flex-wrap items-center gap-1">
        <SectionTitle>{t("fleet.tasksTitle")}</SectionTitle>
        <span className="ml-auto flex items-center gap-1">
          {(["all", "running", "queued", "completed", "failed"] as const).map((s) => (
            <button
              key={s}
              type="button"
              aria-pressed={status === s}
              data-testid={`fleet-task-status-${s}`}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 text-[10px] ${status === s ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
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
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t("fleet.taskSearch")}
            className="w-28 rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
          />
        </span>
      </div>
      <ul className="m-0 flex max-h-64 list-none flex-col gap-1.5 overflow-y-auto p-0">
        {filtered.map((r) => (
          <li key={r.ref} className="rounded-md border border-border-soft bg-panel-2 p-1.5 text-[10.5px]">
            <div className="flex items-center gap-1.5">
              <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: STATUS_COLOR[r.status] }} title={r.status} />
              <span className="min-w-0 flex-1 truncate text-fg" title={r.task}>{shortTask(r.task, 40)}</span>
              {r.model && <span className="shrink-0 font-mono text-[9.5px] text-fg-dim">{r.model}</span>}
              <span className="shrink-0 font-mono text-[9.5px] text-fg-dim">×{r.toolCalls}</span>
            </div>
            {r.answer && <div className="mt-0.5 line-clamp-2 text-[10px] text-fg-dim">{r.answer}</div>}
          </li>
        ))}
        {filtered.length === 0 && <li className="py-3 text-center text-[11px] text-fg-dim">{t("fleet.none")}</li>}
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
  const taskByRef = useMemo(() => {
    const m = new Map<string, string>();
    for (const r of runs) m.set(r.ref, r.task);
    return m;
  }, [runs]);
  const events = family?.events ?? [];
  const filtered = events.filter((e) => {
    if (kind !== "all" && e.kind !== kind) return false;
    if (agent !== "all" && e.agent !== agent) return false;
    const query = q.trim().toLowerCase();
    if (query && !(e.text ?? "").toLowerCase().includes(query)) return false;
    return true;
  });
  const agentLabel = (a: string): string =>
    a === "main" ? t("fleet.agentMain") : shortTask(taskByRef.get(a) ?? a, 14);
  const kindColor: Record<FamilyAgentKind, string> = {
    dispatch: "var(--md-sys-color-primary, var(--color-primary))",
    report: "var(--md-sys-color-success, #22c55e)",
    message: "var(--md-sys-color-outline, #9e9e9e)",
  };
  return (
    <section className="ctx-card p-3" data-testid="fleet-events">
      <div className="mb-1.5 flex flex-wrap items-center gap-1">
        <SectionTitle>{t("fleet.eventsTitle")}</SectionTitle>
        {!!family?.truncated && (
          <span className="text-[10px] text-fg-dim">{t("fleet.truncated", { count: family.truncated })}</span>
        )}
        <span className="ml-auto flex flex-wrap items-center gap-1">
          {(["all", ...(family?.refs ?? ["main"])] as const).map((a) => (
            <button
              key={a}
              type="button"
              aria-pressed={agent === a}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 text-[10px] ${agent === a ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => onAgent(a === agent ? "all" : a)}
            >
              {a === "all" ? t("fleet.agentAll") : agentLabel(a)}
            </button>
          ))}
        </span>
      </div>
      <div className="mb-1.5 flex flex-wrap items-center gap-1">
        {(["all", "message", "dispatch", "report"] as const).map((k) => (
          <button
            key={k}
            type="button"
            aria-pressed={kind === k}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 text-[10px] ${kind === k ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            onClick={() => setKind(k)}
          >
            {k === "all" ? t("fleet.filterAll") : k === "message" ? t("fleet.evMessage") : k === "dispatch" ? t("fleet.evDispatch") : t("fleet.evReport")}
          </button>
        ))}
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t("fleet.taskSearch")}
          className="ml-auto w-28 rounded border border-border-soft bg-transparent px-1.5 py-0.5 text-[10px] text-fg placeholder:text-fg-faint focus:outline-none focus:ring-1 focus:ring-accent/40"
        />
      </div>
      <ul className="m-0 flex max-h-64 list-none flex-col overflow-y-auto p-0">
        {filtered.map((e, i) => (
          <li key={`${e.agent}-${e.kind}-${e.seq}-${i}`} className="ctx-row flex items-start gap-1.5 px-1.5 py-1 text-[10.5px]">
            <span
              className="mt-1 h-2 w-2 shrink-0 rounded-sm"
              style={{ background: kindColor[e.kind] }}
              title={e.kind}
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="shrink-0 rounded bg-bg-soft px-1 font-mono text-[9px] text-fg-dim">{agentLabel(e.agent)}</span>
                {e.kind === "message" && e.role && (
                  <span className="shrink-0 text-[9px] text-fg-faint">{e.role === "user" ? t("fleet.roleUser") : t("fleet.roleAssistant")}</span>
                )}
                {e.kind === "dispatch" && <span className="shrink-0 text-[9px] font-medium text-fg-dim">{t("fleet.evDispatch")}</span>}
                {e.kind === "report" && <span className="shrink-0 text-[9px] font-medium" style={{ color: kindColor.report }}>{t("fleet.evReport")}</span>}
                <span className="ml-auto shrink-0 font-mono text-[9.5px] text-fg-dim">{fmtClock(new Date(e.ts * 1000).toISOString())}</span>
              </div>
              {e.text && (
                <div
                  className={`mt-0.5 text-[10px] text-fg-dim ${e.kind === "report" && onViewContext ? "cursor-pointer hover:text-fg hover:underline" : ""}`}
                  title={e.text}
                  onClick={e.kind === "report" && onViewContext ? () => onViewContext(e.ref || e.agent) : undefined}
                >
                  {e.text}
                </div>
              )}
            </div>
          </li>
        ))}
        {filtered.length === 0 && <li className="py-3 text-center text-[11px] text-fg-dim">{t("fleet.none")}</li>}
      </ul>
    </section>
  );
}
